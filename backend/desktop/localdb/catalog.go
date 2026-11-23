package localdb

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Product is one sellable row for the grid/cart.
type Product struct {
	ID         uint64  `db:"id" json:"id"`
	PublicID   string  `db:"public_id" json:"public_id"`
	SKU        string  `db:"sku" json:"sku"`
	Name       string  `db:"name" json:"name"`
	Kind       string  `db:"kind" json:"kind"`
	CategoryID uint64  `db:"category_id" json:"category_id"`
	UnitCode   string  `db:"unit_code" json:"unit_code"`
	SellBy     string  `db:"sell_by" json:"sell_by"`
	Price      string  `db:"price" json:"price"`
	VatRate    *string `db:"vat_rate" json:"vat_rate"`
	TrackStock bool    `db:"track_stock" json:"track_stock"`

	VariantID   uint64 `db:"variant_id" json:"variant_id"`     // 0 = base product
	VariantName string `db:"variant_name" json:"variant_name"` // "" = base product
}

// Category is a grid filter tile.
type Category struct {
	ID       uint64 `db:"id" json:"id"`
	PublicID string `db:"public_id" json:"public_id"`
	Name     string `db:"name" json:"name"`
	Sort     int    `db:"sort" json:"sort"`
	Count    int    `db:"count" json:"count"`
}

var (
	// ErrNotFound — barcode/SKU/name matched nothing.
	ErrNotFound = errors.New("not found")
	// ErrAmbiguous — the code matched more than one sellable row.
	ErrAmbiguous = errors.New("ambiguous")
)

const productCols = `id, public_id, sku, name, kind, category_id, unit_code,
	sell_by, price, vat_rate, track_stock`

// Categories lists tiles with product counts.
func (d *DB) Categories(ctx context.Context) ([]Category, error) {
	var out []Category
	err := d.SelectContext(ctx, &out,
		`SELECT c.id, c.public_id, c.name, c.sort,
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id) AS count
		 FROM categories c ORDER BY c.sort, c.name`)
	return out, err
}

// Grid lists products for a category (0 = all) for the tile grid.
func (d *DB) Grid(ctx context.Context, categoryID uint64, limit int) ([]Product, error) {
	q := `SELECT ` + productCols + ` FROM products WHERE kind <> 'service'`
	args := []any{}
	if categoryID != 0 {
		q += ` AND category_id = ?`
		args = append(args, categoryID)
	}
	q += ` ORDER BY name LIMIT ?`
	args = append(args, limit)
	var rows []Product
	err := d.SelectContext(ctx, &rows, q, args...)
	return rows, err
}

// Search finds products/variants by name or SKU substring (LIMIT keeps the
// pick list usable on touch screens).
func (d *DB) Search(ctx context.Context, term string, limit int) ([]Product, error) {
	like := "%" + strings.TrimSpace(term) + "%"
	var rows []Product
	err := d.SelectContext(ctx, &rows,
		`SELECT `+productCols+`, 0 AS variant_id, '' AS variant_name FROM products
		 WHERE (name LIKE ? OR sku LIKE ?) AND kind <> 'service'
		 UNION ALL
		 SELECT p.id, p.public_id, p.sku, p.name || ' · ' || v.name, p.kind, p.category_id,
			p.unit_code, p.sell_by, COALESCE(v.price, p.price) AS price, p.vat_rate, p.track_stock,
			v.id AS variant_id, v.name AS variant_name
		 FROM variants v JOIN products p ON p.id = v.product_id
		 WHERE (v.name LIKE ? OR v.sku LIKE ?)
		 ORDER BY name LIMIT ?`,
		like, like, like, like, limit)
	return rows, err
}

// ByBarcode resolves a scanned code to product + variant + pack qty.
func (d *DB) ByBarcode(ctx context.Context, code string) (Product, string, error) {
	var row struct {
		Product
		VariantID   uint64 `db:"variant_id"`
		VariantName string `db:"variant_name"`
		PackQty     string `db:"pack_qty"`
	}
	err := d.GetContext(ctx, &row,
		`SELECT p.`+strings.ReplaceAll(productCols, ",", ", p.")+`,
			b.variant_id, COALESCE(v.name,'') AS variant_name, b.pack_qty
		 FROM barcodes b
		 JOIN products p ON p.id = b.product_id
		 LEFT JOIN variants v ON v.id = b.variant_id
		 WHERE b.barcode = ?`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, "", ErrNotFound
	}
	if err != nil {
		return Product{}, "", err
	}
	p := row.Product
	p.VariantID = row.VariantID
	p.VariantName = row.VariantName
	return p, row.PackQty, nil
}

// ByItemCode resolves a scale-label item code (SKU or barcode prefix match).
func (d *DB) ByItemCode(ctx context.Context, item string) (Product, error) {
	var rows []Product
	err := d.SelectContext(ctx, &rows,
		`SELECT `+productCols+`, 0 AS variant_id FROM products WHERE sku = ? LIMIT 2`, item)
	if err != nil {
		return Product{}, err
	}
	if len(rows) == 1 {
		return rows[0], nil
	}
	// Fall back to a barcode match on the item digits.
	p, _, err := d.ByBarcode(ctx, item)
	return p, err
}

// Setting reads a synced tenant setting ("" when unset).
func (d *DB) Setting(ctx context.Context, key string) (string, error) {
	var v string
	err := d.GetContext(ctx, &v, `SELECT value FROM settings WHERE key = ?`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}
