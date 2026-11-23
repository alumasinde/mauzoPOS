package bindings

import (
	"encoding/json"
	"errors"
	"os"

	"mauzo/desktop/localdb"
	"mauzo/pkg/barcode"
)

// ActivateRequest is the activation form: API URL, one-time code and a name
// for this device.
type ActivateRequest struct {
	APIURL     string `json:"api_url"`
	Code       string `json:"code"`
	DeviceName string `json:"device_name"`
	DeviceID   string `json:"device_id"`
}

// BoundTerminal describes the terminal this device was activated as.
type BoundTerminal struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Branch       string `json:"branch"`
	Tenant       string `json:"tenant"`
	Slug         string `json:"slug"`
	BusinessType string `json:"business_type"`
}

// ActivateResult mirrors the API response.
type ActivateResult struct {
	Token    string        `json:"token"`
	Terminal BoundTerminal `json:"terminal"`
}

// Activate exchanges the one-time code for a device token, stores it in the
// local database and returns the bound terminal descriptor.
func (a *App) Activate(req ActivateRequest) (ActivateResult, error) {
	var out ActivateResult
	if a.db == nil {
		return out, errors.New("local database not ready")
	}
	if req.APIURL != "" {
		a.settings.APIBaseURL = req.APIURL
	}
	if req.DeviceID == "" {
		req.DeviceID = deviceFingerprint()
	}
	err := a.request("POST", "/api/v1/pos/activate", map[string]string{
		"code": req.Code, "device_name": req.DeviceName, "device_id": req.DeviceID,
	}, &out)
	if err != nil {
		return out, err
	}
	// A device may have run a different terminal (or tenant) before — wipe the
	// previous working set so nothing stale survives into the new identity.
	if err := a.db.ResetForActivation(a.ctx); err != nil {
		return out, err
	}
	a.token = out.Token
	meta := map[string]string{
		"device_token":  out.Token,
		"device_id":     req.DeviceID,
		"terminal_id":   out.Terminal.ID,
		"terminal_name": out.Terminal.Name,
		"branch_name":   out.Terminal.Branch,
		"tenant_name":   out.Terminal.Tenant,
		"tenant_slug":   out.Terminal.Slug,
		"business_type": out.Terminal.BusinessType,
		"api_url":       a.settings.APIBaseURL,
	}
	for k, v := range meta {
		if err := a.db.SetMeta(a.ctx, k, v); err != nil {
			return out, err
		}
	}
	return out, nil
}

// deviceFingerprint is a stable per-machine identifier (best effort — it only
// needs to distinguish tills, not survive a reinstall).
func deviceFingerprint() string {
	host, _ := os.Hostname()
	return host + "-" + machineGUID()
}

// machineGUID reads the Windows machine GUID; empty on other platforms.
func machineGUID() string {
	return readMachineGUID()
}

// parsePrefixes decodes the tenant setting JSON array of scale prefixes.
func parsePrefixes(raw string) (barcode.ScaleConfig, bool) {
	var list []string
	if json.Unmarshal([]byte(raw), &list) != nil || len(list) == 0 {
		return barcode.ScaleConfig{}, false
	}
	cfg := barcode.DefaultScale()
	cfg.Prefixes = list
	return cfg, true
}

var _ = localdb.Bundle{} // referenced by sync.go
