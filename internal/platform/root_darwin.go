//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

// rootFor adds the macOS ~/Library roots on top of the XDG-style ones. macOS
// GUI-style apps store under Library; CLI tools typically use the XDG dirs.
func rootFor(kind RootKind) string {
	if v := rootForBase(kind); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	if h == "" {
		return ""
	}
	switch kind {
	case MacAppSupport:
		return filepath.Join(h, "Library", "Application Support")
	case MacCaches:
		return filepath.Join(h, "Library", "Caches")
	case MacPrefs:
		return filepath.Join(h, "Library", "Preferences")
	}
	return ""
}
