package auth

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Auth errors shared by both login contexts.
var (
	ErrInvalidCredentials = httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "Invalid email or password.")
	ErrAccountDisabled    = httpx.NewError(http.StatusForbidden, "account_disabled", "This account is disabled. Contact your administrator.")
	ErrTooManyAttempts    = httpx.NewError(http.StatusTooManyRequests, "too_many_attempts", "Too many failed attempts. Try again later.")
	ErrMFACodeInvalid     = httpx.NewError(http.StatusUnauthorized, "mfa_invalid", "The verification code is invalid.")
	ErrMFATicketInvalid   = httpx.NewError(http.StatusUnprocessableEntity, "mfa_ticket_invalid", "The MFA ticket is invalid or expired. Sign in again.")
	ErrResetTokenInvalid  = httpx.NewError(http.StatusUnprocessableEntity, "reset_token_invalid", "The reset link is invalid or has expired.")
	ErrMFAAlreadyEnabled  = httpx.NewError(http.StatusConflict, "mfa_already_enabled", "Two-factor authentication is already enabled.")
	ErrMFANotPending      = httpx.NewError(http.StatusConflict, "mfa_not_pending", "Start MFA setup first.")
	ErrMFANotEnabled      = httpx.NewError(http.StatusConflict, "mfa_not_enabled", "Two-factor authentication is not enabled.")
	ErrSessionNotFound    = httpx.NewError(http.StatusNotFound, "session_not_found", "The session was not found.")
)
