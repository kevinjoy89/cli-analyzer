//go:build linux

package platform

func rootFor(kind RootKind) string { return rootForBase(kind) }
