//go:build darwin || linux

package platform

import (
	"os"
	"path/filepath"
)

// rootForBase covers the XDG-style roots shared by macOS and Linux.
func rootForBase(kind RootKind) string {
	switch kind {
	case XDGCache:
		return xdgOr("XDG_CACHE_HOME", ".cache")
	case XDGData:
		return xdgOr("XDG_DATA_HOME", ".local/share")
	case XDGConfig:
		return xdgOr("XDG_CONFIG_HOME", ".config")
	case XDGState:
		return xdgOr("XDG_STATE_HOME", ".local/state")
	case Home:
		h, _ := os.UserHomeDir()
		return h
	}
	return ""
}

func xdgOr(env, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return filepath.Join(h, def)
	}
	return ""
}
