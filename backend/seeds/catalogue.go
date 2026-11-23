package seeds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/modules"

	"mauzo/internal/core/security"
)

// Catalogue seeds the reference data every environment needs: the seven
// business types with their default profile templates and the initial
// subscription plans. All writes are idempotent upserts.
type Catalogue struct{}

// Name implements Seeder.
func (Catalogue) Name() string { return "catalogue" }

type seedType struct {
	code, name string
	sort       int
	profile    profileDef
}

type profileDef struct {
	Modules    []string       `json:"modules"`
	Categories []string       `json:"categories"`
	Settings   map[string]any `json:"settings"`
	Receipt    string         `json:"receipt_template"`
	PosLayout  map[string]any `json:"pos_layout"`
}

var retailModules = []string{
	"dashboard", "settings", "users", "roles", "branches", "terminals",
	"catalog", "pricing", "promotions", "inventory", "purchasing",
	"sales", "customers", "payments", "mpesa", "tax", "receipts",
	"shifts", "reports",
}

func withMods(extra ...string) []string {
	return append(append([]string{}, retailModules...), extra...)
}

var seedTypes = []seedType{
	{code: "supermarket", name: "Supermarket", sort: 10, profile: profileDef{
		Receipt:    "supermarket",
		Modules:    withMods("weighing_scales", "loyalty", "etims"),
		Categories: []string{"Groceries", "Fresh Produce", "Bakery", "Household", "Personal Care", "Beverages"},
		Settings: map[string]any{
			"payments.methods":         []string{"cash", "card", "mpesa"},
			"tax.prices_include_vat":   true,
			"tax.default_rate":         16,
			"receipt.paper_width":      80,
			"pos.scale_barcode_prefix": []string{"21", "22"},
		},
		PosLayout: map[string]any{"grid_columns": 5, "show_images": true, "numpad": true},
	}},
	{code: "minimart", name: "Mini Mart / Convenience", sort: 20, profile: profileDef{
		Receipt:    "minimart",
		Modules:    retailModules,
		Categories: []string{"Snacks", "Drinks", "Airtime", "Toiletries", "Household"},
		Settings: map[string]any{
			"payments.methods":       []string{"cash", "mpesa"},
			"tax.prices_include_vat": true,
			"tax.default_rate":       16,
			"receipt.paper_width":    58,
		},
		PosLayout: map[string]any{"grid_columns": 4, "show_images": false, "numpad": true},
	}},
	{code: "butchery", name: "Butchery", sort: 30, profile: profileDef{
		Receipt:    "supermarket",
		Modules:    withMods("weighing_scales", "etims"),
		Categories: []string{"Beef", "Goat", "Pork", "Chicken", "Fish", "Offals"},
		Settings: map[string]any{
			"payments.methods":         []string{"cash", "mpesa"},
			"tax.prices_include_vat":   false,
			"tax.default_rate":         0,
			"receipt.paper_width":      80,
			"pos.scale_barcode_prefix": []string{"21"},
		},
		PosLayout: map[string]any{"grid_columns": 4, "weighted_items": true},
	}},
	{code: "liquor", name: "Liquor Store / Wines & Spirits", sort: 40, profile: profileDef{
		Receipt:    "supermarket",
		Modules:    retailModules,
		Categories: []string{"Beer", "Spirits", "Wines", "Soft Drinks", "Mixers"},
		Settings: map[string]any{
			"payments.methods":       []string{"cash", "card", "mpesa"},
			"tax.prices_include_vat": true,
			"tax.default_rate":       16,
			"receipt.paper_width":    58,
		},
		PosLayout: map[string]any{"grid_columns": 4, "age_check": true},
	}},
	{code: "pharmacy", name: "Pharmacy / Chemist", sort: 50, profile: profileDef{
		Receipt:    "supermarket",
		Modules:    withMods("etims"),
		Categories: []string{"Prescription", "OTC Medicine", "Personal Care", "Baby Care", "Supplements"},
		Settings: map[string]any{
			"payments.methods":       []string{"cash", "card", "mpesa"},
			"tax.prices_include_vat": false,
			"tax.default_rate":       0,
			"receipt.paper_width":    80,
		},
		PosLayout: map[string]any{"grid_columns": 4, "expiry_tracking": true},
	}},
	{code: "restaurant", name: "Restaurant / Eatery", sort: 60, profile: profileDef{
		Receipt:    "restaurant",
		Modules:    withMods("restaurant", "delivery"),
		Categories: []string{"Breakfast", "Mains", "Fast Food", "Drinks", "Desserts"},
		Settings: map[string]any{
			"payments.methods":       []string{"cash", "card", "mpesa"},
			"tax.prices_include_vat": true,
			"tax.default_rate":       16,
			"receipt.paper_width":    80,
		},
		PosLayout: map[string]any{"grid_columns": 4, "tables_first": true},
	}},
	{code: "custom", name: "Other / Custom", sort: 90, profile: profileDef{
		Receipt:    "supermarket",
		Modules:    retailModules,
		Categories: []string{"General"},
		Settings: map[string]any{
			"payments.methods":       []string{"cash", "mpesa"},
			"tax.prices_include_vat": true,
			"tax.default_rate":       16,
			"receipt.paper_width":    80,
		},
		PosLayout: map[string]any{"grid_columns": 4},
	}},
}

