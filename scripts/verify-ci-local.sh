#!/bin/bash
# 本地逐条执行 CI test job 的命令，验证 ci.yml 里写的命令确实能跑。
# 不覆盖 Wails 打包（本地做不了三平台交叉构建）。
# 用法：bash scripts/verify-ci-local.sh
set -u
cd "$(dirname "$0")/.." || exit 1

fail=0
step() {
  local name="$1"; shift
  echo "=== $name"
  if "$@"; then
    echo "    OK"
  else
    echo "    FAILED (exit $?)"
    fail=1
  fi
  echo
}

# --- 对应 ci.yml 的 test job（去掉 setup-* 与 checkout）---
step "frontend: npm ci"        bash -c 'cd frontend && npm ci --silent'
step "frontend: npm run build" bash -c 'cd frontend && npm run build'
step "frontend: npm test"      bash -c 'cd frontend && npm test'

# gofmt 检查用与 ci.yml 相同的 bash 逻辑
step "gofmt check" bash -c '
  unformatted=$(gofmt -l .)
  if [ -n "$unformatted" ]; then
    echo "Files need gofmt:"; echo "$unformatted"; exit 1
  fi
'

step "go vet ./..."       go vet ./...
step "go test ./... -cover" go test ./... -cover
step "contract structure" go test ./internal/cli/ -run TestContract -v
step "go build ./..."     go build ./...

if [ "$fail" -ne 0 ]; then
  echo "RESULT: some steps FAILED"
  exit 1
fi
echo "RESULT: all CI test-job steps passed locally"
