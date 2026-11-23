package modules

// Core modules: always on for every tenant.
var coreModules = []Module{
	{
		Code: "dashboard", Name: "Dashboard", Group: GroupCore, Core: true,
		Description: "Business overview and quick status.",
		Nav:         []NavItem{{Key: "dashboard", Label: "Dashboard", Path: "/", Icon: "home", Section: SectionTop, Permission: "dashboard.view"}},
	},
	{
		Code: "settings", Name: "Business settings", Group: GroupCore, Core: true,
		Description: "Business profile, setup wizard, preferences, modules and audit log.",
		Nav: []NavItem{
			{Key: "settings", Label: "Business settings", Path: "/settings", Icon: "store", Section: SectionConfig, Permission: "settings.business.view", Order: 1},
			{Key: "setup", Label: "Setup wizard", Path: "/onboarding", Icon: "rocket", Section: SectionConfig, Permission: "settings.business.update", Order: 2},
			{Key: "modules", Label: "Modules", Path: "/modules", Icon: "puzzle", Section: SectionConfig, Permission: "settings.module.manage", Order: 3},
			{Key: "integrations", Label: "Integrations", Path: "/settings/integrations", Icon: "plug", Section: SectionConfig, Permission: "settings.business.update", Order: 5},
			{Key: "audit", Label: "Audit log", Path: "/audit", Icon: "shield", Section: SectionTeam, Permission: "audit.log.view", Order: 9},
		},
	},
	{
		Code: "users", Name: "Employees", Group: GroupCore, Core: true,
		Description: "Employees, cashier accounts and branch access.",
		Nav:         []NavItem{{Key: "employees", Label: "Employees", Path: "/employees", Icon: "users", Section: SectionTeam, Permission: "users.employee.view", Order: 1}},
	},
	{
		Code: "roles", Name: "Roles & permissions", Group: GroupCore, Core: true,
		Description: "Role templates, custom roles and permission limits.",
		Nav:         []NavItem{{Key: "roles", Label: "Roles & permissions", Path: "/roles", Icon: "key", Section: SectionTeam, Permission: "roles.role.view", Order: 2}},
	},
	{
		Code: "branches", Name: "Branches", Group: GroupCore, Core: true,
		Description: "Branches and locations.",
		Nav:         []NavItem{{Key: "branches", Label: "Branches", Path: "/branches", Icon: "building", Section: SectionTeam, Permission: "branches.branch.view", Order: 3}},
	},
}

