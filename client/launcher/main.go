package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	initLogger()
	logInfo("main", "Запуск лаунчера версии %s", LauncherVersion)

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "Minecraft",
		Width:         560,
		Height:        540,
		Frameless:     true,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// 20% прозрачность: A=204 (80% непрозрачности), RGBA 0-255
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 254},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableFramelessWindowDecorations: true,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		logError("main", "Ошибка запуска Wails: %v", err)
		println("Error:", err.Error())
	}
}
