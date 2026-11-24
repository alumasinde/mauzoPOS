// Package sales owns the sale ledger: completion in one transaction (items +
// payments + stock decrement + audit), idempotent device uploads, voids and
// returns/refunds. Payment methods are a code registry (paymethods.go).
package sales

import (
	"time"

	"github.com/shopspring/decimal"
)

// Sale statuses.
const (
	StatusCompleted         = "completed"
	StatusVoided            = "voided"
	StatusReturned          = "returned"
	StatusPartiallyReturned = "partially_returned"
)

// Sale sources.
const (
	SourcePOS        = "pos"
	SourceBackoffice = "backoffice"
)

// Payment statuses.
const (
	PayConfirmed = "confirmed"
	// PayUnverified: an M-Pesa code recorded by a till while offline that the
	// server could not match on upload. The sale stands (the goods left);
	// the payment waits in M-Pesa reconciliation.
	PayUnverified = "unverified"
	PayVoided     = "voided"
	PayRefunded   = "refunded"
)

// Counter names in tenant_counters.
const (
	counterReceipt = "sale_receipt"
	counterReturn  = "sale_return"
)

// Sale is a completed (or voided/returned) transaction.
type Sale struct {
	ID           uint64          `db:"id"`
	TenantID     uint64          `db:"tenant_id"`
	BranchID     uint64          `db:"branch_id"`
	TerminalID   *uint64         `db:"terminal_id"`
	CashierID    *uint64         `db:"cashier_id"`
	ShiftID      *uint64         `db:"shift_id"`
	CustomerID   *uint64         `db:"customer_id"`
	PublicID     string          `db:"public_id"`
	ReceiptNo    uint64          `db:"receipt_no"`
	Status       string          `db:"status"`
	Source       string          `db:"source"`
	IdemKey      *string         `db:"idempotency_key"`
	Currency     string          `db:"currency"`
	Subtotal     decimal.Decimal `db:"subtotal"`
	Discount     decimal.Decimal `db:"discount"`
	Tax          decimal.Decimal `db:"tax"`
	Total        decimal.Decimal `db:"total"`
	Paid         decimal.Decimal `db:"paid"`
	ChangeDue    decimal.Decimal `db:"change_due"`
	DiscountNote string          `db:"discount_note"`
	VoidedAt     *time.Time      `db:"voided_at"`
	VoidedBy     *uint64         `db:"voided_by"`
	VoidReason   string          `db:"void_reason"`
	SoldAt       time.Time       `db:"sold_at"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`

	// Joined display fields.
	BranchName   string `db:"branch_name"`
	TerminalName string `db:"terminal_name"`
	CashierName  string `db:"cashier_name"`
	CustomerName string `db:"customer_name"`
}

// SaleItem is one sold line (snapshotted product data).
type SaleItem struct {
	ID            uint64           `db:"id"`
	TenantID      uint64           `db:"tenant_id"`
	SaleID        uint64           `db:"sale_id"`
	LineNo        int              `db:"line_no"`
	ProductID     uint64           `db:"product_id"`
	VariantID     *uint64          `db:"variant_id"`
	Name          string           `db:"name"`
	SKU           string           `db:"sku"`
	Qty           decimal.Decimal  `db:"qty"`
	UnitPrice     decimal.Decimal  `db:"unit_price"`
	Discount      decimal.Decimal  `db:"discount"`
	TaxRate       *decimal.Decimal `db:"tax_rate"`
	Tax           decimal.Decimal  `db:"tax"`
	Total         decimal.Decimal  `db:"total"`
	ReturnedQty   decimal.Decimal  `db:"returned_qty"`
	PriceOverride bool             `db:"price_override"`
	Promotion     string           `db:"promotion"`

	// Joined for restock/lookup (not stored).
	ProductPublicID string `db:"product_public_id"`
	VariantPublicID string `db:"variant_public_id"`
	TrackStock      bool   `db:"track_stock"`
}

// SalePayment is one tendered payment line (split payments supported).
type SalePayment struct {
	ID        uint64           `db:"id"`
	TenantID  uint64           `db:"tenant_id"`
	SaleID    uint64           `db:"sale_id"`
	Method    string           `db:"method"`
	Amount    decimal.Decimal  `db:"amount"`
	Tendered  *decimal.Decimal `db:"tendered"`
	ChangeDue *decimal.Decimal `db:"change_due"`
	Reference string           `db:"reference"`
	Status    string           `db:"status"`
	CreatedBy *uint64          `db:"created_by"`
	CreatedAt time.Time        `db:"created_at"`
}

// SaleReturn is a refund header linked to the original sale.
type SaleReturn struct {
	ID           uint64          `db:"id"`
	TenantID     uint64          `db:"tenant_id"`
	SaleID       uint64          `db:"sale_id"`
	PublicID     string          `db:"public_id"`
	ReturnNo     uint64          `db:"return_no"`
	Reason       string          `db:"reason"`
	RefundMethod string          `db:"refund_method"`
	RefundAmount decimal.Decimal `db:"refund_amount"`
	Status       string          `db:"status"`
	SupervisorID *uint64         `db:"supervisor_id"`
	CreatedBy    *uint64         `db:"created_by"`
	CreatedAt    time.Time       `db:"created_at"`

	// Joined display fields.
	ReceiptNo   uint64 `db:"receipt_no"`
	CashierName string `db:"cashier_name"`
}

// SaleReturnItem is one returned line.
type SaleReturnItem struct {
	ID         uint64          `db:"id"`
	TenantID   uint64          `db:"tenant_id"`
	ReturnID   uint64          `db:"return_id"`
	SaleItemID uint64          `db:"sale_item_id"`
	Qty        decimal.Decimal `db:"qty"`
	Restock    bool            `db:"restock"`
	Amount     decimal.Decimal `db:"amount"`
}

// HeldCart is a parked cart (held_sales, written by the pos module) shown in
// the back-office held-sales page.
type HeldCart struct {
	PublicID     string    `db:"public_id"`
	Label        string    `db:"label"`
	HeldBy       string    `db:"held_by"`
	TerminalName string    `db:"terminal_name"`
	BranchName   string    `db:"branch_name"`
	Cart         string    `db:"cart"`
	Status       string    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
}
