// Package sales will own completed sales, returns, refunds and voids
// (Phase 7). Its permissions are defined now so role templates, the role
// editor, the sidebar and dashboard widgets are complete before it ships.
package sales

import "mauzo/internal/core/authz"

// Sales permissions (canonical set from docs/permissions.md).
const (
	PermSaleView            = "sales.sale.view"
	PermSaleViewAllBranches = "sales.sale.view_all_branches"
	PermSaleVoid            = "sales.sale.void"
	PermRefundCreate        = "sales.refund.create"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermSaleView, Module: "sales", Description: "View sales and transactions in your branches"},
		authz.Permission{Code: PermSaleViewAllBranches, Module: "sales", Description: "View sales across every branch"},
		authz.Permission{Code: PermSaleVoid, Module: "sales", Description: "Void a completed sale", Sensitive: true},
		authz.Permission{Code: PermRefundCreate, Module: "sales", Description: "Process returns and refunds", Sensitive: true},
	)
}
