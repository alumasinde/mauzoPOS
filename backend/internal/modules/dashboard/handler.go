package dashboard

import (
	"net/http"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/modules"
)

// Response is the dashboard layout for the current user.
type Response struct {
	Widgets []modules.Widget `json:"widgets"`
}

// Layout handles GET /t/dashboard.
func Layout(w http.ResponseWriter, r *http.Request) {
	a, ok := identity.ActorFrom(r.Context())
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	httpx.OK(w, Response{Widgets: Widgets(a)})
}
