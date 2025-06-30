package main

import (
	"embed"
	"encoding/json" // Import the JSON package
	"log"           // Import log

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// --- NEW: Embed wails.json to access app info ---
//
//go:embed wails.json
var wailsJSON []byte

func main() {
	app := NewApp()

	var wailsInfo WailsInfo // Using the struct defined in app.go

	err := json.Unmarshal(wailsJSON, &wailsInfo)
	if err != nil {
		log.Fatalf("Failed to parse wails.json: %v", err)
	}
	app.wailsInfo = wailsInfo // Assign the parsed info

	// Create application with options
	err = wails.Run(&options.App{
		Title:     wailsInfo.Info.ProductName, // Use ProductName for the title
		Width:     1280,
		Height:    800,
		Frameless: true, // Frameless window
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
				menu.Text("Load ROM...", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
					go app.LoadROMFromFile()
				}),
				// --- NEW MENU ITEM ---
				menu.Text("Save State", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:savestate")
				}),
				menu.Separator(),
				menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
					runtime.Quit(app.ctx)
				}),
			)),
			menu.SubMenu("Emulation", menu.NewMenuFromItems(
				menu.Text("Pause/Resume", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:pause")
				}),
				// --- NEW MENU ITEMS ---
				menu.Text("Soft Reset", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:softreset")
				}),
				menu.Text("Hard Reset", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:hardreset")
				}),
			)),
			menu.SubMenu("Help", menu.NewMenuFromItems(
				// --- NEW MENU ITEM ---
				menu.Text("Visit GitHub", nil, func(_ *menu.CallbackData) {
					app.OpenGitHubLink()
				}),
				menu.Separator(),

				menu.Text("About", nil, func(_ *menu.CallbackData) {
					app.ShowAboutDialog()
				}),
			)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
