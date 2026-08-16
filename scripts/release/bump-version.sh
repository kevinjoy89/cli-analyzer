#!/usr/bin/env bash
# 更新 wails.json 的 productVersion（发布流程第 2 步的脚本化）。
# 用法：./scripts/release/bump-version.sh 0.3.9
set -euo pipefail
cd "$(dirname "$0")/../.."

ver="${1:-}"
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || {
  echo "非法版本号：$ver（期望如 0.3.9 或 0.3.7.2）"; exit 1
}

# -i.bak 兼容 BSD(macOS)/GNU(Linux) sed；写后删除备份
sed -i.bak "s/\"productVersion\": \"[^\"]*\"/\"productVersion\": \"$ver\"/" wails.json
rm -f wails.json.bak

grep -q "\"productVersion\": \"$ver\"" wails.json \
  || { echo "FAIL: 版本号未写入 wails.json"; exit 1; }
echo "wails.json productVersion → $ver"
