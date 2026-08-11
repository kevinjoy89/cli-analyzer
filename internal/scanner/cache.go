package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"cli-analyzer/internal/platform"
)

// cachePath returns the last-scan cache file location.
func cachePath() string { return filepath.Join(platform.CacheRoot(), "last-scan.json") }

// SaveCache atomically writes the scan result to the cache file.
func SaveCache(res *ScanResult) error {
	dir := filepath.Dir(cachePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	tmp := cachePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cachePath())
}

// LoadCache reads the last scan result, or returns an error when absent.
func LoadCache() (*ScanResult, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, err
	}
	var res ScanResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ClearCache removes the cache file (used by `cache --clear`).
func ClearCache() error { return os.Remove(cachePath()) }

// CacheInfo returns the cache modification time, or ok=false when absent.
func CacheInfo() (string, bool) {
	st, err := os.Stat(cachePath())
	if err != nil {
		return "", false
	}
	return st.ModTime().Format(time.RFC3339), true
}
