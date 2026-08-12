// Package gui is the thin Wails boundary: it is the only package besides the
// root main that imports Wails. All scanning/cleaning logic lives in
// internal/scanner and internal/cleaner.
package gui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	goruntime "runtime"
	"sync"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/history"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
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
}

func NewScannerService() *ScannerService { return &ScannerService{} }

// Startup loads the last scan from cache so the window renders instantly,
// then kicks off a background (silent) update check when enabled.
func (s *ScannerService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if res, err := scanner.LoadCache(); err == nil {
		s.last = res
	}
	// 自动检查更新：异步执行；配置关闭、命中 24h 缓存或网络失败时均静默无提示
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
	}()
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

// CheckForUpdates 手动检查更新（不受 24h 缓存限制），返回 CheckResult JSON；
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

// DownloadUpdate 开始下载最新版安装包（异步）。下载进度经 "update:progress"
// 推送；完成后自动校验 SHA256，成功推 "update:downloaded"，校验失败推
// "update:verify-failed"，取消推 "update:cancelled"，其他错误推 "update:error"。
func (s *ScannerService) DownloadUpdate() string {
	s.mu.Lock()
	res := s.check
	inFlight := s.downloadCancel != nil
	s.mu.Unlock()
	if inFlight {
		return "下载已在进行中"
	}
	if res == nil || !res.UpdateAvailable || res.DownloadURL == "" {
		return "没有可下载的更新"
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.downloadCancel = cancel
	s.mu.Unlock()
	go s.runDownload(ctx, *res, cancel)
	return ""
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
	path, err := updater.DownloadInstaller(ctx, nil, release, asset, func(w, t int64) {
		runtime.EventsEmit(s.ctx, "update:progress", map[string]any{"downloaded": w, "total": t})
	})
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
		return "没有进行中的下载"
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
		return "没有已下载的安装包"
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
