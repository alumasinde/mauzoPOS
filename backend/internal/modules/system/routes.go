// Package system exposes liveness/readiness probes and build info.
package system

import "github.com/go-chi/chi/v5"

// Routes mounts probes at the router root (outside /api/v1) so load balancers
// and Docker health checks can reach them without auth.
func Routes(r chi.Router, h *Handler) {
	r.Get("/healthz", h.Live)
	r.Get("/readyz", h.Ready)
}
