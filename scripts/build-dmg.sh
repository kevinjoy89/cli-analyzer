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
APP="build/bin/CLI Analyzer.app"
OUT="dist/CLI Analyzer-${VERSION}${SUFFIX}.dmg"
STAGE="$(mktemp -d)"

trap 'rm -rf "$STAGE"' EXIT

echo "==> wails build ($WAILS_PLATFORM)"
wails build -platform "$WAILS_PLATFORM"

echo "==> packaging $OUT"
mkdir -p dist
rm -f "$OUT" # create-dmg refuses to overwrite
cp -R "$APP" "$STAGE/"
create-dmg \
  --volname "CLI Analyzer" \
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
