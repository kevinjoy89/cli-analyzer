package main

import (
	"os"
	"testing"

	"cli-analyzer/internal/cli"
)

// TestCLIDispatchWhitelist 守住 main.go 的 CLI 白名单：cli.Run 认识的每个
// 子命令都必须能被 main 路由到 CLI，否则 `cli-analyzer <新命令>` 会落到
// GUI 分支弹窗口（upgrade / residue 就这么漏过一轮）。
//
// 这里不启动 GUI，只校验名单与 cli 包的子命令集合一致。
func TestCLIDispatchWhitelist(t *testing.T) {
	// 与 main.go 的 switch 保持同步；新增子命令时两边都要加。
	whitelist := map[string]bool{
		"scan": true, "clean": true, "cache": true, "trash": true,
		"trends": true, "update": true, "upgrade": true,
		"uninstall": true, "residue": true,
		"version": true, "--version": true, "-v": true,
		"help": true, "-h": true, "--help": true,
	}

	// cli 包对外暴露的子命令清单（由 cli.Subcommands 提供）。
	for _, name := range cli.Subcommands() {
		if !whitelist[name] {
			t.Errorf("subcommand %q is not routed to the CLI by main.go — "+
				"it would open the GUI window instead", name)
		}
	}
}

// TestHelpListsEverySubcommand 断言 usage 文本不会漏掉已注册的子命令。
func TestHelpListsEverySubcommand(t *testing.T) {
	// 只有 help/version 这类不需要在用法里逐个列举的除外。
	skip := map[string]bool{"help": true, "version": true}
	for _, name := range cli.Subcommands() {
		if skip[name] {
			continue
		}
		if !cli.UsageMentions(name) {
			t.Errorf("`cli-analyzer help` does not mention %q", name)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
