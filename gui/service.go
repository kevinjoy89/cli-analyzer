// Package gui is the thin Wails boundary: it is the only package besides the
// root main that imports Wails. All scanning/cleaning logic lives in
// internal/scanner and internal/cleaner.
package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/history"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/probe"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
	"cli-analyzer/internal/uninstall"
	"cli-analyzer/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the app version shown in the UI footer and About dialog,
// sourced from buildinfo (single source).
var AppVersion = buildinfo.Version

// ScannerService is the Wails binding the frontend calls.
type ScannerService struct {
	ctx            context.Context
	mu             sync.Mutex
	last           *scanner.ScanResult
	check          *updater.CheckResult // 最近一次更新检查结果
	downloadedPath string               // 已下载并通过校验的安装包路径
	downloadCancel context.CancelFunc   // 进行中的下载取消句柄
	dlDownloaded   int64                // 下载进度（字节），前端轮询读取
	dlTotal        int64                // 下载总量（字节）
	// ---- uninstall 状态（前端轮询，参考 update 进度经验） ----
	unTool     string // 当前卸载流程的工具名
	unOfficial uninstall.Official
	unRunning  bool
	unDone     bool
	unErr      string
	unOutput   string
}

func NewScannerService() *ScannerService { return &ScannerService{} }

// Startup loads the last scan from cache so the window renders instantly,
// resolves the UI language from config (before the frontend handshake refines it),
// then kicks off a background (silent) update check when enabled.
func (s *ScannerService) Startup(ctx context.Context) {
	s.ctx = ctx
	// 启动语言解析：配置显式语言或 auto→系统探测（供原生菜单/后端错误使用）；
	// 前端 init() 会以 navigator.language 细化并经 SetLanguage 握手。
	i18n.SetLocale(i18n.Resolve(config.Load().Language))
	s.mu.Lock()
	if res, err := scanner.LoadCache(); err == nil {
		s.last = res
	}
	s.mu.Unlock()
	// 启动扫描由前端驱动（init 渲染缓存后调 Scan）：保证扫描动效与按钮禁用
	// 状态与手动扫描一致，也避免 scan:done 事件早于前端监听注册而丢失。
	// 自动检查更新：异步执行；配置关闭、命中 4h 缓存或网络失败时均静默无提示
	go s.autoCheck()
}

// GetLastResult returns the cached scan result as JSON ("" when none yet).
func (s *ScannerService) GetLastResult() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return ""
	}
	b, _ := json.Marshal(s.last)
	return string(b)
}

// Scan starts a background rescan and emits "scan:done" with the JSON result
// (or an {error} object) when finished.
func (s *ScannerService) Scan() {
	go func() {
		res, err := scanner.Scan(scanner.Options{})
		if err != nil {
			runtime.EventsEmit(s.ctx, "scan:done", map[string]any{"error": err.Error()})
			return
		}
		// 追加历史快照（失败静默，不影响扫描主流程）
		_ = history.Record(res)
		s.mu.Lock()
		s.last = res
		s.mu.Unlock()
		b, _ := json.Marshal(res)
		runtime.EventsEmit(s.ctx, "scan:done", string(b))
		// 健康探测：后台并行填充空版本字段（缓存优先，不阻塞任何流程）
		go s.probeAll()
	}()
}

// probeAll 为版本未知的工具后台探测版本（--version/-V/--help，超时+缓存），
// 完成后用更新后的结果发射 probe:done 事件。缓存命中时秒回；挂起工具按
// 3s 超时中断。失败静默，不产生错误事件。
func (s *ScannerService) probeAll() {
	s.mu.Lock()
	res := s.last
	s.mu.Unlock()
	if res == nil {
		return
	}
	type upd struct {
		idx int
		v   string
	}
	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 8)
		results = make([]upd, len(res.Tools))
	)
	for i := range res.Tools {
		t := &res.Tools[i]
		if t.Version != "" || len(t.Binaries) == 0 {
			continue
		}
		b := t.Binaries[0]
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, real string, size int64) {
			defer wg.Done()
			defer func() { <-sem }()
			if v, ok := probe.CachedOrRun(real, size); ok && v != "" {
				results[i] = upd{i, v}
			}
		}(i, b.Real, b.Size)
	}
	wg.Wait()
	probe.Save()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last != res {
		return // 探测期间已发生新的扫描，旧结果作废
	}
	changed := false
	for _, u := range results {
		if u.v != "" && s.last.Tools[u.idx].Version == "" {
			s.last.Tools[u.idx].Version = u.v
			changed = true
		}
	}
	if !changed {
		return
	}
	b, _ := json.Marshal(s.last)
	runtime.EventsEmit(s.ctx, "probe:done", string(b))
}

