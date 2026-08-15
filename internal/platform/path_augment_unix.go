//go:build darwin || linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// augmentUserDirs 把已知的用户二进制目录并入 PATH 发现列表（去重、仅存在者）。
//
// 背景：GUI 从 Finder/Start Menu 启动时，进程 PATH 是系统最小集（不含
// /opt/homebrew/bin、~/.local/bin、go bin、npm globals 等由 shell 配置注入的
// 目录），导致 GUI 扫描漏掉大量工具（实测最小 PATH 52 个 vs 完整 267 个）。
// 这里补齐常见安装位置，使 GUI 扫描与终端扫描结果一致。
func augmentUserDirs(seen map[string]bool, out []string) []string {
	home, _ := os.UserHomeDir()
	var dirs []string
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, ".deno", "bin"),
			filepath.Join(home, ".npm-global", "bin"),
			filepath.Join(home, "npm", "bin"),
			filepath.Join(home, ".node", "bin"),
			filepath.Join(home, "Library", "pnpm"),         // macOS pnpm
			filepath.Join(home, ".local", "share", "pnpm"), // Linux pnpm
			filepath.Join(home, ".pyenv", "shims"),
			filepath.Join(home, ".asdf", "shims"),
			filepath.Join(home, ".local", "share", "mise", "shims"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".go", "bin"),
		)
	}
	// nvm 版本目录：~/.nvm/versions/node/*/bin
	if home != "" {
		if nodes, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin")); err == nil {
			dirs = append(dirs, nodes...)
		}
		// 隐藏目录下的工具 bin：~/.opencode/bin、~/.kimi-code/bin 等一级，
		// 以及 ~/.pi/agent/bin、~/.claude/local/bin 等两级——存在性过滤保证安全
		if one, err := filepath.Glob(filepath.Join(home, ".*", "bin")); err == nil {
			dirs = append(dirs, one...)
		}
		if two, err := filepath.Glob(filepath.Join(home, ".*", "*", "bin")); err == nil {
			dirs = append(dirs, two...)
		}
	}
	// Homebrew：Apple Silicon /opt/homebrew/bin，Intel /usr/local/bin
	dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin")
	// Go：`go env GOPATH`/bin（优先，能覆盖自定义 GOPATH），已解析一次并缓存
	if g := goBinDir(); g != "" {
		dirs = append(dirs, g)
	}
	// GOBIN 自定义时 go install 的二进制在 GOBIN（而非 GOPATH/bin）——
	// goBinDir 的 go env GOPATH/bin 不包含它，GUI 最小 PATH 下会漏扫
	if gb := os.Getenv("GOBIN"); gb != "" {
		dirs = append(dirs, gb)
	}

	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			continue
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			continue // 不存在则跳过（discover 也会跳过，但这里过滤更干净）
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

var (
	goBinOnce sync.Once
	goBinVal  string
)

// goBinDir 返回 `go env GOPATH`/bin；go 不在 PATH 或命令失败时回退空。
func goBinDir() string {
	goBinOnce.Do(func() {
		if goBinVal != "" {
			return
		}
		if p, err := exec.LookPath("go"); err == nil && p != "" {
			if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
				if g := strings.TrimSpace(string(out)); g != "" && g != "go" {
					goBinVal = filepath.Join(g, "bin")
				}
			}
		}
	})
	return goBinVal
}
