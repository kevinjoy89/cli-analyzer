#!/usr/bin/env bash
# 发布前全量检查（docs/release-process.md 检查清单的脚本化）。
# 用法：./scripts/release/check.sh [tag版本号，如 0.3.9]
set -euo pipefail
cd "$(dirname "$0")/../.."

echo "==> 基础检查（gofmt + vet + go test + 前端测试）"
./scripts/test-all.sh

echo "==> 前端类型检查"
(cd frontend && npx tsc --noEmit)

echo "==> 三平台交叉编译"
for os in windows linux darwin; do
  echo "    GOOS=$os"
  GOOS=$os go build ./...
done

# 版本一致性（传 tag 时校验 wails.json productVersion）
if [ $# -ge 1 ]; then
  ver="$1"
  grep -q "\"productVersion\": \"$ver\"" wails.json \
    || { echo "FAIL: wails.json productVersion != $ver"; exit 1; }
  echo "==> wails.json productVersion = $ver ✓"
fi

echo
echo "全部通过 ✓ —— 可以打 tag 推送"
