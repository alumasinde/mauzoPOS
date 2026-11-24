package mpesa

import (
	"github.com/go-chi/chi/v5"

	mw "mauzo/internal/core/middleware"
)

// TenantRoutes wires /t/mpesa — the reconciliation UI and settings.
func TenantRoutes(r chi.Router, h *Handler) {
	r.Route("/mpesa", func(r chi.Router) {
		r.With(mw.Can(PermTransactionView)).Get("/transactions", h.List)
		r.With(mw.Can(PermTransactionView)).Get("/transactions/{id}", h.Get)
		r.With(mw.Can(PermTransactionView)).Get("/summary", h.Summary)
		r.With(mw.Can(PermTransactionView)).Get("/suggestions", h.Suggestions)
		r.With(mw.Can(PermTransactionAllocate)).Post("/transactions/{id}/allocate", h.Allocate)
		r.With(mw.Can(PermTransactionReallocate)).Post("/transactions/{id}/reallocate", h.Reallocate)
		r.With(mw.Can(PermTransactionReallocate)).Post("/transactions/{id}/reverse", h.Reverse)
		r.With(mw.Can(PermConfigManage)).Get("/config", h.Config)
		r.With(mw.Can(PermConfigManage)).Put("/config", h.SaveConfig)
		r.With(mw.Can(PermConfigManage)).Post("/config/register", h.RegisterC2B)
		r.With(mw.Can(PermConfigManage)).Post("/test", h.TestPush)
	})
}

// HookRoutes wires /api/v1/hooks/mpesa/{token}/… — public Daraja callbacks.
// No auth middleware: the token is the credential, so unknown tokens get the
// same 404 as real probes and every body is stored before processing.
func HookRoutes(r chi.Router, h *Handler) {
	r.Route("/mpesa/{token}", func(r chi.Router) {
		r.Post("/stk", h.STKHook)
		r.Post("/c2b/confirmation", h.C2BConfirmationHook)
		r.Post("/c2b/validation", h.C2BValidationHook)
		r.Post("/reversal", h.ReversalHook)
	})
}
