// Package purchasing will own suppliers, purchase orders and goods received
// notes (Phase 11). Its permissions are defined now so role templates, the
// role editor and the sidebar are complete before it ships.
package purchasing

import "mauzo/internal/core/authz"

// Purchasing permissions (canonical set from docs/permissions.md).
const (
	PermSupplierView   = "purchasing.supplier.view"
	PermSupplierManage = "purchasing.supplier.manage"
	PermOrderView      = "purchasing.order.view"
	PermOrderCreate    = "purchasing.order.create"
	PermOrderApprove   = "purchasing.order.approve"
	PermGRNView        = "purchasing.grn.view"
	PermGRNCreate      = "purchasing.grn.create"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermSupplierView, Module: "purchasing", Description: "View suppliers"},
		authz.Permission{Code: PermSupplierManage, Module: "purchasing", Description: "Create and edit suppliers"},
		authz.Permission{Code: PermOrderView, Module: "purchasing", Description: "View purchase orders"},
		authz.Permission{Code: PermOrderCreate, Module: "purchasing", Description: "Create purchase orders"},
		authz.Permission{Code: PermOrderApprove, Module: "purchasing", Description: "Approve purchase orders"},
		authz.Permission{Code: PermGRNView, Module: "purchasing", Description: "View goods received notes"},
		authz.Permission{Code: PermGRNCreate, Module: "purchasing", Description: "Receive goods against purchase orders"},
	)
}
