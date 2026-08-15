package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/uninstall"
)

// runUninstall 处理 `uninstall <tool> [--residue] [--yes] [--json]`。
// 退出码：0 成功 / 1 错误 / 2 黑名单或无该工具。
// 流程：展示标准卸载命令（可代跑）→ 残留检测 → 残留移入内置回收站。
// 残留清理是唯一触碰 USER 级数据的路径，硬约束为必须经回收站（--yes 不豁免）。
func runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	residueOnly := fs.Bool("residue", false, "only list residue, delete nothing")
	yes := fs.Bool("yes", false, "skip interactive prompts (trash constraint unchanged)")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr(), i18n.T("un.usage"))
		return 1
	}
	name := fs.Arg(0)

	if uninstall.IsBlocked(name) {
		return uninstallBlocked(name, *jsonOut)
	}

	res, err := scanner.LoadCache()
	if err != nil {
		// 从未扫描：缓存缺失是状态问题，不是工具不存在——错误信息必须
		// 可区分（此前静默当"未找到工具"处理，用户会被误导去怀疑工具名）
		if *jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": name, "error": i18n.T("cli.noScanResult")})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintln(stderr(), i18n.T("cli.noScanResult"))
		}
		return 1
	}
	tool := findTool(res, name)
	if tool == nil {
		if *jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": name, "error": i18n.T("un.toolNotFound")})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("un.toolNotFound"))
		}
		return 2
	}
	if uninstall.IsBlocked(tool.Name) { // 别名命中黑名单
		return uninstallBlocked(tool.Name, *jsonOut)
	}

	binName := ""
	if len(tool.Binaries) > 0 {
		binName = tool.Binaries[0].Name
	}
	off := uninstall.OfficialCommand(scanner.Installer(tool.Installer), tool.Name, binName)

	if *residueOnly {
		return printResidue(tool.Name, res, *jsonOut)
	}
	if *jsonOut {
		// --json 契约：只输出信息，不执行
		b, _ := json.Marshal(map[string]any{
			"tool": tool.Name, "installer": tool.Installer, "blocked": false,
			"officialCommand": off.Command, "runnable": off.Runnable,
			"residue": uninstall.Residues(tool.Name, res),
		})
		fmt.Fprintln(stdout(), string(b))
		return 0
	}

	// 交互流程：标准卸载命令（可代跑）→ 残留 → 回收站
	fmt.Fprintf(stdout(), "%s: %s\n", i18n.T("un.toolInstaller"), tool.Installer)
	if off.Command != "" {
		fmt.Fprintf(stdout(), "%s\n  %s\n", i18n.T("un.officialCmd"), off.Command)
	} else {
		fmt.Fprintln(stdout(), i18n.T("un.noOfficialCmd"))
	}
	if off.Runnable {
		if !*yes && !confirmPrompt(i18n.T("un.runOfficialPrompt")) {
			fmt.Fprintln(stdout(), i18n.T("un.skipRun"))
		} else {
			if err := runOfficial(off); err != nil {
				fmt.Fprintf(stdout(), "%s: %v\n", i18n.T("un.runFailed"), err)
			} else {
				fmt.Fprintln(stdout(), i18n.T("un.runDone"))
			}
		}
	} else if off.Command != "" {
		fmt.Fprintln(stdout(), i18n.T("un.hintManual"))
	}

	// 残留检测（无论标准卸载成功与否）
	rr := uninstall.Residues(tool.Name, res)
	if len(rr) == 0 {
		fmt.Fprintln(stdout(), i18n.T("un.residueNone"))
		return 0
	}
	printResidueList(rr)
	if !*yes && !confirmPrompt(i18n.T("un.trashPrompt")) {
		fmt.Fprintln(stdout(), i18n.T("un.trashSkipped"))
		return 0
	}
	deleted, errs := uninstall.TrashResidues(rr, tool.Name)
	fmt.Fprintf(stdout(), "%s %d 项\n", i18n.T("un.trashed"), len(deleted))
	for _, e := range errs {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}

func uninstallBlocked(name string, jsonOut bool) int {
	if jsonOut {
		b, _ := json.Marshal(map[string]any{"tool": name, "blocked": true, "error": i18n.T("un.blockedSystem")})
		fmt.Fprintln(stdout(), string(b))
	} else {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("un.blockedSystem"))
	}
	return 2
}

func printResidue(tool string, res *scanner.ScanResult, jsonOut bool) int {
	rr := uninstall.Residues(tool, res)
	if jsonOut {
		b, _ := json.Marshal(rr)
		fmt.Fprintln(stdout(), string(b))
		return 0
	}
	if len(rr) == 0 {
		fmt.Fprintln(stdout(), i18n.T("un.residueNone"))
		return 0
	}
	printResidueList(rr)
	return 0
}

func printResidueList(rr []uninstall.Residue) {
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("un.residueHeader"))
	for _, r := range rr {
		note := ""
		if r.Tier == "user" {
			note = i18n.T("un.residueCredential")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Path, humanBytes(r.Bytes), r.Tier, note)
	}
	w.Flush()
}

// confirmPrompt 从 stdin 读取 y/N；非交互（EOF）时返回 false。
func confirmPrompt(prompt string) bool {
	fmt.Fprintf(stdout(), "%s ", prompt)
	sc := bufio.NewScanner(stdin)
	if !sc.Scan() {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(sc.Text()))
	return ans == "y" || ans == "yes"
}

// runOfficial 代跑标准卸载命令（5 分钟超时，输出流式透传）。
// 与 GUI 行为一致：经（增强的）PATH 解析命令绝对路径并注入完整 PATH——
// CLI 若从最小 PATH 环境启动（GUI 启动的终端/cron），裸 exec "npm" 会报
// "executable file not found in $PATH"，增强目录补齐 shell-only 安装位置。
func runOfficial(off uninstall.Official) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bin := off.Bin
	if resolved, rerr := uninstall.ResolveCommand(off.Bin); rerr == nil {
		bin = resolved
	}
	cmd := exec.CommandContext(ctx, bin, off.Args...)
	cmd.Env = uninstall.WithPath(os.Environ(), uninstall.AugmentedPathEnv())
	cmd.Stdout = stdout()
	cmd.Stderr = stderr()
	return cmd.Run()
}

// findTool 在扫描结果中按名称或别名查找工具。
func findTool(res *scanner.ScanResult, name string) *scanner.Tool {
	if res == nil {
		return nil
	}
	for i := range res.Tools {
		t := &res.Tools[i]
		if t.Name == name {
			return t
		}
		for _, a := range t.Aliases {
			if a == name {
				return t
			}
		}
	}
	return nil
}
