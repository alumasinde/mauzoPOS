package modules

// Core modules: always on for every tenant.
var coreModules = []Module{
	{
		Code: "dashboard", Name: "Dashboard", Group: GroupCore, Core: true,
		Description: "Business overview and quick status.",
		Nav:         []NavItem{{Key: "dashboard", Label: "Dashboard", Path: "/", Icon: "home", Permission: "dashboard.view", Order: 0}},
	},
	{
		Code: "settings", Name: "Business settings", Group: GroupCore, Core: true,
		Description: "Business profile, setup wizard, preferences, modules and audit log.",
		Nav: []NavItem{
			{Key: "setup", Label: "Setup", Path: "/onboarding", Icon: "rocket", Permission: "settings.business.update", Order: 90},
			{Key: "modules", Label: "Modules", Path: "/modules", Icon: "puzzle", Permission: "settings.module.manage", Order: 91},
			{Key: "settings", Label: "Settings", Path: "/settings", Icon: "cog", Permission: "settings.business.view", Order: 92},
			{Key: "audit", Label: "Audit log", Path: "/audit", Icon: "shield", Permission: "audit.log.view", Order: 93},
		},
	},
	{
		Code: "users", Name: "Employees", Group: GroupCore, Core: true,
		Description: "Employees, cashier accounts and branch access.",
		Nav:         []NavItem{{Key: "employees", Label: "Employees", Path: "/employees", Icon: "users", Permission: "users.employee.view", Order: 80}},
	},
	{
		Code: "roles", Name: "Roles & permissions", Group: GroupCore, Core: true,
		Description: "Role templates, custom roles and permission limits.",
		Nav:         []NavItem{{Key: "roles", Label: "Roles", Path: "/roles", Icon: "key", Permission: "roles.role.view", Order: 81}},
	},
	{
		Code: "branches", Name: "Branches", Group: GroupCore, Core: true,
		Description: "Branches and locations.",
	},
}

// Operational modules.
var operationsModules = []Module{
	{Code: "terminals", Name: "POS terminals", Group: GroupOperations, Depends: []string{"branches"},
		Description: "Windows POS terminals, registration and licensing."},
	{Code: "catalog", Name: "Products & catalog", Group: GroupOperations,
		Description: "Products, categories, brands, units and barcodes."},
	{Code: "pricing", Name: "Pricing & price lists", Group: GroupOperations, Depends: []string{"catalog"},
		Description: "Price lists, bulk and tiered pricing."},
	{Code: "promotions", Name: "Promotions", Group: GroupGrowth, Depends: []string{"pricing"},
		Description:    "Discount rules and time-bound promotions.",
		RecommendedFor: []string{"supermarket", "minimart", "liquor"}},
	{Code: "inventory", Name: "Inventory", Group: GroupOperations, Depends: []string{"catalog"},
		Description: "Stock ledger, adjustments, transfers, stocktakes, batches and expiry."},
	{Code: "purchasing", Name: "Purchasing & suppliers", Group: GroupOperations, Depends: []string{"inventory"},
		Description: "Suppliers, purchase orders and goods received."},
	{Code: "sales", Name: "Sales", Group: GroupOperations, Depends: []string{"catalog", "payments"},
		Description: "Sales, returns, refunds and voids."},
	{Code: "customers", Name: "Customers", Group: GroupGrowth,
		Description: "Customer records and purchase history."},
	{Code: "receipts", Name: "Receipts", Group: GroupOperations, Depends: []string{"sales"},
		Description: "Receipt designer and printing."},
	{Code: "shifts", Name: "Cashier shifts", Group: GroupOperations, Depends: []string{"sales", "terminals"},
		Description: "Shift open/close, cash counts, X/Z reports."},
	{Code: "reports", Name: "Reports", Group: GroupOperations,
		Description: "Sales, stock, tax and financial reports."},
}

// Finance modules.
var financeModules = []Module{
	{Code: "payments", Name: "Payments", Group: GroupFinance,
		Description: "Payment methods, split payments and payment records."},
	{Code: "mpesa", Name: "M-Pesa", Group: GroupFinance, Depends: []string{"payments"},
		Description: "Daraja STK Push, Till/Paybill feed and reconciliation."},
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
