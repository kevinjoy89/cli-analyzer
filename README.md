<h1 align="center">CLI Analyzer</h1>

<p align="center">
  Find out which CLI tools are eating your disk — and decide what to reclaim.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-blue" alt="Go 1.26">
  <img src="https://img.shields.io/badge/Wails-v2-8A2BE2" alt="Wails v2">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License: MIT">
</p>

<p align="center">
  <b>English</b> · <a href="README.zh-CN.md">简体中文</a>
</p>

---

Cleanup tools like CleanMyMac show how much disk your **desktop apps** use — but they are blind to **CLI tools**. A CLI tool's footprint is scattered across hidden dotfiles and XDG directories (`~/.claude`, `~/.npm`, `~/.cache/*`), so it gets lumped under "Other" or ignored entirely.

CLI Analyzer scans every CLI tool installed on your machine and attributes its **total** disk footprint (executable + package dirs + data/cache dirs). Every attributed directory is **labeled** (cache / config / data / old version / …) — but whether it should be removed is **your call**: the app never judges "safe to delete" on your behalf. One binary, two interfaces: a terminal CLI and a native dark-mode GUI (Wails v2).

<!-- Screenshots pending: restore this table once docs/screenshots/app-light-en.png and app-dark-en.png exist -->

| Light theme | Dark theme |
|---|---|
| ![Light](./docs/screenshots/app-light-en.png) | ![Dark](./docs/screenshots/app-dark-en.png) |

## Features

- **Detection** — enumerates `$PATH` executables, resolves symlinks, and classifies each by install source (versioned installer / brew formula / npm package / pipx / pyenv shim / go / cargo / other). The Node.js runtime family (node / npm / npx / corepack / node-gyp) is merged into one `nodejs` entry instead of showing every command in the same install dir as a separate tool (common on Windows)
- **Attribution** — total footprint = executable + package dir (`Cellar`, `node_modules`, `versions/…`) + platform data dirs (`~/.cache/<name>`, `~/.config/<name>`, `~/.local/share/<name>`, macOS `~/Library/*`, Windows `%APPDATA%`…)
- **Labeled attribution, your call** — every attributed directory is labeled by kind (cache / config / data / old version / backup / toolchain…). The former SAFE/USER hard gate is gone: nothing is blocked, and nothing is deleted without your explicit choice. Default disposal still goes through the built-in trash (recoverable); permanent deletion is always an explicit `--permanent`/confirm choice
- **Built-in trash** — deletions go to the app's own trash first (same filesystem, instant), recoverable for 7 days (configurable in Preferences); expired items move to the OS trash or are deleted permanently
- **Restore** — restore a trashed item from the GUI trash panel or `cli-analyzer trash restore`; `clean --permanent` skips the built-in trash entirely
- **Usage trends** — every scan appends a snapshot; view total/actionable space over time (SVG chart) and the top actionable growers, with a bell reminder when a tool's actionable space exceeds a configurable limit (click it to list the tools and jump to one)
- **Tree drill-down** — expand an actionable item to its one-level child dirs (`~/.npm` → `_cacache` 10G / `_npx` 764M) and dispose of only the selected children. Sub-path deletion passes the same integrity guards (must be a child of an already-scanned, attributed parent)
- **Built-in updater** — checks GitHub Releases on startup for a new version (toggleable in Preferences, with a 4h rate-limit cache); prompts to download with a progress bar, verifies the SHA256 checksum, then opens the installer for you to complete manually. CLI: `cli-analyzer update check`. Note: within 4h of a release, a cached check may not yet report it — the prompt appears once the cache refreshes
- **Official uninstall** — don't know the uninstall command? The tool detects the install source (brew / npm / pipx / cargo…), shows the standard command and can run it for you, then detects leftover config/cache dirs. Residue disposal defaults to the built-in trash (recoverable); permanently deleting residue is an explicit choice. System-critical tools are blocked. CLI: `cli-analyzer uninstall <tool>`
- **Localized UI** — Simplified Chinese / Traditional Chinese / English; follows the system language by default, switchable in Preferences → Language (applies instantly; macOS native menu applies after restart)
- **Two interfaces** — CLI (`scan` / `clean` / `cache` / `trash` / `trends` / `update` / `version`) + native GUI
- **Unattributed data dirs** — top-level dirs under the data roots that no tool claims (leftovers of removed or never-on-PATH tools) appear in a collapsible "Unattributed data" section; USER-level, move-to-trash only (recoverable), filtered by the non-CLI exclusion system (GUI apps, their data and command-line companions are out of scope)
- **Health probing** — tools with an unknown version are probed in the background (`--version` / `-V` / `--help`, 3s timeout, result cache keyed by binary path/size/mtime, GBK-safe on Windows); failures degrade silently to `—`

