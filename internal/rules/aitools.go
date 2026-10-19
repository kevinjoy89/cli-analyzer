package rules

import (
	"cli-analyzer/internal/platform"
)

// AI 语义 kind 与类别常量（add-ai-tool-catalog）。
//
// AI 编码工具链的占用主体与通用 CLI 不同：会话记录（用户数据，可 resume）、
// 会话数据库（SQLite，rm 等于毁掉对话历史）、可再生代码索引、捆绑运行时
// 各需要不同的处置策略。kind 是自由字符串，经 ui.kindLabel.* 文案展示；
// tier 沿用现有 Safe/User 信息标签。
const (
	// KindSession 会话/转录（projects、sessions、transcripts、rollout）。
	KindSession = "session"
	// KindSessionDB 会话数据库（opencode.db、zcode cli/db、*.sqlite）。
	// 仅报告：不生成可处置项，树下钻中标注说明（见 ui.sessionDb.note）。
	KindSessionDB = "session-db"
	// KindIndex 可再生代码索引/知识图谱（.ckg、search-index、snapshot）。
	KindIndex = "index"
	// KindRuntime 捆绑运行时与 npm 包（bin、node_modules、packages、runtimes）。
	KindRuntime = "runtime"
	// KindPlugins 插件/扩展安装目录。
	KindPlugins = "plugins"
	// KindArtifacts agent 产物（exec、attachments、blobs、computer-use）。
	KindArtifacts = "artifacts"
	// KindLogs 日志与 trace（复用既有 "logs" 标签）。
	KindLogs = "logs"
	// KindTelemetry 遥测缓存。
	KindTelemetry = "telemetry"
	// KindCheckpoint 文件历史/回滚点（file-history、shell-snapshots）。
	KindCheckpoint = "checkpoint"
	// KindWorkspaceData IDE 工作区存储（VS Code workspaceStorage）。
	KindWorkspaceData = "workspace-data"
)

// AI 编目类别：agent = agent harness（以 CLI 为主要形态，桌面端应用归入
// 同一条目）；ai-ide = AI IDE。空字符串 = 通用 CLI 工具。
const (
	CategoryAgent = "agent"
	CategoryAIIde = "ai-ide"
)

// IsReportOnly 报告 kind 是否为「仅报告」目录：归因展示但不生成可处置项。
// 会话数据库包含全部对话历史，rm 式删除等于毁掉 resume 能力，行级清理
// 需要理解各家 schema（见 add-ai-tool-catalog design D2）。
func IsReportOnly(kind string) bool {
	return kind == KindSessionDB
}

