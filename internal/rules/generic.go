package rules

import "cli-analyzer/internal/platform"

// GenericDataDirs returns name-based attribution rules for a tool not in the
// curated table. Each rule resolves to "" on platforms where its root does not
// exist, so the same table works on macOS, Linux and Windows.
func GenericDataDirs(name string) []DataDirRule {
	return []DataDirRule{
		dd(platform.XDGCache, name, TierSafe, "cache"),
		dd(platform.XDGData, name, TierUser, "data"),
		dd(platform.XDGConfig, name, TierUser, "config"),
		dd(platform.Home, "."+name, TierUser, "data"),
		dd(platform.MacCaches, name, TierSafe, "cache"),
		dd(platform.MacAppSupport, name, TierUser, "data"),
		dd(platform.AppData, name, TierUser, "data"),
		dd(platform.LocalAppData, name, TierUser, "data"),
	}
}
