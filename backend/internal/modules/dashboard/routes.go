package dashboard

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/dashboard. Widget data comes from each owning
// module's own endpoints, under that module's permissions.
func TenantRoutes(r chi.Router) {
	r.With(mw.Can(PermView)).Get("/dashboard", Layout)
}
