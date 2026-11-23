// Package terminals implements minimal POS terminal registration — enough for
// onboarding. Device activation and binding land in Phase 6.
package terminals

import "time"

// Terminal statuses.
const (
	StatusPending  = "pending"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// Terminal is one POS device registration at a branch.
type Terminal struct {
	ID          uint64     `db:"id"`
	TenantID    uint64     `db:"tenant_id"`
	BranchID    uint64     `db:"branch_id"`
	PublicID    string     `db:"public_id"`
	Name        string     `db:"name"`
	Status      string     `db:"status"`
	DeviceName  *string    `db:"device_name"`
	ActivatedAt *time.Time `db:"activated_at"`
	LastSeenAt  *time.Time `db:"last_seen_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`

	BranchName string `db:"branch_name"` // joined, list only
}