// Operational modules.
var operationsModules = []Module{
	{Code: "terminals", Name: "POS terminals", Group: GroupOperations, Depends: []string{"branches"},
		Description: "Windows POS terminals, registration and licensing.",
		Nav:         []NavItem{{Key: "terminals", Label: "POS terminals", Path: "/terminals", Icon: "monitor", Section: SectionTeam, Permission: "terminals.terminal.view", Order: 4}},
		Widgets:     []Widget{{Key: "terminal_status", Title: "Terminal status", Size: WidgetNarrow, Permission: "terminals.terminal.view", Order: 31}},
	},
	{Code: "catalog", Name: "Products & catalog", Group: GroupOperations,
		Description: "Products, categories, brands, units and barcodes.",
		Nav: []NavItem{
			{Key: "catalog", Label: "Products", Path: "/catalog", Icon: "box", Section: SectionInventory, Permission: "catalog.product.view", Order: 1},
			{Key: "catalog-taxonomy", Label: "Categories", Path: "/catalog/taxonomy", Icon: "tags", Section: SectionInventory, Permission: "catalog.product.view", Order: 2},
		},
	},
	{Code: "pricing", Name: "Pricing & price lists", Group: GroupOperations, Depends: []string{"catalog"},
		Description: "Price lists, bulk and tiered pricing.",
		Nav: []NavItem{
			{Key: "pricing", Label: "Price lists", Path: "/pricing", Icon: "tag", Section: SectionPricing, Permission: "pricing.price.view", Order: 1},
			{Key: "pricing-history", Label: "Price history", Path: "/pricing/history", Icon: "history", Section: SectionPricing, Permission: "pricing.price.view", Order: 3},
		}},
	{Code: "promotions", Name: "Promotions", Group: GroupGrowth, Depends: []string{"pricing"},
		Description:    "Discount rules and time-bound promotions.",
		RecommendedFor: []string{"supermarket", "minimart", "liquor"},
		Nav: []NavItem{
			{Key: "promotions", Label: "Promotions", Path: "/pricing/promotions", Icon: "percent", Section: SectionPricing, Permission: "promotions.promotion.manage", Order: 2},
		}},
	{Code: "inventory", Name: "Inventory", Group: GroupOperations, Depends: []string{"catalog"},
		Description: "Stock ledger, adjustments, transfers, stocktakes, batches and expiry.",
		Nav: []NavItem{
			{Key: "inventory", Label: "Stock overview", Path: "/inventory", Icon: "layers", Section: SectionInventory, Permission: "inventory.stock.view", Order: 3},
			{Key: "inventory-movements", Label: "Stock movements", Path: "/inventory?tab=movements", Icon: "arrows", Section: SectionInventory, Permission: "inventory.stock.view", Order: 4},
			{Key: "inventory-transfers", Label: "Stock transfers", Path: "/inventory/documents?tab=transfers", Icon: "transfer", Section: SectionInventory, Permission: "inventory.stock.view", Order: 5},
			{Key: "inventory-docs", Label: "Adjustments & counts", Path: "/inventory/documents", Icon: "clipboard", Section: SectionInventory, Permission: "inventory.stock.view", Order: 6},
		},
		Widgets: []Widget{{Key: "inventory_alerts", Title: "Inventory alerts", Size: WidgetNarrow, Permission: "inventory.stock.view", Order: 22}},
	},
	{Code: "purchasing", Name: "Purchasing & suppliers", Group: GroupOperations, Depends: []string{"inventory"},
		Description: "Suppliers, purchase orders and goods received.",
		Nav: []NavItem{
			{Key: "suppliers", Label: "Suppliers", Path: "/purchasing/suppliers", Icon: "truck", Section: SectionPurchasing, Permission: "purchasing.supplier.view", Order: 1},
			{Key: "purchase-orders", Label: "Purchase orders", Path: "/purchasing/orders", Icon: "file", Section: SectionPurchasing, Permission: "purchasing.order.view", Order: 2},
			{Key: "goods-receiving", Label: "Goods receiving", Path: "/purchasing/receiving", Icon: "inbox", Section: SectionPurchasing, Permission: "purchasing.grn.view", Order: 3},
		}},
	{Code: "sales", Name: "Sales", Group: GroupOperations, Depends: []string{"catalog", "payments"},
		Description: "Sales, returns, refunds and voids.",
		Nav: []NavItem{
			{Key: "transactions", Label: "Transactions", Path: "/sales", Icon: "receipt", Section: SectionSales, Permission: "sales.sale.view", Order: 1},
			{Key: "sales-history", Label: "Sales history", Path: "/sales/history", Icon: "history", Section: SectionSales, Permission: "sales.sale.view", Order: 2},
			{Key: "returns", Label: "Returns & refunds", Path: "/sales/returns", Icon: "undo", Section: SectionSales, Permission: "sales.sale.view", Order: 3},
			{Key: "held-sales", Label: "Held sales", Path: "/sales/held", Icon: "pause", Section: SectionSales, Permission: "sales.sale.view", Order: 4},
		},
		Widgets: []Widget{
			{Key: "sales_total", Title: "Total sales", Size: WidgetKPI, Permission: "sales.sale.view", Order: 1},
			{Key: "sales_count", Title: "Transactions", Size: WidgetKPI, Permission: "sales.sale.view", Order: 2},
			{Key: "sales_average", Title: "Average sale", Size: WidgetKPI, Permission: "sales.sale.view", Order: 3},
			{Key: "sales_overview", Title: "Sales overview", Size: WidgetWide, Permission: "sales.sale.view", Order: 20},
			{Key: "recent_sales", Title: "Recent transactions", Size: WidgetWide, Permission: "sales.sale.view", Order: 30},
			{Key: "held_sales", Title: "Held sales", Size: WidgetNarrow, Permission: "sales.sale.view", Order: 32},
		}},
	{Code: "customers", Name: "Customers", Group: GroupGrowth,
		Description: "Customer records and purchase history."},
	{Code: "receipts", Name: "Receipts", Group: GroupOperations, Depends: []string{"sales"},
		Description: "Receipt designer and printing.",
		Nav:         []NavItem{{Key: "receipt-designer", Label: "Receipt designer", Path: "/receipts/designer", Icon: "printer", Section: SectionConfig, Permission: "receipts.template.manage", Order: 4}},
	},
	{Code: "shifts", Name: "Cashier shifts", Group: GroupOperations, Depends: []string{"sales", "terminals"},
		Description: "Shift open/close, cash counts, X/Z reports."},
	{Code: "reports", Name: "Reports", Group: GroupOperations,
		Description: "Sales, stock, tax and financial reports.",
		Nav: []NavItem{
			{Key: "reports-sales", Label: "Sales reports", Path: "/reports/sales", Icon: "chart", Section: SectionReports, Permission: "reports.sales.view", Order: 1},
			{Key: "reports-inventory", Label: "Inventory reports", Path: "/reports/inventory", Icon: "chart-bars", Section: SectionReports, Permission: "reports.inventory.view", Order: 2},
			{Key: "reports-tax", Label: "Tax reports", Path: "/reports/tax", Icon: "calculator", Section: SectionReports, Permission: "reports.tax.view", Order: 3},
		}},
}