// Clean deletes the given SAFE items (by ID) and returns a CleanReport JSON.
// The cleaner layer hard-rejects any non-SAFE item regardless of caller flags.
func (s *ScannerService) Clean(ids []string, dryRun bool) string {
	s.mu.Lock()
	res := s.last
	s.mu.Unlock()
	report := scanner.CleanReport{
		Errors:  []string{"no scan result available"},
		Deleted: []string{},
		Skipped: []string{},
	}
	if res != nil {
		report = cleaner.Clean(res, ids, dryRun)
	}
	b, _ := json.Marshal(report)
	return string(b)
}

// OpenURL opens a URL in the system browser (used for a tool's homepage).
func (s *ScannerService) OpenURL(url string) {
	if s.ctx == nil {
		return
	}
	runtime.BrowserOpenURL(s.ctx, url)
}

// GetVersion returns the app version string (e.g. "0.1.1").
func (s *ScannerService) GetVersion() string { return AppVersion }

// SetTheme 让 Windows 标题栏与原生菜单栏随前端主题切换（内部调 DWM 沉浸式
// 暗色模式）；macOS/Linux 由系统与前端 CSS 处理，此处直接跳过
func (s *ScannerService) SetTheme(mode string) {
	if s.ctx == nil || goruntime.GOOS != "windows" {
		return
	}
	switch mode {
	case "light":
		runtime.WindowSetLightTheme(s.ctx)
	case "dark":
		runtime.WindowSetDarkTheme(s.ctx)
	default: // system
		runtime.WindowSetSystemDefaultTheme(s.ctx)
	}
}

// TrashInfo 返回内置回收站的占用统计 JSON（项数 / 总占用 / 最早到期时间）
func (s *ScannerService) TrashInfo() string {
	b, _ := json.Marshal(trash.Info())
	return string(b)
}

