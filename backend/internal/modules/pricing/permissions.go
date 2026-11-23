package pricing

import "mauzo/internal/core/authz"

// Pricing & promotion permissions (canonical set from docs/permissions.md).
const (
	PermPriceView   = "pricing.price.view"
	PermPriceUpdate = "pricing.price.update"
	PermCostView    = "pricing.cost.view"
	PermPromoManage = "promotions.promotion.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermPriceView, Module: "pricing", Description: "View prices and price lists"},
		authz.Permission{Code: PermPriceUpdate, Module: "pricing", Description: "Manage price lists and change selling prices"},
		authz.Permission{Code: PermCostView, Module: "pricing", Description: "See purchase cost and margins"},
		authz.Permission{Code: PermPromoManage, Module: "promotions", Description: "Manage promotions"},
	)
}
