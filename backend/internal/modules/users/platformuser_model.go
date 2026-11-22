// Package users holds platform operators and tenant employees. Platform users
// get full CRUD here; tenant employee CRUD endpoints arrive in Phase 3 with
// roles, while the repo/service here already backs tenant login (Phase 1).
package users

import "time"

// Platform user statuses.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// PlatformUser is a SaaS operator account.
type PlatformUser struct {
	ID            uint64     `db:"id"`
	PublicID      string     `db:"public_id"`
	Name          string     `db:"name"`
	Email         string     `db:"email"`
	PasswordHash  string     `db:"password_hash"`
	Role          string     `db:"role"`
	Status        string     `db:"status"`
	TOTPSecret    *string    `db:"totp_secret"`
	TOTPEnabled   bool       `db:"totp_enabled"`
	RecoveryCodes []byte     `db:"recovery_codes"`
	LastLoginAt   *time.Time `db:"last_login_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}
