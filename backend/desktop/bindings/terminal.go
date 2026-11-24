package bindings

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"mauzo/desktop/printing"
)

// TerminalSettings is the settings screen payload.
type TerminalSettings struct {
	DeviceName      string `json:"device_name"`
	APIURL          string `json:"api_url"`
	PrinterKind     string `json:"printer_kind"` // none | windows | network | pdf | file
	PrinterName     string `json:"printer_name"` // installed printer (windows) or file path
	PrinterHost     string `json:"printer_host"`
	PrinterPort     int    `json:"printer_port"`
	AutoPrint       string `json:"auto_print"`  // default | always | never
	PaperWidth      int    `json:"paper_width"` // from the branch layout (read-only)
	AutoLockSeconds int    `json:"auto_lock_seconds"`
	BusinessType    string `json:"business_type"`
	SyncCursor      string `json:"sync_cursor"`
}

// POSLayout adapts the checkout screen to the tenant's business profile. The
// profile's pos_layout JSON lands in tenant_settings at onboarding and rides
// the sync bundle into the local settings table.
type POSLayout struct {
	BusinessType   string `json:"business_type"`
	GridColumns    int    `json:"grid_columns"`
	WeightedItems  bool   `json:"weighted_items"`
	TablesFirst    bool   `json:"tables_first"`
	AgeCheck       bool   `json:"age_check"`
	ExpiryTracking bool   `json:"expiry_tracking"`
	ShowImages     bool   `json:"show_images"`
	Numpad         bool   `json:"numpad"`
}

// Layout returns the POS layout config synced from the business profile.
func (a *App) Layout() (POSLayout, error) {
	out := POSLayout{}
	if a.db == nil {
		return out, errors.New("local database not ready")
	}
	out.BusinessType, _ = a.db.Meta(a.ctx, "business_type")
	raw, err := a.db.Setting(a.ctx, "profile.pos_layout")
	if err != nil || raw == "" {
		return out, nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return out, nil
	}
	if n, ok := m["grid_columns"].(float64); ok && n > 0 {
		out.GridColumns = int(n)
	}
	out.WeightedItems = truthy(m["weighted_items"])
	out.TablesFirst = truthy(m["tables_first"])
	out.AgeCheck = truthy(m["age_check"])
	out.ExpiryTracking = truthy(m["expiry_tracking"])
	out.ShowImages = truthy(m["show_images"])
	out.Numpad = truthy(m["numpad"])
	return out, nil
}

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

// GetSettings returns terminal preferences.
func (a *App) GetSettings() (TerminalSettings, error) {
	out := TerminalSettings{APIURL: a.settings.APIBaseURL, AutoLockSeconds: 120}
	if a.db == nil {
		return out, errors.New("local database not ready")
	}
	out.DeviceName, _ = a.db.Meta(a.ctx, "terminal_name")
	out.BusinessType, _ = a.db.Meta(a.ctx, "business_type")
	out.SyncCursor, _ = a.db.Meta(a.ctx, "last_sync")
	if v, _ := a.db.Meta(a.ctx, "api_url"); v != "" {
		out.APIURL = v
	}
	pc := a.printerConfig()
	out.PrinterKind, out.PrinterName, out.PrinterHost, out.PrinterPort = pc.Kind, pc.Name, pc.Host, pc.Port
	out.AutoPrint = autoDefault
	if v, _ := a.db.Meta(a.ctx, "printer_auto"); v != "" {
		out.AutoPrint = v
	}
	out.PaperWidth = a.kit().ParsedTemplate().Paper
	if v, _ := a.db.Meta(a.ctx, "auto_lock_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 15 {
			out.AutoLockSeconds = n
		}
	}
	return out, nil
}

// SaveSettings persists terminal preferences (device identity stays server-side).
func (a *App) SaveSettings(s TerminalSettings) error {
	if a.db == nil {
		return errors.New("local database not ready")
	}
	switch s.PrinterKind {
	case printing.KindNone, printing.KindWindows, printing.KindNetwork, printing.KindFile, printing.KindPDF:
	case "":
		s.PrinterKind = printing.KindNone
	default:
		return errors.New("unknown printer type")
	}
	switch s.AutoPrint {
	case autoDefault, autoAlways, autoNever:
	default:
		s.AutoPrint = autoDefault
	}
	if s.PrinterPort < 0 || s.PrinterPort > 65535 {
		return errors.New("printer port must be 1-65535")
	}
	pairs := map[string]string{
		"printer_kind":      s.PrinterKind,
		"printer_name":      strings.TrimSpace(s.PrinterName),
		"printer_host":      strings.TrimSpace(s.PrinterHost),
		"printer_port":      strconv.Itoa(s.PrinterPort),
		"printer_auto":      s.AutoPrint,
		"auto_lock_seconds": strconv.Itoa(s.AutoLockSeconds),
	}
	defer a.resetProbe()
	if s.APIURL != "" {
		pairs["api_url"] = s.APIURL
		a.settings.APIBaseURL = s.APIURL
	}
	if s.DeviceName != "" {
		pairs["terminal_name"] = s.DeviceName
	}
	for k, v := range pairs {
		if err := a.db.SetMeta(a.ctx, k, v); err != nil {
			return err
		}
	}
	return nil
}
