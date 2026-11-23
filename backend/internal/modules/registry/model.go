// Package registry owns module enablement: the platform-wide modules switch
// (modules table) and per-tenant enablement + config (tenant_modules). Module
// definitions come from core/modules.
package registry

import "time"

// Enablement sources.
const (
	SourceTenant   = "tenant"   // enabled by the tenant, gated by the plan
	SourcePlatform = "platform" // platform override, bypasses the plan
)

// TenantModule is one tenant_modules row.
type TenantModule struct {
	TenantID  uint64    `db:"tenant_id"`
	Module    string    `db:"module"`
	Enabled   bool      `db:"enabled"`
	Source    string    `db:"source"`
	Config    []byte    `db:"config"` // NULL when never configured
	UpdatedAt time.Time `db:"updated_at"`
}

// PlatformModule is one modules row.
type PlatformModule struct {
	Code      string    `db:"code"`
	IsActive  bool      `db:"is_active"`
	UpdatedAt time.Time `db:"updated_at"`
}
