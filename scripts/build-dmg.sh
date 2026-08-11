#!/usr/bin/env bash
# Build the universal macOS app and package it into a dmg installer.
#
# Prereqs: Go + Wails CLI + create-dmg (brew install create-dmg).
# Output: dist/CLI Analyzer-<version>.dmg (universal arm64 + x86_64).
set -euo pipefail
cd "$(dirname "$0")/.."

export PATH="/opt/homebrew/bin:$PATH:$HOME/go/bin"

VERSION="$(grep -m1 '"productVersion"' wails.json | sed -E 's/.*: *"([^"]+)".*/\1/')"
APP="build/bin/CLI Analyzer.app"
OUT="dist/CLI Analyzer-${VERSION}.dmg"
STAGE="$(mktemp -d)"

trap 'rm -rf "$STAGE"' EXIT

echo "==> wails build (darwin/universal)"
wails build -platform darwin/universal

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
