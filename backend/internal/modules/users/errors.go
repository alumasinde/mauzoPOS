package users

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Module errors.
var (
	ErrUserNotFound    = httpx.NewError(http.StatusNotFound, "user_not_found", "The user was not found.")
	ErrEmailTaken      = httpx.NewError(http.StatusConflict, "email_taken", "A user with that email already exists.")
	ErrWeakPassword    = httpx.NewError(http.StatusUnprocessableEntity, "weak_password", "Password must be at least 8 characters.")
	ErrInvalidRole     = httpx.NewError(http.StatusUnprocessableEntity, "invalid_role", "Unknown role.")
	ErrLastSuperAdmin  = httpx.NewError(http.StatusConflict, "last_super_admin", "Cannot remove the last active super admin.")
	ErrAlreadyDisabled = httpx.NewError(http.StatusConflict, "already_disabled", "The user is already disabled.")
	ErrAlreadyEnabled  = httpx.NewError(http.StatusConflict, "already_enabled", "The user is already active.")
)
