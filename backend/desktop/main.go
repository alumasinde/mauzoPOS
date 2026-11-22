// Command desktop is the Wails host for the Mauzo POS Windows terminal (.exe).
// Build with `wails build` from this directory.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"mauzo/desktop/bindings"
	"mauzo/desktop/settings"
)

// Built by `pnpm build` in frontend/apps/pos (outDir points here).
//
//go:embed all:frontend/dist
var assets embed.FS

// Version is set at build time: wails build -ldflags "-X main.Version=1.2.3"
var Version = "dev"

func main() {
	cfg := settings.Load()
	app := bindings.NewApp(Version, cfg)

	err := wails.Run(&options.App{
		Title:            "Mauzo POS",
		Width:            1366,
		Height:           768,
		MinWidth:         1024,
		MinHeight:        700,
		WindowStartState: options.Maximised,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.Startup,
		Bind:             []any{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
