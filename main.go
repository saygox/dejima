package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

// buildAppMenu creates a minimal application menu that does not
// intercept Cmd+C / Cmd+V / Cmd+A etc., so those key combos
// reach the frontend and get forwarded to the remote machine.
func buildAppMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	if runtime.GOOS == "darwin" {
		// macOS requires an app menu; keep only Quit
		sub := appMenu.AddSubmenu("KVM-Like")
		sub.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			// Wails handles quit
		})
	}

	return appMenu
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "KVM-Like",
		Width:  1280,
		Height: 720,
		Menu:   buildAppMenu(),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Mac: &mac.Options{
			Preferences: &mac.Preferences{
				FullscreenEnabled: mac.Enabled,
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
