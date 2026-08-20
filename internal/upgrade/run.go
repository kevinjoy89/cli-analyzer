package upgrade

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"cli-analyzer/internal/cmdexec"
)

// runTimeout 是代跑升级命令的总超时（与 uninstall 代跑一致，5 分钟）。
// brew upgrade / cargo install 编译可能更久，超时由调用方按错误提示；
// GUI 侧以轮询状态展示输出，CLI 侧流式透传。
const runTimeout = 5 * time.Minute

// RunOfficial 代跑官方升级命令：经（增强的）PATH 解析命令绝对路径并注入
// 完整 PATH（与 uninstall 代跑一致）。输出按 out/err 分流：GUI 传同一
// 并发安全缓冲，CLI 分别传 stdout/stderr。
func RunOfficial(cmd Command, out, errw io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	bin := cmd.Bin
	if resolved, rerr := cmdexec.ResolveCommand(cmd.Bin); rerr == nil {
		bin = resolved
	}
	// Windows 上 .cmd/.bat（npm.cmd 等全局 shim）不能直接 exec（CreateProcess
	// 不解析脚本），需经 cmd.exe /c 包装；.exe 直接跑。探测路径的 Bin 是
	// 绝对路径（.exe），ResolveCommand 找不到时会原样保留，同样直接跑。
	if isCmdShim(bin) {
		return runCmdShim(ctx, bin, cmd.Args, out, errw)
	}
	c := exec.CommandContext(ctx, bin, cmd.Args...)
	c.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
	c.Stdout = out
	c.Stderr = errw
	return c.Run()
}

// isCmdShim 报告路径是否为 Windows 批处理脚本（.cmd/.bat）。
func isCmdShim(bin string) bool {
	low := strings.ToLower(bin)
	return strings.HasSuffix(low, ".cmd") || strings.HasSuffix(low, ".bat")
}

// runCmdShim 经 cmd.exe /c 执行批处理脚本（Windows .cmd/.bat shim）。
// ponytail: 参数里的绝对路径若含空格，cmd /c 的整行解析可能拆分——升级
// 命令参数（包名/工具名）极少含空格，暂不逐参数引号化；遇真实场景再加。
func runCmdShim(ctx context.Context, script string, args []string, out, errw io.Writer) error {
	// cmd.exe /c "<script>" <args...> —— 脚本路径带引号防空格路径拆分
	cmdArgs := append([]string{"/c", script}, args...)
	c := exec.CommandContext(ctx, "cmd.exe", cmdArgs...)
	c.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
	c.Stdout = out
	c.Stderr = errw
	return c.Run()
}
