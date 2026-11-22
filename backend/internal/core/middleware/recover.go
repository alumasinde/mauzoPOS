package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	chimw "github.com/go-chi/chi/v5/middleware"

	"mauzo/internal/core/httpx"
)

// Recoverer converts panics into a logged 500 with the standard error envelope.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				slog.ErrorContext(r.Context(), "panic",
					"request_id", chimw.GetReqID(r.Context()),
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()))
				httpx.WriteError(w, r, httpx.ErrInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
