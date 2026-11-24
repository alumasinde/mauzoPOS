package pricing_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/pricing"
	"mauzo/internal/testutil"
	"mauzo/pkg/pricecalc"
)

type fixture struct {
	svc       *pricing.Service
	promos    *pricing.PromoService
	catalog   *catalog.Service
	branches  *branches.Service
	conn      *sqlx.DB
	tenant    uint64
	branchPub string
	branchID  uint64
	actor     identity.Actor
}

func setup(t *testing.T) fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"price_history", "promotion_targets", "promotions", "price_list_items", "price_lists",
		"product_images", "product_components", "product_barcodes",
		"product_variants", "products", "categories", "brands",
		"branches", "tenants")
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, `DELETE FROM units WHERE tenant_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status) VALUES (?, 'Price Shop', 'price', 'active')`,
		security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug='price'`); err != nil {
		t.Fatal(err)
	}
	images, err := catalog.NewDiskImageStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	taxRepo := catalog.NewTaxonomyRepo(conn)
	catSvc := catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, images)
	taxSvc := catalog.NewTaxonomyService(taxRepo)
	branchSvc := branches.NewService(conn, branches.NewRepo(conn))
	repo := pricing.NewRepo(conn)
	svc := pricing.NewService(conn, repo, catSvc, branchSvc)
	promos := pricing.NewPromoService(conn, repo, catSvc, taxSvc)
	catSvc.WatchPrices(svc.RecordPrice)

	b, err := branchSvc.Create(ctx, tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	bid, err := branchSvc.BranchByPublicID(ctx, tenantID, b.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.Actor{
		Type: identity.ActorUser, TenantID: tenantID, Role: "owner",
		Access: identity.Access{Perms: authz.AllOf(), AllBranches: true},
	}
	return fixture{svc: svc, promos: promos, catalog: catSvc, branches: branchSvc, conn: conn,
		tenant: tenantID, branchPub: b.PublicID, branchID: bid, actor: actor}
}

func product(t *testing.T, f fixture, name, price string) catalog.Product {
	t.Helper()
	p, err := f.catalog.Create(context.Background(), f.tenant, catalog.ProductRequest{
		Name: name, SKU: name, Price: price,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPriceListTiersAndResolution(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	milk := product(t, f, "Milk", "65")

	l, err := f.svc.CreateList(ctx, f.actor, pricing.ListRequest{
		Name: "Wholesale", Priority: 5,
		Items: []pricing.ListItemRequest{
			{ProductID: milk.PublicID, MinQty: "1", Price: "55"},
			{ProductID: milk.PublicID, MinQty: "10", Price: "50"},
		},
	})
	if err != nil {
		t.Fatalf("create list: %v", err)
	}
	if len(l.Items) != 2 {
		t.Fatalf("items=%d", len(l.Items))
	}

	// Below the 10-tier: pays the 1-tier price.
	rp, err := f.svc.Resolve(ctx, f.tenant, f.branchID, "", milk, 0, decimal.NewFromInt(3))
	if err != nil || rp.UnitPrice.String() != "55" || rp.Source != pricing.HistoryPriceList {
		t.Fatalf("resolve qty3: %v %+v", err, rp)
	}
	// At/above the 10-tier: the deeper tier wins.
	rp, err = f.svc.Resolve(ctx, f.tenant, f.branchID, "", milk, 0, decimal.NewFromInt(12))
	if err != nil || rp.UnitPrice.String() != "50" {
		t.Fatalf("resolve qty12: %v %+v", err, rp)
	}
	// A group-scoped list does not leak to other groups.
	l2, err := f.svc.CreateList(ctx, f.actor, pricing.ListRequest{
		Name: "VIP", CustomerGroup: ptr("vip"), Priority: 9,
		Items: []pricing.ListItemRequest{{ProductID: milk.PublicID, Price: "40"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rp, _ = f.svc.Resolve(ctx, f.tenant, f.branchID, "retail", milk, 0, decimal.NewFromInt(12))
	if rp.ListName != "Wholesale" {
		t.Fatalf("group leak: %+v", rp)
	}
	rp, _ = f.svc.Resolve(ctx, f.tenant, f.branchID, "vip", milk, 0, decimal.NewFromInt(12))
	if rp.UnitPrice.String() != "40" || rp.ListName != "VIP" {
		t.Fatalf("vip resolve: %+v", rp)
	}
	_ = l2
}

func TestPromotionCRUDAndQuote(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	milk := product(t, f, "Milk", "100")
	bread := product(t, f, "Bread", "50")

	p, err := f.promos.CreatePromotion(ctx, f.actor, pricing.PromotionRequest{
		Name: "10% milk", Kind: "percent",
		Config:  pricecalc.Config{Percent: decimal.NewFromInt(10)},
		Targets: []pricing.TargetRequest{{ProductID: milk.PublicID}},
	})
	if err != nil || len(p.Targets) != 1 {
		t.Fatalf("create promo: %v %+v", err, p)
	}
	if _, err := f.promos.CreatePromotion(ctx, f.actor, pricing.PromotionRequest{
		Name: "bad", Kind: "bogus",
	}); err == nil {
		t.Fatal("invalid kind accepted")
	}

	out, err := pricing.Quote(ctx, f.actor, f.svc, f.promos, f.branches, pricing.QuoteRequest{
		BranchID: f.branchPub,
		Lines: []pricing.QuoteLine{
			{ProductID: milk.PublicID, Qty: "2"},
			{ProductID: bread.PublicID, Qty: "1"},
		},
	})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if out.Subtotal != "250" || out.Discount != "20" || out.Total != "230" {
		t.Fatalf("totals: %+v", out)
	}
	var milkLine *pricing.QuotedLine
	for i := range out.Lines {
		if out.Lines[i].ProductID == milk.PublicID {
			milkLine = &out.Lines[i]
		}
	}
	if milkLine == nil || milkLine.Promotion != "10% milk" || milkLine.FinalPrice != "90" {
		t.Fatalf("milk line: %+v", milkLine)
	}

	// An expired window disables the rule.
	yesterday := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	if _, err := f.promos.UpdatePromotion(ctx, f.actor, p.PublicID, pricing.PromotionRequest{
		Name: "10% milk", Kind: "percent",
		Config:  pricecalc.Config{Percent: decimal.NewFromInt(10)},
		EndsAt:  &yesterday,
		Targets: []pricing.TargetRequest{{ProductID: milk.PublicID}},
	}); err != nil {
		t.Fatal(err)
	}
	out, _ = pricing.Quote(ctx, f.actor, f.svc, f.promos, f.branches, pricing.QuoteRequest{
		Lines: []pricing.QuoteLine{{ProductID: milk.PublicID, Qty: "2"}},
	})
	if out.Discount != "0" {
		t.Fatalf("expired promo still applied: %+v", out)
	}
}

func TestPriceHistoryFromCatalogHook(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := product(t, f, "Flour", "120")

	// The hook is registered in setup; a catalog price change must record.
	ctxA := identity.WithActor(ctx, f.actor)
	if _, err := f.catalog.Update(ctxA, f.tenant, p.PublicID, catalog.ProductRequest{
		Name: "Flour", SKU: "Flour", Price: "135",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	rows, total, err := f.svc.History(ctx, f.tenant, p.PublicID, httpx.Page{Page: 1, PerPage: 10})
	if err != nil || total != 1 {
		t.Fatalf("history: %v total=%d", err, total)
	}
	if rows[0].OldPrice.String() != "120" || rows[0].NewPrice.String() != "135" {
		t.Fatalf("row: %+v", rows[0])
	}
}

func ptr[T any](v T) *T { return &v }
