// Package pricing implements price lists (per branch/customer group with
// quantity tiers), promotions, price resolution and price history.
package pricing

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// List kinds/scopes.
const (
	HistoryBase      = "base"
	HistoryPriceList = "price_list"
)

// PriceList is a named set of prices that may apply to one branch and/or
// customer group, optionally inside a validity window.
type PriceList struct {
	ID            uint64     `db:"id"`
	TenantID      uint64     `db:"tenant_id"`
	PublicID      string     `db:"public_id"`
	Name          string     `db:"name"`
	BranchID      *uint64    `db:"branch_id"`
	CustomerGroup *string    `db:"customer_group"`
	Priority      int        `db:"priority"`
	IsActive      bool       `db:"is_active"`
	ValidFrom     *time.Time `db:"valid_from"`
	ValidTo       *time.Time `db:"valid_to"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`

	BranchPublicID string          `db:"-"`
	Items          []PriceListItem `db:"-"`
}

// PriceListItem is one tiered price row; several rows per product create
// quantity tiers (largest min_qty ≤ qty wins).
type PriceListItem struct {
	ID        uint64          `db:"id"`
	ListID    uint64          `db:"list_id"`
	ProductID uint64          `db:"product_id"`
	VariantID uint64          `db:"variant_id"`
	MinQty    decimal.Decimal `db:"min_qty"`
	Price     decimal.Decimal `db:"price"`

	ProductPublicID string `db:"product_public_id"`
	VariantPublicID string `db:"variant_public_id"`
	ProductName     string `db:"product_name"`
}

// Promotion is a persisted pricecalc promotion.
type Promotion struct {
	ID        uint64          `db:"id"`
	TenantID  uint64          `db:"tenant_id"`
	PublicID  string          `db:"public_id"`
	Name      string          `db:"name"`
	Kind      string          `db:"kind"`
	Config    json.RawMessage `db:"config"`
	StartsAt  *time.Time      `db:"starts_at"`
	EndsAt    *time.Time      `db:"ends_at"`
	DaysMask  *uint8          `db:"days_mask"`
	Priority  int             `db:"priority"`
	IsActive  bool            `db:"is_active"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`

	Targets []PromotionTarget `db:"-"`
}

// PromotionTarget scopes a promotion; a row with both NULLs means the whole
// catalog.
type PromotionTarget struct {
	ID          uint64  `db:"id"`
	PromotionID uint64  `db:"promotion_id"`
	ProductID   *uint64 `db:"product_id"`
	CategoryID  *uint64 `db:"category_id"`

	ProductPublicID  string `db:"product_public_id"`
	CategoryPublicID string `db:"category_public_id"`
}

// HistoryRow is one recorded price change.
type HistoryRow struct {
	ID          uint64           `db:"id"`
	TenantID    uint64           `db:"tenant_id"`
	ProductID   uint64           `db:"product_id"`
	VariantID   uint64           `db:"variant_id"`
	Source      string           `db:"source"`
	RefPublicID *string          `db:"ref_public_id"`
	OldPrice    *decimal.Decimal `db:"old_price"`
	NewPrice    decimal.Decimal  `db:"new_price"`
	ActorID     *uint64          `db:"actor_id"`
	CreatedAt   time.Time        `db:"created_at"`

	ProductName string `db:"product_name"`
}
