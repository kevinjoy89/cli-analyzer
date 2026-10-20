package trash

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// 可逆性加固：删除前把「要删什么」写进清单。
//
// 内置回收站本身可恢复，但恢复之后没人能验证「真的全回来了」。清单记录
// 移入回收站时的路径与文件数，Restore 之后可以逐条核对——恢复不是「命令
// 成功了」，而是「东西确实在」。
//
// 清单是**辅助证据**，不是恢复的前提：清单缺失（旧版本移入的项目）时
// Restore 照常工作，只是不做校验。

// Manifest 是一个回收站项目的处置清单。
type Manifest struct {
	// Files 是移入时统计到的文件数（目录按遍历计数）。
	Files int `json:"files"`
	// TopPaths 是移入目录的直接子项（最多 maxTopPaths 个），用于恢复后
	// 快速判断内容是否齐全。
	TopPaths []string `json:"topPaths,omitempty"`
	// TotalBytes 是移入时的字节数（与 Item.Bytes 同口径）。
	TotalBytes int64 `json:"totalBytes"`
}

// maxTopPaths 限制清单体积：这是一份「够核对」的摘要，不是完整目录树。
const maxTopPaths = 200

func manifestPathOf(itemDir string) string { return filepath.Join(itemDir, "manifest.json") }

// buildManifest 在移入**之前**统计 path（此时它还在原位置）。
func buildManifest(path string, bytes int64) Manifest {
	m := Manifest{TotalBytes: bytes}
	st, err := os.Lstat(path)
	if err != nil {
		return m
	}
	if !st.IsDir() {
		m.Files = 1
		return m
	}
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		m.Files++
		return nil
	})
	entries, err := os.ReadDir(path)
	if err == nil {
		for _, e := range entries {
			if len(m.TopPaths) >= maxTopPaths {
				break
			}
			m.TopPaths = append(m.TopPaths, e.Name())
		}
	}
	return m
}

func writeManifest(itemDir string, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPathOf(itemDir), b, 0o644)
}

// readManifest 返回清单；不存在或损坏时返回 (零值, false)。
func readManifest(itemDir string) (Manifest, bool) {
	b, err := os.ReadFile(manifestPathOf(itemDir))
	if err != nil {
		return Manifest{}, false
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, false
	}
	return m, true
}

// VerifyResult 是恢复后的核对结果。
type VerifyResult struct {
	// Path 是内容实际落回的路径。
	Path string `json:"path"`
	// Checked 表示是否找到清单（false = 旧项目，未做校验）。
	Checked bool `json:"checked"`
	// Files 是恢复后实际统计到的文件数。
	Files int `json:"files"`
	// ExpectedFiles 是清单记录的文件数（Checked 时有效）。
	ExpectedFiles int `json:"expectedFiles,omitempty"`
	// Missing 是清单里有、恢复后找不到的顶层项。
	Missing []string `json:"missing,omitempty"`
	// OK 为 true 表示文件数与顶层项都对得上。
	OK bool `json:"ok"`
}
