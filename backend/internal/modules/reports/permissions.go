// Package reports will own sales, stock, tax and financial reports
// (Phase 13). Its permissions are defined now so role templates, the role
// editor and the sidebar are complete before it ships.
package reports

import "mauzo/internal/core/authz"

// Report permissions (canonical set from docs/permissions.md).
const (
	PermSalesView     = "reports.sales.view"
	PermInventoryView = "reports.inventory.view"
	PermTaxView       = "reports.tax.view"
	PermFinancialView = "reports.financial.view"
	PermExport        = "reports.any.export"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermSalesView, Module: "reports", Description: "View sales reports"},
		authz.Permission{Code: PermInventoryView, Module: "reports", Description: "View inventory reports"},
		authz.Permission{Code: PermTaxView, Module: "reports", Description: "View tax reports"},
		authz.Permission{Code: PermFinancialView, Module: "reports", Description: "View financial reports"},
		authz.Permission{Code: PermExport, Module: "reports", Description: "Export reports"},
	)
}
