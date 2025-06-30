package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure.
	// NewApp() now correctly initializes the CPU core.
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "chip8-wails",
		Width:            1280,
		Height:           800,
		Frameless:        true, // Frameless window
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 44, G: 62, B: 80, A: 1}, // Matches bg-[#2c3e50]
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon: icon,
		},
		Menu: menu.NewMenuFromItems(
			menu.SubMenu("File", menu.NewMenuFromItems(
								menu.Text("Load ROM", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
					go app.LoadROMFromFile() // Run in a goroutine to not block the UI
				}),
				menu.Separator(),
				menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.Quit(app.ctx)
					}
				}),
			)),
			menu.SubMenu("Emulation", menu.NewMenuFromItems(
				menu.Text("Pause/Resume", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.EventsEmit(app.ctx, "menu:pause")
					}
				}),
			)),
			menu.SubMenu("Help", menu.NewMenuFromItems(
				menu.Text("About", nil, func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
							Type:    runtime.InfoDialog,
							Title:   "About CHIP-8 Emulator",
							Message: "A CHIP-8 emulator built with Wails and Svelte.",
						})
					}
				}),
			)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
