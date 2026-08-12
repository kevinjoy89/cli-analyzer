#!/usr/bin/env bash
# Build the macOS app and package it into a dmg installer.
#
# Usage: ./scripts/build-dmg.sh [universal|arm64|amd64]
#   universal (default): universal binary (arm64 + x86_64)
#   arm64 | amd64      : single-architecture binary
#
# Prereqs: Go + Wails CLI + create-dmg (brew install create-dmg).
# Output: dist/CLI Analyzer-<version>[-arch].dmg
set -euo pipefail
cd "$(dirname "$0")/.."

export PATH="/opt/homebrew/bin:$PATH:$HOME/go/bin"

PLATFORM="${1:-universal}"
case "$PLATFORM" in
  universal) WAILS_PLATFORM="darwin/universal"; SUFFIX="" ;;
  arm64|amd64) WAILS_PLATFORM="darwin/$PLATFORM"; SUFFIX="-$PLATFORM" ;;
  *) echo "usage: $0 [universal|arm64|amd64]" >&2; exit 2 ;;
esac

VERSION="$(grep -m1 '"productVersion"' wails.json | sed -E 's/.*: *"([^"]+)".*/\1/')"
# 版本单一来源 + 安装来源标识（design D1/D2）：macOS 无歧义，注入仅为 version 输出统一
LDFLAGS="-X cli-analyzer/internal/buildinfo.Version=${VERSION} -X cli-analyzer/internal/buildinfo.InstallSource=dmg"
APP="build/bin/CLI Analyzer.app"
# 卷名可覆盖（VOLNAME 环境变量）：本地测试构建时用不同卷名，避免与
# 已挂载的同名 dmg 冲突（create-dmg 卸载临时卷会因重名失败）。
VOLNAME="${VOLNAME:-CLI Analyzer}"
OUT="dist/CLI Analyzer-${VERSION}${SUFFIX}.dmg"
STAGE="$(mktemp -d)"

trap 'rm -rf "$STAGE"' EXIT

echo "==> wails build ($WAILS_PLATFORM)"
wails build -platform "$WAILS_PLATFORM" -ldflags "$LDFLAGS"

echo "==> packaging $OUT"
mkdir -p dist
rm -f "$OUT" # create-dmg refuses to overwrite
cp -R "$APP" "$STAGE/"
create-dmg \
  --volname "$VOLNAME" \
  --volicon "$APP/Contents/Resources/iconfile.icns" \
  --window-size 600 400 \
  --icon-size 100 \
  --icon "CLI Analyzer.app" 175 190 \
  --app-drop-link 425 190 \
  --hide-extension "CLI Analyzer.app" \
  --no-internet-enable \
  "$OUT" "$STAGE"

# The built .app is only a staging copy; the dmg is the deliverable. Removing
# it also keeps the artifact out of Spotlight (duplicate app in search results).
rm -rf "$APP"
echo "==> done: $OUT"
