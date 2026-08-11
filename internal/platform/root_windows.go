//go:build windows

package platform

import "os"

// rootFor maps Windows %APPDATA%/%LOCALAPPDATA% roots.
func rootFor(kind RootKind) string {
	switch kind {
	case AppData:
		return os.Getenv("APPDATA")
	case LocalAppData:
		return os.Getenv("LOCALAPPDATA")
	case Home:
		h, _ := os.UserHomeDir()
		return h
	}
	return ""
}
