package main

import (
	"embed"
	"os"

	"cli-analyzer/gui"
	"cli-analyzer/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

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
		OnStartup:        srv.Startup,
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
