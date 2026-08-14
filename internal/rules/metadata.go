package rules

// Meta is the display metadata shown in the GUI detail panel.
type Meta struct {
	Homepage    string
	Description string
}

// metaByName holds optional Homepage/Description for both curated and generic
// tools. It never affects attribution — only the detail panel.
var metaByName = map[string]Meta{
	"claude":        {"https://docs.anthropic.com/en/docs/claude-code", "tool.desc.claude"},
	"kimi":          {"https://kimi.moonshot.cn", "tool.desc.kimi"},
	"mimocode":      {"https://www.zhipuai.cn", "tool.desc.mimocode"},
	"opencode":      {"https://opencode.ai", "tool.desc.opencode"},
	"mavis":         {"", "tool.desc.mavis"},
	"nodejs":        {"https://nodejs.org", "tool.desc.nodejs"},
	"npm":           {"https://www.npmjs.com", "tool.desc.npm"},
	"node":          {"https://nodejs.org", "tool.desc.node"},
	"bun":           {"https://bun.sh", "tool.desc.bun"},
	"yarn":          {"https://yarnpkg.com", "tool.desc.yarn"},
	"pnpm":          {"https://pnpm.io", "tool.desc.pnpm"},
	"gh":            {"https://cli.github.com", "tool.desc.gh"},
	"uv":            {"https://docs.astral.sh/uv", "tool.desc.uv"},
	"huggingface":   {"https://huggingface.co", "tool.desc.huggingface"},
	"docker":        {"https://docs.docker.com/engine", "tool.desc.docker"},
	"brew":          {"https://brew.sh", "tool.desc.brew"},
	"go":            {"https://go.dev", "tool.desc.go"},
	"pyenv":         {"https://github.com/pyenv/pyenv", "tool.desc.pyenv"},
	"rustup":        {"https://rustup.rs", "tool.desc.rustup"},
	"p10k":          {"https://github.com/romkatv/powerlevel10k", "tool.desc.p10k"},
	"prisma":        {"https://www.prisma.io", "tool.desc.prisma"},
	"puppeteer":     {"https://pptr.dev", "tool.desc.puppeteer"},
	"codex":         {"https://openai.com/codex", "tool.desc.codex"},
	"git":           {"https://git-scm.com", "tool.desc.git"},
	"pip":           {"https://pypi.org", "tool.desc.pip"},
	"trae-cn":       {"https://www.trae.com.cn", "tool.desc.trae-cn"},
	"summarize":     {"", "tool.desc.summarize"},
	"codegraph":     {"", "tool.desc.codegraph"},
	"skillhub":      {"", "tool.desc.skillhub"},
	"agent-browser": {"", "tool.desc.agent-browser"},
}
