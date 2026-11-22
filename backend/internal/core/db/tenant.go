package db

import (
	"context"
	"errors"

	"mauzo/internal/core/tenancy"
)

// ErrNoTenant is returned when tenant-scoped data access is attempted without an
// authenticated tenant in the context. It is a programming error, never a user error.
var ErrNoTenant = errors.New("db: no tenant in context")

// Tenant wraps a Querier and binds it to the tenant taken from the request context.
// Repositories for tenant-owned tables MUST use Tenant and include
// "tenant_id = ?" with t.ID() in every statement. The tenant ID is never read
// from request payloads.
//
// Keeping this indirection lets us later route large tenants to dedicated
// databases by swapping the Querier per tenant.
type Tenant struct {
	Q  Querier
	id uint64
}

// ForTenant binds q to the tenant in ctx, or returns ErrNoTenant.
func ForTenant(ctx context.Context, q Querier) (Tenant, error) {
	id, ok := tenancy.TenantID(ctx)
	if !ok || id == 0 {
		return Tenant{}, ErrNoTenant
	}
	return Tenant{Q: q, id: id}, nil
}

// ID returns the bound tenant ID for use in "tenant_id = ?" clauses.
func (t Tenant) ID() uint64 { return t.id }
