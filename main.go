package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	goruntime "runtime"
	"syscall"

	"github.com/saygox/dejima/internal/lock"
	"github.com/saygox/dejima/internal/procgroup"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// buildAppMenu creates the application menu.
// The Tools submenu emits Wails events so the frontend can open dialogs.
// No keyboard shortcuts are assigned to avoid conflicts with HID forwarding.
func buildAppMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	if goruntime.GOOS == "darwin" {
		// macOS requires an app menu; keep only Quit
		sub := appMenu.AddSubmenu("Dejima")
		sub.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			// Wails handles quit
		})
	}

	tools := appMenu.AddSubmenu("Tools")
	tools.AddText("Type Text...", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:typeText")
	})
	tools.AddText("Get Clipboard", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:getClipboard")
	})
	tools.AddSeparator()

	sendKey := tools.AddSubmenu("Send Key")
	for _, item := range []struct {
		label string
		keyID string
	}{
		{"Escape", "escape"},
		{"Ctrl+Alt+Delete", "ctrl-alt-del"},
		{"Alt+Tab", "alt-tab"},
		{"Alt+F4", "alt-f4"},
		{"PrintScreen", "printscreen"},
		{"Insert", "insert"},
		{"ScrollLock", "scrolllock"},
		{"Pause/Break", "pause"},
	} {
		id := item.keyID // capture for closure
		sendKey.AddText(item.label, nil, func(_ *menu.CallbackData) {
			wailsRuntime.EventsEmit(app.ctx, "menu:sendKey", id)
		})
	}

	tools.AddSeparator()
	tools.AddText("Test Clipboard Pipeline...", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:testClipboard")
	})
	tools.AddText("Video Diagnostics...", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:videoDiag")
	})
	tools.AddText("RPi Diagnostics...", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(app.ctx, "menu:rpiDiag")
	})

	return appMenu
}

func main() {
	release, err := lock.Acquire()
	if err != nil {
		log.Printf("dejima: %v", err)
		os.Exit(1)
	}
	defer release()

	_ = procgroup.Init()

	app := NewApp()

	// Catch SIGINT/SIGTERM to clean up serial port before exit.
	// Wails' OnShutdown is not called on crash or taskkill /F.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("signal received, shutting down...")
		app.shutdown(context.Background())
		os.Exit(0)
	}()

	err = wails.Run(&options.App{
		Title:     "Dejima",
		Width:     1280,
		Height:    720,
		Frameless: goruntime.GOOS == "windows",
		Menu:      buildAppMenu(app),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
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
