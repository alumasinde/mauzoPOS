package localdb

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Bundle mirrors GET /pos/sync. Decimal fields stay strings end-to-end.
type Bundle struct {
	ServerTime time.Time `json:"server_time"`
	Cursor     time.Time `json:"cursor"`
	Categories []struct {
		ID       uint64 `json:"id"`
		PublicID string `json:"public_id"`
		ParentID uint64 `json:"parent_id"`
		Name     string `json:"name"`
		Sort     int    `json:"sort"`
	} `json:"categories"`
	Units []struct {
		ID     uint64 `json:"id"`
		Code   string `json:"code"`
		Name   string `json:"name"`
		Kind   string `json:"kind"`
		Factor string `json:"factor"`
	} `json:"units"`
	Products []struct {
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
	} `json:"products"`
	Variants []struct {
		ID        uint64  `json:"id"`
		ProductID uint64  `json:"product_id"`
		PublicID  string  `json:"public_id"`
		SKU       string  `json:"sku"`
		Name      string  `json:"name"`
		Price     *string `json:"price"`
	} `json:"variants"`
	Barcodes []struct {
		ProductID uint64 `json:"product_id"`
		VariantID uint64 `json:"variant_id"`
		Barcode   string `json:"barcode"`
		PackQty   string `json:"pack_qty"`
	} `json:"barcodes"`
	PriceLists []struct {
		ID            uint64     `json:"id"`
		BranchID      uint64     `json:"branch_id"`
		CustomerGroup string     `json:"customer_group"`
		Priority      int        `json:"priority"`
		ValidFrom     *time.Time `json:"valid_from"`
		ValidTo       *time.Time `json:"valid_to"`
	} `json:"price_lists"`
	PriceItems []struct {
		ListID    uint64 `json:"list_id"`
		ProductID uint64 `json:"product_id"`
		VariantID uint64 `json:"variant_id"`
		MinQty    string `json:"min_qty"`
		Price     string `json:"price"`
	} `json:"price_items"`
	Promotions []struct {
		ID       uint64     `json:"id"`
		Name     string     `json:"name"`
		Kind     string     `json:"kind"`
		Config   string     `json:"config"`
		StartsAt *time.Time `json:"starts_at"`
		EndsAt   *time.Time `json:"ends_at"`
		DaysMask *uint8     `json:"days_mask"`
		Priority int        `json:"priority"`
	} `json:"promotions"`
	PromoTargets []struct {
		PromotionID uint64 `json:"promotion_id"`
		ProductID   uint64 `json:"product_id"`
		CategoryID  uint64 `json:"category_id"`
	} `json:"promotion_targets"`
	Cashiers []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		RoleName    string   `json:"role_name"`
		PINHash     *string  `json:"pin_hash"`
		Permissions []string `json:"permissions"`
		MaxDiscount *string  `json:"max_discount_percent"`
	} `json:"cashiers"`
	Customers []struct {
		ID            uint64  `json:"id"`
		PublicID      string  `json:"public_id"`
		Name          string  `json:"name"`
		Phone         string  `json:"phone"`
		Code          string  `json:"code"`
		CreditLimit   *string `json:"credit_limit"`
		CreditBalance string  `json:"credit_balance"`
	} `json:"customers"`
	Settings []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"settings"`
	Modules []string `json:"modules"`
}

// ApplySync writes a bundle into the local mirror in one transaction.
// Products are upserted (incremental); the small reference tables are
// rebuilt so deletions propagate.
func (d *DB) ApplySync(ctx context.Context, b Bundle) error {
	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rebuild := []string{
		"categories", "units", "barcodes", "price_lists", "price_items",
		"promotions", "promo_targets", "cashiers", "settings", "modules", "variants", "customers",
	}
	for _, t := range rebuild {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+t); err != nil {
			return err
		}
	}
	if err := applyCatalog(ctx, tx, b); err != nil {
		return err
	}
	if err := applyPricing(ctx, tx, b); err != nil {
		return err
	}
	if err := applyAccess(ctx, tx, b); err != nil {
		return err
	}
	return tx.Commit()
}

// applyCatalog mirrors categories, units, products, variants, and barcodes.
func applyCatalog(ctx context.Context, tx *sqlx.Tx, b Bundle) error {
	for _, c := range b.Categories {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO categories (id, public_id, parent_id, name, sort) VALUES (?,?,?,?,?)`,
			c.ID, c.PublicID, c.ParentID, c.Name, c.Sort); err != nil {
			return err
		}
	}
	for _, u := range b.Units {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO units (id, code, name, kind, factor) VALUES (?,?,?,?,?)`,
			u.ID, u.Code, u.Name, u.Kind, u.Factor); err != nil {
			return err
		}
	}
	for _, p := range b.Products {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO products (id, public_id, sku, name, kind, category_id, unit_code,
				sell_by, price, vat_rate, track_stock, updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
			 ON CONFLICT(id) DO UPDATE SET public_id=excluded.public_id, sku=excluded.sku,
				name=excluded.name, kind=excluded.kind, category_id=excluded.category_id,
				unit_code=excluded.unit_code, sell_by=excluded.sell_by, price=excluded.price,
				vat_rate=excluded.vat_rate, track_stock=excluded.track_stock, updated_at=excluded.updated_at`,
			p.ID, p.PublicID, p.SKU, p.Name, p.Kind, p.CategoryID, p.UnitCode,
			p.SellBy, p.Price, p.VatRate, b2i(p.TrackStock), p.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	for _, v := range b.Variants {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO variants (id, product_id, public_id, sku, name, price) VALUES (?,?,?,?,?,?)`,
			v.ID, v.ProductID, v.PublicID, v.SKU, v.Name, v.Price); err != nil {
			return err
		}
	}
	for _, bc := range b.Barcodes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO barcodes (barcode, product_id, variant_id, pack_qty) VALUES (?,?,?,?)`,
			bc.Barcode, bc.ProductID, bc.VariantID, bc.PackQty); err != nil {
			return err
		}
	}
	return nil
}

