// Package tenancy carries the authenticated tenant/branch identity through
// request contexts. Values are set only by auth middleware.
package tenancy

import "context"

type ctxKey int

const (
	tenantKey ctxKey = iota
	branchKey
)

// WithTenant stores the authenticated tenant ID in ctx.
func WithTenant(ctx context.Context, tenantID uint64) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantID returns the tenant ID stored by WithTenant.
func TenantID(ctx context.Context) (uint64, bool) {
	v, ok := ctx.Value(tenantKey).(uint64)
	return v, ok
}

// WithBranch stores the active branch ID in ctx.
func WithBranch(ctx context.Context, branchID uint64) context.Context {
	return context.WithValue(ctx, branchKey, branchID)
}

// BranchID returns the branch ID stored by WithBranch.
func BranchID(ctx context.Context) (uint64, bool) {
	v, ok := ctx.Value(branchKey).(uint64)
	return v, ok
}
