package sales

import (
	"errors"

	"mauzo/internal/core/httpx"
)

// MethodSpec is one payment method's validation rules. M-Pesa joins this
// registry in Phase 9 with an async confirmation state.
type MethodSpec struct {
	Label             string `json:"label"`
	RequiresReference bool   `json:"requires_reference"` // card slip / bank transfer ref
	RequiresCustomer  bool   `json:"requires_customer"`  // credit account sales
	Tendered          bool   `json:"tendered"`           // accepts a tendered amount and computes change
}

// methods is the payment-method registry.
var methods = map[string]MethodSpec{
	"cash":   {Label: "Cash", Tendered: true},
	"card":   {Label: "Card", RequiresReference: true},
	"bank":   {Label: "Bank transfer", RequiresReference: true},
	"credit": {Label: "Customer credit", RequiresCustomer: true},
	// mpesa: reference is the M-Pesa transaction code (e.g. QKR7ABC123); a
	// confirmed feed transaction with that code is consumed at completion.
	"mpesa": {Label: "M-Pesa", RequiresReference: true},
}

// Methods returns the registry for API clients.
func Methods() map[string]MethodSpec {
	out := make(map[string]MethodSpec, len(methods))
	for k, v := range methods {
		out[k] = v
	}
	return out
}

// Sentinel errors.
var (
	ErrSaleNotFound    = httpx.ErrNotFound
	ErrNoLines         = httpx.NewError(422, "no_lines", "A sale needs at least one line.")
	ErrNoPayments      = httpx.NewError(422, "no_payments", "A sale needs at least one payment.")
	ErrUnderpaid       = httpx.NewError(422, "underpaid", "Payments do not cover the sale total.")
	ErrOverpaid        = httpx.NewError(422, "overpaid", "Payments exceed the sale total.")
	ErrUnknownMethod   = httpx.NewError(422, "unknown_method", "Unknown payment method.")
	ErrNeedReference   = httpx.NewError(422, "reference_required", "This payment method needs a reference.")
	ErrNeedCustomer    = httpx.NewError(422, "customer_required", "Credit sales need a customer account.")
	ErrUnderTendered   = httpx.NewError(422, "under_tendered", "Tendered amount is less than the payment.")
	ErrNotVoidable     = httpx.NewError(422, "not_voidable", "Only completed sales can be voided.")
	ErrNotReturnable   = httpx.NewError(422, "not_returnable", "This sale has nothing left to return.")
	ErrOverReturn      = httpx.NewError(422, "over_return", "Return quantity exceeds what was sold.")
	ErrRefundTooLarge  = httpx.NewError(422, "refund_over_limit", "Refund is above this role's limit — a supervisor must authorize it.")
	ErrDiscountOver    = httpx.NewError(422, "discount_over_limit", "Discount is above this role's limit — a supervisor must authorize it.")
	ErrPriceOverride   = httpx.NewError(403, "price_override_denied", "This cashier may not override prices.")
	ErrCashierNotFound = httpx.NewError(422, "cashier_invalid", "The cashier is not active on this tenant.")
	ErrCreditExceeded  = httpx.NewError(422, "credit_limit", "This sale would exceed the customer's credit limit.")
)

// ErrDuplicateSale flags an idempotency-key collision inside the transaction —
// Complete converts it into the deduplicated response.
var ErrDuplicateSale = errors.New("duplicate sale")
