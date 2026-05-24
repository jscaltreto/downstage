package main

import (
	"github.com/jscaltreto/downstage/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	// Create an instance of the app structure
	app := desktop.NewApp()

	// Build the native menu before wails.Run so the menu bar is present
	// from the first frame. The menu's click callbacks close over the
	// app struct and reference app.ctx, which isn't live until OnStartup
	// — but the callbacks fire only after the user interacts with the
	// menu, which Wails guarantees is after OnStartup has run. The app
	// also stashes the built menu via SetInitialMenu so the
	// SetDisabledCommands rebuild path has a starting tree.
	builtMenu := desktop.BuildMenu(app, nil)
	app.SetInitialMenu(builtMenu)

	// Pick the initial window size from persisted state if it looks
	// sane. Fall back to the default when no state exists, when the
	// stored dimensions are suspiciously small, or when reading fails.
	// Only size is persisted — see WindowState's comment for why
	// position and maximize state are deliberately omitted.
	width, height := 1024, 768
	if ws, err := app.GetWindowState(); err == nil {
		if ws.Width >= 400 && ws.Height >= 400 {
			width = ws.Width
			height = ws.Height
		}
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Downstage Write",
		Width:  width,
		Height: height,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Menu:             builtMenu,
		OnStartup:        app.Startup,
		OnBeforeClose:    app.BeforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
