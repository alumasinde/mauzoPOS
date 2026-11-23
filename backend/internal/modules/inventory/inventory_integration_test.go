package inventory_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/inventory"
	"mauzo/internal/testutil"
)

type fixture struct {
	svc        *inventory.Service
	conn       *sqlx.DB
	tenant     uint64
	branch     uint64
	branch2    uint64
	branchPub  string
	branch2Pub string
	actor      identity.Actor
	catalog    *catalog.Service
}

func setup(t *testing.T) fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"price_history", "promotion_targets", "promotions", "price_list_items", "price_lists",
		"reorder_levels", "stock_count_items", "stock_counts", "stock_transfer_items",
		"stock_transfers", "stock_adjustment_items", "stock_adjustments",
		"stock_balances", "stock_movements", "product_batches", "stock_locations",
		"product_images", "product_components", "product_barcodes",
		"product_variants", "products", "categories", "brands",
		"branches", "tenants")
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, `DELETE FROM units WHERE tenant_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status) VALUES (?, 'Inv Shop', 'inv', 'active')`,
		security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug='inv'`); err != nil {
		t.Fatal(err)
	}

	images, err := catalog.NewDiskImageStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	taxRepo := catalog.NewTaxonomyRepo(conn)
	catSvc := catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, images)
	branchSvc := branches.NewService(branches.NewRepo(conn))
	svc := inventory.NewService(conn, inventory.NewRepo(conn),
		inventory.NewDocsRepo(conn), catSvc, branchSvc)

	actor := identity.Actor{
		Type: identity.ActorUser, TenantID: tenantID, Role: "owner",
		Access: identity.Access{Perms: authz.AllOf(), AllBranches: true},
	}
	b1, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "EAST", Name: "East"})
	if err != nil {
		t.Fatal(err)
	}
	b1id, err := branchSvc.BranchByPublicID(ctx, tenantID, b1.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	b2id, err := branchSvc.BranchByPublicID(ctx, tenantID, b2.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{svc: svc, conn: conn, tenant: tenantID, catalog: catSvc,
		branch: b1id, branch2: b2id, branchPub: b1.PublicID, branch2Pub: b2.PublicID, actor: actor}
}

func product(t *testing.T, f fixture, name, price string) catalog.Product {
	t.Helper()
	p, err := f.catalog.Create(context.Background(), f.tenant, catalog.ProductRequest{
		Name: name, SKU: name, Price: price, TrackStock: boolP(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func boolP(b bool) *bool { return &b }

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestReceiveBalancesAndLedger(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Rice 5kg", "400")
	exp := "2027-01-15"

	err := f.svc.Receive(ctx, f.actor, f.branchPub, []inventory.LineRequest{
		{ProductID: p.PublicID, Qty: "30", UnitCost: ptrS("300"), BatchNo: "B1", Expiry: &exp},
	}, "opening")
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	rows, _, err := f.svc.Balances(ctx, f.actor, inventory.BalanceFilter{BranchID: f.branch}, page())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Qty.String() != "30" || rows[0].AvgCost.String() != "300" {
		t.Fatalf("balance rows: %+v", rows)
	}
	// A second receipt at a different cost blends the weighted average.
	if err := f.svc.Receive(ctx, f.actor, f.branchPub, []inventory.LineRequest{
		{ProductID: p.PublicID, Qty: "10", UnitCost: ptrS("600")},
	}, ""); err != nil {
		t.Fatal(err)
	}
	rows, _, _ = f.svc.Balances(ctx, f.actor, inventory.BalanceFilter{BranchID: f.branch}, page())
	var total decimal.Decimal
	for _, r := range rows {
		total = total.Add(r.Qty.Mul(r.AvgCost))
	}
	if total.String() != "15000" { // 30*300 + 10*600
		t.Fatalf("weighted avg wrong, total=%s", total)
	}

	movs, _, err := f.svc.Movements(ctx, f.actor, inventory.MovementFilter{BranchID: f.branch}, page())
	if err != nil || len(movs) != 2 {
		t.Fatalf("movements: %v len=%d", err, len(movs))
	}
	for _, m := range movs {
		if m.Kind != inventory.MoveReceive {
			t.Fatalf("expected receive movement, got %s", m.Kind)
		}
	}
}

func TestAdjustmentApprovalFlow(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Sugar 1kg", "150")
	mustReceive(t, f, p.PublicID, "20")

	a, err := f.svc.CreateAdjustment(ctx, f.actor, inventory.AdjustmentRequest{
		BranchID: f.branchPub, Reason: "damage",
		Items: []inventory.AdjustmentItemRequest{{ProductID: p.PublicID, Qty: "-5"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.PostAdjustment(ctx, f.actor, a.PublicID); err == nil {
		t.Fatal("posted an unapproved adjustment")
	}
	if _, err := f.svc.ApproveAdjustment(ctx, f.actor, a.PublicID, true); err == nil {
		t.Fatal("creator approved own document")
	}
	approver := f.actor
	approver.ID = 99
	if _, err := f.svc.ApproveAdjustment(ctx, approver, a.PublicID, true); err != nil {
		t.Fatalf("approve: %v", err)
	}
	posted, err := f.svc.PostAdjustment(ctx, approver, a.PublicID)
	if err != nil || posted.Status != inventory.StatusPosted {
		t.Fatalf("post: %v status=%s", err, posted.Status)
	}
	qty := balanceQty(t, f, p.ID)
	if qty.String() != "15" {
		t.Fatalf("qty after adjustment = %s", qty)
	}
}

func TestTransferLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Oil 3L", "800")
	mustReceive(t, f, p.PublicID, "25")

	tr, err := f.svc.CreateTransfer(ctx, f.actor, inventory.TransferRequest{
		FromBranchID: f.branchPub, ToBranchID: f.branch2Pub,
		Items: []inventory.TransferItemRequest{{ProductID: p.PublicID, Qty: "8"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.ReceiveTransfer(ctx, f.actor, tr.PublicID, inventory.ReceiveRequest{}); err == nil {
		t.Fatal("received an unsent transfer")
	}
	if _, err := f.svc.SendTransfer(ctx, f.actor, tr.PublicID); err != nil {
		t.Fatalf("send: %v", err)
	}
	if _, err := f.svc.SendTransfer(ctx, f.actor, tr.PublicID); err == nil {
		t.Fatal("double-send allowed")
	}
	done, err := f.svc.ReceiveTransfer(ctx, f.actor, tr.PublicID, inventory.ReceiveRequest{})
	if err != nil || done.Status != inventory.StatusReceived {
		t.Fatalf("receive: %v status=%s", err, done.Status)
	}
	if q := branchQty(t, f, f.branch2, p.ID); q.String() != "8" {
		t.Fatalf("dest qty=%s", q)
	}
	if q := branchQty(t, f, f.branch, p.ID); q.String() != "17" {
		t.Fatalf("source qty=%s", q)
	}
}

func TestStocktakeAndWastage(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Soap", "80")
	mustReceive(t, f, p.PublicID, "10")

	c, err := f.svc.CreateCount(ctx, f.actor, f.branchPub, "", "cycle")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if len(c.Items) == 0 {
		t.Fatal("count snapshot empty")
	}
	counts := map[string]string{}
	for _, it := range c.Items {
		counts[it.PublicID] = "7" // physical count finds 7 of 10
	}
	if _, err := f.svc.SetCounted(ctx, f.actor, c.PublicID, counts); err != nil {
		t.Fatalf("set counted: %v", err)
	}
	if _, err := f.svc.PostCount(ctx, f.actor, c.PublicID); err != nil {
		t.Fatalf("post count: %v", err)
	}
	if q := branchQty(t, f, f.branch, p.ID); q.String() != "7" {
		t.Fatalf("qty after count=%s", q)
	}

	if err := f.svc.Wastage(ctx, f.actor, f.branchPub, "damage",
		[]inventory.LineRequest{{ProductID: p.PublicID, Qty: "2"}}, ""); err != nil {
		t.Fatalf("wastage: %v", err)
	}
	if q := branchQty(t, f, f.branch, p.ID); q.String() != "5" {
		t.Fatalf("qty after wastage=%s", q)
	}
	if err := f.svc.Wastage(ctx, f.actor, f.branchPub, "bogus",
		[]inventory.LineRequest{{ProductID: p.PublicID, Qty: "1"}}, ""); err == nil {
		t.Fatal("invalid wastage reason accepted")
	}
}

func TestBatchFEFOAndReorder(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Yogurt", "60")
	near, far := "2027-01-10", "2027-09-01"
	if err := f.svc.Receive(ctx, f.actor, f.branchPub, []inventory.LineRequest{
		{ProductID: p.PublicID, Qty: "5", BatchNo: "FAR", Expiry: &far},
		{ProductID: p.PublicID, Qty: "4", BatchNo: "NEAR", Expiry: &near},
	}, ""); err != nil {
		t.Fatal(err)
	}
	// Outbound without batch picks the earliest-expiry batch first.
	if err := f.svc.Wastage(ctx, f.actor, f.branchPub, "expired",
		[]inventory.LineRequest{{ProductID: p.PublicID, Qty: "3"}}, ""); err != nil {
		t.Fatal(err)
	}
	var nearQty, farQty decimal.Decimal
	rows, _, _ := f.svc.Balances(ctx, f.actor, inventory.BalanceFilter{BranchID: f.branch}, page())
	batches, err := f.svc.Batches(ctx, f.tenant, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[uint64]string{}
	for _, b := range batches {
		byID[b.ID] = b.BatchNo
	}
	for _, r := range rows {
		switch byID[r.BatchID] {
		case "NEAR":
			nearQty = r.Qty
		case "FAR":
			farQty = r.Qty
		}
	}
	if nearQty.String() != "1" || farQty.String() != "5" {
		t.Fatalf("FEFO wrong: NEAR=%s FAR=%s", nearQty, farQty)
	}

	if err := f.svc.SetReorder(ctx, f.tenant, inventory.ReorderLevel{
		BranchID: f.branch, ProductID: p.ID,
		MinQty: dec(t, "5"), TargetQty: dec(t, "12"),
	}); err != nil {
		t.Fatal(err)
	}
	ros, err := f.svc.Reorders(ctx, f.tenant, f.branch)
	if err != nil || len(ros) != 1 || ros[0].Qty.String() != "6" {
		t.Fatalf("reorder rows: %v %+v", err, ros)
	}

	ex, err := f.svc.Expiring(ctx, f.tenant, 60)
	if err != nil || len(ex) != 1 || ex[0].BatchNo != "NEAR" {
		t.Fatalf("expiring: %v %+v", err, ex)
	}

	vals, err := f.svc.Valuations(ctx, f.tenant)
	if err != nil || len(vals) != 1 || vals[0].SkuCount == 0 {
		t.Fatalf("valuation: %v %+v", err, vals)
	}
}

// ---- helpers --------------------------------------------------------------------

func page() httpx.Page { return httpx.Page{Page: 1, PerPage: 50} }

func ptrS(s string) *string { return &s }

func mustReceive(t *testing.T, f fixture, productPub, qty string) {
	t.Helper()
	err := f.svc.Receive(context.Background(), f.actor, f.branchPub,
		[]inventory.LineRequest{{ProductID: productPub, Qty: qty}}, "")
	if err != nil {
		t.Fatalf("receive %s: %v", qty, err)
	}
}

func branchQty(t *testing.T, f fixture, branchID, productID uint64) decimal.Decimal {
	t.Helper()
	var q decimal.NullDecimal
	err := f.conn.GetContext(context.Background(), &q,
		`SELECT COALESCE(SUM(qty),0) FROM stock_balances
		 WHERE tenant_id=? AND branch_id=? AND product_id=?`, f.tenant, branchID, productID)
	if err != nil {
		t.Fatal(err)
	}
	if !q.Valid {
		return decimal.Zero
	}
	return q.Decimal
}

func balanceQty(t *testing.T, f fixture, productID uint64) decimal.Decimal {
	t.Helper()
	return branchQty(t, f, f.branch, productID)
}
