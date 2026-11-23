// Package tenants implements tenant management in the platform auth context.
package tenants

import "time"

// Tenant lifecycle statuses.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// Tenant is a business that uses the POS.
type Tenant struct {
	ID             uint64     `db:"id"`
	PublicID       string     `db:"public_id"`
	Name           string     `db:"name"`
	Slug           string     `db:"slug"`
	Email          *string    `db:"email"`
	Phone          *string    `db:"phone"`
	Status         string     `db:"status"`
	BusinessType   *string    `db:"business_type"`
	KRAPin         *string    `db:"kra_pin"`
	VATRegistered  bool       `db:"vat_registered"`
	Address        *string    `db:"address"`
	OnboardingStep int        `db:"onboarding_step"`
	OnboardedAt    *time.Time `db:"onboarded_at"`
	SuspendedAt    *time.Time `db:"suspended_at"`
	SuspendReason  *string    `db:"suspend_reason"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
