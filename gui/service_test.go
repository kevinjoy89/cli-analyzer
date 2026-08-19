package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
	"cli-analyzer/internal/upgrade"
)

// writeScript 写出一个可探测版本的可执行 shell 脚本到指定路径。
func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// useTempTrash 将内置回收站根指向临时目录，避免测试写入真实回收站
func useTempTrash(t *testing.T) {
	t.Helper()
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := trash.Root
	trash.Root = func() string { return rootDir }
	t.Cleanup(func() { trash.Root = orig })
}

// TestOrphanTrashRejectsNonUnattributed 验证 OrphanTrash 只允许移入当前扫描
// 结果中 Unattributed 列表内的路径：前端（或任何调用方）传入任意路径时，
// 不在列表内的路径必须被拒绝，不能直接移入回收站。
func TestOrphanTrashRejectsNonUnattributed(t *testing.T) {
	useTempTrash(t)
	s := NewScannerService()
	base := t.TempDir()
	// 合法孤儿：出现在扫描结果的 Unattributed 中
	realOrphan := filepath.Join(base, "real-orphan")
	if err := os.MkdirAll(realOrphan, 0o755); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.last = &scanner.ScanResult{
		Unattributed: []scanner.DataDir{{Path: realOrphan, Bytes: 1}},
	}
	s.mu.Unlock()

	// 非法路径：不在 Unattributed 中（模拟调用方传入任意目录）
	evil := filepath.Join(base, "evil-orphan")
	if err := os.MkdirAll(evil, 0o755); err != nil {
		t.Fatal(err)
	}

	out := s.OrphanTrash([]string{realOrphan, evil})
	var r struct {
		Trashed []string `json:"trashed"`
		Errors  []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if len(r.Trashed) != 1 || r.Trashed[0] != realOrphan {
		t.Errorf("trashed = %v, want only %s", r.Trashed, realOrphan)
	}
	if len(r.Errors) == 0 {
		t.Errorf("errors 应为空或包含拒绝信息, got %v", r.Errors)
	}
	// 合法路径已被移入回收站；非法路径必须原封不动
	if _, err := os.Stat(realOrphan); !os.IsNotExist(err) {
		t.Errorf("unattributed dir should have been trashed, stat err=%v", err)
	}
	if _, err := os.Stat(evil); err != nil {
		t.Errorf("non-unattributed dir must NOT be touched, stat err=%v", err)
	}
}

// TestOrphanTrashNoScanRejectsAll 验证无扫描结果（s.last 为空）时全部拒绝，
// 防止空快照误放行任意路径。
func TestOrphanTrashNoScanRejectsAll(t *testing.T) {
	useTempTrash(t)
	s := NewScannerService()
	base := t.TempDir()
	p := filepath.Join(base, "orphan")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	out := s.OrphanTrash([]string{p})
	var r struct {
		Trashed []string `json:"trashed"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Trashed) != 0 {
		t.Errorf("无扫描结果时不应移入任何路径, trashed=%v", r.Trashed)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("无扫描结果时路径必须保持原样, stat err=%v", err)
	}
}

// cacheEnv 隔离扫描缓存根（unix: XDG_CACHE_HOME；Windows: LOCALAPPDATA），
// 避免测试读写真实用户缓存（last-scan.json）。
func cacheEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "localappdata"))
}

// TestCheckToolUpdateNoScan 验证无扫描结果时 CheckToolUpdate 返回错误 JSON
// 而非 panic。
func TestCheckToolUpdateNoScan(t *testing.T) {
	s := NewScannerService() // s.last 为空
	out := s.CheckToolUpdate("ripgrep")
	var r struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if r.Name != "ripgrep" || r.Error == "" {
		t.Errorf("no-scan result = %+v, want name + error", r)
	}
}

// TestCheckToolUpdateToolNotFound 验证工具不在扫描结果时返回未找到错误。
func TestCheckToolUpdateToolNotFound(t *testing.T) {
	cacheEnv(t)
	s := NewScannerService()
	s.mu.Lock()
	s.last = &scanner.ScanResult{Tools: []scanner.Tool{{Name: "ripgrep", Installer: string(scanner.InstBrew)}}}
	s.mu.Unlock()
	out := s.CheckToolUpdate("no-such-tool")
	var r struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if r.Error == "" {
		t.Errorf("tool-not-found result = %+v, want error", r)
	}
}

// TestCheckToolUpdateGoSource 验证 go 来源工具返回 detected=false 且携带
// 命令提示（不执行任何网络查询——go 来源无检测能力）。
func TestCheckToolUpdateGoSource(t *testing.T) {
	s := NewScannerService()
	s.mu.Lock()
	s.last = &scanner.ScanResult{Tools: []scanner.Tool{{Name: "dlv", Installer: string(scanner.InstGo)}}}
	s.mu.Unlock()
	out := s.CheckToolUpdate("dlv")
	var r struct {
		Name      string `json:"name"`
		Detected  bool   `json:"detected"`
		HasUpdate bool   `json:"hasUpdate"`
		Command   string `json:"command"`
		Runnable  bool   `json:"runnable"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if r.Detected || r.HasUpdate {
		t.Errorf("go source should be detected=false hasUpdate=false, got %+v", r)
	}
	if r.Command == "" || r.Runnable {
		t.Errorf("go source should carry hint command, non-runnable, got %+v", r)
	}
	// 检测后应记录 ugTool/ugCommand（供 RunToolUpgrade 代跑）
	if s.ugTool != "dlv" {
		t.Errorf("ugTool = %q, want dlv", s.ugTool)
	}
}

// TestRunToolUpgradeGuard 验证 RunToolUpgrade 拒绝与最近检测工具不一致的
// 请求（防页面守卫丢弃后的陈旧命令被代跑）。
func TestRunToolUpgradeGuard(t *testing.T) {
	s := NewScannerService()
	s.mu.Lock()
	s.ugTool = "ripgrep"
	s.ugCommand = upgrade.Command{Command: "brew upgrade ripgrep", Runnable: true, Bin: "brew", Args: []string{"upgrade", "ripgrep"}}
	s.mu.Unlock()
	// 不同工具 → 拒绝
	if out := s.RunToolUpgrade("git"); out == "" {
		t.Errorf("RunToolUpgrade(git) should be rejected, got %q", out)
	}
	// 非可代跑命令 → 拒绝
	s.mu.Lock()
	s.ugTool = "dlv"
	s.ugCommand = upgrade.Command{Command: "hint", Runnable: false}
	s.mu.Unlock()
	if out := s.RunToolUpgrade("dlv"); out == "" {
		t.Errorf("RunToolUpgrade(non-runnable) should be rejected, got %q", out)
	}
}

// TestGetUpgradeStatusContract 验证 GetUpgradeStatus 返回结构化的
// {running, done, output, error} JSON 契约。
func TestGetUpgradeStatusContract(t *testing.T) {
	s := NewScannerService()
	s.mu.Lock()
	s.ugRunning, s.ugDone, s.ugErr, s.ugOutput = true, false, "", "partial output"
	s.mu.Unlock()
	out := s.GetUpgradeStatus()
	var r struct {
		Running bool   `json:"running"`
		Done    bool   `json:"done"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if !r.Running || r.Done || r.Output != "partial output" {
		t.Errorf("status = %+v, want running=true done=false output set", r)
	}
}

// TestCheckToolUpdateStaleDoesNotOverwrite 验证并发检测时「最近发起」的检测
// 胜出：先发起的 A 检测（慢）后返回不得覆盖后发起的 B 记录（否则 B 的代跑
// 会被误拒）。CheckToolUpdate 先记录 ugTool/ugCommand 再查询，故 A 后返回
// 不覆盖。用 go 来源（无网络查询）模拟：两次调用后 ugTool 应等于最近一次。
func TestCheckToolUpdateStaleDoesNotOverwrite(t *testing.T) {
	s := NewScannerService()
	s.mu.Lock()
	s.last = &scanner.ScanResult{Tools: []scanner.Tool{
		{Name: "dlv", Installer: string(scanner.InstGo)},
		{Name: "kimi", Installer: string(scanner.InstOther)},
	}}
	s.mu.Unlock()
	// 先发起 A 的检测（慢），再发起 B 的检测（快）；B 后发起应胜出
	_ = s.CheckToolUpdate("dlv")
	_ = s.CheckToolUpdate("kimi")
	s.mu.Lock()
	got := s.ugTool
	s.mu.Unlock()
	if got != "kimi" {
		t.Errorf("ugTool = %q, want kimi（最近发起胜出）", got)
	}
}

// TestReprobeUsesRunStartTool 验证升级完成后的重探测目标是「代跑发起时的工具」
// 而非「完成时的 s.ugTool」：代跑 A 期间用户检查 B（CheckToolUpdate 覆盖
// ugTool=B），A 升级完成后必须重探测 A（刚被升级）而非 B（code review #7 修复）。
//
// 种子化 probe 缓存（无子进程，确定性）：A 的真实路径命中缓存返回版本 2.0.0；
// B 的路径不存在（os.Stat 失败 → CachedOrRun 返回空，也不跑子进程）。若回归成用
// s.ugTool（=B）重探测，B 会被命中而 A 保持空——断言 A.Version==2.0.0 即暴露。
func TestReprobeUsesRunStartTool(t *testing.T) {
	cacheEnv(t)
	aBin := filepath.Join(t.TempDir(), "a-tool")
	// A 是真实文件（供 os.Stat 取 size/mtime）
	if err := os.WriteFile(aBin, []byte("#!/bin/sh\necho a-tool 9.9.9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(aBin)
	if err != nil {
		t.Fatal(err)
	}
	// 种子化 probe 缓存：A 命中返回 2.0.0（无需执行子进程）。
	// cacheOnce 在本包首个触发（gui 包其它测试不触碰 probe 缓存），
	// 因此 CachedOrRun 首次调用会读取本种子文件。
	const cacheVersion = 2 // 与 internal/probe.cacheVersion 保持一致
	type probeEntry struct {
		Size    int64  `json:"size"`
		MtimeNs int64  `json:"mtimeNs"`
		Version string `json:"version,omitempty"`
		Ok      bool   `json:"ok"`
	}
	cf := struct {
		V       int                   `json:"v"`
		Entries map[string]probeEntry `json:"entries"`
	}{
		V: cacheVersion,
		Entries: map[string]probeEntry{
			aBin: {Size: st.Size(), MtimeNs: st.ModTime().UnixNano(), Version: "2.0.0", Ok: true},
		},
	}
	seed := filepath.Join(platform.CacheRoot(), "probe-versions.json")
	if err := os.MkdirAll(filepath.Dir(seed), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(cf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seed, b, 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewScannerService()
	s.mu.Lock()
	bBin := filepath.Join(t.TempDir(), "b-tool") // 不存在：即使误探测 B 也只返回空
	s.last = &scanner.ScanResult{Tools: []scanner.Tool{
		{Name: "a-tool", Installer: string(scanner.InstBrew), Binaries: []scanner.Binary{{Name: "a-tool", Path: aBin, Real: aBin, Size: st.Size()}}},
		{Name: "b-tool", Installer: string(scanner.InstBrew), Binaries: []scanner.Binary{{Name: "b-tool", Path: bBin, Real: bBin}}},
	}}
	// 模拟：代跑发起时目标是 A；随后用户检查 B（CheckToolUpdate 覆盖 ugTool=B）
	s.ugTool = "b-tool"
	s.mu.Unlock()

	// 用「发起时的工具名 A」重探测：命中 A（缓存返回 2.0.0），B 保持空
	s.reprobeToolVersion("a-tool")

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tt := range s.last.Tools {
		switch tt.Name {
		case "a-tool":
			// 种子命中返回 2.0.0；若 cacheOnce 已被本进程其它迭代触发（种子未读）
			// 则走真实脚本探测回显 9.9.9——两者都证明命中了 A。
			if tt.Version != "2.0.0" && tt.Version != "9.9.9" {
				t.Errorf("a-tool.Version = %q, want 2.0.0 或 9.9.9（probe 命中 A）", tt.Version)
			}
		case "b-tool":
			if tt.Version != "" {
				t.Errorf("b-tool.Version = %q, want empty（不应重探测 B）", tt.Version)
			}
		}
	}
}
