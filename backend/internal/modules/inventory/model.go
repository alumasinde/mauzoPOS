// Package inventory owns stock: locations, the append-only movement ledger,
// derived balances, adjustment/transfer/stocktake documents, batches and
// reorder alerts. The ledger is the only write path that changes stock.
package inventory

import (
	"time"

	"github.com/shopspring/decimal"
)

// Movement kinds (stock_movements.kind).
const (
	MoveReceive     = "receive"
	MoveSale        = "sale"
	MoveAdjust      = "adjust"
	MoveTransferOut = "transfer_out"
	MoveTransferIn  = "transfer_in"
	MoveCount       = "count"
	MoveWaste       = "waste"
	MoveReturn      = "return"
)

// AdjustReasons constrains adjustment causes so reports can group them.
var AdjustReasons = []string{
	"correction", "opening_stock", "damage", "theft", "expired", "stocktake", "other",
}

// WastageReasons are the accepted reasons for a wastage posting.
var WastageReasons = []string{"damage", "expired", "spoiled", "staff_meal", "sample", "other"}

// Document statuses.
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusPosted    = "posted"
	StatusRejected  = "rejected"
	StatusDraft     = "draft"
	StatusSent      = "sent"
	StatusReceived  = "received"
	StatusOpen      = "open"
	StatusCancelled = "cancelled"
)

// Location is a stock-holding place inside a branch.
type Location struct {
	ID        uint64    `db:"id"`
	TenantID  uint64    `db:"tenant_id"`
	BranchID  uint64    `db:"branch_id"`
	PublicID  string    `db:"public_id"`
	Name      string    `db:"name"`
	Kind      string    `db:"kind"`
	IsDefault bool      `db:"is_default"`
	CreatedAt time.Time `db:"created_at"`
}

// Movement is one ledger row: a signed delta in the product's base unit.
type Movement struct {
	ID          uint64           `db:"id"`
	TenantID    uint64           `db:"tenant_id"`
	BranchID    uint64           `db:"branch_id"`
	LocationID  uint64           `db:"location_id"`
	ProductID   uint64           `db:"product_id"`
	VariantID   uint64           `db:"variant_id"`
	BatchID     uint64           `db:"batch_id"`
	Qty         decimal.Decimal  `db:"qty"`
	UnitCost    *decimal.Decimal `db:"unit_cost"`
	Kind        string           `db:"kind"`
	RefType     *string          `db:"ref_type"`
	RefPublicID *string          `db:"ref_public_id"`
	Reason      *string          `db:"reason"`
	Note        *string          `db:"note"`
	ActorID     *uint64          `db:"actor_id"`
	CreatedAt   time.Time        `db:"created_at"`

	// Joined display fields (list queries only).
	ProductName  string `db:"product_name"`
	ProductSKU   string `db:"product_sku"`
	VariantName  string `db:"variant_name"`
	LocationName string `db:"location_name"`
	BranchName   string `db:"branch_name"`
}

// Balance is a derived quantity (+ weighted-average cost) per location row.
type Balance struct {
	BranchID   uint64          `db:"branch_id"`
	LocationID uint64          `db:"location_id"`
	ProductID  uint64          `db:"product_id"`
	VariantID  uint64          `db:"variant_id"`
	BatchID    uint64          `db:"batch_id"`
	Qty        decimal.Decimal `db:"qty"`
	AvgCost    decimal.Decimal `db:"avg_cost"`

	// Joined display fields.
	ProductName      string `db:"product_name"`
	ProductSKU       string `db:"product_sku"`
	ProductPublicID  string `db:"product_public_id"`
	VariantPublicID  string `db:"variant_public_id"`
	VariantName      string `db:"variant_name"`
	LocationName     string `db:"location_name"`
	LocationPublicID string `db:"location_public_id"`
	BranchName       string `db:"branch_name"`
	BranchPublicID   string `db:"branch_public_id"`
}

