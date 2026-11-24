package bindings

import (
	"time"

	"mauzo/desktop/localdb"
)

// SyncResult reports what a sync pass pulled.
type SyncResult struct {
	OK         bool   `json:"ok"`
	At         string `json:"at"`
	Products   int    `json:"products"`
	Categories int    `json:"categories"`
	Cashiers   int    `json:"cashiers"`
	Error      string `json:"error,omitempty"`
}

// Sync pulls the catalog/pricing/cashier bundle and applies it to the local
// working set. Incremental after the first pass (server filters by cursor).
func (a *App) Sync() SyncResult {
	res := SyncResult{At: time.Now().Format("15:04:05")}
	if a.db == nil || a.token == "" {
		res.Error = "Terminal not activated"
		return res
	}
	since, _ := a.db.Meta(a.ctx, "sync_cursor")
	var b localdb.Bundle
	path := "/api/v1/pos/sync"
	if since != "" {
		path += "?since=" + urlQuery(since)
	}
	if err := a.request("GET", path, nil, &b); err != nil {
		res.Error = err.Error()
		return res
	}
	if err := a.db.ApplySync(a.ctx, b); err != nil {
		res.Error = err.Error()
		return res
	}
	_ = a.db.SetMeta(a.ctx, "sync_cursor", b.Cursor.Format(time.RFC3339Nano))
	_ = a.db.SetMeta(a.ctx, "last_sync", res.At)
	res.OK = true
	res.Products = len(b.Products)
	res.Categories = len(b.Categories)
	res.Cashiers = len(b.Cashiers)
	return res
}

// Status is the top-bar indicator state.
type Status struct {
	Online       bool   `json:"online"`
	LastSync     string `json:"last_sync"`
	Printer      string `json:"printer"` // none | ready | not_configured
	PendingSales int    `json:"pending_sales"`
	FailedSales  int    `json:"failed_sales"` // refused on upload — needs attention
}

// Status returns connectivity + sync state for the header.
func (a *App) Status() Status {
	st := Status{Printer: "not_configured"}
	if a.db == nil {
		return st
	}
	st.Online = a.online()
	st.LastSync, _ = a.db.Meta(a.ctx, "last_sync")
	st.Printer = a.printerStatus()
	var n int
	_ = a.db.GetContext(a.ctx, &n, `SELECT COUNT(*) FROM pending_sales WHERE uploaded = 0`)
	st.PendingSales = n
	st.FailedSales, _ = a.db.FailedCount(a.ctx)
	return st
}

// LastSync returns the cursor for UI display.
func (a *App) LastSync() string {
	if a.db == nil {
		return ""
	}
	v, _ := a.db.Meta(a.ctx, "last_sync")
	return v
}