// TrashList 返回回收站项目列表 JSON（按移入时间倒序）
func (s *ScannerService) TrashList() string {
	items, err := trash.List()
	if err != nil {
		b, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// Restore 恢复回收站中指定项目，返回 {"restored": 实际路径} 或 {"error": ...}
func (s *ScannerService) Restore(id string) string {
	restored, err := trash.Restore(id)
	if err != nil {
		b, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(map[string]any{"restored": restored})
	return string(b)
}

// PurgeNow 立即彻底删除回收站中的指定项目（跳过系统回收站）
func (s *ScannerService) PurgeNow(ids []string) string {
	deleted, errs := trash.Purge(ids)
	b, _ := json.Marshal(map[string]any{"deleted": deleted, "errors": errs})
	return string(b)
}

// OrphanTrash 将孤儿数据目录移入内置回收站（USER 级，可恢复，无永久删除）。
func (s *ScannerService) OrphanTrash(paths []string) string {
	sizer := &disk.Sizer{Skip: map[string]bool{trash.Root(): true}}
	trashed := []string{}
	errs := []string{}
	for _, p := range paths {
		if err := trash.Trash(p, trash.Item{Tool: "orphan", Kind: "data", Bytes: sizer.WalkSize(p)}); err != nil {
			errs = append(errs, p+": "+err.Error())
			continue
		}
		trashed = append(trashed, p)
	}
	s.Scan() // 后台重扫，刷新孤儿列表
	b, _ := json.Marshal(map[string]any{"trashed": trashed, "errors": errs})
	return string(b)
}

// GetTrashConfig 返回当前回收站配置 JSON
func (s *ScannerService) GetTrashConfig() string {
	b, _ := json.Marshal(config.Load().Trash)
	return string(b)
}

// SetTrashConfig 保存回收站配置；成功返回 ""，失败返回错误信息
func (s *ScannerService) SetTrashConfig(cfgJSON string) string {
	var tc config.TrashConfig
	if err := json.Unmarshal([]byte(cfgJSON), &tc); err != nil {
		return err.Error()
	}
	c := config.Load()
	c.Trash = tc
	if err := config.Save(c); err != nil {
		return err.Error()
	}
	return ""
}

// GetTrends 返回最近 days 天内的占用趋势 JSON（points + topGrowers）
func (s *ScannerService) GetTrends(days int) string {
	tr, err := history.Trends(days)
	if err != nil {
		b, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(tr)
	return string(b)
}

// GetTopGrowers 返回最近 30 天 cleanable 增量 Top 5
func (s *ScannerService) GetTopGrowers() string {
	tr, err := history.Trends(30)
	if err != nil {
		b, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(b)
	}
	b, _ := json.Marshal(tr.TopGrowers)
	return string(b)
}

// GetReminderConfig 返回 cleanable 阈值提醒配置 JSON
func (s *ScannerService) GetReminderConfig() string {
	b, _ := json.Marshal(config.Load().Reminder)
	return string(b)
}

// SetReminderConfig 保存阈值提醒配置；成功返回 ""，失败返回错误信息
func (s *ScannerService) SetReminderConfig(cfgJSON string) string {
	var rc config.ReminderConfig
	if err := json.Unmarshal([]byte(cfgJSON), &rc); err != nil {
		return err.Error()
	}
	c := config.Load()
	c.Reminder = rc
	if err := config.Save(c); err != nil {
		return err.Error()
	}
	return ""
}

// ---- update 检查/下载/安装 ----

// autoCheck 是启动时的后台自动检查：失败或无需更新时静默；
// 发现更新时向前端推送 "update:available" 事件。
func (s *ScannerService) autoCheck() {
	if !config.Load().Update.CheckUpdatesEnabled() {
		return
	}
	res := updater.CheckForUpdates(context.Background(), false)
	s.mu.Lock()
	s.check = &res
	s.mu.Unlock()
	if res.Error != "" || !res.UpdateAvailable {
		return // 静默：网络失败不打扰，已是最新也不打扰
	}
	b, _ := json.Marshal(res)
	runtime.EventsEmit(s.ctx, "update:available", string(b))
}

// CheckForUpdates 手动检查更新（不受限流缓存限制），返回 CheckResult JSON；
// 同时推送 "update:check-done" 事件（与返回值为同一结果）。
func (s *ScannerService) CheckForUpdates() string {
	res := updater.CheckForUpdates(context.Background(), true)
	s.mu.Lock()
	s.check = &res
	s.mu.Unlock()
	b, _ := json.Marshal(res)
	runtime.EventsEmit(s.ctx, "update:check-done", string(b))
	return string(b)
}

// GetUpdateStatus 返回最近一次（自动/手动）检查的结果 JSON；未检查过返回 ""。
// 前端 init 完成后主动拉取：启动自动检查可能命中缓存而瞬时完成，事件早于
// 前端监听器注册而丢失（"打开软件不弹更新提示"的根因之一）。
func (s *ScannerService) GetUpdateStatus() string {
	s.mu.Lock()
	res := s.check
	s.mu.Unlock()
	if res == nil {
		return ""
	}
	b, _ := json.Marshal(res)
	return string(b)
}

// DownloadUpdate 开始下载最新版安装包（异步）。下载进度经 "update:progress"
// 推送；完成后自动校验 SHA256，成功推 "update:downloaded"，校验失败推
// "update:verify-failed"，取消推 "update:cancelled"，其他错误推 "update:error"。
func (s *ScannerService) DownloadUpdate() string {
	s.mu.Lock()
	// 检查与占位在锁内原子完成，杜绝双启动竞态（两次调用各起一个下载 goroutine
	// 会互写进度，表现为进度条前进后回退）
	if s.downloadCancel != nil {
		s.mu.Unlock()
		return i18n.T("upd.downloadInProgress")
	}
	res := s.check
	if res == nil || !res.UpdateAvailable || res.DownloadURL == "" {
		s.mu.Unlock()
		return i18n.T("upd.nothingToDownload")
	}
	s.dlDownloaded = 0
	s.dlTotal = 0
	ctx, cancel := context.WithCancel(context.Background())
	s.downloadCancel = cancel
	s.mu.Unlock()
	go s.runDownload(ctx, *res, cancel)
	return ""
}

// progressFn 更新下载进度状态（前端轮询 GetDownloadProgress 读取）。
// 单调守卫：忽略回退值（w 小于当前已读），即使出现双写也不会让进度倒退。
func (s *ScannerService) progressFn() updater.ProgressFunc {
	return func(w, t int64) {
		s.mu.Lock()
		if w > s.dlDownloaded || t != s.dlTotal {
			s.dlDownloaded = w
			s.dlTotal = t
		}
		s.mu.Unlock()
	}
}

// GetDownloadProgress 返回进行中下载的进度 JSON {downloaded, total}；
// 无进行中下载时返回 ""。前端在下载期间每 ~200ms 轮询一次。
func (s *ScannerService) GetDownloadProgress() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.downloadCancel == nil {
		return ""
	}
	b, _ := json.Marshal(map[string]any{"downloaded": s.dlDownloaded, "total": s.dlTotal})
	return string(b)
}

// runDownload 执行下载 + 校验，并通过事件汇报各阶段结果。
func (s *ScannerService) runDownload(ctx context.Context, res updater.CheckResult, cancel context.CancelFunc) {
	defer func() {
		cancel()
		s.mu.Lock()
		s.downloadCancel = nil
		s.mu.Unlock()
	}()
	// 重新拉取 release 以拿到 checksums.txt 附件（检查阶段只缓存了摘要）
	release, err := updater.LatestRelease(ctx, nil)
	if err != nil {
		runtime.EventsEmit(s.ctx, "update:error", map[string]any{"error": err.Error()})
		return
	}
	asset, err := updater.SelectAsset(release, goruntime.GOOS, goruntime.GOARCH, res.InstallSource)
	if err != nil {
		runtime.EventsEmit(s.ctx, "update:error", map[string]any{"error": err.Error(), "releaseURL": res.ReleaseURL})
		return
	}
	path, err := updater.DownloadInstaller(ctx, nil, release, asset, s.progressFn())
	if err != nil {
		if path != "" {
			// 下载成功但校验失败（checksums 缺失或哈希不匹配）：安全优先，不给安装入口
			runtime.EventsEmit(s.ctx, "update:verify-failed", map[string]any{
				"error": err.Error(), "releaseURL": res.ReleaseURL, "path": path,
			})
			return
		}
		if errors.Is(err, context.Canceled) {
			runtime.EventsEmit(s.ctx, "update:cancelled", map[string]any{})
		} else {
			runtime.EventsEmit(s.ctx, "update:error", map[string]any{"error": err.Error()})
		}
		return
	}
	s.mu.Lock()
	s.downloadedPath = path
	s.mu.Unlock()
	exe, _ := os.Executable()
	runtime.EventsEmit(s.ctx, "update:downloaded", map[string]any{
		"path": path, "releaseURL": res.ReleaseURL,
		"executablePath": exe, "installSource": res.InstallSource,
	})
}

// CancelDownload 取消进行中的下载；无下载时返回提示。
func (s *ScannerService) CancelDownload() string {
	s.mu.Lock()
	cancel := s.downloadCancel
	s.mu.Unlock()
	if cancel == nil {
		return i18n.T("upd.noDownloadInProgress")
	}
	cancel()
	return ""
}

// InstallUpdate 打开已下载并通过校验的安装包，随后退出应用（design D7：
// 先打开后退出，打开失败则保留应用并返回错误信息）。
func (s *ScannerService) InstallUpdate() string {
	s.mu.Lock()
	path := s.downloadedPath
	s.mu.Unlock()
	if path == "" {
		return i18n.T("upd.noInstallerDownloaded")
	}
	if err := updater.OpenInstaller(path); err != nil {
		return err.Error()
	}
	runtime.Quit(s.ctx)
	return ""
}

// GetUpdateConfig 返回更新相关配置 JSON（自动检查开关、忽略版本等）。
func (s *ScannerService) GetUpdateConfig() string {
	b, _ := json.Marshal(config.Load().Update)
	return string(b)
}

// SetUpdateConfig 保存更新配置；成功返回 ""，失败返回错误信息。
func (s *ScannerService) SetUpdateConfig(cfgJSON string) string {
	var uc config.UpdateConfig
	if err := json.Unmarshal([]byte(cfgJSON), &uc); err != nil {
		return err.Error()
	}
	c := config.Load()
	c.Update = uc
	if err := config.Save(c); err != nil {
		return err.Error()
	}
	return ""
}

// IgnoreVersion 持久化“忽略该版本”：在出现比它更新的版本前不再提示。
func (s *ScannerService) IgnoreVersion(version string) string {
	c := config.Load()
	c.Update.IgnoredVersion = version
	if err := config.Save(c); err != nil {
		return err.Error()
	}
	return ""
}

// ---- i18n ----

// GetLanguage 返回配置中显式设置的语言（"auto" 表示跟随系统）。
func (s *ScannerService) GetLanguage() string {
	return config.Load().Language
}

// SetLanguage 同步运行时的生效语言（前端以 navigator.language 细化后调用）。
// 只改内存中的 i18n 状态，不持久化（持久化由首选项保存负责）。
func (s *ScannerService) SetLanguage(locale string) string {
	if i18n.IsSupported(locale) {
		i18n.SetLocale(locale)
		return ""
	}
	return "unsupported locale: " + locale
}

// SetLanguagePreference 持久化语言偏好（auto | zh-CN | zh-TW | en）；
// 成功返回 ""，失败返回错误信息。
func (s *ScannerService) SetLanguagePreference(locale string) string {
	c := config.Load()
	c.Language = locale
	if err := config.Save(c); err != nil {
		return err.Error()
	}
	// 同步运行时语言：显式语言直接生效，auto 按系统探测
	i18n.SetLocale(i18n.Resolve(locale))
	return ""
}

// GetTranslations 返回指定语言的完整翻译字典 JSON（前端 t() 的数据来源）；
// 非法语言回退 zh-CN。
func (s *ScannerService) GetTranslations(locale string) string {
	if !i18n.IsSupported(locale) {
		locale = "zh-CN"
	}
	b, _ := json.Marshal(map[string]any{
		"locale": locale,
		"dict":   translationsFor(locale),
	})
	return string(b)
}

// translationsFor 是 i18n.Dict 的薄封装（命名对齐 GetTranslations 语义）。
func translationsFor(locale string) map[string]string {
	return i18n.Dict(locale)
}

// ---- uninstall ----

// UninstallStart 返回卸载起始信息（标准卸载命令、黑名单拦截、占用摘要），
// 并记录当前卸载流程的工具。返回 JSON：{tool, installer, blocked,
// blockedReason, officialCommand, runnable, footprint, userBytes} 或 {error}。
func (s *ScannerService) UninstallStart(tool string) string {
	if uninstall.IsBlocked(tool) {
		b, _ := json.Marshal(map[string]any{"tool": tool, "blocked": true, "blockedReason": i18n.T("un.guiBlocked")})
		return string(b)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var t *scanner.Tool
	if s.last != nil {
		for i := range s.last.Tools {
			if s.last.Tools[i].Name == tool {
				t = &s.last.Tools[i]
				break
			}
		}
	}
	if t == nil {
		b, _ := json.Marshal(map[string]any{"tool": tool, "error": i18n.T("un.toolNotFound")})
		return string(b)
	}
	bin := ""
	if len(t.Binaries) > 0 {
		bin = t.Binaries[0].Name
	}
	// 缓存陈旧：工具曾有过二进制但现在都不在了（已卸载）→ 提示并触发重扫，
	// 避免对已消失的工具跑注定失败的卸载命令。
	if len(t.Binaries) > 0 {
		gone := true
		for _, b := range t.Binaries {
			if _, err := os.Stat(b.Real); err == nil {
				gone = false
				break
			}
		}
		if gone {
			b, _ := json.Marshal(map[string]any{"tool": t.Name, "stale": true, "error": i18n.T("un.toolGone")})
			return string(b)
		}
	}
	off := uninstall.OfficialCommand(scanner.Installer(t.Installer), t.Name, bin)
	s.unTool = t.Name
	s.unOfficial = off
	s.unRunning, s.unDone, s.unErr, s.unOutput = false, false, "", ""
	b, _ := json.Marshal(map[string]any{
		"tool": t.Name, "installer": t.Installer, "blocked": false,
		"officialCommand": off.Command, "runnable": off.Runnable,
		"footprint": t.Footprint, "userBytes": t.User,
	})
	return string(b)
}

// UninstallBlocked 报告该工具是否命中系统关键工具黑名单（前端据此禁用卸载按钮）。
func (s *ScannerService) UninstallBlocked(tool string) string {
	b, _ := json.Marshal(map[string]any{"tool": tool, "blocked": uninstall.IsBlocked(tool)})
	return string(b)
}

// UninstallRunOfficial 异步代跑标准卸载命令；状态经 GetUninstallStatus 轮询读取。
func (s *ScannerService) UninstallRunOfficial() string {
	s.mu.Lock()
	off := s.unOfficial
	if !off.Runnable {
		s.mu.Unlock()
		return i18n.T("un.notRunnable")
	}
	if s.unRunning {
		s.mu.Unlock()
		return i18n.T("un.alreadyRunning")
	}
	s.unRunning, s.unDone, s.unErr, s.unOutput = true, false, "", ""
	s.mu.Unlock()
	go s.runUninstallOfficial(off)
	return ""
}

func (s *ScannerService) runUninstallOfficial(off uninstall.Official) {
	defer func() {
		s.mu.Lock()
		s.unRunning = false
		s.unDone = true
		s.mu.Unlock()
		runtime.EventsEmit(s.ctx, "uninstall:done", map[string]any{"done": true})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// GUI 启动 PATH 是最小集：先经（增强的）PATH 解析出命令绝对路径，
	// 并给子进程注入完整 PATH（npm 内部的 node 等子进程依赖它）。
	bin := off.Bin
	if resolved, rerr := uninstall.ResolveCommand(off.Bin); rerr == nil {
		bin = resolved
	}
	cmd := exec.CommandContext(ctx, bin, off.Args...)
	platform.HideConsoleWindow(cmd) // Windows: 不闪控制台窗口
	cmd.Env = withPath(os.Environ(), uninstall.AugmentedPathEnv())
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	s.mu.Lock()
	s.unOutput = buf.String()
	uninstalled := err == nil
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.unErr = i18n.T("un.runTimeout")
		} else {
			s.unErr = err.Error()
		}
	}
	s.mu.Unlock()
	// 卸载成功后立即重扫：让列表/缓存反映工具已消失，无需手动刷新
	if uninstalled {
		s.Scan()
	}
}

// GetUninstallStatus 返回代跑状态 JSON：{running, done, output, error}。
// 前端轮询（macOS WKWebView 对高频事件不可靠，沿用 update 进度方案）。
func (s *ScannerService) GetUninstallStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(map[string]any{
		"running": s.unRunning, "done": s.unDone, "output": s.unOutput, "error": s.unErr,
	})
	return string(b)
}

// UninstallResidue 返回当前工具的残留列表 JSON（双源检测：规则表 + 扫描快照）。
func (s *ScannerService) UninstallResidue() string {
	s.mu.Lock()
	tool, last := s.unTool, s.last
	s.mu.Unlock()
	if tool == "" {
		return `{"error":"` + i18n.T("un.noTool") + `"}`
	}
	rr := uninstall.Residues(tool, last)
	b, _ := json.Marshal(rr)
	return string(b)
}

// UninstallTrashResidues 将选中的残留路径移入内置回收站（可恢复），随后后台重扫。
func (s *ScannerService) UninstallTrashResidues(paths []string) string {
	s.mu.Lock()
	tool, last := s.unTool, s.last
	s.mu.Unlock()
	if tool == "" {
		b, _ := json.Marshal(map[string]any{"error": i18n.T("un.noTool")})
		return string(b)
	}
	all := uninstall.Residues(tool, last)
	want := map[string]bool{}
	for _, p := range paths {
		want[p] = true
	}
	var sel []uninstall.Residue
	for _, r := range all {
		if want[r.Path] {
			sel = append(sel, r)
		}
	}
	deleted, errs := uninstall.TrashResidues(sel, tool)
	s.Scan() // 后台重扫，让主界面刷新
	b, _ := json.Marshal(map[string]any{"deleted": deleted, "errors": errs})
	return string(b)
}

// withPath 返回替换 PATH 后的环境变量切片（保留其余环境）。
func withPath(env []string, path string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "PATH="+path)
}
