package branches

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/branches (list + create; full CRUD lands in Phase 12).
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/branches", func(r chi.Router) {
		r.With(mw.Can(PermBranchView)).Get("/", h.List)
		r.With(mw.Can(PermBranchManage)).Post("/", h.Create)
	})
}
