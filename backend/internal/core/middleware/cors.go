package middleware

import (
	"net/http"
	"slices"
	"strings"
)

// CORS allows only the configured browser origins. The POS desktop app talks to
// the API from Go, so it never needs CORS.
func CORS(allowed []string) func(http.Handler) http.Handler {
	allowHeaders := strings.Join([]string{
		"Authorization", "Content-Type", "Idempotency-Key", "X-Request-Id", "X-Branch-Id",
	}, ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowed, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Add("Vary", "Origin")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", allowHeaders)
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