// Batch tracks a dated lot of a product for expiry reporting.
type Batch struct {
	ID         uint64     `db:"id"`
	TenantID   uint64     `db:"tenant_id"`
	ProductID  uint64     `db:"product_id"`
	VariantID  uint64     `db:"variant_id"`
	BatchNo    string     `db:"batch_no"`
	ExpiryDate *time.Time `db:"expiry_date"`
	ReceivedAt time.Time  `db:"received_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

// Adjustment is a counted reason-coded stock correction that must be
// approved before it posts to the ledger.
type Adjustment struct {
	ID         uint64     `db:"id"`
	TenantID   uint64     `db:"tenant_id"`
	BranchID   uint64     `db:"branch_id"`
	PublicID   string     `db:"public_id"`
	Status     string     `db:"status"`
	Reason     string     `db:"reason"`
	Note       *string    `db:"note"`
	CreatedBy  uint64     `db:"created_by"`
	ApprovedBy *uint64    `db:"approved_by"`
	PostedAt   *time.Time `db:"posted_at"`
	CreatedAt  time.Time  `db:"created_at"`

	BranchPublicID string           `db:"branch_public_id"`
	BranchName     string           `db:"branch_name"`
	Items          []AdjustmentItem `db:"-"`
}

// AdjustmentItem is one signed line of an Adjustment.
type AdjustmentItem struct {
	ID           uint64          `db:"id"`
	AdjustmentID uint64          `db:"adjustment_id"`
	LocationID   uint64          `db:"location_id"`
	ProductID    uint64          `db:"product_id"`
	VariantID    uint64          `db:"variant_id"`
	BatchID      uint64          `db:"batch_id"`
	Qty          decimal.Decimal `db:"qty"`

	// Joined display fields.
	ProductPublicID  string `db:"product_public_id"`
	VariantPublicID  string `db:"variant_public_id"`
	ProductName      string `db:"product_name"`
	VariantName      string `db:"variant_name"`
	LocationPublicID string `db:"location_public_id"`
	LocationName     string `db:"location_name"`
}

// Transfer moves stock between two branches via a sent/received workflow.
type Transfer struct {
	ID           uint64     `db:"id"`
	TenantID     uint64     `db:"tenant_id"`
	PublicID     string     `db:"public_id"`
	FromBranchID uint64     `db:"from_branch_id"`
	ToBranchID   uint64     `db:"to_branch_id"`
	Status       string     `db:"status"`
	Note         *string    `db:"note"`
	CreatedBy    uint64     `db:"created_by"`
	SentAt       *time.Time `db:"sent_at"`
	ReceivedAt   *time.Time `db:"received_at"`
	CreatedAt    time.Time  `db:"created_at"`

	FromPublicID string         `db:"from_public_id"`
	FromName     string         `db:"from_name"`
	ToPublicID   string         `db:"to_public_id"`
	ToName       string         `db:"to_name"`
	Items        []TransferItem `db:"-"`
}

// TransferItem is one line of a Transfer.
type TransferItem struct {
	ID          uint64           `db:"id"`
	PublicID    string           `db:"public_id"`
	TransferID  uint64           `db:"transfer_id"`
	ProductID   uint64           `db:"product_id"`
	VariantID   uint64           `db:"variant_id"`
	QtySent     decimal.Decimal  `db:"qty_sent"`
	QtyReceived *decimal.Decimal `db:"qty_received"`

	// Joined display fields.
	ProductPublicID string `db:"product_public_id"`
	VariantPublicID string `db:"variant_public_id"`
	ProductName     string `db:"product_name"`
	VariantName     string `db:"variant_name"`
}

// Count is a stocktake session; posting writes 'count' movements for the
// variance between expected and counted quantities.
type Count struct {
	ID         uint64     `db:"id"`
	TenantID   uint64     `db:"tenant_id"`
	BranchID   uint64     `db:"branch_id"`
	PublicID   string     `db:"public_id"`
	Status     string     `db:"status"`
	LocationID *uint64    `db:"location_id"`
	Note       *string    `db:"note"`
	CreatedBy  uint64     `db:"created_by"`
	PostedAt   *time.Time `db:"posted_at"`
	CreatedAt  time.Time  `db:"created_at"`

	BranchPublicID   string      `db:"branch_public_id"`
	BranchName       string      `db:"branch_name"`
	LocationPublicID string      `db:"location_public_id"`
	Items            []CountItem `db:"-"`
}

// CountItem snapshots the expected balance for a counted row.
type CountItem struct {
	ID          uint64           `db:"id"`
	PublicID    string           `db:"public_id"`
	CountID     uint64           `db:"count_id"`
	LocationID  uint64           `db:"location_id"`
	ProductID   uint64           `db:"product_id"`
	VariantID   uint64           `db:"variant_id"`
	BatchID     uint64           `db:"batch_id"`
	ExpectedQty decimal.Decimal  `db:"expected_qty"`
	CountedQty  *decimal.Decimal `db:"counted_qty"`

	// Joined display fields.
	ProductPublicID  string `db:"product_public_id"`
	VariantPublicID  string `db:"variant_public_id"`
	ProductName      string `db:"product_name"`
	VariantName      string `db:"variant_name"`
	LocationPublicID string `db:"location_public_id"`
	LocationName     string `db:"location_name"`
}

// ReorderLevel is the low-stock threshold for a product in a branch.
type ReorderLevel struct {
	BranchID  uint64          `db:"branch_id"`
	ProductID uint64          `db:"product_id"`
	VariantID uint64          `db:"variant_id"`
	MinQty    decimal.Decimal `db:"min_qty"`
	TargetQty decimal.Decimal `db:"target_qty"`
}
