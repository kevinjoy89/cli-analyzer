#!/usr/bin/env bash
# 一键跑全量检查与测试：gofmt + go vet + Go 单测 + 前端单测。
# 用法：./scripts/test-all.sh（从任意目录可运行）
set -euo pipefail
cd "$(dirname "$0")/.."

# 兼容本机 GOROOT 指向旧工具链导致 go 命令版本不匹配的场景：
# 让 go 从自身二进制路径推断 GOROOT（仅影响本脚本子进程）
unset GOROOT 2>/dev/null || true

echo "==> gofmt 检查"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "以下文件需 gofmt:"
  echo "$unformatted"
  exit 1
fi
echo "gofmt 干净"

echo "==> go vet ./..."
go vet ./...

echo "==> go test ./..."
go test ./...

echo "==> 前端单测 (vitest)"
(cd frontend && npm test)

echo
echo "全部通过 ✓"
