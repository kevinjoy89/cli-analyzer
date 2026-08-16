package scanner

import (
	"os"
	"path/filepath"
	"sort"

	"cli-analyzer/internal/platform"
)

// FingerprintEntry 记录一个测量路径的变更指纹（mtime+size，仅 stat 不递归）。
type FingerprintEntry struct {
	Path  string `json:"path"`
	MTime int64  `json:"mtime"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"dir"`
}

// statEntry 对单个路径 stat；路径不存在返回 nil（条目缺失由比较方判定为变更）。
// MTime 用纳秒精度（与 probe 缓存一致）：同一秒内同大小的替换也能被检测。
func statEntry(p string) *FingerprintEntry {
	st, err := os.Stat(p)
	if err != nil {
		return nil
	}
	return &FingerprintEntry{
		Path:  filepath.Clean(p),
		MTime: st.ModTime().UnixNano(),
		Size:  st.Size(),
		IsDir: st.IsDir(),
	}
}

// measurePaths 收集扫描结果中全部被测量的顶层路径：二进制 real、
// dataDirs、cleanables 与孤儿路径（去重）。
func measurePaths(res *ScanResult) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	if res != nil {
		for i := range res.Tools {
			t := &res.Tools[i]
			for j := range t.Binaries {
				add(t.Binaries[j].Real)
			}
			for j := range t.DataDirs {
				add(t.DataDirs[j].Path)
			}
			for j := range t.Cleanables {
				add(t.Cleanables[j].Path)
			}
		}
		for i := range res.Unattributed {
			add(res.Unattributed[i].Path)
		}
	}
	return out
}

// pathDirFingerprints 收集 PATH 发现目录的 stat 条目（与工具发现同一来源，
// 含 augmentUserDirs 补齐目录）：新二进制/新工具进入 PATH 目录时目录 mtime
// 变化，指纹不一致 → 自动全量。仅 stat，不 ReadDir 目录内容。
// 不存在的目录不产生条目（创建后条目出现 → 变更，保守方向）。
func pathDirFingerprints() []FingerprintEntry {
	dirs := platform.PathDirs(true)
	out := make([]FingerprintEntry, 0, len(dirs))
	for _, d := range dirs {
		if e := statEntry(d); e != nil {
			out = append(out, *e)
		}
	}
	return out
}

// ComputeFingerprint 返回扫描结果全部测量路径 + PATH 发现目录的当前指纹
// （按路径排序，序列化稳定）。不存在的路径不产生条目 ——
// FingerprintsEqual 把条目缺失判定为变更。
func ComputeFingerprint(res *ScanResult) []FingerprintEntry {
	paths := measurePaths(res)
	out := make([]FingerprintEntry, 0, len(paths)+16)
	for _, p := range paths {
		if e := statEntry(p); e != nil {
			out = append(out, *e)
		}
	}
	out = append(out, pathDirFingerprints()...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// FingerprintsEqual 比较两份指纹：条目数、路径集合与每个条目的
// mtime/size/isDir 全部一致才相等（顺序无关）。
func FingerprintsEqual(a, b []FingerprintEntry) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]FingerprintEntry(nil), a...)
	sb := append([]FingerprintEntry(nil), b...)
	sort.Slice(sa, func(i, j int) bool { return sa[i].Path < sa[j].Path })
	sort.Slice(sb, func(i, j int) bool { return sb[i].Path < sb[j].Path })
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}
