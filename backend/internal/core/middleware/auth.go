package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/tenancy"
)

// SessionAuthenticator resolves a bearer token to an authenticated actor.
// Implemented by modules/auth.
type SessionAuthenticator interface {
	Authenticate(ctx context.Context, token string) (identity.Actor, error)
}

// ErrUnauthenticated is returned when no valid bearer token is presented.
var ErrUnauthenticated = httpx.ErrUnauthorized

// Authenticate resolves the Authorization: Bearer token through a and stores
// the actor (and tenant, for user actors) in the request context. It does not
// check the actor type — pair it with RequireActor per route group.
func Authenticate(a SessionAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.WriteError(w, r, ErrUnauthenticated)
				return
			}
			actor, err := a.Authenticate(r.Context(), token)
			if err != nil {
				var appErr *httpx.Error
				if !errors.As(err, &appErr) {
					appErr = ErrUnauthenticated
				}
				httpx.WriteError(w, r, appErr)
				return
			}
			ctx := identity.WithActor(r.Context(), actor)
			if actor.TenantID != 0 {
				ctx = tenancy.WithTenant(ctx, actor.TenantID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireActor rejects actors of the wrong type (e.g. a tenant session on a
// /platform route) with 403.
func RequireActor(t identity.ActorType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, ok := identity.ActorFrom(r.Context())
			if !ok || a.Type != t {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if tok, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(tok)
	}
	return ""
}
