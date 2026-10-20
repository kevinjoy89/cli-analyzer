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
		Name: "claude", DisplayName: "Claude Code", Aliases: []string{"claude-code"}, Installer: "versioned", Category: CategoryAgent,
		DataDirs: []DataDirRule{
			// 配置/状态混合根（settings.json、todos、cache、backups 等）
			dd(platform.Home, ".claude", TierUser, "data"),
			// AI 语义子目录（2026-09-30 实测：projects 17M、transcripts 39M、
			// telemetry 7.6M、file-history/shell-snapshots 回滚点）
			dd(platform.Home, ".claude/projects", TierUser, KindSession),
			dd(platform.Home, ".claude/transcripts", TierUser, KindSession),
			dd(platform.Home, ".claude/file-history", TierUser, KindCheckpoint),
			dd(platform.Home, ".claude/shell-snapshots", TierUser, KindCheckpoint),
			dd(platform.Home, ".claude/plugins", TierUser, KindPlugins),
			dd(platform.Home, ".claude/telemetry", TierSafe, KindTelemetry),
			// 版本化安装基目录（versions/ 617M 级）：必须标 install 而非
			// data——标 data 会把安装本体列为可处置项（clean --include-data
			// 误删 claude 安装），且安装占用列被吞成 0。旧版本清理由
			// deriveOldVersions 基于 installRoot 生成，不受此影响。
			dd(platform.XDGData, "claude", TierUser, "install"),
			dd(platform.XDGConfig, "claude", TierUser, "config"),
			dd(platform.XDGCache, "claude", TierSafe, "cache"),
		},
	},
	{
		Name: "kimi", DisplayName: "Kimi Code", Aliases: []string{"kimi-code"}, Installer: "other", Category: CategoryAgent,
		// 桌面端为 Kimi Code.app（Electron）；kimi-desktop 是 Kimi 聊天助手，
		// 不属于本条目。
		Apps: []string{"Kimi Code"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".kimi-code", TierUser, "data"), // config.toml, sessions, telemetry, install mix
			// AI 语义子目录（2026-09-30 实测：bin 353M、sessions 28M、search-index 3.3M）
			dd(platform.Home, ".kimi-code/sessions", TierUser, KindSession),
			dd(platform.Home, ".kimi-code/search-index", TierSafe, KindIndex),
			dd(platform.Home, ".kimi-code/bin", TierSafe, KindRuntime),
			// 伴随命令的独立目录（kimi-webbridge 浏览器桥 / kimi-slides），
			// 由 companion 家族归并进 kimi 后随行认领
			dd(platform.Home, ".kimi-webbridge", TierUser, "data"),
			dd(platform.Home, ".kimi-work", TierUser, "data"),
			// Kimi Code 桌面端数据
			ddDesktop(platform.MacAppSupport, "kimi-code-app", TierUser, "data"),
		},
	},
	{
		Name: "mimocode", DisplayName: "mimocode", Aliases: []string{"mimo"}, Installer: "other", Category: CategoryAgent,
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "mimocode", TierUser, "data"),
			// 会话快照/索引（opencode fork 同构；mimocode.db 会话数据库文件
			// 位于父目录，由树下钻 subKind 标注 session-db 仅报告）
			dd(platform.XDGData, "mimocode/snapshot", TierSafe, KindIndex),
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
		Name: "opencode", DisplayName: "OpenCode", Installer: "other", Category: CategoryAgent,
		// OpenCode.app 存在时把桌面端包计入安装占用并读出版本（双形态工具）。
		Apps: []string{"OpenCode"},
		DataDirs: []DataDirRule{
			dd(platform.XDGData, "opencode", TierUser, "data"),
			// AI 语义子目录（2026-09-30 实测：snapshot 158M、storage 94M、
			// bin 123M、log 59M；opencode.db 会话数据库文件位于父目录，
			// 由树下钻 subKind 标注 session-db 仅报告）
			dd(platform.XDGData, "opencode/snapshot", TierSafe, KindIndex),
			dd(platform.XDGData, "opencode/storage", TierUser, KindSession),
			dd(platform.XDGData, "opencode/bin", TierSafe, KindRuntime),
			dd(platform.XDGData, "opencode/log", TierSafe, KindLogs),
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
			// OpenCode.app 桌面端（2026-09-30 实测 ai.opencode.desktop 1.1G）
			ddDesktop(platform.MacAppSupport, "ai.opencode.desktop", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "@opencode-ai", TierUser, "data"),
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
		Name: "dsh", DisplayName: "DeepSeek Harness", Installer: "npm", Category: CategoryAgent,
		// 桌面端为 DeepSeek Harness.app（@deepseek-ai 数据目录）
		Apps: []string{"DeepSeek Harness"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".dsh", TierUser, "data"), // profiles/sessions/agent/telemetry
			// AI 语义子目录（2026-09-30 实测：dsh-runtimes 361M、sessions 84M、
			// profiles 27M、attachments 16M）
			dd(platform.Home, ".dsh/dsh-runtimes", TierSafe, KindRuntime),
			dd(platform.Home, ".dsh/sessions", TierUser, KindSession),
			dd(platform.Home, ".dsh/profiles", TierUser, "config"),
			dd(platform.Home, ".dsh/attachments", TierUser, KindArtifacts),
			// DeepSeek Harness 桌面端数据
			ddDesktop(platform.MacAppSupport, "@deepseek-ai", TierUser, "data"),
		},
	},
	{
		Name: "pi", Installer: "npm", Category: CategoryAgent,
		DataDirs: []DataDirRule{
			dd(platform.Home, ".pi", TierUser, "data"),
			// AI 语义子目录（2026-09-30 实测：agent/npm 375M、agent/sessions 175M）
			dd(platform.Home, ".pi/agent/npm", TierSafe, KindRuntime),
			dd(platform.Home, ".pi/agent/sessions", TierUser, KindSession),
			dd(platform.Home, ".pi/context-mode", TierSafe, "cache"),
		},
	},
	{
		// oh-my-pi 的 CLI（@oh-my-pi/pi-coding-agent，binary 名 omp）
		Name: "omp", Aliases: []string{"oh-my-pi"}, Installer: "npm",
	},
	{
		Name: "codex", DisplayName: "Codex CLI", Installer: "other", Category: CategoryAgent,
		// 桌面端承载于 ChatGPT.app（bundle id com.openai.codex）
		Apps: []string{"ChatGPT"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".codex", TierUser, "data"),
			// AI 语义子目录（2026-09-30 实测：plugins 316M、packages 311M、
			// sessions 176M、archived_sessions 25M；logs_2.sqlite 与
			// thread_history_1.sqlite 会话数据库文件由树下钻 subKind 标注）
			dd(platform.Home, ".codex/sessions", TierUser, KindSession),
			dd(platform.Home, ".codex/archived_sessions", TierUser, KindSession),
			dd(platform.Home, ".codex/plugins", TierSafe, KindRuntime),
			dd(platform.Home, ".codex/packages", TierSafe, KindRuntime),
			// Codex 桌面端数据（Electron Chromium profile）
			ddDesktop(platform.MacAppSupport, "Codex", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "com.openai.codex", TierUser, "data"),
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

// ddDesktop 是桌面端形态（Form = desktop）的目录规则：桌面端 AppSupport
// 数据、Windows 桌面端镜像目录等。
func ddDesktop(root platform.RootKind, sub, tier, kind string) DataDirRule {
	r := dd(root, sub, tier, kind)
	r.Form = FormDesktop
	return r
}

func cl(root platform.RootKind, sub, tier, kind, keep, desc string) CleanRule {
	return CleanRule{Root: root, Sub: sub, Tier: tier, Kind: kind, Keep: keep, Desc: desc}
}
