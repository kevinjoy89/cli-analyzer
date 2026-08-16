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

// fingerprintPath 返回变更指纹文件路径（独立于 last-scan.json，避免格式迁移）。
func fingerprintPath() string { return filepath.Join(platform.CacheRoot(), "last-scan.fp.json") }

// writeJSONAtomic 原子写 JSON 文件：唯一临时名 + Sync + RenameReplace。
// GUI 与 CLI 并发写同一文件时固定 ".tmp" 会互相覆盖，必须用唯一临时名。
func writeJSONAtomic(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
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
	return platform.RenameReplace(tmpName, path)
}

// SaveCache atomically writes the scan result to the cache file.
func SaveCache(res *ScanResult) error {
	return writeJSONAtomic(cachePath(), res)
}

// SaveFingerprint 原子写变更指纹文件。
func SaveFingerprint(entries []FingerprintEntry) error {
	return writeJSONAtomic(fingerprintPath(), entries)
}

// LoadFingerprint 读取变更指纹；文件缺失返回 error。
func LoadFingerprint() ([]FingerprintEntry, error) {
	data, err := os.ReadFile(fingerprintPath())
	if err != nil {
		return nil, err
	}
	var out []FingerprintEntry
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
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

// ClearCache removes the cache file and the fingerprint file
// (used by `cache --clear`).
// 无缓存时视为已清除（幂等）：首次运行/已清过的机器上"清除失败"是误导。
func ClearCache() error {
	err := os.Remove(cachePath())
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return err
	}
	if err := os.Remove(fingerprintPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CacheInfo returns the cache modification time, or ok=false when absent.
func CacheInfo() (string, bool) {
	st, err := os.Stat(cachePath())
	if err != nil {
		return "", false
	}
	return st.ModTime().Format(time.RFC3339), true
}
