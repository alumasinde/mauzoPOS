// Package customers implements basic customer accounts — needed by credit
// sales now and loyalty/statements later.
package customers

import (
	"time"

	"github.com/shopspring/decimal"
)

// Customer statuses.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Customer is a tenant customer account.
type Customer struct {
	ID          uint64           `db:"id"`
	TenantID    uint64           `db:"tenant_id"`
	PublicID    string           `db:"public_id"`
	Code        string           `db:"code"`
	Name        string           `db:"name"`
	Phone       string           `db:"phone"`
	Email       string           `db:"email"`
	KRAPin      string           `db:"kra_pin"`
	CreditLimit *decimal.Decimal `db:"credit_limit"`
	Notes       *string          `db:"notes"`
	Status      string           `db:"status"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`

	CreditBalance *decimal.Decimal `db:"credit_balance"` // joined: outstanding credit
}
