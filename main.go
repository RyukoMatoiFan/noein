package main

import (
	"embed"
	"log"
	"noein/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create application instance
	application := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Noein - Frame-Perfect Video Editor",
		Width:  1600,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: application.Startup,
		Bind: []interface{}{
			application,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			WebviewBrowserPath:   "",
		},
		EnableDefaultContextMenu: true,
		EnableFraudulentWebsiteDetection: false,
	})

	if err != nil {
		log.Fatal(err)
	}
}
