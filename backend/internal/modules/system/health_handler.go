package system

import (
	"context"
	"net/http"
	"time"

	"mauzo/internal/core/httpx"
)

// Pinger is satisfied by *sqlx.DB (and later the Redis client).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Handler serves the liveness and readiness probes.
type Handler struct {
	version string
	checks  map[string]Pinger
}

// NewHandler builds a probe handler with named dependency checks.
func NewHandler(version string, checks map[string]Pinger) *Handler {
	return &Handler{version: version, checks: checks}
}

type healthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Live reports that the process is up. It never touches dependencies.
func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, healthResponse{Status: "ok", Version: h.version})
}

// Ready reports whether dependencies (database, ...) are reachable.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	res := healthResponse{Status: "ok", Version: h.version, Checks: map[string]string{}}
	status := http.StatusOK
	for name, p := range h.checks {
		if err := p.PingContext(ctx); err != nil {
			res.Checks[name] = "unavailable"
			res.Status = "degraded"
			status = http.StatusServiceUnavailable
			continue
		}
		res.Checks[name] = "ok"
	}
	httpx.JSON(w, status, res)
}
