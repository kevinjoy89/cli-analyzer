// Package gui is the thin Wails boundary: it is the only package besides the
// root main that imports Wails. All scanning/cleaning logic lives in
// internal/scanner and internal/cleaner.
package gui

import (
	"context"
	"encoding/json"
	"sync"

	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/scanner"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the app version shown in the UI footer and About dialog.
const AppVersion = "0.1.0"

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

// GetVersion returns the app version string (e.g. "0.1.0").
func (s *ScannerService) GetVersion() string { return AppVersion }
