// Package bindings contains the Go methods exposed to the POS Svelte UI via
// Wails (window.go.bindings.App.*). Keep each concern in its own file:
// app.go (lifecycle/info), server.go (API connectivity), later cart.go,
// scanner.go, printer.go, shift.go ...
package bindings

import (
	"context"
	"net/http"

	"mauzo/desktop/settings"
)

// App is the Wails-bound object whose methods are callable from the UI.
type App struct {
	ctx      context.Context
	version  string
	settings settings.Settings
	http     *http.Client
}

// NewApp builds the bindings object for the Wails app.
func NewApp(version string, s settings.Settings) *App {
	return &App{
		ctx:      context.Background(),
		version:  version,
		settings: s,
		http:     &http.Client{Timeout: s.HTTPTimeout},
	}
}

// Startup is called by Wails when the window is ready.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

// TerminalInfo is the static terminal metadata shown in the UI header.
type TerminalInfo struct {
	Version    string `json:"version"`
	APIBaseURL string `json:"api_base_url"`
	Activated  bool   `json:"activated"`
}

// Info returns static terminal information for the UI header.
func (a *App) Info() TerminalInfo {
	return TerminalInfo{
		Version:    a.version,
		APIBaseURL: a.settings.APIBaseURL,
		Activated:  false, // terminal activation: Phase 6
	}
}
