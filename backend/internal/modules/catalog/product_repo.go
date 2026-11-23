package catalog

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// ProductRepo is the products store (plus variants, barcodes, components,
// images — all child rows of a product).
type ProductRepo struct {
	db *sqlx.DB
}

// NewProductRepo builds a ProductRepo.
func NewProductRepo(conn *sqlx.DB) *ProductRepo { return &ProductRepo{db: conn} }

const prodCols = `id, tenant_id, public_id, sku, name, description, kind,
	category_id, brand_id, unit_id, sell_by, price, cost, vat_rate, deposit,
	track_stock, attributes, status, created_at, updated_at`

// ListFilter narrows a product list.
type ListFilter struct {
	Query      string
	CategoryID uint64 // public-id resolved by the service
	BrandID    uint64 // public-id resolved by the handler
	Kind       string
	Status     string
}

// ListRow is a product plus its resolved taxonomy names for list display.
type ListRow struct {
	Product
	CategoryPublicID *string          `db:"category_public_id"`
	CategoryName     *string          `db:"category_name"`
	BrandPublicID    *string          `db:"brand_public_id"`
	BrandName        *string          `db:"brand_name"`
	UnitCode         *string          `db:"unit_code"`
	ImageID          *uint64          `db:"image_id"`
	Stock            *decimal.Decimal `db:"stock"`
	LowStock         bool             `db:"low_stock"`
}

// List returns a page of products matching f with taxonomy names resolved.
func (r *ProductRepo) List(ctx context.Context, tenantID uint64, f ListFilter, p httpx.Page) ([]ListRow, int, error) {
	where := `p.tenant_id = ?`
	args := []any{tenantID}
	if f.Query != "" {
		where += ` AND (p.name LIKE ? OR p.sku LIKE ?)`
		args = append(args, "%"+f.Query+"%", "%"+f.Query+"%")
	}
	if f.CategoryID != 0 {
		where += ` AND p.category_id = ?`
		args = append(args, f.CategoryID)
	}
	if f.BrandID != 0 {
		where += ` AND p.brand_id = ?`
		args = append(args, f.BrandID)
	}
	if f.Kind != "" {
		where += ` AND p.kind = ?`
		args = append(args, f.Kind)
	}
	if f.Status != "" {
		where += ` AND p.status = ?`
		args = append(args, f.Status)
	}
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM products p WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	var out []ListRow
	err := r.db.SelectContext(ctx, &out,
		`SELECT p.`+strings.ReplaceAll(prodCols, ", ", ", p.")+`,
				c.public_id AS category_public_id, c.name AS category_name,
				b.public_id AS brand_public_id, b.name AS brand_name,
				u.code AS unit_code,
				(SELECT i.id FROM product_images i WHERE i.product_id = p.id
				 ORDER BY i.sort, i.id LIMIT 1) AS image_id,
				CASE WHEN p.track_stock = 1 THEN
					(SELECT COALESCE(SUM(sb.qty),0) FROM stock_balances sb
					 WHERE sb.tenant_id = p.tenant_id AND sb.product_id = p.id)
				END AS stock,
				EXISTS (
					SELECT 1 FROM reorder_levels rl
					WHERE rl.tenant_id = p.tenant_id AND rl.product_id = p.id AND rl.min_qty > 0
					  AND rl.min_qty > COALESCE((SELECT SUM(sb.qty) FROM stock_balances sb
						WHERE sb.tenant_id = rl.tenant_id AND sb.branch_id = rl.branch_id
						  AND sb.product_id = rl.product_id AND sb.variant_id = rl.variant_id), 0)
				) AS low_stock
		 FROM products p
		 LEFT JOIN categories c ON c.id = p.category_id
		 LEFT JOIN brands b ON b.id = p.brand_id
		 LEFT JOIN units u ON u.id = p.unit_id
		 WHERE `+where+` ORDER BY p.name LIMIT ? OFFSET ?`,
		append(args, p.Limit(), p.Offset())...)
	return out, total, err
}

