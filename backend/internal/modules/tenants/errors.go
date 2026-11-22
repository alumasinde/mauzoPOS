package tenants

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Module errors.
var (
	ErrTenantNotFound   = httpx.NewError(http.StatusNotFound, "tenant_not_found", "The tenant was not found.")
	ErrSlugTaken        = httpx.NewError(http.StatusConflict, "slug_taken", "That sign-in code is already in use.")
	ErrTenantSuspended  = httpx.NewError(http.StatusForbidden, "tenant_suspended", "This business is suspended. Contact support.")
	ErrAlreadySuspended = httpx.NewError(http.StatusConflict, "already_suspended", "The tenant is already suspended.")
	ErrNotSuspended     = httpx.NewError(http.StatusConflict, "not_suspended", "The tenant is not suspended.")
)
