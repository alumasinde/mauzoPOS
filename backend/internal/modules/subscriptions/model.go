// Package subscriptions implements plans and tenant subscriptions.
package subscriptions

import (
	"time"

	"github.com/shopspring/decimal"
)

// Subscription lifecycle statuses.
const (
	SubStatusTrialing  = "trialing"
	SubStatusActive    = "active"
	SubStatusPastDue   = "past_due"
	SubStatusCancelled = "cancelled"
)

// Billing intervals.
const (
	IntervalMonthly = "monthly"
	IntervalYearly  = "yearly"
)

// Plan is one subscription tier.
type Plan struct {
	ID              uint64          `db:"id"`
	PublicID        string          `db:"public_id"`
	Name            string          `db:"name"`
	Description     *string         `db:"description"`
	PriceKES        decimal.Decimal `db:"price_kes"`
	BillingInterval string          `db:"billing_interval"`
	ModulesJSON     []byte          `db:"modules"`
	Modules         []string        `db:"-"`
	MaxBranches     *int            `db:"max_branches"`
	MaxUsers        *int            `db:"max_users"`
	IsActive        bool            `db:"is_active"`
	Sort            int             `db:"sort"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// Subscription is one tenant_subscriptions row; Plan fields join for reads.
type Subscription struct {
	ID          uint64     `db:"id"`
	TenantID    uint64     `db:"tenant_id"`
	PlanID      uint64     `db:"plan_id"`
	Status      string     `db:"status"`
	StartsAt    time.Time  `db:"starts_at"`
	TrialEndsAt *time.Time `db:"trial_ends_at"`
	EndsAt      *time.Time `db:"ends_at"`
	CreatedAt   time.Time  `db:"created_at"`

	PlanPublicID string `db:"plan_public_id"`
	PlanName     string `db:"plan_name"`
}