// aiCurated 是第一批 AI 编目条目（本机实测，2026-09-30，macOS；实测占用
// 记录在每条注释里，schema 变动时便于复核）。目录规则沿用 dd() 模式，
// 跨平台根在对应平台不存在时 Resolve 返回 "" 自动跳过。
var aiCurated = []Rule{
	{
		// ZCode：agent harness（CLI 数据 + ZCode.app 桌面端）。cli/agents
		// 1.2G 的构成待确认（见 design Open Questions），先随父目录归因。
		Name: "zcode", DisplayName: "ZCode", Installer: "other", Category: CategoryAgent,
		Apps: []string{"ZCode"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".zcode", TierUser, "data"),
			// 会话数据库（2026-09-30 实测 478M）：仅报告不清理
			dd(platform.Home, ".zcode/cli/db", TierUser, KindSessionDB),
			dd(platform.Home, ".zcode/cli/artifacts", TierUser, KindArtifacts),
			dd(platform.Home, ".zcode/cli/exec", TierUser, KindArtifacts),
			dd(platform.Home, ".zcode/cli/plugins", TierUser, KindPlugins),
			dd(platform.Home, ".zcode/cli/rollout", TierUser, KindSession),
			dd(platform.Home, ".zcode/cli/image-cache", TierSafe, "cache"),
			dd(platform.Home, ".zcode/computer-use", TierUser, KindArtifacts),
			ddDesktop(platform.MacAppSupport, "ZCode", TierUser, "data"),
		},
	},
	{
		// Antigravity：Google 的 agent（agy CLI + Antigravity.app 桌面端 +
		// Antigravity Tools.app 伴侣）。CLI 数据位于 ~/.gemini 根目录
		Name: "agy", DisplayName: "Antigravity", Aliases: []string{"antigravity"}, Installer: "local-bin",
		Category: CategoryAgent,
		Apps:     []string{"Antigravity", "Antigravity Tools"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".gemini", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "Antigravity", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "com.lbjlaq.antigravity-tools", TierUser, "data"),
			dd(platform.AppData, "Antigravity", TierUser, "data"),
		},
	},
	{
		// WorkBuddy：腾讯 agent 桌面端（Electron）。logs 696M + traces 266M
		// 是单体最大的纯日志堆积（2026-09-30 实测）
		Name: "workbuddy", DisplayName: "WorkBuddy", Installer: "other", Category: CategoryAgent,
		Apps: []string{"WorkBuddy"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".workbuddy", TierUser, "data"),
			dd(platform.Home, ".workbuddy/binaries", TierSafe, KindRuntime),
			dd(platform.Home, ".workbuddy/logs", TierSafe, KindLogs),
			dd(platform.Home, ".workbuddy/traces", TierSafe, KindLogs),
			dd(platform.Home, ".workbuddy/plugins", TierUser, KindPlugins),
			dd(platform.Home, ".workbuddy/blobs", TierUser, KindArtifacts),
			// projects 为会话/工作区混合数据，语义待确认，先按会话标注
			dd(platform.Home, ".workbuddy/projects", TierUser, KindSession),
			ddDesktop(platform.AppData, "WorkBuddy", TierUser, "data"),
		},
	},
	{
		// u1s1：pi 生态的桌面发行版（io.u1s1.app，主二进制 pi-agent-desktop）
		Name: "u1s1", DisplayName: "u1s1", Installer: "other", Category: CategoryAgent,
		Apps: []string{"u1s1"},
		DataDirs: []DataDirRule{
			dd(platform.Home, ".u1s1", TierUser, "data"),
			// 捆绑的 pi 运行时（2026-09-30 实测 461M）
			dd(platform.Home, ".u1s1/agent/npm", TierSafe, KindRuntime),
			dd(platform.Home, ".u1s1/agent/sessions", TierUser, KindSession),
			ddDesktop(platform.AppData, "u1s1", TierUser, "data"),
		},
	},
	{
		Name: "copilot", DisplayName: "GitHub Copilot", Installer: "other", Category: CategoryAgent,
		DataDirs: []DataDirRule{
			// GitHub Copilot CLI 配置（2026-09-30 实测 84K）
			dd(platform.Home, ".copilot", TierUser, "config"),
			dd(platform.AppData, "copilot", TierUser, "data"),
		},
	},
	// ---- AI IDE ----
	{
		// VS Code：AI IDE 宿主
		// workspaceStorage 1.3G 是 IDE 工作区存储（2026-09-30 实测）
		Name: "code", DisplayName: "Visual Studio Code", Aliases: []string{"vscode"}, Installer: "other",
		Category: CategoryAIIde,
		Apps:     []string{"Visual Studio Code"},
		DataDirs: []DataDirRule{
			ddDesktop(platform.MacAppSupport, "Code", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "Code/User/workspaceStorage", TierUser, KindWorkspaceData),
			ddDesktop(platform.MacAppSupport, "Code/CachedData", TierSafe, "cache"),
			// Windows 官方数据路径镜像
			ddDesktop(platform.AppData, "Code", TierUser, "data"),
			ddDesktop(platform.AppData, `Code\User\workspaceStorage`, TierUser, KindWorkspaceData),
			ddDesktop(platform.LocalAppData, "Code", TierUser, "data"),
		},
	},
	{
		// JoyCoder：独立的 AI IDE
		Name: "joycoder", DisplayName: "JoyCoder", Installer: "other",
		Category: CategoryAIIde,
		Apps:     []string{"JoyCoder"},
		DataDirs: []DataDirRule{
			ddDesktop(platform.MacAppSupport, "JoyCoder", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "JoyCoder/CachedData", TierSafe, "cache"),
			ddDesktop(platform.AppData, "JoyCoder", TierUser, "data"),
		},
	},
	{
		// CodeBuddy：独立的 AI IDE
		Name: "codebuddy", DisplayName: "CodeBuddy", Installer: "other",
		Category: CategoryAIIde,
		Apps:     []string{"CodeBuddy", "CodeBuddyExtension"},
		DataDirs: []DataDirRule{
			ddDesktop(platform.MacAppSupport, "CodeBuddyExtension", TierUser, "data"),
			ddDesktop(platform.AppData, "CodeBuddyExtension", TierUser, "data"),
		},
	},
	{
		// Trae CN：字节 AI IDE（本机已卸载，数据残留约 1.2G）
		Name: "trae-cn", DisplayName: "Trae CN", Installer: "other", Category: CategoryAIIde,
		Apps: []string{"Trae CN"},
		DataDirs: []DataDirRule{
			ddDesktop(platform.Home, ".trae-cn", TierUser, "data"),
			ddDesktop(platform.MacAppSupport, "Trae CN", TierUser, "data"),
			ddDesktop(platform.AppData, "Trae CN", TierUser, "data"),
		},
	},
	{
		// MarsCode：豆包 AI IDE（Trae 前身，本机已卸载）。残留几乎全是
		// .ckg 代码知识图谱索引（2026-09-30 实测 3.9G，可再生）
		Name: "marscode", DisplayName: "MarsCode", Installer: "other", Category: CategoryAIIde,
		Apps: []string{"MarsCode"},
		DataDirs: []DataDirRule{
			ddDesktop(platform.Home, ".marscode", TierUser, "data"),
			ddDesktop(platform.Home, ".marscode/.ckg", TierSafe, KindIndex),
		},
	},
}

// FormDesktop 是桌面端形态的 Form 标注值（add-form-factor-split）。
const FormDesktop = "desktop"
