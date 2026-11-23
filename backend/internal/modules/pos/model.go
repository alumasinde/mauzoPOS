// Package pos implements the terminal-side API: device activation and token
// auth, catalog/price/cashier sync bundles, PIN sign-in and supervisor
// authorisation, and held-sale parking. The desktop terminal (backend/desktop)
// is the only intended caller.
package pos

import (
	"context"
	"time"
)

// Device is the authenticated terminal on a request. Device tokens are opaque
// bearer secrets minted at activation; only their SHA-256 hash is stored.
type Device struct {
	TerminalID uint64
	TenantID   uint64
	BranchID   uint64
}

type deviceKey struct{}

// WithDevice stores the authenticated terminal on the context.
func WithDevice(ctx context.Context, d Device) context.Context {
	return context.WithValue(ctx, deviceKey{}, d)
}

// DeviceFrom returns the authenticated terminal.
func DeviceFrom(ctx context.Context) (Device, bool) {
	d, ok := ctx.Value(deviceKey{}).(Device)
	return d, ok
}

// HeldSale is a parked cart shared by every terminal on a branch.
type HeldSale struct {
	ID         uint64    `db:"id" json:"id"`
	TenantID   uint64    `db:"tenant_id" json:"tenant_id"`
	BranchID   uint64    `db:"branch_id" json:"branch_id"`
	TerminalID uint64    `db:"terminal_id" json:"terminal_id"`
	UserID     uint64    `db:"user_id" json:"user_id"`
	PublicID   string    `db:"public_id" json:"public_id"`
	Label      string    `db:"label" json:"label"`
	Cart       string    `db:"cart" json:"cart"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`

	HeldBy   string `db:"held_by" json:"held_by"`
	Terminal string `db:"terminal_name" json:"terminal_name"`
}

// Sync row types — flat shapes the terminal writes into its local SQLite.

// SyncCategory is a catalog category for the tile grid.
type SyncCategory struct {
	ID       uint64 `db:"id" json:"id"`
	PublicID string `db:"public_id" json:"public_id"`
	ParentID uint64 `db:"parent_id" json:"parent_id"` // 0 = root
	Name     string `db:"name" json:"name"`
	Sort     int    `db:"sort" json:"sort"`
}

// SyncUnit is a unit of measure used by sellable products.
type SyncUnit struct {
	ID     uint64 `db:"id" json:"id"`
	Code   string `db:"code" json:"code"`
	Name   string `db:"name" json:"name"`
	Kind   string `db:"kind" json:"kind"`
	Factor string `db:"factor" json:"factor"`
}

// SyncProduct is a sellable product with its resolved unit code.
type SyncProduct struct {
	ID         uint64    `db:"id" json:"id"`
	PublicID   string    `db:"public_id" json:"public_id"`
	SKU        string    `db:"sku" json:"sku"`
	Name       string    `db:"name" json:"name"`
	Kind       string    `db:"kind" json:"kind"`
	CategoryID uint64    `db:"category_id" json:"category_id"`
	UnitCode   string    `db:"unit_code" json:"unit_code"`
	SellBy     string    `db:"sell_by" json:"sell_by"`
	Price      string    `db:"price" json:"price"`
	VatRate    *string   `db:"vat_rate" json:"vat_rate"`
	TrackStock bool      `db:"track_stock" json:"track_stock"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// SyncVariant is a purchasable variant of a product.
type SyncVariant struct {
	ID        uint64  `db:"id" json:"id"`
	ProductID uint64  `db:"product_id" json:"product_id"`
	PublicID  string  `db:"public_id" json:"public_id"`
	SKU       string  `db:"sku" json:"sku"`
	Name      string  `db:"name" json:"name"`
	Price     *string `db:"price" json:"price"`
}

// SyncBarcode is a barcode mapping to a product or variant with pack quantity.
type SyncBarcode struct {
	ProductID uint64 `db:"product_id" json:"product_id"`
	VariantID uint64 `db:"variant_id" json:"variant_id"` // 0 = product-level
	Barcode   string `db:"barcode" json:"barcode"`
	PackQty   string `db:"pack_qty" json:"pack_qty"`
}

// SyncPriceList is a price list header applicable to the terminal branch.
type SyncPriceList struct {
	ID            uint64     `db:"id" json:"id"`
	BranchID      uint64     `db:"branch_id" json:"branch_id"` // 0 = all branches
	CustomerGroup string     `db:"customer_group" json:"customer_group"`
	Priority      int        `db:"priority" json:"priority"`
	ValidFrom     *time.Time `db:"valid_from" json:"valid_from"`
	ValidTo       *time.Time `db:"valid_to" json:"valid_to"`
}

// SyncPriceItem is a price-list line (tiered by MinQty).
type SyncPriceItem struct {
	ListID    uint64 `db:"list_id" json:"list_id"`
	ProductID uint64 `db:"product_id" json:"product_id"`
	VariantID uint64 `db:"variant_id" json:"variant_id"`
	MinQty    string `db:"min_qty" json:"min_qty"`
	Price     string `db:"price" json:"price"`
}

// SyncPromotion is a promotion definition with its scheduling window.
type SyncPromotion struct {
	ID       uint64     `db:"id" json:"id"`
	Name     string     `db:"name" json:"name"`
	Kind     string     `db:"kind" json:"kind"`
	Config   string     `db:"config" json:"config"`
	StartsAt *time.Time `db:"starts_at" json:"starts_at"`
	EndsAt   *time.Time `db:"ends_at" json:"ends_at"`
	DaysMask *uint8     `db:"days_mask" json:"days_mask"`
	Priority int        `db:"priority" json:"priority"`
}

// SyncPromoTarget is a product/category target row for a promotion.
type SyncPromoTarget struct {
	PromotionID uint64 `db:"promotion_id" json:"promotion_id"`
	ProductID   uint64 `db:"product_id" json:"product_id"`   // 0 = not product-targeted
	CategoryID  uint64 `db:"category_id" json:"category_id"` // 0 = not category-targeted
}

// SyncCustomer is a customer account for the credit-payment picker.
type SyncCustomer struct {
	ID            uint64  `db:"id" json:"id"`
	PublicID      string  `db:"public_id" json:"public_id"`
	Name          string  `db:"name" json:"name"`
	Phone         string  `db:"phone" json:"phone"`
	Code          string  `db:"code" json:"code"`
	CreditLimit   *string `db:"credit_limit" json:"credit_limit"`
	CreditBalance string  `db:"credit_balance" json:"credit_balance"`
}

// SyncCashier is an employee allowed to sign in at the terminal.
type SyncCashier struct {
	ID          uint64  `db:"id" json:"id"`
	PublicID    string  `db:"public_id" json:"public_id"`
	Name        string  `db:"name" json:"name"`
	RoleName    string  `db:"role_name" json:"role_name"`
	PINHash     *string `db:"pin_hash" json:"pin_hash"`
	AllBranches bool    `db:"all_branches" json:"all_branches"`
	MaxDiscount *string `db:"max_discount_percent" json:"max_discount_percent"`
}

// SyncCashierBranch is a branch-assignment row for restricted cashiers.
type SyncCashierBranch struct {
	UserID   uint64 `db:"user_id" json:"user_id"`
	BranchID uint64 `db:"branch_id" json:"branch_id"`
}

// SyncSetting is a tenant setting key/value needed by the POS.
type SyncSetting struct {
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
}
