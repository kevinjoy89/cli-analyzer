// Package probe 探测 CLI 工具的版本/描述：依次尝试 --version / -V / --help，
// 带超时、结果缓存与 Windows 编码处理。失败静默降级，不阻塞调用方。
package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"cli-analyzer/internal/platform"
)

// timeout 是单条探测命令的超时上限（spec：3 秒）。
const timeout = 3 * time.Second

// ProbeVersion 探测单个二进制的版本：按 --version / -V / --help 顺序，
// 取首个成功（0 退出且非空输出）的首行。ok=false 表示失败/超时/无输出。
func ProbeVersion(bin string) (string, bool) {
	for _, args := range [][]string{{"--version"}, {"-V"}, {"--help"}} {
		out, ok, timedOut := runWithTimeout(bin, args, timeout)
		if timedOut {
			// 一条命令挂起说明该工具不适合探测：不再浪费 -V/--help
			return "", false
		}
		if ok {
			if v := extractVersion(out); v != "" {
				return v, true
			}
		}
	}
	return "", false
}

// syncBuf 是并发安全的输出缓冲：子进程 stdout/stderr 由两个复制 goroutine
// 并行写入同一 buffer（Start+Wait 分离后无法用 CombinedOutput 的锁定合并）——
// 管道缓冲在多数场景串行化写入，但无锁 buffer 仍是理论竞态，加锁防御。
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Bytes()
}

func runWithTimeout(bin string, args []string, d time.Duration) (out []byte, ok, timedOut bool) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	platform.HideConsoleWindow(cmd)
	setupProcessGroup(cmd)

	var buf syncBuf
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		// 启动失败（命令不存在/不可执行）按非 0 退出处理
		return nil, false, false
	}
	// 超时后连同进程组一起终止（防止 --version 派生的子进程残留）。
	// 必须在 Start 返回后才读 cmd.Process——Start 之前启动 goroutine 会与
	// exec 内部写 cmd.Process 形成数据竞态（race 检测实锤，超时瞬间触发）。
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			killGroup(cmd)
		case <-done:
		}
	}()

	err := cmd.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, true // 超时
		}
		return nil, false, false // 非 0 退出
	}
	b := buf.Bytes()
	if len(strings.TrimSpace(cleanOutput(b))) == 0 {
		return nil, false, false // 空输出不算成功
	}
	return b, true, false
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// cleanOutput 转码（GBK → UTF-8）、剥离 ANSI 与控制字符。
func cleanOutput(b []byte) string {
	if !utf8.Valid(b) {
		if dec, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), b); err == nil {
			b = dec
		}
	}
	s := ansiRe.ReplaceAllString(string(b), "")
	var sb strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || r >= 0x20 {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// extractVersion 返回输出的首个非空行（trim 后）。
func extractVersion(b []byte) string {
	for _, line := range strings.Split(cleanOutput(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// ---- 结果缓存 ----

type entry struct {
	Size    int64  `json:"size"`
	MtimeNs int64  `json:"mtimeNs"`
	Version string `json:"version,omitempty"`
	Ok      bool   `json:"ok"`
}

var (
	cacheOnce sync.Once
	cacheMu   sync.Mutex
	cache     = map[string]entry{}
)

func cachePath() string { return filepath.Join(platform.CacheRoot(), "probe-versions.json") }

func loadCache() {
	cacheOnce.Do(func() {
		b, err := os.ReadFile(cachePath())
		if err != nil {
			return
		}
		if err := json.Unmarshal(b, &cache); err != nil {
			cache = map[string]entry{}
		}
	})
}

// Save 把缓存写回磁盘（在探测批次完成后调用一次，避免逐条写盘）。
func Save() {
	cacheMu.Lock()
	b, err := json.Marshal(cache)
	cacheMu.Unlock()
	if err != nil {
		return
	}
	p := cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	// 唯一临时文件名：多实例并发 Save 时固定 ".tmp" 会互相覆盖
	tmp, err := os.CreateTemp(filepath.Dir(p), "probe-*.tmp")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return
	}
	_ = os.Rename(tmpName, p)
}

// CachedOrRun 优先命中缓存（键 = real path + size + mtime）；未命中则探测并
// 写回内存缓存。ok=false 表示失败（失败结果同样缓存，二进制不变不重探）。
func CachedOrRun(real string, size int64) (string, bool) {
	st, err := os.Stat(real)
	if err != nil {
		return "", false
	}
	mtime := st.ModTime().UnixNano()
	loadCache()
	cacheMu.Lock()
	e, hit := cache[real]
	cacheMu.Unlock()
	if hit && e.Size == size && e.MtimeNs == mtime {
		return e.Version, e.Ok
	}
	v, ok := ProbeVersion(real)
	cacheMu.Lock()
	cache[real] = entry{Size: size, MtimeNs: mtime, Version: v, Ok: ok}
	cacheMu.Unlock()
	return v, ok
}
