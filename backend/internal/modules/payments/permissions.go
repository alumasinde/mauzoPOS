// Package payments will own the payment engine and payment records
// (Phase 7). Its permissions are defined now so role templates, the role
// editor, the sidebar and dashboard widgets are complete before it ships.
package payments

import "mauzo/internal/core/authz"

// Payment permissions (canonical set from docs/permissions.md).
const (
	PermPaymentView = "payments.payment.view"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermPaymentView, Module: "payments", Description: "View payment records and cash collections"},
	)
}
