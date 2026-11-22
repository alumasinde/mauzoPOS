package auth

import "github.com/go-chi/chi/v5"

// PublicRoutes wires unauthenticated endpoints. ctx is "platform" or "tenant"
// and picks which login variant the POST /login route calls.
func PublicRoutes(r chi.Router, h *Handler, ctx string) {
	if ctx == "platform" {
		r.Post("/auth/login", h.LoginPlatform)
		r.Post("/auth/password/forgot", h.ForgotPasswordPlatform)
	} else {
		r.Post("/auth/login", h.LoginTenant)
		r.Post("/auth/password/forgot", h.ForgotPasswordTenant)
	}
	r.Post("/auth/mfa/verify", h.VerifyMFA)
	r.Post("/auth/password/reset", h.ResetPassword)
}

// PrivateRoutes wires endpoints that need an authenticated session. They are
// identical for both contexts; the actor comes from the middleware context.
func PrivateRoutes(r chi.Router, h *Handler) {
	r.Post("/auth/logout", h.Logout)
	r.Get("/auth/me", h.Me)
	r.Post("/auth/mfa/setup", h.MFASetup)
	r.Post("/auth/mfa/enable", h.MFAEnable)
	r.Post("/auth/mfa/disable", h.MFADisable)
	r.Get("/auth/sessions", h.Sessions)
	r.Delete("/auth/sessions/{id}", h.RevokeSession)
	r.Get("/auth/history", h.History)
}
