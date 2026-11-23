// Package mpesa will own the Daraja integration: STK Push, the Till/Paybill
// feed and reconciliation (Phase 9). Its permissions are defined now so role
// templates, the role editor, the sidebar and dashboard widgets are complete
// before it ships.
package mpesa

import "mauzo/internal/core/authz"

// M-Pesa permissions (canonical set from docs/permissions.md).
const (
	PermTransactionView       = "mpesa.transaction.view"
	PermTransactionAllocate   = "mpesa.transaction.allocate"
	PermTransactionReallocate = "mpesa.transaction.reallocate"
	PermConfigManage          = "mpesa.config.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermTransactionView, Module: "mpesa", Description: "View the incoming M-Pesa feed"},
		authz.Permission{Code: PermTransactionAllocate, Module: "mpesa", Description: "Allocate an M-Pesa payment to a sale"},
		authz.Permission{Code: PermTransactionReallocate, Module: "mpesa", Description: "Correct a wrong M-Pesa allocation", Sensitive: true},
		authz.Permission{Code: PermConfigManage, Module: "mpesa", Description: "Manage Daraja (Till/Paybill) configuration", Sensitive: true},
	)
}
