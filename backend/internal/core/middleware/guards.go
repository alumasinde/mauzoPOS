package middleware

import (
	"context"
	"net/http"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/modules"
	"mauzo/internal/core/tenancy"
)

// ErrModuleDisabled tells the client the feature is switched off for the
// tenant (distinct from a missing permission so the UI can explain it).
var ErrModuleDisabled = httpx.NewError(http.StatusForbidden, "module_disabled",
	"This feature is not enabled for your business.")

// ErrBranchForbidden rejects requests for a branch the user cannot access.
var ErrBranchForbidden = httpx.NewError(http.StatusForbidden, "branch_forbidden",
	"You do not have access to that branch.")

// Can allows the request when the actor holds perm. For tenant users this
// also requires the permission's module to be enabled (see identity.Actor.Can).
// It panics on unregistered permissions — a programming error caught at boot.
func Can(perm string) func(http.Handler) http.Handler {
	p, ok := authz.Lookup(perm)
	if !ok {
		panic("middleware: unregistered permission " + perm)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, ok := identity.ActorFrom(r.Context())
			if !ok {
				httpx.WriteError(w, r, httpx.ErrUnauthorized)
				return
			}
			if a.Type == identity.ActorUser && p.Module != "" && !a.HasModule(p.Module) {
				httpx.WriteError(w, r, ErrModuleDisabled)
				return
			}
			if !a.Can(perm) {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireModule rejects tenant requests when module code is not enabled. It
// panics on unknown module codes.
func RequireModule(code string) func(http.Handler) http.Handler {
	if !modules.Known(code) {
		panic("middleware: unknown module " + code)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a, ok := identity.ActorFrom(r.Context())
			if !ok || !a.HasModule(code) {
				httpx.WriteError(w, r, ErrModuleDisabled)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BranchResolver maps a branch public id to its id within a tenant.
type BranchResolver interface {
	BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

// BranchHeader is the request header selecting the working branch.
const BranchHeader = "X-Branch-Id"

// BranchScope reads the optional X-Branch-Id header, verifies the user may
// access that branch and stores it in the context (tenancy.BranchID).
// Handlers that need a branch call RequireBranch(ctx).
func BranchScope(res BranchResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pid := r.Header.Get(BranchHeader)
			if pid == "" {
				next.ServeHTTP(w, r)
				return
			}
			a, ok := identity.ActorFrom(r.Context())
			if !ok || a.Type != identity.ActorUser {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			id, err := res.BranchByPublicID(r.Context(), a.TenantID, pid)
			if err != nil || !a.CanAccessBranch(id) {
				httpx.WriteError(w, r, ErrBranchForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(tenancy.WithBranch(r.Context(), id)))
		})
	}
}
