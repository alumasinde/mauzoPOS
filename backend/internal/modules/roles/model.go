// Package roles owns tenant roles, role permissions, the permissions table
// mirror of core/authz, role templates and per-request access resolution.
package roles

import (
	"time"

	"github.com/shopspring/decimal"
)

// Role is one tenant role.
type Role struct {
	ID                 uint64              `db:"id"`
	TenantID           uint64              `db:"tenant_id"`
	PublicID           string              `db:"public_id"`
	Code               string              `db:"code"`
	Name               string              `db:"name"`
	Description        *string             `db:"description"`
	IsSystem           bool                `db:"is_system"`
	IsLocked           bool                `db:"is_locked"`
	Customized         bool                `db:"customized"`
	MaxDiscountPercent decimal.NullDecimal `db:"max_discount_percent"`
	MaxRefundAmount    decimal.NullDecimal `db:"max_refund_amount"`
	MaxCashOutAmount   decimal.NullDecimal `db:"max_cash_out_amount"`
	CreatedAt          time.Time           `db:"created_at"`
	UpdatedAt          time.Time           `db:"updated_at"`
	UserCount          int                 `db:"user_count"`

	Permissions []string `db:"-"`
}

// Limits are the per-role monetary caps.
type Limits struct {
	MaxDiscountPercent decimal.NullDecimal
	MaxRefundAmount    decimal.NullDecimal
	MaxCashOutAmount   decimal.NullDecimal
}

// Limits returns the role's caps.
func (r Role) Limits() Limits {
	return Limits{r.MaxDiscountPercent, r.MaxRefundAmount, r.MaxCashOutAmount}
}
