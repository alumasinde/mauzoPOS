// Package localdb is the terminal's local SQLite working set: the synced
// catalog/pricing/cashier mirror, held carts, completed sales awaiting upload
// and terminal settings. Everything the checkout screen needs must work
// against this file with the network down.
package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // driver "sqlite"
)

// DB wraps the local SQLite handle.
type DB struct {
	*sqlx.DB
}

const schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
    id        INTEGER PRIMARY KEY,
    public_id TEXT NOT NULL,
    parent_id INTEGER NOT NULL DEFAULT 0,
    name      TEXT NOT NULL,
    sort      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS units (
    id     INTEGER PRIMARY KEY,
    code   TEXT NOT NULL,
    name   TEXT NOT NULL,
    kind   TEXT NOT NULL,
    factor TEXT NOT NULL DEFAULT '1'
);

CREATE TABLE IF NOT EXISTS products (
    id          INTEGER PRIMARY KEY,
    public_id   TEXT NOT NULL,
    sku         TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    category_id INTEGER NOT NULL DEFAULT 0,
    unit_code   TEXT NOT NULL DEFAULT 'pc',
    sell_by     TEXT NOT NULL DEFAULT 'unit',
    price       TEXT NOT NULL,
    vat_rate    TEXT,
    track_stock INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_products_name ON products(name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS ix_products_cat ON products(category_id);

CREATE TABLE IF NOT EXISTS variants (
    id         INTEGER PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    public_id  TEXT NOT NULL,
    sku        TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL,
    price      TEXT
);
CREATE INDEX IF NOT EXISTS ix_variants_product ON variants(product_id);

CREATE TABLE IF NOT EXISTS barcodes (
    barcode    TEXT PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id INTEGER NOT NULL DEFAULT 0,
    pack_qty   TEXT NOT NULL DEFAULT '1'
);

CREATE TABLE IF NOT EXISTS price_lists (
    id             INTEGER PRIMARY KEY,
    branch_id      INTEGER NOT NULL DEFAULT 0,
    customer_group TEXT NOT NULL DEFAULT '',
    priority       INTEGER NOT NULL DEFAULT 0,
    valid_from     TEXT,
    valid_to       TEXT
);

CREATE TABLE IF NOT EXISTS price_items (
    list_id    INTEGER NOT NULL REFERENCES price_lists(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL,
    variant_id INTEGER NOT NULL DEFAULT 0,
    min_qty    TEXT NOT NULL,
    price      TEXT NOT NULL,
    PRIMARY KEY (list_id, product_id, variant_id, min_qty)
);

CREATE TABLE IF NOT EXISTS promotions (
    id        INTEGER PRIMARY KEY,
    name      TEXT NOT NULL,
    kind      TEXT NOT NULL,
    config    TEXT NOT NULL,
    starts_at TEXT,
    ends_at   TEXT,
    days_mask INTEGER,
    priority  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS promo_targets (
    promotion_id INTEGER NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    product_id   INTEGER NOT NULL DEFAULT 0,
    category_id  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS cashiers (
    id          TEXT PRIMARY KEY, -- user public id
    name        TEXT NOT NULL,
    role_name   TEXT NOT NULL,
    pin_hash    TEXT NOT NULL,
    permissions TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS modules (
    code TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS held_carts (
    id         TEXT PRIMARY KEY,
    label      TEXT NOT NULL DEFAULT '',
    cart       TEXT NOT NULL,
    held_by    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    synced     INTEGER NOT NULL DEFAULT 0 -- 1 = also parked server-side
);

CREATE TABLE IF NOT EXISTS pending_sales (
    id         TEXT PRIMARY KEY,
    cart       TEXT NOT NULL,
    totals     TEXT NOT NULL,
    payment    TEXT NOT NULL DEFAULT 'cash',
    cashier_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    uploaded   INTEGER NOT NULL DEFAULT 0 -- Phase 7 uploads these
);
`

// Open opens (and if needed creates) the local database at path.
func Open(ctx context.Context, path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", path)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite serialises writers anyway
	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("localdb schema: %w", err)
	}
	return &DB{db}, nil
}

// Meta reads a meta key; missing keys return ("", nil).
func (d *DB) Meta(ctx context.Context, key string) (string, error) {
	var v string
	err := d.GetContext(ctx, &v, `SELECT value FROM meta WHERE key = ?`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetMeta upserts a meta key.
func (d *DB) SetMeta(ctx context.Context, key, value string) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO meta (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}
