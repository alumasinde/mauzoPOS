package bindings

// Heartbeat pings the API, refreshing the terminal's last_seen. The UI calls
// it on an interval; the result doubles as the online indicator.
func (a *App) Heartbeat() bool {
	var out map[string]any
	err := a.request("POST", "/api/v1/pos/heartbeat", nil, &out)
	return err == nil
}

// PendingSales is the offline-upload queue depth for the status bar.
func (a *App) PendingSales() int {
	if a.db == nil {
		return 0
	}
	n, err := a.db.PendingCount(a.ctx)
	if err != nil {
		return 0
	}
	return n
}
