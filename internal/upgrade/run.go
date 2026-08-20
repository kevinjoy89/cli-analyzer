package upgrade

import (
	"context"
	"io"
	"os"
	"os/exec"
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
	// Windows .cmd/.bat shim（npm 全局等）经 cmd.exe /c 执行（CreateProcess
	// 不解析批处理脚本）；探测路径的 Bin 是绝对路径（.exe），直接跑。
	bin, args := cmdexec.WrapShim(bin, cmd.Args)
	c := exec.CommandContext(ctx, bin, args...)
	c.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
	c.Stdout = out
	c.Stderr = errw
	return c.Run()
}
