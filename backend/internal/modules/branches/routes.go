package branches

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/branches (list + create; full CRUD lands in Phase 12).
// /accessible needs no permission: every employee may see the branches they
// work in (the back-office branch picker).
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/branches", func(r chi.Router) {
		r.With(mw.Can(PermBranchView)).Get("/", h.List)
		r.Get("/accessible", h.Accessible)
		r.With(mw.Can(PermBranchManage)).Post("/", h.Create)
	})
}
