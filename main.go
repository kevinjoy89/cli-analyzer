package main

import (
	"context"
	"embed"
	"os"

	"cli-analyzer/gui"
	"cli-analyzer/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
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

// buildMenu builds the macOS menu bar. A custom menu replaces Wails' default
// one, which drops only the stock "Edit" menu; the standard App and Window
// menus are kept, and a Help menu links to the project's GitHub repo and
// issue tracker.
func buildMenu() *menu.Menu {
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())    // standard app menu: About, Hide, Quit
	appMenu.Append(menu.WindowMenu()) // standard window menu: Minimize, Zoom, …

	help := menu.NewMenu()
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

// main dispatches on argv: scan/clean/cache/version run as a CLI; gui or no
// arguments open the Wails desktop window. One binary, both interfaces.
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scan", "clean", "cache", "version", "--version", "-v", "help", "-h", "--help":
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
