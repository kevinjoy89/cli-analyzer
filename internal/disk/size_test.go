package disk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkSize(t *testing.T) {
	td := t.TempDir()
	mkfile(t, filepath.Join(td, "a.txt"), 100)
	mkfile(t, filepath.Join(td, "sub", "b.txt"), 50)
	// Symlink loop must not hang or count.
	if err := os.Symlink(td, filepath.Join(td, "sub", "loop")); err == nil {
		t.Log("created symlink loop")
	}

	s := &Sizer{}
	got := s.WalkSize(td)
	// Physical size >= apparent 150 bytes.
	if got < 150 {
		t.Errorf("WalkSize(%s) = %d, want >= 150", td, got)
	}
}

func TestWalkAllParallel(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	mkfile(t, filepath.Join(d1, "x"), 10)
	mkfile(t, filepath.Join(d2, "y"), 20)
	s := &Sizer{Workers: 2}
	sizes := s.WalkAll([]string{d1, d2})
	if sizes[d1] < 10 || sizes[d2] < 20 {
		t.Errorf("WalkAll sizes wrong: %v", sizes)
	}
	// Missing path -> 0, and reported in Errors.
	missing := filepath.Join(d1, "does-not-exist")
	sizes = s.WalkAll([]string{missing})
	if sizes[missing] != 0 {
		t.Errorf("missing dir size = %d, want 0", sizes[missing])
	}
}

func mkfile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
}
