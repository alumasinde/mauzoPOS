package authz

import "github.com/shopspring/decimal"

// RoleTemplate is a default tenant role. Every tenant gets one role per
// template. A system role follows its template (re-expanded at boot, so
// permissions added by later phases flow in) until the tenant edits it; after
// that the tenant's choice wins.
type RoleTemplate struct {
	Code        string
	Name        string
	Description string
	// Locked roles (owner) cannot be edited, deleted or emptied and implicitly
	// hold every tenant permission.
	Locked bool
	// AllBranches is the default branch access for users created in this role.
	AllBranches bool
	// Grants are Expand patterns.
	Grants []string
	// Limits (nil = unlimited) enforced by sales/POS in later phases.
	MaxDiscountPercent *decimal.Decimal
	MaxRefundAmount    *decimal.Decimal
	MaxCashOutAmount   *decimal.Decimal
}

// Role template codes.
const (
	RoleOwner            = "owner"
	RoleAdmin            = "admin"
	RoleManager          = "manager"
	RoleAccountant       = "accountant"
	RoleInventoryManager = "inventory_manager"
	RoleSupervisor       = "supervisor"
	RoleCashier          = "cashier"
)

func dec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

// posBasics are what every till operator needs.
var posBasics = []string{
	"pos.access", "pos.sale.create", "pos.sale.hold", "pos.sale.recall", "pos.sale.cancel",
	"pos.discount.apply", "pos.shift.open", "pos.shift.close",
	"mpesa.transaction.view", "mpesa.transaction.allocate",
}

// RoleTemplates is the default role set (see docs/permissions.md §3).
var RoleTemplates = []RoleTemplate{
	{
		Code: RoleOwner, Name: "Owner", Locked: true, AllBranches: true,
		Description: "Full control of the business, including billing and roles.",
		Grants:      []string{"*"},
	},
	{
		Code: RoleAdmin, Name: "Administrator", AllBranches: true,
		Description: "Manages settings, employees, roles and every module.",
		Grants:      []string{"*"},
	},
	{
		Code: RoleManager, Name: "Manager", AllBranches: true,
		Description:        "Runs day-to-day operations across branches.",
		MaxDiscountPercent: dec("50"),
		Grants: []string{
			"dashboard.view", "settings.business.view", "audit.log.view",
			"users.employee.*", "branches.*", "terminals.terminal.view", "terminals.terminal.manage",
			"catalog.*", "pricing.*", "promotions.*", "inventory.*", "purchasing.*",
			"sales.*", "customers.*", "payments.*", "shifts.*", "reports.*",
			"mpesa.transaction.*", "pos.*",
		},
	},
	{
		Code: RoleAccountant, Name: "Accountant", AllBranches: true,
		Description: "Finance: payments, tax, fiscal invoices and reports.",
		Grants: []string{
			"dashboard.view", "audit.log.view", "sales.sale.view", "sales.sale.view_all_branches",
			"payments.*", "mpesa.transaction.view", "mpesa.transaction.reallocate",
			"tax.*", "etims.*", "shifts.shift.view", "shifts.variance.approve",
			"pricing.cost.view", "purchasing.*.view", "reports.*",
		},
	},
	{
		Code: RoleInventoryManager, Name: "Inventory manager", AllBranches: false,
		Description: "Products, stock, purchasing and suppliers.",
		Grants: []string{
			"dashboard.view", "catalog.*", "inventory.stock.view", "inventory.adjustment.create",
			"inventory.transfer.*", "inventory.stocktake.manage", "inventory.wastage.create",
			"purchasing.*", "pricing.price.view", "pricing.cost.view", "reports.inventory.view",
		},
	},
	{
		Code: RoleSupervisor, Name: "Supervisor", AllBranches: false,
		Description:        "Oversees tills and authorises sensitive actions.",
		MaxDiscountPercent: dec("20"),
		MaxRefundAmount:    dec("5000"),
		MaxCashOutAmount:   dec("5000"),
		Grants: append([]string{
			"dashboard.view", "sales.sale.view", "sales.sale.void", "sales.refund.create",
			"shifts.shift.view", "reports.sales.view", "pos.*", "mpesa.transaction.*",
		}, posBasics...),
	},
	{
		Code: RoleCashier, Name: "Cashier", AllBranches: false,
		Description:        "Rings up sales at the till.",
		MaxDiscountPercent: dec("5"),
		Grants:             posBasics,
	},
}

// Template returns the template for code.
func Template(code string) (RoleTemplate, bool) {
	for _, t := range RoleTemplates {
		if t.Code == code {
			return t, true
		}
	}
	return RoleTemplate{}, false
}
