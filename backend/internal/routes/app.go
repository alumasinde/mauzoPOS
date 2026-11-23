// Package routes is the master router. It applies global middleware and mounts
// every module's routes under the correct authentication context. Modules own
// their route definitions in modules/<name>/routes.go.
package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"mauzo/internal/app"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	mw "mauzo/internal/core/middleware"
	auditmod "mauzo/internal/modules/audit"
	"mauzo/internal/modules/auth"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/businesstypes"
	"mauzo/internal/modules/onboarding"
	"mauzo/internal/modules/settings"
	"mauzo/internal/modules/subscriptions"
	"mauzo/internal/modules/system"
	"mauzo/internal/modules/tenants"
	"mauzo/internal/modules/terminals"
	"mauzo/internal/modules/users"
)

// New builds the HTTP router: global middleware and module routes.
func New(c *app.Container) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(mw.RequestLogger(c.Log))
	r.Use(mw.Recoverer)
	r.Use(mw.CORS(c.Config.HTTP.AllowedOrigins))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.WriteError(w, r, httpx.ErrNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, r, httpx.NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed."))
	})

	system.Routes(r, c.System)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(c.RateLimiter.Handler)
		r.Route("/platform", func(r chi.Router) { platformRoutes(r, c) })
		r.Route("/t", func(r chi.Router) { tenantRoutes(r, c) })
		r.Route("/pos", func(r chi.Router) { posRoutes(r, c) })
		r.Route("/hooks", func(r chi.Router) { webhookRoutes(r, c) })
	})
	return r
}

// platformRoutes: platform-admin auth context. Public auth endpoints sit next
// to a group guarded by platform sessions.
func platformRoutes(r chi.Router, c *app.Container) {
	auth.PublicRoutes(r, c.Auth, "platform")
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate(c.AuthSvc))
		r.Use(mw.RequireActor(identity.ActorPlatformUser))
		auth.PrivateRoutes(r, c.Auth)
		tenants.Routes(r, c.Tenants)
		users.PlatformRoutes(r, c.PlatformUsers)
		auditmod.PlatformRoutes(r, c.AuditLogs)
		businesstypes.PlatformRoutes(r, c.BusinessTypes)
		subscriptions.PlatformRoutes(r, c.Plans)
	})
}

// tenantRoutes: tenant employee auth context.
func tenantRoutes(r chi.Router, c *app.Container) {
	auth.PublicRoutes(r, c.Auth, "tenant")
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate(c.AuthSvc))
		r.Use(mw.RequireActor(identity.ActorUser))
		auth.PrivateRoutes(r, c.Auth)
		auditmod.TenantRoutes(r, c.AuditLogs)
		settings.TenantRoutes(r, c.Settings)
		branches.TenantRoutes(r, c.Branches)
		terminals.TenantRoutes(r, c.Terminals)
		onboarding.TenantRoutes(r, c.Onboarding)
	})
}

// posRoutes: terminal + cashier auth context (Phase 6+).
func posRoutes(_ chi.Router, _ *app.Container) {}

// webhookRoutes: provider callbacks — M-Pesa, eTIMS (Phase 9+).
func webhookRoutes(_ chi.Router, _ *app.Container) {}
