// Package branches implements minimal tenant branch management — enough for
// onboarding (first/main branch) and back-office settings.
package branches

import "time"

// Branch statuses.
const (
	StatusActive = "active"
	StatusClosed = "closed"
)

// Branch is one physical location of a tenant.
type Branch struct {
	ID        uint64    `db:"id"`
	TenantID  uint64    `db:"tenant_id"`
	PublicID  string    `db:"public_id"`
	Code      string    `db:"code"`
	Name      string    `db:"name"`
	Phone     *string   `db:"phone"`
	Location  *string   `db:"location"`
	IsMain    bool      `db:"is_main"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
