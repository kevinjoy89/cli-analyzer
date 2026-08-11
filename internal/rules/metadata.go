package rules

// Meta is the display metadata shown in the GUI detail panel.
type Meta struct {
	Homepage    string
	Description string
}

// metaByName holds optional Homepage/Description for both curated and generic
// tools. It never affects attribution — only the detail panel.
var metaByName = map[string]Meta{
	"claude":     {"https://docs.anthropic.com/en/docs/claude-code", "Anthropic 的 AI 编程助手 CLI"},
	"kimi":       {"https://kimi.moonshot.cn", "月之暗面 Kimi 的 AI 编程助手"},
	"mimocode":   {"https://www.zhipuai.cn", "智谱的 AI 编程智能体（GLM）"},
	"opencode":   {"https://opencode.ai", "开源 AI 编程智能体"},
	"mavis":      {"", "AI 编程智能体"},
	"npm":        {"https://www.npmjs.com", "Node.js 包管理器"},
	"node":       {"https://nodejs.org", "JavaScript 运行时"},
	"bun":        {"https://bun.sh", "极速 JavaScript 运行时与包管理器"},
	"yarn":       {"https://yarnpkg.com", "JavaScript 包管理器"},
	"pnpm":       {"https://pnpm.io", "高效的 JavaScript 包管理器"},
	"gh":         {"https://cli.github.com", "GitHub 官方命令行工具"},
	"uv":         {"https://docs.astral.sh/uv", "极速 Python 包与项目管理器"},
	"huggingface": {"https://huggingface.co", "机器学习模型与数据集平台"},
	"docker":     {"https://docs.docker.com/engine", "容器引擎"},
	"brew":       {"https://brew.sh", "macOS / Linux 包管理器（Homebrew）"},
	"go":         {"https://go.dev", "Go 语言工具链"},
	"pyenv":      {"https://github.com/pyenv/pyenv", "Python 版本管理器"},
	"rustup":     {"https://rustup.rs", "Rust 工具链管理器"},
	"p10k":       {"https://github.com/romkatv/powerlevel10k", "Zsh 主题 Powerlevel10k"},
	"prisma":     {"https://www.prisma.io", "Node.js ORM 引擎"},
	"puppeteer":  {"https://pptr.dev", "Headless Chrome 控制库"},
	"codex":      {"https://openai.com/codex", "OpenAI 的编程智能体"},
	"git":        {"https://git-scm.com", "分布式版本控制系统"},
	"pip":        {"https://pypi.org", "Python 包安装器"},
	"trae-cn":    {"https://www.trae.com.cn", "字节跳动 Trae AI IDE 的 CLI"},
	"summarize":  {"", "代码总结工具"},
	"codegraph":  {"", "代码知识图谱工具"},
	"skillhub":   {"", "技能（Skill）管理工具"},
	"agent-browser": {"", "AI 浏览器控制工具"},
}
