// Package cleaner deletes cleanable items with a hard two-tier safety gate.
package cleaner

import (
	"os"
	"path/filepath"
	"strings"

	"cli-analyzer/internal/config"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// subItem pairs a sub entry with its owning cleanable, so a child path can be
// deleted on its own while inheriting the parent's SAFE gate and guards.
type subItem struct {
	parent *scanner.Cleanable
	sub    *scanner.SubEntry
}

// Clean removes the cleanable items identified by ids (looked up in result).
//
// An id is either a full cleanable (deletes its whole path) or a sub-entry id
// ("<cleanable ID>::<child path>") which deletes just that child — verified to
// be one of the children the scan actually attributed to a SAFE cleanable.
//
// Safety model:
//   - Only Tier "safe" items are ever removed; anything else lands in Skipped
//     regardless of caller intent or flags.
//   - Guards are re-checked at delete time (not just scan time): the path must
//     be absolute, free of "..", outside forbidden system roots, not the trash
//     root, and — for old-version items — not equal to the tool's current
//     version path.
//   - Deletion is deferred by default: SAFE items move into the built-in trash
//     (recoverable within the retention window) unless the config disables it.
//
// When dryRun is true nothing is touched and Deleted stays empty.
func Clean(result *scanner.ScanResult, ids []string, dryRun bool) scanner.CleanReport {
	return clean(result, ids, dryRun, false)
}

// CleanPermanent 强制直接删除（跳过内置回收站），供 `clean --permanent` 使用；
// SAFE 门禁与 guard 检查不变
func CleanPermanent(result *scanner.ScanResult, ids []string, dryRun bool) scanner.CleanReport {
	return clean(result, ids, dryRun, true)
}

// clean 是 Clean / CleanPermanent 的公共实现；permanent 为 true 时直接删除
func clean(result *scanner.ScanResult, ids []string, dryRun bool, permanent bool) scanner.CleanReport {
	report := scanner.CleanReport{
		DryRun:  dryRun,
		Deleted: []string{},
		Skipped: []string{},
		Errors:  []string{},
	}
	if result == nil {
		report.Errors = append(report.Errors, "no scan result available")
		return report
	}

	byID := map[string]*scanner.Cleanable{}
	subs := map[string]*subItem{}
	for i := range result.Tools {
		for j := range result.Tools[i].Cleanables {
			c := &result.Tools[i].Cleanables[j]
			byID[c.ID] = c
			for k := range c.Sub {
				s := &c.Sub[k]
				subs[s.ID] = &subItem{parent: c, sub: s}
			}
		}
	}

	// 默认延迟删除；仅当永久模式或配置禁用回收站时才真正删除
	useTrash := !permanent && config.Load().Trash.UseTrash
	remove := func(path string, bytes int64, tool, kind string) {
		if dryRun {
			report.Deleted = append(report.Deleted, path)
			report.Freed += bytes
			return
		}
		if useTrash {
			if err := trash.Trash(path, trash.Item{Tool: tool, Kind: kind, Bytes: bytes}); err != nil {
				report.Errors = append(report.Errors, path+": "+err.Error())
				return
			}
		} else {
			if err := os.RemoveAll(path); err != nil {
				report.Errors = append(report.Errors, path+": "+err.Error())
				return
			}
		}
		report.Deleted = append(report.Deleted, path)
		report.Freed += bytes
	}

	for _, id := range ids {
		if c, ok := byID[id]; ok {
			if c.Tier != scanner.TierSafe {
				report.Skipped = append(report.Skipped, c.Path+" (USER data — not auto-deletable)")
				continue
			}
			if reason := guard(c); reason != "" {
				report.Skipped = append(report.Skipped, c.Path+" ("+reason+")")
				continue
			}
			remove(c.Path, c.Bytes, c.Tool, c.Kind)
			continue
		}
		if si, ok := subs[id]; ok {
			if si.parent.Tier != scanner.TierSafe {
				report.Skipped = append(report.Skipped, si.sub.Path+" (USER data — not auto-deletable)")
				continue
			}
			if reason := guardSub(si.parent.Path, si.sub.Path); reason != "" {
				report.Skipped = append(report.Skipped, si.sub.Path+" ("+reason+")")
				continue
			}
			// 子项用扫描器给出的精确类型（~/.npm/_logs → logs，而非父项的 cache）；
			// 旧缓存无 Kind 字段时回退父项类型。
			kind := si.sub.Kind
			if kind == "" {
				kind = si.parent.Kind
			}
			remove(si.sub.Path, si.sub.Bytes, si.parent.Tool, kind)
			continue
		}
		report.Skipped = append(report.Skipped, id+" (unknown item)")
	}
	return report
}

// guard returns "" when the item may be deleted, else a reason string.
func guard(c *scanner.Cleanable) string {
	if !filepath.IsAbs(c.Path) {
		return "path is not absolute"
	}
	clean := filepath.Clean(c.Path)
	if clean != c.Path {
		return "path contains .. or redundant segments"
	}
	if forbidden(clean) {
		return "path is a forbidden system root"
	}
	if clean == trash.Root() {
		return "path is the trash root"
	}
	if c.CurrentPath != "" && clean == filepath.Clean(c.CurrentPath) {
		return "refusing to delete the current version"
	}
	return ""
}

// guardSub verifies a sub-entry path may be deleted: it must be absolute,
// clean, not itself a forbidden root, and strictly inside its owning cleanable
// directory (so an id can never reach outside the parent).
func guardSub(parent, p string) string {
	if !filepath.IsAbs(p) {
		return "path is not absolute"
	}
	clean := filepath.Clean(p)
	if clean != p {
		return "path contains .. or redundant segments"
	}
	if forbidden(clean) {
		return "path is a forbidden system root"
	}
	rel, err := filepath.Rel(parent, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "path is not inside its cleanable"
	}
	return ""
}

// forbiddenRoots are paths that must never be deleted themselves. Deleting
// well-formed descendants (e.g. a brew Cellar/<formula>/<old-version>) is
// allowed — those are precisely the cleanables the scanner produces.
var forbiddenRoots = []string{
	"/", "/usr", "/usr/bin", "/usr/sbin", "/bin", "/sbin", "/System",
	"/Library", "/etc", "/var", "/Volumes", "/Applications",
	"/opt", "/opt/homebrew", "/usr/local",
}

// forbidden reports whether p is one of the protected system roots. Note this
// is an exact match on purpose: /opt/homebrew is protected, but
// /opt/homebrew/Cellar/<f>/<v> is a legitimate cleanable.
func forbidden(p string) bool {
	for _, r := range forbiddenRoots {
		if p == r {
			return true
		}
	}
	// The user's home dir itself must never be deleted.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p == home {
			return true
		}
	}
	return false
}