// ByPublicID loads a product without its child rows.
func (r *ProductRepo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Product, error) {
	var p Product
	err := r.db.GetContext(ctx, &p,
		`SELECT `+prodCols+` FROM products WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return p, err
}

// ByID loads a product by internal id, scoped to a tenant.
func (r *ProductRepo) ByID(ctx context.Context, tenantID, id uint64) (Product, error) {
	var p Product
	err := r.db.GetContext(ctx, &p,
		`SELECT `+prodCols+` FROM products WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return p, err
}

// BySKU finds a product by SKU.
func (r *ProductRepo) BySKU(ctx context.Context, tenantID uint64, sku string) (Product, error) {
	var p Product
	err := r.db.GetContext(ctx, &p,
		`SELECT `+prodCols+` FROM products WHERE tenant_id = ? AND sku = ?`, tenantID, sku)
	return p, err
}

// Create inserts a product inside tx and hydrates it.
func (r *ProductRepo) Create(ctx context.Context, q db.Querier, p *Product) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO products (tenant_id, public_id, sku, name, description, kind, category_id,
			brand_id, unit_id, sell_by, price, cost, vat_rate, deposit, track_stock, attributes, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.TenantID, p.PublicID, p.SKU, p.Name, p.Description, p.Kind, p.CategoryID, p.BrandID,
		p.UnitID, p.SellBy, p.Price, p.Cost, p.VatRate, p.Deposit, p.TrackStock, p.Attributes, p.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = db.U64(id)
	// Hydrate inside the same querier so callers in a transaction see the row.
	var fresh Product
	if err := q.GetContext(ctx, &fresh,
		`SELECT `+prodCols+` FROM products WHERE tenant_id = ? AND public_id = ?`,
		p.TenantID, p.PublicID); err != nil {
		return err
	}
	*p = fresh
	return nil
}

// Update rewrites the editable columns of a product inside tx.
func (r *ProductRepo) Update(ctx context.Context, q db.Querier, p *Product) error {
	_, err := q.ExecContext(ctx,
		`UPDATE products SET sku = ?, name = ?, description = ?, kind = ?, category_id = ?,
			brand_id = ?, unit_id = ?, sell_by = ?, price = ?, cost = ?, vat_rate = ?, deposit = ?,
			track_stock = ?, attributes = ?, status = ? WHERE id = ?`,
		p.SKU, p.Name, p.Description, p.Kind, p.CategoryID, p.BrandID, p.UnitID, p.SellBy,
		p.Price, p.Cost, p.VatRate, p.Deposit, p.TrackStock, p.Attributes, p.Status, p.ID)
	return err
}

// Archive soft-deletes a product.
func (r *ProductRepo) Archive(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE products SET status = 'archived' WHERE id = ?`, id)
	return err
}

const variantCols = `id, tenant_id, product_id, public_id, sku, name, price, attributes`

// VariantsOf returns a product's variants.
func (r *ProductRepo) VariantsOf(ctx context.Context, productID uint64) ([]Variant, error) {
	var out []Variant
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+variantCols+` FROM product_variants WHERE product_id = ? ORDER BY id`, productID)
	return out, err
}

// ReplaceVariants deletes all variants of a product and inserts vs inside tx.
func (r *ProductRepo) ReplaceVariants(ctx context.Context, q db.Querier, tenantID, productID uint64, vs []Variant) error {
	if _, err := q.ExecContext(ctx,
		`DELETE FROM product_variants WHERE product_id = ?`, productID); err != nil {
		return err
	}
	for i := range vs {
		vs[i].TenantID = tenantID
		vs[i].ProductID = productID
		res, err := q.ExecContext(ctx,
			`INSERT INTO product_variants (tenant_id, product_id, public_id, sku, name, price, attributes)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tenantID, productID, vs[i].PublicID, vs[i].SKU, vs[i].Name, vs[i].Price, vs[i].Attributes)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		vs[i].ID = db.U64(id)
	}
	return nil
}

const barcodeCols = `id, tenant_id, product_id, variant_id, barcode, pack_qty`

// BarcodesOf returns a product's barcodes.
func (r *ProductRepo) BarcodesOf(ctx context.Context, productID uint64) ([]Barcode, error) {
	var out []Barcode
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+barcodeCols+` FROM product_barcodes WHERE product_id = ? ORDER BY id`, productID)
	return out, err
}

// BarcodeByCode finds the barcode row for a code, scoped to a tenant.
func (r *ProductRepo) BarcodeByCode(ctx context.Context, tenantID uint64, code string) (Barcode, error) {
	var b Barcode
	err := r.db.GetContext(ctx, &b,
		`SELECT `+barcodeCols+` FROM product_barcodes WHERE tenant_id = ? AND barcode = ?`,
		tenantID, code)
	return b, err
}

// ReplaceBarcodes rewrites a product's barcode rows inside tx.
func (r *ProductRepo) ReplaceBarcodes(ctx context.Context, q db.Querier, tenantID, productID uint64, bs []Barcode) error {
	if _, err := q.ExecContext(ctx,
		`DELETE FROM product_barcodes WHERE product_id = ?`, productID); err != nil {
		return err
	}
	for i := range bs {
		_, err := q.ExecContext(ctx,
			`INSERT INTO product_barcodes (tenant_id, product_id, variant_id, barcode, pack_qty)
			 VALUES (?, ?, ?, ?, ?)`,
			tenantID, productID, bs[i].VariantID, bs[i].Code, bs[i].PackQty)
		if err != nil {
			return err
		}
	}
	return nil
}

