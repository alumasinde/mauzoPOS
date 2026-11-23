package inventory

import "mauzo/internal/core/authz"

// Inventory permissions (canonical set from docs/permissions.md).
const (
	PermStockView         = "inventory.stock.view"
	PermAdjustmentCreate  = "inventory.adjustment.create"
	PermAdjustmentApprove = "inventory.adjustment.approve"
	PermTransferCreate    = "inventory.transfer.create"
	PermTransferReceive   = "inventory.transfer.receive"
	PermStocktakeManage   = "inventory.stocktake.manage"
	PermWastageCreate     = "inventory.wastage.create"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermStockView, Module: "inventory", Description: "View stock balances, movements, locations and alerts"},
		authz.Permission{Code: PermAdjustmentCreate, Module: "inventory", Description: "Create stock adjustments, locations, reorder levels and receive stock"},
		authz.Permission{Code: PermAdjustmentApprove, Module: "inventory", Description: "Approve and post stock adjustments"},
		authz.Permission{Code: PermTransferCreate, Module: "inventory", Description: "Create, send and cancel branch transfers"},
		authz.Permission{Code: PermTransferReceive, Module: "inventory", Description: "Receive branch transfers"},
		authz.Permission{Code: PermStocktakeManage, Module: "inventory", Description: "Run stocktakes and cycle counts"},
		authz.Permission{Code: PermWastageCreate, Module: "inventory", Description: "Record wastage"},
	)
}
