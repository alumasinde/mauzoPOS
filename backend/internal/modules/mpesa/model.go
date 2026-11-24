// Package mpesa owns the Daraja integration: per-tenant credentials, STK
// push initiation and callbacks, the C2B confirmation feed, allocation of
// incoming payments to sales, reversals and reconciliation.
package mpesa

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction kinds.
const (
	KindSTK      = "stk"
	KindC2B      = "c2b"
	KindReversal = "reversal"
)

// Transaction statuses.
const (
	StInitiated  = "initiated" // push queued at Daraja, no result yet
	StPending    = "pending"   // prompt on the customer's phone
	StSuccess    = "success"   // money collected
	StFailed     = "failed"    // Daraja/issuer failure
	StCancelled  = "cancelled" // customer dismissed the prompt
	StTimeout    = "timeout"   // no result in the window
	StRevPending = "reversal_pending"
	StReversed   = "reversed"
)

// Event kinds for mpesa_events.
const (
	EvSTKCallback    = "stk_callback"
	EvC2BConfirm     = "c2b_confirmation"
	EvC2BValidation  = "c2b_validation"
	EvReversalResult = "reversal_result"
	EvAllocation     = "allocation"
	EvReversalReq    = "reversal_request"
	EvStatusQuery    = "status_query"
	EvConfig         = "config"
	EvError          = "error"
)

// Transaction is one incoming (or reversing) M-Pesa payment.
type Transaction struct {
	ID                uint64          `db:"id" json:"id"`
	TenantID          uint64          `db:"tenant_id" json:"-"`
	PublicID          string          `db:"public_id" json:"public_id"`
	BranchID          *uint64         `db:"branch_id" json:"branch_id"`
	Kind              string          `db:"kind" json:"kind"`
	Status            string          `db:"status" json:"status"`
	TransID           *string         `db:"trans_id" json:"trans_id"`
	CheckoutRequestID *string         `db:"checkout_request_id" json:"checkout_request_id"`
	MerchantRequestID *string         `db:"merchant_request_id" json:"merchant_request_id"`
	Phone             string          `db:"phone" json:"phone"`
	PayerName         string          `db:"payer_name" json:"payer_name"`
	Amount            decimal.Decimal `db:"amount" json:"amount"`
	AccountRef        string          `db:"account_ref" json:"account_ref"`
	ShortCode         string          `db:"short_code" json:"short_code"`
	ResultCode        *int            `db:"result_code" json:"result_code"`
	ResultDesc        string          `db:"result_desc" json:"result_desc"`
	AllocatedSaleID   *uint64         `db:"allocated_sale_id" json:"allocated_sale_id"`
	AllocatedBy       *uint64         `db:"allocated_by" json:"allocated_by"`
	AllocatedAt       *time.Time      `db:"allocated_at" json:"allocated_at"`
	TransTime         *time.Time      `db:"trans_time" json:"trans_time"`
	InitiatedBy       *uint64         `db:"initiated_by" json:"initiated_by"`
	LastQueriedAt     *time.Time      `db:"last_queried_at" json:"last_queried_at"`
	Raw               []byte          `db:"raw" json:"-"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

// Credentials is the stored Daraja configuration (secrets sealed).
type Credentials struct {
	ID                    uint64     `db:"id"`
	TenantID              uint64     `db:"tenant_id"`
	Env                   string     `db:"env"`
	ShortCode             string     `db:"short_code"`
	ShortCodeKind         string     `db:"short_code_kind"`
	ConsumerKeyEnc        []byte     `db:"consumer_key_enc"`
	ConsumerSecretEnc     []byte     `db:"consumer_secret_enc"`
	PasskeyEnc            []byte     `db:"passkey_enc"`
	InitiatorName         string     `db:"initiator_name"`
	SecurityCredentialEnc []byte     `db:"security_credential_enc"`
	CallbackToken         string     `db:"callback_token"`
	C2BRegisteredAt       *time.Time `db:"c2b_registered_at"`
	STKPushesToday        int        `db:"stk_pushes_today"`
	STKPushesOn           *time.Time `db:"stk_pushes_on"`
	CreatedAt             time.Time  `db:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"`
}

// Reversal tracks one merchant-initiated reversal request.
type Reversal struct {
	ID                       uint64          `db:"id" json:"id"`
	TenantID                 uint64          `db:"tenant_id" json:"-"`
	TxnID                    uint64          `db:"txn_id" json:"txn_id"`
	OriginatorConversationID string          `db:"originator_conversation_id" json:"originator_conversation_id"`
	ConversationID           string          `db:"conversation_id" json:"conversation_id"`
	Amount                   decimal.Decimal `db:"amount" json:"amount"`
	Reason                   string          `db:"reason" json:"reason"`
	Status                   string          `db:"status" json:"status"`
	ResultCode               *int            `db:"result_code" json:"result_code"`
	ResultDesc               string          `db:"result_desc" json:"result_desc"`
	RequestedBy              *uint64         `db:"requested_by" json:"requested_by"`
	CreatedAt                time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time       `db:"updated_at" json:"updated_at"`
}

// Event is an audit row for inbound callbacks and admin actions.
type Event struct {
	ID        uint64    `db:"id" json:"id"`
	TenantID  uint64    `db:"tenant_id" json:"-"`
	TxnID     *uint64   `db:"txn_id" json:"txn_id"`
	Kind      string    `db:"kind" json:"kind"`
	Ref       string    `db:"ref" json:"ref"`
	Detail    string    `db:"detail" json:"detail"`
	Payload   []byte    `db:"payload" json:"payload"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ListFilter selects feed rows.
type ListFilter struct {
	Status    string
	Allocated string // "" | "yes" | "no"
	Kind      string
	Query     string // phone / trans_id / account_ref search
	From, To  string
	BranchID  uint64
}

// Summary feeds the dashboard KPI + page stat cards.
type Summary struct {
	CollectedToday decimal.Decimal `db:"collected_today" json:"collected_today"`
	CountToday     int             `db:"count_today" json:"count_today"`
	Unallocated    int             `db:"unallocated" json:"unallocated"`
	Pending        int             `db:"pending" json:"pending"`
	Failed7d       int             `db:"failed_7d" json:"failed_7d"`
	Reversed       int             `db:"reversed" json:"reversed"`
}
