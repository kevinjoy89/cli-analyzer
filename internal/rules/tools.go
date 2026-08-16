package rules

import (
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
)

// curated is the built-in two-tier rule table. Data-dir tiers are set here;
// any DataDir with Kind "cache" automatically becomes a SAFE cleanable during
// scanning. Versioned/brew/pyenv old-version cleanables are derived by the
// scanner from the installer type, not listed here.
var curated = []Rule{
	{
		Name: "claude", Aliases: []string{"claude-code"}, Installer: "versioned",
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "claude", TierUser, "data"),
			dd(platform.XDGConfig, "claude", TierUser, "config"),
			dd(platform.XDGCache, "claude", TierSafe, "cache"),
		},
	},
	{
		Name: "kimi", Aliases: []string{"kimi-code"}, Installer: "other",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".kimi-code", TierUser, "data"), // config.toml, sessions, telemetry, install mix
			// 伴随命令的独立目录（kimi-webbridge 浏览器桥 / kimi-slides），
			// 由 companion 家族归并进 kimi 后随行认领
			dd(platform.Home, ".kimi-webbridge", TierUser, "data"),
			dd(platform.Home, ".kimi-work", TierUser, "data"),
		},
	},
	{
		Name: "mimocode", Aliases: []string{"mimo"}, Installer: "other",
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "mimocode", TierUser, "data"),
			dd(platform.XDGConfig, "mimocode", TierUser, "config"),
			// mimocode is an opencode fork and stores its plugins in the cache
			// dir the same way — treat as USER data, not cleanable cache.
			dd(platform.XDGCache, "mimocode", TierUser, "data"),
			// 安装目录（bin + node_modules + package.json，143MB 级）：
			// mimo 二进制经 companion 家族归并进 mimocode，安装目录随行认领
			dd(platform.Home, ".mimocode", TierUser, "install"),
		},
	},
	{
		Name: "opencode", Installer: "other",
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "opencode", TierUser, "data"),
			dd(platform.XDGConfig, "opencode", TierUser, "config"),
			// oh-my-opencode 是 opencode 的插件/主题框架（配置目录），归属 opencode
			dd(platform.XDGConfig, "oh-my-opencode", TierUser, "config"),
			// ~/.cache/opencode is NOT cache — it is opencode's plugin/extension
			// install dir (MCP servers, language servers, skills, package.json
			// manifest). Deleting it breaks those plugins and loses the manifest,
			// so it must stay USER (never auto-deletable).
			dd(platform.XDGCache, "opencode", TierUser, "data"),
			// 安装目录（bin + node_modules，194MB 级）：opencode-plugins 等
			// 伴随命令经 companion 家族归并后随行认领
			dd(platform.Home, ".opencode", TierUser, "install"),
		},
	},
	{
		Name: "mavis", Installer: "versioned",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".mavis", TierUser, "data"),
			dd(platform.XDGCache, "mavis", TierSafe, "cache"),
		},
	},
	{
		// nodejs 是 Node.js 运行时家族（node/npm/npx/corepack/node-gyp）的合并
		// 工具：Windows 官方安装器 / nvm-windows / Volta / scoop 目录里每个
		// 命令原本各自成行，扫描器把它们归并为一条 nodejs（见 scanner 的
		// nodejsFamily）。brew 公式 node 不走合并，保持 "node" 原名。
		// 注意：%APPDATA%\npm（Windows npm 全局前缀）不再计入 nodejs——
		// 其中的 -g 包（opencode、pi 等）由扫描器归为各自的 npm 工具
		// （安装根 = npm\node_modules\<pkg>），避免双重计数。
		Name: "nodejs", Aliases: []string{"node", "npm", "npx", "corepack", "node-gyp"}, Installer: "nodejs",
		Cleanables: []CleanRule{
			cl(platform.Home, ".npm", TierSafe, "cache", "", "npm cache (~/.npm) — safe to clear with npm cache clean or deleting the dir"),
			cl(platform.LocalAppData, "npm-cache", TierSafe, "cache", "", "npm cache (%LocalAppData%\\npm-cache) — safe to clear with npm cache clean"),
			cl(platform.LocalAppData, "node-addon-native-custom-loader", TierSafe, "cache", "", "node-addon-require-builtin native addon cache (%LocalAppData%\\node-addon-native-custom-loader) — safe to delete, re-downloaded on demand"),
		},
	},
	{
		Name: "gh", Installer: "brew",
		DataDirs: []DataDirRule{
			dd(platform.XDGConfig, "gh", TierUser, "config"),
			dd(platform.XDGData, "gh", TierUser, "data"),
			dd(platform.XDGCache, "gh", TierSafe, "cache"),
		},
	},
	{
		Name: "clink", Installer: "other",
		DataDirs: []DataDirRule{
			// clink（cmd 增强）安装目录即数据目录：clink.exe 与历史/日志共存于
			// %LocalAppData%\clink，二进制经注册表注入 cmd 不在 PATH（发现由
			// platform.augmentUserDirs windows 补充）。
			dd(platform.LocalAppData, "clink", TierUser, "data"),
		},
	},
	{
		Name: "uv", Installer: "local-bin",
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "uv", TierUser, "data"),
			dd(platform.XDGCache, "uv", TierSafe, "cache"),
			dd(platform.MacCaches, "uv", TierSafe, "cache"),
		},
	},
	{
		Name: "huggingface", Aliases: []string{"hf"}, Installer: "other",
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "huggingface", TierUser, "data"),
		},
		Cleanables: []CleanRule{
			cl(platform.XDGCache, "huggingface", TierSafe, "download", "", "HuggingFace model cache — re-downloadable; offline users may want to keep it"),
		},
	},
	{
		Name: "docker", Installer: "brew",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".docker", TierUser, "data"),
			dd(platform.XDGConfig, "docker", TierUser, "config"),
		},
		// v1: no auto-prune; docker system prune -a is shown as advice only.
	},
	{
		Name: "brew", Installer: "brew",
		DataDirs: []DataDirRule{
			dd(platform.XDGConfig, "brew", TierUser, "config"),
		},
	},
	{
		Name: "go", Installer: "go",
		DataDirs: []DataDirRule{
			dd(platform.Home, "go/pkg/mod", TierUser, "data"),
		},
		Cleanables: []CleanRule{
			cl(platform.MacCaches, "go-build", TierSafe, "cache", "", "Go build cache (go clean -cache)"),
			cl(platform.XDGCache, "go-build", TierSafe, "cache", "", "Go build cache (go clean -cache)"),
			cl(platform.Home, "go/pkg/mod/cache/download", TierSafe, "download", "", "Downloaded Go module cache (re-downloadable)"),
		},
	},
	{
		Name: "pyenv", Installer: "pyenv",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".pyenv", TierUser, "data"),
		},
		// Old toolchains (non-default versions) derived by scanner.
	},
	{
		Name: "rustup", Installer: "rustup",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".cargo", TierUser, "data"),
			dd(platform.Home, ".rustup", TierUser, "data"),
		},
		Cleanables: []CleanRule{
			cl(platform.Home, ".cargo/registry/cache", TierSafe, "download", "", "Cargo registry cache (re-fetchable)"),
		},
	},
	{
		Name: "p10k", Installer: "other",
		Cleanables: []CleanRule{
			{Root: platform.XDGCache, Sub: "p10k-*", Tier: TierSafe, Kind: "cache", Desc: "Powerlevel10k cache dumps"},
		},
	},
	{
		Name: "prisma", Installer: "npm",
		Cleanables: []CleanRule{
			cl(platform.XDGCache, "prisma", TierSafe, "cache", "", "Prisma engine cache (re-downloadable)"),
		},
	},
	{
		Name: "puppeteer", Installer: "npm",
		Cleanables: []CleanRule{
			cl(platform.XDGCache, "puppeteer", TierSafe, "download", "", "Puppeteer Chromium binary (re-downloadable)"),
		},
	},
	{
		// codegraph 双安装形态归一：~/.codegraph/versions 版本化安装（binary
		// 名 codegraph，installer versioned）与 npm @colbymchenry/codegraph
		// 归并为一行（classify 两侧映射）。数据/运行状态在 ~/.codegraph。
		Name: "codegraph", Installer: "versioned",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".codegraph", TierUser, "data"), // current/daemons/telemetry/update-check
		},
	},
	{
		Name: "dsh", Installer: "npm",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".dsh", TierUser, "data"), // profiles/sessions/agent/telemetry
		},
	},
	{
		Name: "pi", Installer: "npm",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".pi", TierUser, "data"),
		},
	},
	{
		// oh-my-pi 的 CLI（@oh-my-pi/pi-coding-agent，binary 名 omp）
		Name: "omp", Aliases: []string{"oh-my-pi"}, Installer: "npm",
	},
	{
		Name: "codex", Installer: "other",
		DataDirs: []DataDirRule{
			dd(platform.Home, ".codex", TierUser, "data"),
		},
		Cleanables: []CleanRule{
			// Only leftover install-staging dirs are cleanable; the active
			// codex-primary-runtime MUST be preserved or codex breaks.
			{Root: platform.XDGCache, Sub: "codex-runtimes/codex-runtime-install-*", Tier: TierSafe, Kind: "cache", Desc: i18n.T("ui.kind.codexStaging")},
		},
	},
	{
		Name: "git", Installer: "brew",
		DataDirs: []DataDirRule{
			dd(platform.XDGConfig, "git", TierUser, "config"),
			dd(platform.Home, ".gitconfig", TierUser, "config"),
		},
	},
	{
		Name: "pip", Aliases: []string{"pip3"}, Installer: "pip",
		Cleanables: []CleanRule{
			cl(platform.MacCaches, "pip", TierSafe, "cache", "", "pip cache"),
			cl(platform.XDGCache, "pip", TierSafe, "cache", "", "pip cache"),
		},
	},
	{
		Name: "yarn", Installer: "npm",
		Cleanables: []CleanRule{
			cl(platform.MacCaches, "Yarn", TierSafe, "cache", "", "Yarn global cache"),
			cl(platform.XDGCache, "yarn", TierSafe, "cache", "", "Yarn cache"),
		},
	},
	{
		Name: "pnpm", Installer: "npm",
		Cleanables: []CleanRule{
			cl(platform.XDGCache, "pnpm", TierSafe, "cache", "", "pnpm metadata cache"),
			cl(platform.XDGState, "pnpm", TierSafe, "cache", "", "pnpm content-addressable store (pnpm store prune)"),
		},
	},
}

func dd(root platform.RootKind, sub, tier, kind string) DataDirRule {
	return DataDirRule{Root: root, Sub: sub, Tier: tier, Kind: kind}
}

func cl(root platform.RootKind, sub, tier, kind, keep, desc string) CleanRule {
	return CleanRule{Root: root, Sub: sub, Tier: tier, Kind: kind, Keep: keep, Desc: desc}
}
