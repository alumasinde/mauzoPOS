package shifts

import (
	"time"

	"github.com/shopspring/decimal"
)

// Shift statuses.
const (
	StOpen   = "open"
	StClosed = "closed"
)

// Movement kinds.
const (
	KindIn     = "cash_in"
	KindOut    = "cash_out"
	KindDrawer = "drawer_open"
)

// Variance review outcomes.
const (
	VarPending  = "pending"
	VarApproved = "approved"
	VarFlagged  = "flagged"
)

// Shift is one cashier's drawer session on a terminal.
type Shift struct {
	ID             uint64           `db:"id" json:"-"`
	TenantID       uint64           `db:"tenant_id" json:"-"`
	BranchID       uint64           `db:"branch_id" json:"-"`
	TerminalID     uint64           `db:"terminal_id" json:"-"`
	PublicID       string           `db:"public_id" json:"id"`
	OpenedBy       uint64           `db:"opened_by" json:"-"`
	ClosedBy       *uint64          `db:"closed_by" json:"-"`
	Status         string           `db:"status" json:"status"`
	FloatAmount    decimal.Decimal  `db:"float_amount" json:"float"`
	OpenedAt       time.Time        `db:"opened_at" json:"opened_at"`
	ClosedAt       *time.Time       `db:"closed_at" json:"closed_at"`
	Expected       *decimal.Decimal `db:"expected_amount" json:"expected"`
	Declared       *decimal.Decimal `db:"declared_amount" json:"declared"`
	Variance       *decimal.Decimal `db:"variance" json:"variance"`
	CloseNote      string           `db:"close_note" json:"close_note"`
	VarianceStatus string           `db:"variance_status" json:"variance_status"`
	VarianceNote   string           `db:"variance_note" json:"variance_note"`
	ReviewedBy     *uint64          `db:"reviewed_by" json:"-"`
	ReviewedAt     *time.Time       `db:"reviewed_at" json:"reviewed_at"`

	// Joined display fields.
	OpenedByName string `db:"opened_by_name" json:"opened_by_name"`
	ClosedByName string `db:"closed_by_name" json:"closed_by_name"`
	Terminal     string `db:"terminal_name" json:"terminal"`
	Branch       string `db:"branch_name" json:"branch"`
}

// Movement is a manual drawer event inside (or beside, for drawer kicks) a
// shift: cash added, cash removed, or a no-sale drawer open.
type Movement struct {
	ID         uint64          `db:"id" json:"-"`
	TenantID   uint64          `db:"tenant_id" json:"-"`
	ShiftID    *uint64         `db:"shift_id" json:"-"`
	BranchID   uint64          `db:"branch_id" json:"-"`
	TerminalID uint64          `db:"terminal_id" json:"-"`
	PublicID   string          `db:"public_id" json:"id"`
	Kind       string          `db:"kind" json:"kind"`
	Amount     decimal.Decimal `db:"amount" json:"amount"`
	Reason     string          `db:"reason" json:"reason"`
	CreatedBy  uint64          `db:"created_by" json:"-"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`

	CreatedByName string `db:"created_by_name" json:"created_by_name"`
}

// MethodTotal is one payment method's applied total inside a shift.
type MethodTotal struct {
	Method string          `json:"method"`
	Count  int             `json:"count"`
	Total  decimal.Decimal `json:"total"`
}

// Report is the X-report payload for an open shift and the Z-report payload
// for a closed one. Expected = float + cash applied + cash in − cash out −
// cash refunds processed in the window.
type Report struct {
	Shift      Shift           `json:"shift"`
	SalesCount int             `json:"sales_count"`
	SalesTotal decimal.Decimal `json:"sales_total"`
	Methods    []MethodTotal   `json:"methods"`
	CashSales  decimal.Decimal `json:"cash_sales"`
	CashRefund decimal.Decimal `json:"cash_refunded"`
	CashIn     decimal.Decimal `json:"cash_in"`
	CashOut    decimal.Decimal `json:"cash_out"`
	Expected   decimal.Decimal `json:"expected"`
	Movements  []Movement      `json:"movements"`
}
