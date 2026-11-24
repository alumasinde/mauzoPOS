// Package bindings contains the Go methods exposed to the POS Svelte UI via
// Wails (window.go.bindings.App.*). Keep each concern in its own file:
// app.go (lifecycle/info), server.go (API connectivity), activation.go,
// sync.go, auth.go, catalog.go, cart.go, terminal.go.
package bindings

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"mauzo/desktop/localdb"
	"mauzo/desktop/settings"
	"mauzo/pkg/barcode"
)

// App is the Wails-bound object whose methods are callable from the UI.
type App struct {
	ctx      context.Context
	version  string
	settings settings.Settings
	http     *http.Client

	mu      sync.Mutex // guards session + fails
	db      *localdb.DB
	token   string // device bearer token
	session *Session
	scale   barcode.ScaleConfig
	fails   map[string]*pinFail
	print   printState
}

// Session is the signed-in cashier.
type Session struct {
	CashierID   string   `json:"cashier_id"`
	Name        string   `json:"name"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
	Since       string   `json:"since"`
}

type pinFail struct {
	count int
	until time.Time
}

// NewApp builds the bindings object for the Wails app.
func NewApp(version string, s settings.Settings) *App {
	return &App{
		ctx:      context.Background(),
		version:  version,
		settings: s,
		http:     &http.Client{Timeout: s.HTTPTimeout},
		scale:    barcode.DefaultScale(),
		fails:    map[string]*pinFail{},
	}
}

// Startup is called by Wails when the window is ready. It opens the local
// database (creating it on first run) and restores the device token.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	dir, err := settings.DataDir()
	if err != nil {
		return
	}
	db, err := localdb.Open(ctx, filepath.Join(dir, "terminal.db"))
	if err != nil {
		return
	}
	a.db = db
	if tok, err := db.Meta(ctx, "device_token"); err == nil {
		a.token = tok
	}
	// Scale-label prefixes are a tenant setting: pos.scale_barcode_prefix.
	if raw, err := db.Setting(ctx, "pos.scale_barcode_prefix"); err == nil && raw != "" {
		if cfg, ok := parsePrefixes(raw); ok {
			a.scale = cfg
		}
	}
}

// Quit exits the app — the frameless kiosk build has no close button, so the
// More sheet calls this (Alt+F4 also still works).
func (a *App) Quit() { runtime.Quit(a.ctx) }

// TerminalInfo is the terminal metadata shown in the UI header.
type TerminalInfo struct {
	Version      string `json:"version"`
	APIBaseURL   string `json:"api_base_url"`
	Activated    bool   `json:"activated"`
	Terminal     string `json:"terminal"`
	Branch       string `json:"branch"`
	Tenant       string `json:"tenant"`
	BusinessType string `json:"business_type"`
}

// Info returns terminal information for the UI header.
func (a *App) Info() TerminalInfo {
	info := TerminalInfo{Version: a.version, APIBaseURL: a.settings.APIBaseURL}
	if a.db == nil {
		return info
	}
	info.Activated = a.token != ""
	info.Terminal, _ = a.db.Meta(a.ctx, "terminal_name")
	info.Branch, _ = a.db.Meta(a.ctx, "branch_name")
	info.Tenant, _ = a.db.Meta(a.ctx, "tenant_name")
	info.BusinessType, _ = a.db.Meta(a.ctx, "business_type")
	return info
}
