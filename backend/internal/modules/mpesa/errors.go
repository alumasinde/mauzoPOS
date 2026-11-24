package mpesa

import "mauzo/internal/core/httpx"

// Sentinel errors for the mpesa module.
var (
	ErrNotConfigured    = httpx.NewError(422, "mpesa_not_configured", "M-Pesa is not configured for this business.")
	ErrNoTransaction    = httpx.NewError(422, "mpesa_txn_not_found", "No M-Pesa transaction with that reference.")
	ErrNotConfirmed     = httpx.NewError(422, "mpesa_not_confirmed", "This M-Pesa payment has not completed.")
	ErrAlreadyAllocated = httpx.NewError(422, "mpesa_already_allocated", "This M-Pesa payment is already allocated to a sale.")
	ErrAmountMismatch   = httpx.NewError(422, "mpesa_amount_mismatch", "The M-Pesa amount does not match the sale payment.")
	ErrInvalidPhone     = httpx.NewError(422, "mpesa_invalid_phone", "Enter a Safaricom number — 07…, 01… or 2547…")
	ErrInvalidAmount    = httpx.NewError(422, "mpesa_invalid_amount", "Amount must be a whole-shilling value of at least 1.")
	ErrRateLimited      = httpx.NewError(429, "mpesa_rate_limited", "Too many M-Pesa prompts today for this business.")
	ErrNoInitiator      = httpx.NewError(422, "mpesa_no_initiator", "Reversal needs initiator credentials in M-Pesa settings.")
	ErrNoReversal       = httpx.NewError(422, "mpesa_not_reversible", "Only confirmed, un-reversed collections can be reversed.")
	ErrBadKind          = httpx.NewError(422, "mpesa_bad_kind", "short_code_kind must be 'till' or 'paybill'.")
	ErrBranchMismatch   = httpx.NewError(422, "mpesa_branch_mismatch", "This collection was taken at a different branch.")
	ErrSaleNotPayable   = httpx.NewError(422, "mpesa_sale_not_payable", "Only completed sales can receive an M-Pesa allocation.")
	ErrInvalidShortCode = httpx.NewError(422, "mpesa_invalid_short_code", "Each till / store number must be digits and listed once.")
)
