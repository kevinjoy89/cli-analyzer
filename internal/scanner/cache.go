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
	// 唯一临时文件名：GUI 与 CLI 并发扫描写缓存时固定 ".tmp" 会互相覆盖
	tmp, err := os.CreateTemp(dir, "last-scan-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, cachePath())
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
// 无缓存时视为已清除（幂等）：首次运行/已清过的机器上"清除失败"是误导。
func ClearCache() error {
	err := os.Remove(cachePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// CacheInfo returns the cache modification time, or ok=false when absent.
func CacheInfo() (string, bool) {
	st, err := os.Stat(cachePath())
	if err != nil {
		return "", false
	}
	return st.ModTime().Format(time.RFC3339), true
}
