// Package disk measures directory sizes without external tools (no `du`).
package disk

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Sizer computes sizes for a set of top-level paths in parallel.
//
// Size semantics: physical bytes (allocated blocks) on Unix, apparent size on
// Windows. Hard-linked files are counted once per path (v1 accepted
// limitation). Symlinks are never followed.
type Sizer struct {
	// Workers caps concurrent directory walks; 0 means GOMAXPROCS.
	Workers int
	// Errors counts unreadable subtrees encountered during walks.
	Errors int

	// Skip, when non-empty, lists absolute directory paths to exclude from
	// measurement（如内置回收站根目录）。遍历期间只读。
	Skip map[string]bool

	mu sync.Mutex
}

func (s *Sizer) countErr() {
	s.mu.Lock()
	s.Errors++
	s.mu.Unlock()
}

// WalkSize returns the total size of path (file or dir), skipping unreadable
// entries without following symlinks.
func (s *Sizer) WalkSize(path string) int64 {
	info, err := os.Lstat(path)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return fileSize(info)
	}
	return s.walkDir(path)
}

func (s *Sizer) walkDir(root string) int64 {
	var total int64
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			s.countErr()
			return nil // keep walking siblings
		}
		// 跳过配置的排除目录（如内置回收站），不进入其子树
		if d.IsDir() && s.Skip[p] {
			return filepath.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		// Never follow symlinks; WalkDir won't descend into dir symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		total += fileSize(info)
		return nil
	})
	return total
}

// WalkAll returns the size of each dir, computed in parallel. dirs should be a
// deduplicated set of top-level paths.
func (s *Sizer) WalkAll(dirs []string) map[string]int64 {
	workers := s.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	out := make(map[string]int64, len(dirs))
	var mu sync.Mutex
	for _, d := range dirs {
		if d == "" {
			continue
		}
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			n := s.WalkSize(dir)
			mu.Lock()
			out[dir] = n
			mu.Unlock()
		}(d)
	}
	wg.Wait()
	return out
}

// ChildrenSizes returns the size of each direct child (file or dir) under dir.
// It is used to render one level of breakdown for a cleanable item in the UI.
// Returns nil when dir is not a readable directory. Symlinks are never
// followed: a symlinked child is reported by its link size, matching WalkSize.
func (s *Sizer) ChildrenSizes(dir string) map[string]int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make(map[string]int64, len(entries))
	var dirs []string
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			dirs = append(dirs, p)
		} else {
			if st, err := os.Lstat(p); err == nil {
				out[p] = fileSize(st)
			}
		}
	}
	for p, n := range s.WalkAll(dirs) {
		out[p] = n
	}
	return out
}