type seedPlan struct {
	name     string
	price    string
	interval string
	modules  []string
	branches *int
	users    *int
	sort     int
}

func intPtr(n int) *int { return &n }

var seedPlans = []seedPlan{
	{name: "Starter", price: "1500", interval: "monthly", sort: 10,
		modules: []string{"dashboard", "settings", "users", "branches", "terminals",
			"catalog", "inventory", "sales", "payments", "mpesa", "receipts"},
		branches: intPtr(1), users: intPtr(3)},
	{name: "Business", price: "4000", interval: "monthly", sort: 20,
		modules:  withMods("etims", "loyalty"),
		branches: intPtr(3), users: intPtr(15)},
	{name: "Enterprise", price: "12000", interval: "monthly", sort: 30,
		modules: withMods("weighing_scales", "restaurant", "loyalty", "credit",
			"delivery", "etims")},
}

// Seed implements Seeder.
func (Catalogue) Seed(ctx context.Context, conn *sqlx.DB) error {
	for _, t := range seedTypes {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO business_types (code, name, sort) VALUES (?, ?, ?)
			 ON DUPLICATE KEY UPDATE name = VALUES(name), sort = VALUES(sort)`,
			t.code, t.name, t.sort); err != nil {
			return fmt.Errorf("business type %s: %w", t.code, err)
		}
		var typeID uint64
		if err := conn.GetContext(ctx, &typeID,
			`SELECT id FROM business_types WHERE code = ?`, t.code); err != nil {
			return err
		}
		def, err := json.Marshal(t.profile)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO business_profiles (public_id, business_type_id, name, is_default, definition)
			 VALUES (?, ?, 'Standard', 1, ?)
			 ON DUPLICATE KEY UPDATE definition = VALUES(definition), is_default = 1`,
			security.NewPublicID(), typeID, def); err != nil {
			return fmt.Errorf("profile %s: %w", t.code, err)
		}
	}
	for _, p := range seedPlans {
		closure, err := modules.Closure(p.modules) // a plan always includes dependencies
		if err != nil {
			return fmt.Errorf("plan %s: %w", p.name, err)
		}
		mods, err := json.Marshal(closure)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (public_id, name, description, price_kes, billing_interval,
			 modules, max_branches, max_users, is_active, sort)
			 VALUES (?, ?, NULL, ?, ?, ?, ?, ?, 1, ?)
			 ON DUPLICATE KEY UPDATE price_kes = VALUES(price_kes),
			 billing_interval = VALUES(billing_interval), modules = VALUES(modules),
			 max_branches = VALUES(max_branches), max_users = VALUES(max_users),
			 is_active = 1, sort = VALUES(sort)`,
			security.NewPublicID(), p.name, p.price, p.interval, mods,
			p.branches, p.users, p.sort); err != nil {
			return fmt.Errorf("plan %s: %w", p.name, err)
		}
	}
	return nil
}

func init() { Register(Catalogue{}) }
