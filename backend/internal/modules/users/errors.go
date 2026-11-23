package users

import (
	"net/http"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// Module errors.
var (
	ErrUserNotFound     = httpx.NewError(http.StatusNotFound, "user_not_found", "The user was not found.")
	ErrEmailTaken       = httpx.NewError(http.StatusConflict, "email_taken", "A user with that email already exists.")
	ErrUsernameTaken    = httpx.NewError(http.StatusConflict, "username_taken", "That username is already taken.")
	ErrWeakPassword     = httpx.NewError(http.StatusUnprocessableEntity, "weak_password", "Password must be at least 8 characters.")
	ErrInvalidRole      = httpx.NewError(http.StatusUnprocessableEntity, "invalid_role", "Unknown role.")
	ErrLastSuperAdmin   = httpx.NewError(http.StatusConflict, "last_super_admin", "Cannot remove the last active super admin.")
	ErrAlreadyDisabled  = httpx.NewError(http.StatusConflict, "already_disabled", "The user is already disabled.")
	ErrAlreadyEnabled   = httpx.NewError(http.StatusConflict, "already_enabled", "The user is already active.")
	ErrInvalidEmail     = httpx.NewError(http.StatusUnprocessableEntity, "invalid_email", "Enter a valid email address.")
	ErrInvalidUsername  = httpx.NewError(http.StatusUnprocessableEntity, "invalid_username", "Usernames cannot contain @ and are at most 60 characters.")
	ErrIdentityRequired = httpx.NewError(http.StatusUnprocessableEntity, "identity_required", "Give the employee an email or a username.")
	ErrNameRequired     = httpx.NewError(http.StatusUnprocessableEntity, "name_required", "Name is required (at most 120 characters).")
	ErrWeakPIN          = httpx.NewError(http.StatusUnprocessableEntity, "weak_pin", "PIN must be 4–6 digits and not a repeated or sequential number.")
	ErrSelfChange       = httpx.NewError(http.StatusConflict, "self_change", "You cannot change your own role, branch access or status.")
	ErrLastOwner        = httpx.NewError(http.StatusConflict, "last_owner", "The business must keep at least one active owner.")
	ErrOutranked        = httpx.NewError(http.StatusForbidden, "user_outranks_you", "You cannot manage an employee whose role has permissions you lack.")
	ErrRoleEscalation   = httpx.NewError(http.StatusForbidden, "role_escalation", "You can only assign roles whose permissions you hold.")
	ErrBranchNotAllowed = httpx.NewError(http.StatusForbidden, "branch_forbidden", "You can only grant access to branches you can access.")
	ErrUnknownBranch    = httpx.NewError(http.StatusUnprocessableEntity, "branch_unknown", "Unknown branch.")
	ErrSeatLimit        = httpx.NewError(http.StatusPaymentRequired, "seat_limit", "Your plan's employee limit has been reached.")
)

// mapDup turns a unique-key race into the matching conflict error.
func mapDup(err error) error {
	if db.IsDuplicate(err) {
		return ErrEmailTaken.WithDetails(map[string]any{"field": "email or username"})
	}
	return err
}
