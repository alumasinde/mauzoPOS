package bindings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ServerStatus reports the result of an API readiness probe.
type ServerStatus struct {
	Reachable bool   `json:"reachable"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// CheckServer calls the API readiness probe and reports connectivity.
func (a *App) CheckServer() ServerStatus {
	start := time.Now()
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, a.settings.APIBaseURL+"/readyz", nil)
	if err != nil {
		return ServerStatus{Error: err.Error()}
	}
	res, err := a.http.Do(req)
	if err != nil {
		return ServerStatus{Error: "Cannot reach server: " + err.Error()}
	}
	defer func() { _ = res.Body.Close() }()

	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return ServerStatus{Error: fmt.Sprintf("Unexpected response (HTTP %d)", res.StatusCode)}
	}
	st := ServerStatus{
		Reachable: true,
		Status:    body.Status,
		Version:   body.Version,
		LatencyMS: time.Since(start).Milliseconds(),
	}
	if res.StatusCode != http.StatusOK {
		st.Error = "Server is degraded"
	}
	return st
}