## Installation

> **Releases**: installers for macOS / Windows / Linux are published on the [Releases page](https://github.com/kevinjoy89/cli-analyzer/releases). Building from source also works.
>
> **Updates**: starting from the release that ships this feature, the app checks for new versions automatically on startup (disable in Preferences → Update) and prompts you to download when one is available. Downloads land in `~/Downloads` and are verified against the published SHA256 checksums before the installer is opened — installation itself stays with the system flow. macOS builds are currently **unsigned**, so Gatekeeper may require right-click → Open on first launch of a downloaded copy.

Requirements: **Go ≥ 1.26** and **npm**.

```bash
brew install go
export PATH="/opt/homebrew/opt/go/bin:$HOME/go/bin:$PATH"
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0

# CLI only
go build -o bin/cli-analyzer .

# GUI app (the same binary also works as a CLI)
wails build        # → build/bin/cli-analyzer
```

Cross-platform installers (macOS dmg / Windows installer / Linux deb + AppImage) are covered in [docs/packaging.md](docs/packaging.md); one-shot macOS packaging via `./scripts/build-dmg.sh`.

> **Mirror note**: This project does **not** depend on any specific Go mirror — the default proxy works everywhere, no config needed. Users on mainland-China networks who hit download timeouts can run once:
> `go env -w GOPROXY=https://goproxy.cn,direct`
> That setting lives in your own environment (`~/.config/go/env`), is not committed, and does not affect other developers; restore with `go env -u GOPROXY`.

## Usage

```bash
cli-analyzer scan                    # scan (cached when nothing changed; auto-rescans when files changed)
cli-analyzer scan --refresh --json   # force a rescan, JSON output (includes unattributed + probed versions)
cli-analyzer clean                   # interactive, per-item disposal (into built-in trash)
cli-analyzer clean --dry-run --all   # show the plan only, delete nothing
cli-analyzer clean --yes kimi        # dispose of all cleanup-kind items for one tool
cli-analyzer clean --all --include-data  # batch also includes config/data/state items
cli-analyzer clean --permanent       # delete immediately, skip the built-in trash
cli-analyzer trash list              # list built-in trash items
cli-analyzer trash restore <id>      # restore an item to its original path
cli-analyzer trash empty             # empty the built-in trash (permanent)
cli-analyzer trends [days]           # usage trends over the last N days (default 30)
cli-analyzer update check           # check for a new version (exit: 0 up-to-date / 2 update / 1 error)
cli-analyzer uninstall <tool>       # standard uninstall + residue cleanup (built-in trash by default)
cli-analyzer uninstall <tool> --permanent  # permanently delete residue (unrecoverable)
cli-analyzer version                # show version and install source (e.g. 0.3.8 (darwin, dmg))
cli-analyzer cache --clear           # clear the scan cache
cli-analyzer                         # open the GUI
```

Example scan output:

```
Tool   Cmd   Total   Actionable   Install   Source
nodejs 5     10.5 GB  10.5 GB     89.7 MB   nodejs
opencode -   8.2 GB   0 B         8.2 GB    other
uv     -     7.3 GB   7.2 GB      57.0 MB   other
codex  -     2.0 GB   288.7 MB    1.8 GB    other
pip    -     1.1 GB   1.1 GB      0 B       pip
go     -     1.0 GB   753.7 MB    282.3 MB  go
pyenv  -     759.8 MB 0 B         759.8 MB  pyenv
...
Total  -     31.5 GB  19.9 GB     11.7 GB   -
```

(The Node.js runtime family — node / npm / npx / corepack / node-gyp — is merged into a single `nodejs` entry; its detail panel lists every bundled command and binary path, and the `~/.npm` cache stays an actionable cache item.)

**Disposal model**: the app attributes and labels each directory (cache / config / data / old version / backup / toolchain…); deletion is your call. `clean --all` batches only cleanup-kind items (cache / old-version / backup / download) by default — config/data/state need an explicit `--include-data`. Old-version disposal always keeps the current version (e.g. the symlink target for `claude`); the cleaner's integrity guards still refuse system roots, `..` paths, the trash root, and the current version path.

**Deferred deletion**: disposed items are moved into the app's built-in trash first — same filesystem, instant, and recoverable. They stay there for the retention window (default 7 days, configurable in Preferences) and are then purged: by default into the OS trash, or permanently if configured. The GUI status bar shows the trash occupancy and the earliest expiry, so "disposed" and "space released" stay distinct. Use `clean --permanent` to bypass the built-in trash.

### Disposal guidance — labeled, not judged

The app no longer auto-deletes anything, so nothing is "refused". These directories keep their risk labels as guidance for when you pick them yourself:

| Path | What to know before disposing |
|---|---|
| `~/.cache/opencode`, `~/.cache/mimocode` | named "cache" but are actually **plugin install dirs** of opencode-family tools (MCP servers, language servers, skills, `package.json` manifests) |
| pyenv / rustup old toolchains (non-current) | pip-installed commands hardcode the interpreter path in their shebang (e.g. 16 commands → `~/.pyenv/versions/3.6.15/bin/python3.6`); projects may pin a toolchain via `rust-toolchain.toml` |
| brew non-current Cellar versions | may be depended on by other formulae (e.g. fontconfig); `brew cleanup` handles them safely |
| `~/.cache/codex-runtimes/codex-primary-runtime` | the runtime codex is actively using; only `codex-runtime-install-*` staging dirs are meant to be cleaned |

## Cross-platform

- The scan core is pure Go stdlib — `GOOS=linux/windows go build ./...` cross-compiles directly
- **macOS**: additionally scans `~/Library/{Caches, Application Support, Preferences}`
- **Linux**: XDG dirs; GUI build needs `libwebkit2gtk-4.1`, `libgtk-3`
- **Windows**: `%APPDATA%` / `%LOCALAPPDATA%`; GUI requires building on a Windows host (WebView2)

## Known limitations (v1)

- Hard-linked files are counted once per path (sizes slightly inflated); future: inode-based dedup
- Scan results are a snapshot; run `--refresh` (or hit "Rescan" in the GUI) after files change
- Startup/`scan` skip a full rescan when nothing changed (mtime-based fingerprint); in-place file edits inside a data dir may not trigger a rescan — "Rescan" or `scan --refresh` always forces one

## Project layout

```
main.go             # argv dispatch: scan/clean/cache/trash/trends/version → CLI; otherwise Wails GUI
gui/service.go      # Wails bindings (the only file importing wails)
internal/scanner/   # discover → classify → attribute → cleanability (pure core)
internal/rules/     # curated + generic attribution rules
internal/platform/  # per-OS data roots & executable detection (build tags)
internal/disk/      # parallel directory size measurement (no du dependency)
internal/cleaner/   # integrity guards + deferred deletion (built-in trash)
internal/trash/     # built-in trash: defer/restore/sweep + per-OS system trash
internal/config/    # local config (retention, expire action, reminder threshold)
internal/history/   # scan snapshots in SQLite for usage trends
internal/cli/       # scan / clean / cache / trash / trends subcommands
```

## Contributing

Bug reports and feature ideas are welcome — open an [issue](https://github.com/kevinjoy89/cli-analyzer/issues). Pull requests are appreciated.

## License

[MIT](LICENSE) © 2026 kevinjoy89