// Finance modules.
var financeModules = []Module{
	{Code: "payments", Name: "Payments", Group: GroupFinance,
		Description: "Payment methods, split payments and payment records.",
		Nav:         []NavItem{{Key: "cash-management", Label: "Cash management", Path: "/payments/cash", Icon: "wallet", Section: SectionPayments, Permission: "payments.payment.view", Order: 2}},
		Widgets: []Widget{
			{Key: "cash_collections", Title: "Cash collections", Size: WidgetKPI, Permission: "payments.payment.view", Order: 5},
			{Key: "payment_methods", Title: "Payment methods", Size: WidgetNarrow, Permission: "payments.payment.view", Order: 21},
		}},
	{Code: "mpesa", Name: "M-Pesa", Group: GroupFinance, Depends: []string{"payments"},
		Description: "Daraja STK Push, Till/Paybill feed and reconciliation.",
		Nav:         []NavItem{{Key: "mpesa", Label: "M-Pesa reconciliation", Path: "/mpesa", Icon: "phone", Section: SectionPayments, Permission: "mpesa.transaction.view", Order: 1}},
		Widgets:     []Widget{{Key: "mpesa_collections", Title: "M-Pesa collections", Size: WidgetKPI, Permission: "mpesa.transaction.view", Order: 4}},
	},
	{Code: "tax", Name: "Tax", Group: GroupFinance,
		Description: "Tax categories, effective-dated rates and VAT breakdowns."},
	{Code: "etims", Name: "eTIMS fiscalisation", Group: GroupFinance, Depends: []string{"tax", "sales"},
		Description: "KRA eTIMS invoice submission and fiscal references."},
	{Code: "credit", Name: "Customer credit", Group: GroupFinance, Depends: []string{"customers", "payments"},
		Description: "Customer accounts, credit limits and statements.",
		ConfigSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"default_limit_kes": map[string]any{"type": "number", "minimum": 0},
			},
		}},
}

// Industry & growth modules.
var industryModules = []Module{
	{Code: "weighing_scales", Name: "Weighing scales", Group: GroupIndustry, Depends: []string{"catalog"},
		Description:    "Scale integration and price/weight-embedded barcodes.",
		RecommendedFor: []string{"butchery", "supermarket", "grocery"},
		ConfigSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"barcode_prefixes": map[string]any{"type": "array", "maxItems": 10,
					"items": map[string]any{"type": "string", "minLength": 2, "maxLength": 3}},
				"embedded": map[string]any{"type": "string", "enum": []any{"price", "weight"}},
			},
		}},
	{Code: "restaurant", Name: "Restaurant (tables, KOT)", Group: GroupIndustry, Depends: []string{"catalog", "sales"},
		Description:    "Tables, menus, kitchen orders, recipes and service charge.",
		RecommendedFor: []string{"restaurant"},
		ConfigSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"service_charge_percent": map[string]any{"type": "number", "minimum": 0, "maximum": 25},
				"tables_enabled":         map[string]any{"type": "boolean"},
			},
		}},
	{Code: "loyalty", Name: "Loyalty & rewards", Group: GroupGrowth, Depends: []string{"customers", "sales"},
		Description: "Points earning and redemption.",
		ConfigSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"points_per_100_kes": map[string]any{"type": "number", "minimum": 0},
				"point_value_kes":    map[string]any{"type": "number", "minimum": 0},
			},
		}},
	{Code: "delivery", Name: "Delivery management", Group: GroupGrowth, Depends: []string{"sales", "customers"},
		Description: "Delivery orders, riders and fees.",
		ConfigSchema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"fee_kes": map[string]any{"type": "number", "minimum": 0},
			},
		}},
}

func init() {
	register(coreModules...)
	register(operationsModules...)
	register(financeModules...)
	register(industryModules...)
	validateGraph()
}
