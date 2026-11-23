package terminals

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/terminals (list + create; activation lands in Phase 6).
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/terminals", func(r chi.Router) {
		r.With(mw.Can(PermTerminalView)).Get("/", h.List)
		r.With(mw.Can(PermTerminalManage)).Post("/", h.Create)
		r.With(mw.Can(PermTerminalActivate)).Post("/{id}/code", h.IssueCode)
	})
}