// ComponentsOf returns a composite product's ingredient rows with names.
func (r *ProductRepo) ComponentsOf(ctx context.Context, productID uint64) ([]Component, error) {
	var rows []struct {
		ProductID   uint64          `db:"product_id"`
		ComponentID uint64          `db:"component_id"`
		Qty         decimal.Decimal `db:"qty"`
		PublicID    string          `db:"public_id"`
		SKU         string          `db:"sku"`
		Name        string          `db:"name"`
	}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT c.product_id, c.component_id, c.qty, p.public_id, p.sku, p.name
		 FROM product_components c JOIN products p ON p.id = c.component_id
		 WHERE c.product_id = ? ORDER BY p.name`, productID)
	if err != nil {
		return nil, err
	}
	out := make([]Component, len(rows))
	for i, row := range rows {
		out[i] = Component{ProductID: row.ProductID, ComponentID: row.ComponentID,
			Qty: row.Qty, ComponentPublicID: row.PublicID, ComponentSKU: row.SKU, ComponentName: row.Name}
	}
	return out, nil
}

// ReplaceComponents rewrites a product's component list inside tx.
func (r *ProductRepo) ReplaceComponents(ctx context.Context, q db.Querier, productID uint64, cs []Component) error {
	if _, err := q.ExecContext(ctx,
		`DELETE FROM product_components WHERE product_id = ?`, productID); err != nil {
		return err
	}
	for _, c := range cs {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO product_components (product_id, component_id, qty) VALUES (?, ?, ?)`,
			productID, c.ComponentID, c.Qty); err != nil {
			return err
		}
	}
	return nil
}

// KindOf returns a product's kind — used to forbid composite components.
func (r *ProductRepo) KindOf(ctx context.Context, tenantID, id uint64) (string, error) {
	var kind string
	err := r.db.GetContext(ctx, &kind,
		`SELECT kind FROM products WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return kind, err
}

const imageCols = `id, product_id, path, sort`

// ImagesOf returns a product's images.
func (r *ProductRepo) ImagesOf(ctx context.Context, productID uint64) ([]Image, error) {
	var out []Image
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+imageCols+` FROM product_images WHERE product_id = ? ORDER BY sort, id`, productID)
	return out, err
}

// AddImage records a stored image and returns its id.
func (r *ProductRepo) AddImage(ctx context.Context, img *Image) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO product_images (product_id, path, sort) VALUES (?, ?, ?)`,
		img.ProductID, img.Path, img.Sort)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	img.ID = db.U64(id)
	return nil
}

// ImageByID loads one image row.
func (r *ProductRepo) ImageByID(ctx context.Context, id uint64) (Image, error) {
	var img Image
	err := r.db.GetContext(ctx, &img,
		`SELECT `+imageCols+` FROM product_images WHERE id = ?`, id)
	return img, err
}

// DeleteImage removes an image row.
func (r *ProductRepo) DeleteImage(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM product_images WHERE id = ?`, id)
	return err
}

// ExportRows returns flat rows for CSV export in SKU order.
func (r *ProductRepo) ExportRows(ctx context.Context, tenantID uint64) ([]ExportRow, error) {
	var rows []ExportRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT p.sku, p.name, p.kind, p.sell_by, p.price, p.cost, p.vat_rate, p.deposit,
				p.track_stock, p.status, c.name AS category, b.name AS brand, u.code AS unit,
				(SELECT GROUP_CONCAT(bc.barcode ORDER BY bc.id SEPARATOR '|')
				 FROM product_barcodes bc WHERE bc.product_id = p.id) AS barcodes
		 FROM products p
		 LEFT JOIN categories c ON c.id = p.category_id
		 LEFT JOIN brands b ON b.id = p.brand_id
		 LEFT JOIN units u ON u.id = p.unit_id
		 WHERE p.tenant_id = ? AND p.status <> 'archived'
		 ORDER BY p.sku`, tenantID)
	return rows, err
}

// ExportRow is one flat CSV line.
type ExportRow struct {
	SKU        string  `db:"sku"`
	Name       string  `db:"name"`
	Kind       string  `db:"kind"`
	SellBy     string  `db:"sell_by"`
	Price      string  `db:"price"`
	Cost       *string `db:"cost"`
	VatRate    *string `db:"vat_rate"`
	Deposit    *string `db:"deposit"`
	TrackStock bool    `db:"track_stock"`
	Status     string  `db:"status"`
	Category   *string `db:"category"`
	Brand      *string `db:"brand"`
	Unit       *string `db:"unit"`
	Barcodes   *string `db:"barcodes"`
}
