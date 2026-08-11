package main

import (
	"context"
	"embed"
	"os"
	goruntime "runtime"

	"cli-analyzer/gui"
	"cli-analyzer/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// repoURL is the project's GitHub home; the Help menu links to it.
const repoURL = "https://github.com/kevinjoy89/cli-analyzer"

// appCtx is set in OnStartup and used by menu callbacks (e.g. open a URL).
var appCtx context.Context

// buildMenu builds the menu bar. On macOS a custom menu replaces Wails' default
// one (dropping only the stock "Edit" menu) and keeps the standard App and
// Window menus. Windows/Linux have no app/window role menus, so they get a
// File (Quit) menu instead. Every platform shares the Help menu linking to the
// project's GitHub repo and issue tracker.
func buildMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// prefsCallback 通知前端打开首选项面板（前端监听 "open-prefs" 事件）
	prefsCallback := func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "open-prefs")
		}
	}

	if goruntime.GOOS == "darwin" {
		// 手动构建应用菜单（App Menu）：AppMenuRole 无法在 About 与 Quit 之间
		// 插入"首选项"，故用子菜单重建，顶层标题即应用名"CLI Analyzer"
		app := menu.NewMenu()
		app.Append(menu.Text("About CLI Analyzer", nil, func(_ *menu.CallbackData) {
			if appCtx != nil {
				_, _ = runtime.MessageDialog(appCtx, runtime.MessageDialogOptions{
					Title:   "About CLI Analyzer",
					Message: "CLI Analyzer " + gui.AppVersion + "\n\n扫描并清理 CLI 工具的磁盘占用。",
					Icon:    appIcon,
				})
			}
		}))
		app.Append(menu.Separator())
		app.Append(menu.Text("首选项…", keys.CmdOrCtrl(","), prefsCallback))
		app.Append(menu.Separator())
		app.Append(menu.Text("Hide CLI Analyzer", keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Hide(appCtx)
			}
		}))
		app.Append(menu.Text("Quit CLI Analyzer", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Quit(appCtx)
			}
		}))
		appMenu.Append(menu.SubMenu("CLI Analyzer", app))
		appMenu.Append(menu.WindowMenu()) // standard window menu: Minimize, Zoom, …
	} else {
		file := menu.NewMenu()
		// 首选项位于退出按钮上方（Windows/Linux）
		file.Append(menu.Text("首选项…", keys.CmdOrCtrl(","), prefsCallback))
		file.Append(menu.Separator())
		file.Append(menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Quit(appCtx)
			}
		}))
		appMenu.Append(menu.SubMenu("File", file))
	}

	help := menu.NewMenu()
	if goruntime.GOOS != "darwin" {
		// macOS's About lives in the AppMenu role; Windows/Linux have no such
		// role, so surface the About dialog at the top of Help.
		help.Append(menu.Text("About CLI Analyzer", nil, func(_ *menu.CallbackData) {
			if appCtx != nil {
				_, _ = runtime.MessageDialog(appCtx, runtime.MessageDialogOptions{
					Title:   "About CLI Analyzer",
					Message: "CLI Analyzer " + gui.AppVersion + "\n\n扫描并清理 CLI 工具的磁盘占用。",
					Icon:    appIcon,
				})
			}
		}))
		help.Append(menu.Separator())
	}
	help.Append(menu.Text("GitHub Repository", nil, func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.BrowserOpenURL(appCtx, repoURL)
		}
	}))
	help.Append(menu.Separator())
	help.Append(menu.Text("Report an Issue", nil, func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.BrowserOpenURL(appCtx, repoURL+"/issues/new")
		}
	}))
	appMenu.Append(menu.SubMenu("Help", help))

	return appMenu
}

// main dispatches on argv: scan/clean/cache/trash/trends/version run as a CLI;
// gui or no arguments open the Wails desktop window. One binary, both interfaces.
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scan", "clean", "cache", "trash", "trends", "version", "--version", "-v", "help", "-h", "--help":
			os.Exit(cli.Run(os.Args[1:]))
		}
		// "gui" and anything else fall through to the window.
	}

	srv := gui.NewScannerService()
	err := wails.Run(&options.App{
		Title:  "CLI Analyzer",
		Width:  1180,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 26, B: 32, A: 1},
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
			srv.Startup(ctx)
		},
		Menu: buildMenu(),
		Bind: []interface{}{
			srv,
		},
		Windows: &windows.Options{
			// 强制 Windows 标题栏与原生菜单栏使用沉浸式暗色模式，
			// 与应用内的暗色 UI 一致（默认 SystemDefault 会跟随系统浅色主题）
			Theme: windows.Dark,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "CLI Analyzer",
				Message: "CLI Analyzer " + gui.AppVersion + "\n\n扫描并清理 CLI 工具的磁盘占用。",
				Icon:    appIcon,
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
