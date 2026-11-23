package catalog_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/security"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/testutil"
	"mauzo/pkg/barcode"
)

func setup(t *testing.T) (*catalog.Service, *catalog.TaxonomyService, *sqlx.DB, uint64) {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"product_images", "product_components", "product_barcodes",
		"product_variants", "products", "categories", "brands", "tenants")
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, `DELETE FROM units WHERE tenant_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	pub := security.NewPublicID()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status) VALUES (?, 'Test Shop', 'test', 'active')`, pub); err != nil {
		t.Fatal(err)
	}
	var tenantID uint64
	if err := conn.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE slug = 'test'`); err != nil {
		t.Fatal(err)
	}
	images, err := catalog.NewDiskImageStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	taxRepo := catalog.NewTaxonomyRepo(conn)
	svc := catalog.NewService(conn, catalog.NewProductRepo(conn), taxRepo, images)
	return svc, catalog.NewTaxonomyService(taxRepo), conn, tenantID
}

func seedBasics(t *testing.T, tax *catalog.TaxonomyService, tenantID uint64) (cat, brand string) {
	t.Helper()
	ctx := context.Background()
	c, err := tax.CreateCategory(ctx, tenantID, catalog.CategoryRequest{Name: "Butchery"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := tax.CreateBrand(ctx, tenantID, catalog.BrandRequest{Name: "Farmers"})
	if err != nil {
		t.Fatal(err)
	}
	return c.PublicID, b.PublicID
}

func TestTaxonomyTreeAndUnits(t *testing.T) {
	svc, tax, _, tenantID := setup(t)
	ctx := context.Background()

	parent, err := tax.CreateCategory(ctx, tenantID, catalog.CategoryRequest{Name: "Grocery"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := tax.CreateCategory(ctx, tenantID, catalog.CategoryRequest{Name: "Cereals", ParentID: &parent.PublicID})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatal("child should point at the parent's internal id")
	}
	// Self-parenting rejected.
	if _, err = tax.UpdateCategory(ctx, tenantID, parent.PublicID,
		catalog.CategoryRequest{Name: "Grocery", ParentID: &parent.PublicID}); err == nil {
		t.Fatal("self-parenting must fail")
	}
	// Delete with children rejected; deleting the child then parent works.
	if err = tax.DeleteCategory(ctx, tenantID, parent.PublicID); !errors.Is(err, catalog.ErrCategoryInUse) {
		t.Fatalf("want category_in_use, got %v", err)
	}
	if err = tax.DeleteCategory(ctx, tenantID, child.PublicID); err != nil {
		t.Fatal(err)
	}
	if err = tax.DeleteCategory(ctx, tenantID, parent.PublicID); err != nil {
		t.Fatal(err)
	}

	// Built-in units are visible; conversion works inside a kind.
	units, err := tax.ListUnits(ctx, tenantID)
	if err != nil || len(units) < 6 {
		t.Fatalf("want built-ins, got %v %v", len(units), err)
	}
	q, err := tax.Convert(ctx, tenantID, mustDec(t, "2.5"), "kg", "g")
	if err != nil || q.String() != "2500" {
		t.Fatalf("2.5 kg → %v g, err %v", q, err)
	}
	if _, err = tax.Convert(ctx, tenantID, mustDec(t, "1"), "kg", "l"); !errors.Is(err, catalog.ErrBadUnit) {
		t.Fatal("cross-kind conversion must fail")
	}
	// Tenant unit anchored to a built-in base.
	if _, err = tax.CreateUnit(ctx, tenantID, catalog.UnitRequest{
		Code: "gram30", Name: "30-gram tray", Kind: catalog.UnitKindMass,
		BaseCode: "kg", Factor: "0.03",
	}); err != nil {
		t.Fatal(err)
	}
	q, err = tax.Convert(ctx, tenantID, mustDec(t, "10"), "gram30", "kg")
	if err != nil || q.String() != "0.3" {
		t.Fatalf("10 trays → %v kg, err %v", q, err)
	}
	_ = svc
}

func TestProductLifecycle(t *testing.T) {
	svc, tax, _, tenantID := setup(t)
	ctx := context.Background()
	cat, brand := seedBasics(t, tax, tenantID)

	p, err := svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "BEEF", Name: "Minced Beef", CategoryID: &cat, BrandID: &brand,
		UnitCode: "kg", SellBy: catalog.SellByWeight, Price: "850", Cost: "700",
		Barcodes: []catalog.BarcodeRequest{{Code: "00123"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.SellBy != catalog.SellByWeight || p.UnitCode == nil || *p.UnitCode != "kg" {
		t.Fatalf("weighted product not kept: %+v", p)
	}
	// Duplicate SKU rejected.
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "BEEF", Name: "Dup", Price: "1"}); !errors.Is(err, catalog.ErrSKUTaken) {
		t.Fatalf("want sku_taken, got %v", err)
	}
	// Weighted product requires a mass unit.
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "X1", Name: "x", UnitCode: "pc", SellBy: catalog.SellByWeight, Price: "1"}); !errors.Is(err, catalog.ErrBadUnit) {
		t.Fatalf("want invalid_unit, got %v", err)
	}
	// Archive hides it from lookup.
	if err = svc.Archive(ctx, tenantID, p.PublicID); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Lookup(ctx, tenantID, "00123"); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("archived product must not be scannable, got %v", err)
	}
}

func TestVariantsPackBarcodesAndLookup(t *testing.T) {
	svc, _, _, tenantID := setup(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "SODA", Name: "Soda", Price: "60",
		Variants: []catalog.VariantRequest{{Name: "Crate of 24", SKU: "SODA-C24", Price: strp("1350")}},
		Barcodes: []catalog.BarcodeRequest{
			{Code: "6161103000015"},
			{Code: "6161103000244", VariantID: strp("Crate of 24"), PackQty: strp("24")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Variants) != 1 || len(p.Barcodes) != 2 {
		t.Fatalf("children not persisted: %+v", p)
	}
	// Pack barcode resolves variant + qty.
	res, err := svc.Lookup(ctx, tenantID, "6161103000244")
	if err != nil {
		t.Fatal(err)
	}
	if res.Quantity != "24" || res.Price != "1350" || res.Variant == nil {
		t.Fatalf("pack lookup = qty %s price %s variant %+v", res.Quantity, res.Price, res.Variant)
	}
	// SKU lookup hits the product.
	res, err = svc.Lookup(ctx, tenantID, "SODA")
	if err != nil || res.Source != "sku" || res.Price != "60" {
		t.Fatalf("sku lookup = %+v, err %v", res, err)
	}
	// Duplicate barcode across products rejected.
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "OTHER", Name: "o", Price: "1",
		Barcodes: []catalog.BarcodeRequest{{Code: "6161103000015"}},
	}); !errors.Is(err, catalog.ErrBarcodeTaken) {
		t.Fatalf("want barcode_taken, got %v", err)
	}
	// Bad check digit on a GTIN rejected.
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "BAD", Name: "b", Price: "1",
		Barcodes: []catalog.BarcodeRequest{{Code: "6161103000016"}},
	}); !errors.Is(err, catalog.ErrBadBarcode) {
		t.Fatalf("want invalid_barcode, got %v", err)
	}
	// Scale-prefix EANs are reserved for scale labels.
	payload := "210012315050"
	scaleCode := payload + string(rune('0'+barcode.CheckDigit(payload)))
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "SC", Name: "s", Price: "1",
		Barcodes: []catalog.BarcodeRequest{{Code: scaleCode}},
	}); !errors.Is(err, catalog.ErrBadBarcode) {
		t.Fatalf("scale-prefixed code must be rejected, got %v", err)
	}
}

func TestScaleLabelLookup(t *testing.T) {
	svc, _, _, tenantID := setup(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "BEEF", Name: "Minced Beef", UnitCode: "kg",
		SellBy: catalog.SellByWeight, Price: "850",
		Barcodes: []catalog.BarcodeRequest{{Code: "00123"}},
	}); err != nil {
		t.Fatal(err)
	}
	// Scale label: 21 + item 00123 + price 15050 (KES 150.50) + check.
	payload := "210012315050"
	code := payload + string(rune('0'+barcode.CheckDigit(payload)))
	res, err := svc.Lookup(ctx, tenantID, code)
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "scale" || res.Price != "150.5" || res.Scale == nil || res.Scale.ItemCode != "00123" {
		t.Fatalf("scale lookup = %+v", res)
	}
}

func TestComposite(t *testing.T) {
	svc, _, _, tenantID := setup(t)
	ctx := context.Background()

	buns, err := svc.Create(ctx, tenantID, catalog.ProductRequest{SKU: "BUNS", Name: "Buns", Price: "120"})
	if err != nil {
		t.Fatal(err)
	}
	soda, err := svc.Create(ctx, tenantID, catalog.ProductRequest{SKU: "SD", Name: "Soda", Price: "60"})
	if err != nil {
		t.Fatal(err)
	}
	combo, err := svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "COMBO", Name: "Combo", Kind: catalog.KindComposite, Price: "200",
		Components: []catalog.ComponentRequest{
			{ProductID: buns.PublicID, Qty: "1"}, {ProductID: soda.PublicID, Qty: "2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(combo.Components) != 2 || combo.Components[0].ComponentPublicID == "" {
		t.Fatalf("components not hydrated: %+v", combo.Components)
	}
	// Composite components must exist and not be composite themselves.
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "C2", Name: "c2", Kind: catalog.KindComposite, Price: "1",
		Components: []catalog.ComponentRequest{{ProductID: combo.PublicID, Qty: "1"}},
	}); !errors.Is(err, catalog.ErrBadComposite) {
		t.Fatalf("nested composite must fail, got %v", err)
	}
	if _, err = svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "C3", Name: "c3", Kind: catalog.KindComposite, Price: "1",
		Components: []catalog.ComponentRequest{{ProductID: "nope", Qty: "1"}},
	}); !errors.Is(err, catalog.ErrBadComposite) {
		t.Fatalf("unknown component must fail, got %v", err)
	}
}

func TestImportExportRoundTrip(t *testing.T) {
	svc, _, _, tenantID := setup(t)
	ctx := context.Background()

	csvData := "sku,name,kind,sell_by,price,cost,category,brand,unit,barcodes\n" +
		"BREAD,White Bread,standard,unit,55,40,Bakery,Broadways,pc,\n" +
		"MILK,Fresh Milk,standard,unit,65,,Dairy,,pc,\n"
	res, err := svc.ImportCSV(ctx, tenantID, strings.NewReader(csvData))
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 2 || len(res.Errors) != 0 {
		t.Fatalf("import = %+v", res)
	}
	// Re-import updates by SKU.
	res, err = svc.ImportCSV(ctx, tenantID, strings.NewReader(csvData))
	if err != nil || res.Updated != 2 {
		t.Fatalf("re-import = %+v, err %v", res, err)
	}
	// Export round-trips the header + rows.
	var buf bytes.Buffer
	if err := svc.ExportCSV(ctx, tenantID, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "sku,name,") || !strings.Contains(out, "BREAD,White Bread,standard,unit,55") {
		t.Fatalf("unexpected export:\n%s", out)
	}
	if !strings.Contains(out, "Bakery") || !strings.Contains(out, "Broadways") {
		t.Fatal("auto-created taxonomy missing from export")
	}
	// XLSX export produces a readable workbook, and re-importing it updates by
	// SKU rather than creating duplicates.
	var xbuf bytes.Buffer
	if err := svc.ExportXLSX(ctx, tenantID, &xbuf); err != nil {
		t.Fatal(err)
	}
	res, err = svc.ImportXLSX(ctx, tenantID, bytes.NewReader(xbuf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 2 || res.Created != 0 || len(res.Errors) != 0 {
		t.Fatalf("xlsx round-trip = %+v", res)
	}
}

func TestLabels(t *testing.T) {
	svc, _, _, tenantID := setup(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, tenantID, catalog.ProductRequest{
		SKU: "LBL", Name: "Labelled", Price: "99",
		Barcodes: []catalog.BarcodeRequest{{Code: "6161103000015"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	labels, err := svc.Labels(ctx, tenantID, []string{p.PublicID, "missing"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 3 || labels[0].Barcode != "6161103000015" || len(labels[0].Pattern) == 0 {
		t.Fatalf("labels = %+v", labels)
	}
}

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func strp(s string) *string { return &s }
