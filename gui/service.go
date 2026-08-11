// Package gui is the thin Wails boundary: it is the only package besides the
// root main that imports Wails. All scanning/cleaning logic lives in
// internal/scanner and internal/cleaner.
package gui

import (
	"context"
	"encoding/json"
	"sync"

	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/history"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the app version shown in the UI footer and About dialog.
const AppVersion = "0.1.1"

// ScannerService is the Wails binding the frontend calls.
type ScannerService struct {
	ctx  context.Context
	mu   sync.Mutex
	last *scanner.ScanResult
}

func NewScannerService() *ScannerService { return &ScannerService{} }

// Startup loads the last scan from cache so the window renders instantly.
func (s *ScannerService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if res, err := scanner.LoadCache(); err == nil {
		s.last = res
	}
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
