package uninstall

import (
	"os"
	"path/filepath"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/rules"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// Residue 是卸载后仍存在的一个数据目录（残留）。
type Residue struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Tier  string `json:"tier"` // safe | user
	Kind  string `json:"kind"` // config|data|cache|install|state
}

// Residues 双源检测某工具的残留目录：
//   - 规则表数据目录（含 generic 规则，确定性基础，即使从未扫描过也成立）
//   - 扫描快照中归因到该工具的 DataDirs（兜住规则表外的自定义目录）
//
// 仅返回仍存在的路径；去重按绝对路径。
func Residues(name string, snapshot *scanner.ScanResult) []Residue {
	seen := map[string]bool{}
	var out []Residue
	sizer := &disk.Sizer{}
	add := func(path, tier, kind string) {
		path = filepath.Clean(path)
		if path == "" || path == "." || seen[path] {
			return
		}
		st, err := os.Stat(path)
		if err != nil {
			return // 已不存在 → 不是残留
		}
		seen[path] = true
		bytes := int64(0)
		if st.IsDir() {
			bytes = sizer.WalkSize(path)
		} else {
			bytes = st.Size()
		}
		out = append(out, Residue{Path: path, Bytes: bytes, Tier: tier, Kind: kind})
	}

	// 源 1：规则表（curated + generic）
	t := rules.Load()
	if r := t.Lookup(name); r != nil {
		for _, dr := range r.DataDirs {
			if p := resolveRule(dr); p != "" {
				add(p, dr.Tier, dr.Kind)
			}
		}
	}
	for _, dr := range rules.GenericDataDirs(name) {
		if p := resolveRule(dr); p != "" {
			add(p, dr.Tier, dr.Kind)
		}
	}

	// 源 2：扫描快照归因目录
	if snapshot != nil {
		for _, tool := range snapshot.Tools {
			if tool.Name != name {
				continue
			}
			for _, d := range tool.DataDirs {
				add(d.Path, string(d.Tier), d.Kind)
			}
		}
	}
	return out
}

// resolveRule 将数据目录规则解析为绝对路径（Path 优先，否则 Root+Sub）。
func resolveRule(dr rules.DataDirRule) string {
	if dr.Path != "" {
		return dr.Path
	}
	if dr.Root == "" {
		return ""
	}
	base := platform.Root(dr.Root)
	if base == "" {
		return ""
	}
	return filepath.Join(base, dr.Sub)
}

// TrashResidues 将残留项移入内置回收站（可恢复）。返回成功移入的路径与失败项。
// 这是 USER 级数据唯一允许的删除路径：本包不提供永久删除变体。
func TrashResidues(res []Residue, tool string) (deleted []string, errs []error) {
	for _, r := range res {
		if err := trash.Trash(r.Path, trash.Item{
			Tool:  tool,
			Kind:  r.Kind,
			Bytes: r.Bytes,
		}); err != nil {
			errs = append(errs, err)
			continue
		}
		deleted = append(deleted, r.Path)
	}
	return
}
