package dashboard_test

import (
	"testing"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/identity"
	"mauzo/internal/modules/dashboard"
	_ "mauzo/internal/modules/registerall"
)

func keys(a identity.Actor) []string {
	var out []string
	for _, w := range dashboard.Widgets(a) {
		out = append(out, w.Key)
	}
	return out
}

func user(perms authz.Set, mods ...string) identity.Actor {
	m := map[string]bool{"dashboard": true}
	for _, c := range mods {
		m[c] = true
	}
	return identity.Actor{Type: identity.ActorUser, TenantID: 1, ID: 1,
		Access: identity.Access{Perms: perms, Modules: m}}
}

func TestWidgetsFollowModulesAndPermissions(t *testing.T) {
	// Owner with nothing enabled sees nothing.
	if got := keys(user(authz.AllOf())); len(got) != 0 {
		t.Fatalf("no modules: got %v", got)
	}
	// Owner with sales + payments + mpesa sees the KPI row in order.
	got := keys(user(authz.AllOf(), "sales", "payments", "mpesa", "catalog"))
	want := []string{"sales_total", "sales_count", "sales_average", "mpesa_collections", "cash_collections"}
	for i, k := range want {
		if i >= len(got) || got[i] != k {
			t.Fatalf("order: got %v, want prefix %v", got, want)
		}
	}
	// Inventory manager without sales permissions sees only inventory alerts.
	got = keys(user(authz.NewSet("dashboard.view", "inventory.stock.view"), "sales", "inventory", "catalog"))
	if len(got) != 1 || got[0] != "inventory_alerts" {
		t.Fatalf("inventory manager: got %v", got)
	}
}
