package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Application menu
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())
	appMenu.Append(menu.EditMenu())
	appMenu.Append(menu.WindowMenu())

	// Create application with options supporting background tray daemon and dynamic mini prompt
	err := wails.Run(&options.App{
		Title:             "Candak Download Manager",
		Width:             1080,
		Height:            720,
		MinWidth:          800,
		MinHeight:         500,
		HideWindowOnClose: true,
		Menu:              appMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 250, B: 250, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			app.ClosePromptWindow(false, "", 0)
			return false
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.candak.downloadmanager",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				app.OpenMainWindow()
			},
		},
		EnableDefaultContextMenu: true,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
