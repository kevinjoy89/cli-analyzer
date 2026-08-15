package main

import (
	"context"
	"embed"
	"os"
	goruntime "runtime"
	"strings"

	"cli-analyzer/gui"
	"cli-analyzer/internal/cli"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"

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
	// Windows 使用前端自绘菜单条（原生 Win32 菜单栏无法跟随应用主题），
	// 返回 nil 以移除原生菜单栏
	if goruntime.GOOS == "windows" {
		return nil
	}
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
		app.Append(menu.Text(i18n.T("menu.about"), nil, func(_ *menu.CallbackData) {
			if appCtx != nil {
				_, _ = runtime.MessageDialog(appCtx, runtime.MessageDialogOptions{
					Title:   i18n.T("menu.about"),
					Message: "CLI Analyzer " + gui.AppVersion + "\n\n" + i18n.T("menu.aboutBody"),
					Icon:    appIcon,
				})
			}
		}))
		app.Append(menu.Separator())
		app.Append(menu.Text(i18n.T("menu.prefs"), keys.CmdOrCtrl(","), prefsCallback))
		app.Append(menu.Separator())
		app.Append(menu.Text(i18n.T("menu.hide"), keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Hide(appCtx)
			}
		}))
		app.Append(menu.Text(i18n.T("menu.quitApp"), keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Quit(appCtx)
			}
		}))
		appMenu.Append(menu.SubMenu("CLI Analyzer", app))
		appMenu.Append(menu.WindowMenu()) // standard window menu: Minimize, Zoom, …
	} else {
		file := menu.NewMenu()
		// 首选项位于退出按钮上方（Windows/Linux）
		file.Append(menu.Text(i18n.T("menu.prefs"), keys.CmdOrCtrl(","), prefsCallback))
		file.Append(menu.Separator())
		file.Append(menu.Text(i18n.T("menu.quit"), keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.Quit(appCtx)
			}
		}))
		appMenu.Append(menu.SubMenu("File", file))
	}

	help := menu.NewMenu()
	if goruntime.GOOS != "darwin" {
		// macOS 的 About 在 AppMenu role 里；Windows/Linux 没有这样的 role，
		// 所以 Help 顶部放一个 About 项——弹应用内模态框（居中 + logo），
		// 而非原生 MessageBox（不支持自定义图标、不居中）。
		help.Append(menu.Text(i18n.T("menu.about"), nil, func(_ *menu.CallbackData) {
			if appCtx != nil {
				runtime.EventsEmit(appCtx, "open-about")
			}
		}))
		help.Append(menu.Separator())
	}
	// 手动检查更新：前端监听 "check-updates" 事件，走与自动检查一致的提示流程
	help.Append(menu.Text(i18n.T("menu.checkUpdates"), nil, func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "check-updates")
		}
	}))
	help.Append(menu.Separator())
	help.Append(menu.Text(i18n.T("menu.github"), nil, func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.BrowserOpenURL(appCtx, repoURL)
		}
	}))
	help.Append(menu.Separator())
	help.Append(menu.Text(i18n.T("menu.issue"), nil, func(_ *menu.CallbackData) {
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
		case "scan", "clean", "cache", "trash", "trends", "update", "uninstall", "version", "--version", "-v", "help", "-h", "--help":
			os.Exit(cli.Run(os.Args[1:]))
		}
		// 横线开头的参数不是 GUI 标志（Wails 窗口不接收 CLI 参数）：交给
		// CLI 报"未知命令"与 usage——此前落到 GUI 分支，在无 build tags 的
		// CLI 构建下显示 "Wails applications will not build..." 的误导错误
		if strings.HasPrefix(os.Args[1], "-") {
			os.Exit(cli.Run(os.Args[1:]))
		}
		// "gui" and anything else fall through to the window.
	}

	// 原生菜单在 wails.Run 前构建（Menu 是启动选项的一部分），语言必须提前解析：
	// OnStartup 里的 SetLocale 晚于菜单构建，会导致菜单永远停留在默认 zh-CN。
	i18n.SetLocale(i18n.Resolve(config.Load().Language))

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
			// 启动时跟随系统主题；运行时由前端主题切换驱动
			// （gui.ScannerService.SetTheme 调 DWM 沉浸式暗色模式）
			Theme: windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "CLI Analyzer",
				Message: "CLI Analyzer " + gui.AppVersion + "\n\n" + i18n.T("menu.aboutBody"),
				Icon:    appIcon,
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
