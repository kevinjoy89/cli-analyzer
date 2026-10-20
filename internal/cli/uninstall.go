package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"cli-analyzer/internal/cmdexec"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/uninstall"
)

// UninstallReport is the machine-readable contract of `uninstall`.
type UninstallReport struct {
	Tool            string              `json:"tool"`
	Installer       string              `json:"installer"`
	Blocked         bool                `json:"blocked"`
	OfficialCommand string              `json:"officialCommand"`
	Runnable        bool                `json:"runnable"`
	DryRun          bool                `json:"dryRun"`
	Residue         []uninstall.Residue `json:"residue"`
	// Deleted lists the residue paths actually disposed of (empty on dry-run).
	Deleted []string `json:"deleted,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// runUninstall 处理 `uninstall <tool> [--residue] [--yes] [--permanent] [--dry-run] [--json]`。
// 退出码：0 成功 / 1 错误 / 2 黑名单或无该工具。
// 流程：展示标准卸载命令（可代跑）→ 残留检测 → 残留处置（默认移入内置
// 回收站，可恢复；--permanent 为用户显式选择的永久删除）。
// --dry-run 只输出计划，不动文件系统、不代跑命令、不写回收站索引。
func runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	residueOnly := fs.Bool("residue", false, "only list residue, delete nothing")
	yes := fs.Bool("yes", false, "skip interactive prompts")
	permanent := fs.Bool("permanent", false, "permanently delete residue (default: built-in trash, recoverable)")
	dryRun := fs.Bool("n", false, "dry run: print the plan, touch nothing")
	fs.BoolVar(dryRun, "dry-run", false, "dry run")
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
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
		return uninstallBlocked(name, *jsonOut, *pretty)
	}

	res, err := scanner.LoadCache()
	if err != nil {
		// 从未扫描：缓存缺失是状态问题，不是工具不存在——错误信息必须
		// 可区分（此前静默当"未找到工具"处理，用户会被误导去怀疑工具名）
		if *jsonOut {
			emitJSONError(KindUninstall, map[string]any{"tool": name, "message": i18n.T("cli.noScanResult")}, *pretty)
		} else {
			fmt.Fprintln(stderr(), i18n.T("cli.noScanResult"))
		}
		return 1
	}
	tool := findTool(res, name)
	if tool == nil {
		if *jsonOut {
			emitJSONError(KindUninstall, map[string]any{"tool": name, "message": i18n.T("un.toolNotFound")}, *pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("un.toolNotFound"))
		}
		return 2
	}
	if uninstall.IsBlocked(tool.Name) { // 别名命中黑名单
		return uninstallBlocked(tool.Name, *jsonOut, *pretty)
	}

	off := uninstall.OfficialFor(*tool)
	residues := uninstall.Residues(tool.Name, res)

	// 记录卸载事件（残留闭环）：即使后续处置失败，下次 `residue` 仍能提醒。
	// 预演不记（预演意味着「我还没决定」）。
	if !*dryRun && !*residueOnly {
		paths := make([]string, 0, len(residues))
		for _, r := range residues {
			paths = append(paths, r.Path)
		}
		_ = uninstall.RecordEvent(tool.Name, paths)
	}

	if *residueOnly {
		return printResidue(tool.Name, residues, *jsonOut, *pretty)
	}

	report := UninstallReport{
		Tool:            tool.Name,
		Installer:       tool.Installer,
		OfficialCommand: off.Command,
		Runnable:        off.Runnable,
		DryRun:          *dryRun,
		Residue:         residues,
	}

	// --dry-run：只输出计划。不代跑命令、不动文件、不写回收站索引——
	// 预演与真实执行同口径（同一份 residues 列表），但零副作用。
	if *dryRun {
		if *jsonOut {
			_ = emitJSON(KindUninstall, report, *pretty)
			return 0
		}
		fmt.Fprintf(stdout(), "%s: %s\n", i18n.T("un.toolInstaller"), tool.Installer)
		if off.Command != "" {
			fmt.Fprintf(stdout(), "%s\n  %s\n", i18n.T("un.officialCmd"), off.Command)
		} else {
			fmt.Fprintln(stdout(), i18n.T("un.noOfficialCmd"))
		}
		if len(residues) == 0 {
			fmt.Fprintln(stdout(), i18n.T("un.residueNone"))
		} else {
			printResidueList(residues)
		}
		fmt.Fprintln(stdout(), i18n.T("un.dryRunPlan"))
		return 0
	}

	if *jsonOut {
		// --json 契约：只输出信息，不执行（镜像 trash/upgrade）
		_ = emitJSON(KindUninstall, report, *pretty)
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
	if *permanent {
		if !*yes && !confirmPrompt(i18n.T("un.deletePermanentPrompt")) {
			fmt.Fprintln(stdout(), i18n.T("un.trashSkipped"))
			return 0
		}
		deleted, errs := uninstall.RemoveResidues(rr, tool.Name)
		fmt.Fprintln(stdout(), i18n.T("un.deletedPermanentN", map[string]any{"n": len(deleted)}))
		for _, e := range errs {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
		}
		if len(errs) > 0 {
			return 1
		}
		return 0
	}
	if !*yes && !confirmPrompt(i18n.T("un.trashPrompt")) {
		fmt.Fprintln(stdout(), i18n.T("un.trashSkipped"))
		return 0
	}
	deleted, errs := uninstall.TrashResidues(rr, tool.Name)
	fmt.Fprintln(stdout(), i18n.T("un.trashedN", map[string]any{"n": len(deleted)}))
	for _, e := range errs {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}

func uninstallBlocked(name string, jsonOut, pretty bool) int {
	if jsonOut {
		emitJSONError(KindUninstall, map[string]any{
			"tool": name, "blocked": true, "message": i18n.T("un.blockedSystem"),
		}, pretty)
	} else {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("un.blockedSystem"))
	}
	return 2
}

func printResidue(tool string, rr []uninstall.Residue, jsonOut, pretty bool) int {
	if jsonOut {
		if rr == nil {
			rr = []uninstall.Residue{}
		}
		_ = emitJSON(KindUninstall, map[string]any{"tool": tool, "residue": rr}, pretty)
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
	if resolved, rerr := cmdexec.ResolveCommand(off.Bin); rerr == nil {
		bin = resolved
	}
	// Windows .cmd/.bat shim（npm 全局）经 cmd.exe /c 执行。
	bin, args := cmdexec.WrapShim(bin, off.Args)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
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
