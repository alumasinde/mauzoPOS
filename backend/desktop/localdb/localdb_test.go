package localdb

import (
	"context"
	"testing"
	"time"
)

func open(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), t.TempDir()+"/t.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSyncAndLookup(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	b := Bundle{
		ServerTime: time.Now(), Cursor: time.Now(),
	}
	b.Categories = append(b.Categories, struct {
		ID       uint64 `json:"id"`
		PublicID string `json:"public_id"`
		ParentID uint64 `json:"parent_id"`
		Name     string `json:"name"`
		Sort     int    `json:"sort"`
	}{ID: 1, PublicID: "cat1", Name: "Dairy"})
	b.Products = append(b.Products, struct {
		ID         uint64    `json:"id"`
		PublicID   string    `json:"public_id"`
		SKU        string    `json:"sku"`
		Name       string    `json:"name"`
		Kind       string    `json:"kind"`
		CategoryID uint64    `json:"category_id"`
		UnitCode   string    `json:"unit_code"`
		SellBy     string    `json:"sell_by"`
		Price      string    `json:"price"`
		VatRate    *string   `json:"vat_rate"`
		TrackStock bool      `json:"track_stock"`
		UpdatedAt  time.Time `json:"updated_at"`
	}{ID: 10, PublicID: "p1", SKU: "MILK500", Name: "Fresh Milk 500ml", Kind: "stock",
		CategoryID: 1, UnitCode: "pc", SellBy: "unit", Price: "110.00", TrackStock: true,
		UpdatedAt: time.Now()})
	b.Barcodes = append(b.Barcodes, struct {
		ProductID uint64 `json:"product_id"`
		VariantID uint64 `json:"variant_id"`
		Barcode   string `json:"barcode"`
		PackQty   string `json:"pack_qty"`
	}{ProductID: 10, Barcode: "6001234567890", PackQty: "1"})
	b.Settings = append(b.Settings, struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{Key: "business.currency", Value: `"KES"`})
	b.Modules = []string{"catalog", "inventory"}

	if err := db.ApplySync(ctx, b); err != nil {
		t.Fatal(err)
	}

	p, pack, err := db.ByBarcode(ctx, "6001234567890")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Fresh Milk 500ml" || pack != "1" {
		t.Fatalf("got %+v pack %s", p, pack)
	}

	cats, _ := db.Categories(ctx)
	if len(cats) != 1 || cats[0].Count != 1 {
		t.Fatalf("cats: %+v", cats)
	}

	found, err := db.Search(ctx, "milk", 10)
	if err != nil || len(found) != 1 {
		t.Fatalf("search: %v %+v", err, found)
	}

	v, _ := db.Setting(ctx, "business.currency")
	if v != `"KES"` {
		t.Fatalf("setting %q", v)
	}
}

func TestQuoteTiersAndPromos(t *testing.T) {
	ctx := context.Background()
	db := open(t)

	var b Bundle
	vat := "16"
	b.Products = append(b.Products, struct {
		ID         uint64    `json:"id"`
		PublicID   string    `json:"public_id"`
		SKU        string    `json:"sku"`
		Name       string    `json:"name"`
		Kind       string    `json:"kind"`
		CategoryID uint64    `json:"category_id"`
		UnitCode   string    `json:"unit_code"`
		SellBy     string    `json:"sell_by"`
		Price      string    `json:"price"`
		VatRate    *string   `json:"vat_rate"`
		TrackStock bool      `json:"track_stock"`
		UpdatedAt  time.Time `json:"updated_at"`
	}{ID: 10, PublicID: "p1", SKU: "M", Name: "Milk", Kind: "stock", UnitCode: "pc",
		SellBy: "unit", Price: "65", VatRate: &vat, UpdatedAt: time.Now()})
	b.PriceLists = append(b.PriceLists, struct {
		ID            uint64     `json:"id"`
		BranchID      uint64     `json:"branch_id"`
		CustomerGroup string     `json:"customer_group"`
		Priority      int        `json:"priority"`
		ValidFrom     *time.Time `json:"valid_from"`
		ValidTo       *time.Time `json:"valid_to"`
	}{ID: 5, BranchID: 0, Priority: 1})
	b.PriceItems = append(b.PriceItems, struct {
		ListID    uint64 `json:"list_id"`
		ProductID uint64 `json:"product_id"`
		VariantID uint64 `json:"variant_id"`
		MinQty    string `json:"min_qty"`
		Price     string `json:"price"`
	}{ListID: 5, ProductID: 10, VariantID: 0, MinQty: "12", Price: "50"})
	b.Promotions = append(b.Promotions, struct {
		ID       uint64     `json:"id"`
		Name     string     `json:"name"`
		Kind     string     `json:"kind"`
		Config   string     `json:"config"`
		StartsAt *time.Time `json:"starts_at"`
		EndsAt   *time.Time `json:"ends_at"`
		DaysMask *uint8     `json:"days_mask"`
		Priority int        `json:"priority"`
	}{ID: 9, Name: "10% off", Kind: "percent", Config: `{"percent":10}`, Priority: 0})

	if err := db.ApplySync(ctx, b); err != nil {
		t.Fatal(err)
	}

	q, err := db.QuoteCart(ctx, []QuoteLine{{ProductID: 10, Qty: "12"}})
	if err != nil {
		t.Fatal(err)
	}
	if q.Subtotal != "600.00" || q.Discount != "60.00" || q.Total != "540.00" {
		t.Fatalf("quote %+v", q)
	}
	if q.Lines[0].UnitPrice != "50.00" || q.Lines[0].Source != "price_list" {
		t.Fatalf("line %+v", q.Lines[0])
	}
	// VAT inclusive-of-total at 16%: 540 × 16/116 = 74.48
	if q.Vat != "74.48" {
		t.Fatalf("vat %s", q.Vat)
	}
}

func TestHeldAndPending(t *testing.T) {
	ctx := context.Background()
	db := open(t)
	if err := db.HoldCart(ctx, "h1", "table 4", `[{"a":1}]`, "Jane", false); err != nil {
		t.Fatal(err)
	}
	held, _ := db.HeldCarts(ctx)
	if len(held) != 1 || held[0].Label != "table 4" {
		t.Fatalf("held %+v", held)
	}
	if err := db.RecordPendingSale(ctx, "s1", "[]", `{"total":"100"}`, "cash", "u1"); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = db.GetContext(ctx, &n, `SELECT COUNT(*) FROM pending_sales WHERE uploaded = 0`)
	if n != 1 {
		t.Fatalf("pending %d", n)
	}
	if err := db.DeleteHeldCart(ctx, "h1"); err != nil {
		t.Fatal(err)
	}
}
