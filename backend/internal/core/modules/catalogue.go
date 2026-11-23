// Package modules holds the canonical catalogue of feature module codes. Plans
// and profile templates allow-list against it so a typo fails validation instead
// of silently enabling nothing. Phase 3 grows this into the module registry
// (deps, permissions, availability) backed by a table.
package modules

// Module describes one enableable feature area.
type Module struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Catalogue is the known set of module codes.
var Catalogue = []Module{
	{Code: "dashboard", Name: "Dashboard"},
	{Code: "settings", Name: "Business settings"},
	{Code: "users", Name: "Employees"},
	{Code: "roles", Name: "Roles & permissions"},
	{Code: "branches", Name: "Branches"},
	{Code: "terminals", Name: "POS terminals"},
	{Code: "catalog", Name: "Products & catalog"},
	{Code: "pricing", Name: "Pricing & price lists"},
	{Code: "promotions", Name: "Promotions"},
	{Code: "inventory", Name: "Inventory"},
	{Code: "purchasing", Name: "Purchasing & suppliers"},
	{Code: "sales", Name: "Sales"},
	{Code: "customers", Name: "Customers"},
	{Code: "payments", Name: "Payments"},
	{Code: "mpesa", Name: "M-Pesa"},
	{Code: "tax", Name: "Tax"},
	{Code: "etims", Name: "eTIMS fiscalisation"},
	{Code: "receipts", Name: "Receipts"},
	{Code: "shifts", Name: "Cashier shifts"},
	{Code: "reports", Name: "Reports"},
	{Code: "weighing_scales", Name: "Weighing scales"},
	{Code: "restaurant", Name: "Restaurant (tables, KOT)"},
	{Code: "loyalty", Name: "Loyalty & rewards"},
	{Code: "credit", Name: "Customer credit"},
	{Code: "delivery", Name: "Delivery management"},
}

var codes = func() map[string]struct{} {
	m := make(map[string]struct{}, len(Catalogue))
	for _, mod := range Catalogue {
		m[mod.Code] = struct{}{}
	}
	return m
}()

// Known reports whether code is in the catalogue.
func Known(code string) bool {
	_, ok := codes[code]
	return ok
}

// Codes returns every catalogue code in order.
func Codes() []string {
	out := make([]string, len(Catalogue))
	for i, mod := range Catalogue {
		out[i] = mod.Code
	}
	return out
}
