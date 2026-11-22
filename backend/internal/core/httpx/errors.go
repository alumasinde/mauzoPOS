package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// Error is a client-facing error with a stable machine-readable code.
// Services return *Error for expected failures; anything else becomes a 500.
type Error struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// NewError creates a client-facing error with an HTTP status and stable code.
func NewError(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

// WithDetails returns a copy of e carrying structured detail fields.
func (e *Error) WithDetails(d map[string]any) *Error {
	c := *e
	c.Details = d
	return &c
}

// Common errors. Modules define their own in errors.go.
var (
	ErrBadRequest     = NewError(http.StatusBadRequest, "bad_request", "The request is invalid.")
	ErrUnauthorized   = NewError(http.StatusUnauthorized, "unauthorized", "Authentication is required.")
	ErrForbidden      = NewError(http.StatusForbidden, "forbidden", "You do not have permission to perform this action.")
	ErrNotFound       = NewError(http.StatusNotFound, "not_found", "The requested resource was not found.")
	ErrConflict       = NewError(http.StatusConflict, "conflict", "The request conflicts with the current state.")
	ErrValidation     = NewError(http.StatusUnprocessableEntity, "validation_failed", "Some fields are invalid.")
	ErrTooManyRequest = NewError(http.StatusTooManyRequests, "rate_limited", "Too many requests. Please slow down.")
	ErrInternal       = NewError(http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
)

type errorBody struct {
	Error *Error `json:"error"`
}

// WriteError renders err using the standard error envelope. Unknown errors are
// logged with the request ID and hidden from the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		slog.ErrorContext(r.Context(), "unhandled error",
			"err", err, "request_id", middleware.GetReqID(r.Context()),
			"method", r.Method, "path", r.URL.Path)
		appErr = ErrInternal
	}
	JSON(w, appErr.Status, errorBody{Error: appErr})
}