// applyPricing mirrors price lists/tiers and promotions with their targets.
func applyPricing(ctx context.Context, tx *sqlx.Tx, b Bundle) error {
	for _, l := range b.PriceLists {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO price_lists (id, branch_id, customer_group, priority, valid_from, valid_to)
			 VALUES (?,?,?,?,?,?)`,
			l.ID, l.BranchID, l.CustomerGroup, l.Priority, timePtr(l.ValidFrom), timePtr(l.ValidTo)); err != nil {
			return err
		}
	}
	for _, i := range b.PriceItems {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO price_items (list_id, product_id, variant_id, min_qty, price)
			 VALUES (?,?,?,?,?)`,
			i.ListID, i.ProductID, i.VariantID, i.MinQty, i.Price); err != nil {
			return err
		}
	}
	for _, p := range b.Promotions {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO promotions (id, name, kind, config, starts_at, ends_at, days_mask, priority)
			 VALUES (?,?,?,?,?,?,?,?)`,
			p.ID, p.Name, p.Kind, p.Config, timePtr(p.StartsAt), timePtr(p.EndsAt),
			u8Ptr(p.DaysMask), p.Priority); err != nil {
			return err
		}
	}
	for _, t := range b.PromoTargets {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO promo_targets (promotion_id, product_id, category_id) VALUES (?,?,?)`,
			t.PromotionID, t.ProductID, t.CategoryID); err != nil {
			return err
		}
	}
	return nil
}

// applyAccess mirrors cashiers (with permission sets), settings, and modules.
func applyAccess(ctx context.Context, tx *sqlx.Tx, b Bundle) error {
	for _, c := range b.Cashiers {
		perms, err := json.Marshal(c.Permissions)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO cashiers (id, name, role_name, pin_hash, permissions, max_discount) VALUES (?,?,?,?,?,?)`,
			c.ID, c.Name, c.RoleName, c.PINHash, string(perms), c.MaxDiscount); err != nil {
			return err
		}
	}
	for _, cu := range b.Customers {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO customers (id, public_id, name, phone, code, credit_limit, credit_balance)
			 VALUES (?,?,?,?,?,?,?)`,
			cu.ID, cu.PublicID, cu.Name, cu.Phone, cu.Code, cu.CreditLimit, cu.CreditBalance); err != nil {
			return err
		}
	}
	for _, s := range b.Settings {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO settings (key, value) VALUES (?,?)`, s.Key, s.Value); err != nil {
			return err
		}
	}
	for _, m := range b.Modules {
		if _, err := tx.ExecContext(ctx, `INSERT INTO modules (code) VALUES (?)`, m); err != nil {
			return err
		}
	}
	return nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339Nano)
	return &s
}

func u8Ptr(u *uint8) *int {
	if u == nil {
		return nil
	}
	i := int(*u)
	return &i
}

// ResetForActivation wipes every tenant-scoped table so activating a device as
// a new terminal (or a different tenant's till) never sees the previous
// terminal's catalog, cashiers or parked work.
func (d *DB) ResetForActivation(ctx context.Context) error {
	tables := []string{
		"categories", "units", "products", "variants", "barcodes",
		"price_lists", "price_items", "promotions", "promo_targets",
		"cashiers", "settings", "modules", "held_carts", "pending_sales",
	}
	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range tables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+t); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM meta WHERE key IN ('sync_cursor','last_sync','device_token','device_id')`); err != nil {
		return err
	}
	return tx.Commit()
}
