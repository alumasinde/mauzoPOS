package pos

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// HeldResponse is the API view of a parked cart.
type HeldResponse struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Cart     string `json:"cart"`
	HeldBy   string `json:"held_by"`
	Terminal string `json:"terminal"`
	HeldAt   string `json:"held_at"`
}

func toHeldResponse(h HeldSale) HeldResponse {
	return HeldResponse{
		ID: h.PublicID, Label: h.Label, Cart: h.Cart,
		HeldBy: h.HeldBy, Terminal: h.Terminal,
		HeldAt: h.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// PublicRoutes mounts the unauthenticated activation endpoint.
func PublicRoutes(r chi.Router, h *Handler) {
	r.Post("/activate", h.Activate)
}

// DeviceRoutes mounts endpoints behind device-token auth.
func DeviceRoutes(r chi.Router, h *Handler) {
	r.Get("/whoami", h.Whoami)
	r.Get("/sync", h.Sync)
	r.Post("/signin", h.SignIn)
	r.Post("/authorize", h.Authorize)
	r.Get("/held", h.Held)
	r.Post("/held", h.Held)
	r.Post("/held/{id}/recall", h.RecallHeld)
	r.Delete("/held/{id}", h.CancelHeld)
}

// DeviceAuth resolves the terminal bearer token. The device token
// authenticates the till itself; cashier identity is a PIN away and is not
// required at this layer.
func DeviceAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			d, err := svc.AuthenticateDevice(r.Context(), token)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			ctx := WithDevice(r.Context(), d)
			ctx = identity.WithActor(ctx, identity.Actor{
				Type: identity.ActorTerminal, ID: d.TerminalID, TenantID: d.TenantID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func publicIDParam(r *http.Request) string { return chi.URLParam(r, "id") }
