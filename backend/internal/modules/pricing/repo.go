package pricing

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// Repo persists price lists, promotions and history.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo {
	return &Repo{db: conn}
}

// ---- price lists ---------------------------------------------------------------

const listCols = `id, tenant_id, public_id, name, branch_id, customer_group,
	priority, is_active, valid_from, valid_to, created_at, updated_at`

// CreateList inserts a price list and its items inside q's transaction.
func (r *Repo) CreateList(ctx context.Context, q db.Querier, l *PriceList) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO price_lists
		 (tenant_id, public_id, name, branch_id, customer_group, priority, is_active, valid_from, valid_to)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		l.TenantID, l.PublicID, l.Name, l.BranchID, l.CustomerGroup, l.Priority,
		l.IsActive, l.ValidFrom, l.ValidTo)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = db.U64(id)
	return r.ReplaceItems(ctx, q, l.ID, l.Items)
}

// ReplaceItems rewrites a list's items inside q's transaction.
func (r *Repo) ReplaceItems(ctx context.Context, q db.Querier, listID uint64, items []PriceListItem) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM price_list_items WHERE list_id=?`, listID); err != nil {
		return err
	}
	for i := range items {
		items[i].ListID = listID
		res, err := q.ExecContext(ctx,
			`INSERT INTO price_list_items (list_id, product_id, variant_id, min_qty, price)
			 VALUES (?,?,?,?,?)`,
			listID, items[i].ProductID, items[i].VariantID, items[i].MinQty, items[i].Price)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		items[i].ID = db.U64(id)
	}
	return nil
}

// ListByPublicID loads a list with its items (hydrated for display).
func (r *Repo) ListByPublicID(ctx context.Context, q db.Querier, tenantID uint64, publicID string) (PriceList, error) {
	var l PriceList
	err := q.GetContext(ctx, &l,
		`SELECT `+listCols+` FROM price_lists WHERE tenant_id=? AND public_id=?`, tenantID, publicID)
	if err != nil {
		return l, err
	}
	var items []PriceListItem
	if err := q.SelectContext(ctx, &items,
		`SELECT i.*, p.public_id AS product_public_id, p.name AS product_name,
			COALESCE(v.public_id,'') AS variant_public_id
		 FROM price_list_items i
		 JOIN products p ON p.id=i.product_id
		 LEFT JOIN product_variants v ON v.id=i.variant_id
		 WHERE i.list_id=? ORDER BY p.name, i.min_qty`, l.ID); err != nil {
		return l, err
	}
	l.Items = items
	var pub sql.NullString
	if l.BranchID != nil {
		_ = q.GetContext(ctx, &pub, `SELECT public_id FROM branches WHERE id=?`, *l.BranchID)
	}
	l.BranchPublicID = pub.String
	return l, nil
}

// Lists returns all lists for a tenant, newest first.
func (r *Repo) Lists(ctx context.Context, tenantID uint64) ([]PriceList, error) {
	var out []PriceList
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+listCols+` FROM price_lists WHERE tenant_id=? ORDER BY priority DESC, name`, tenantID)
	return out, err
}

// UpdateList updates list head fields inside q's transaction (items are
// rewritten via ReplaceItems).
func (r *Repo) UpdateList(ctx context.Context, q db.Querier, l *PriceList) error {
	res, err := q.ExecContext(ctx,
		`UPDATE price_lists SET name=?, branch_id=?, customer_group=?, priority=?,
		 is_active=?, valid_from=?, valid_to=? WHERE id=?`,
		l.Name, l.BranchID, l.CustomerGroup, l.Priority, l.IsActive,
		l.ValidFrom, l.ValidTo, l.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteList removes a list (items cascade).
func (r *Repo) DeleteList(ctx context.Context, tenantID uint64, id uint64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM price_lists WHERE tenant_id=? AND id=?`, tenantID, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ApplicableLists returns active lists that apply to a branch + customer
// group at now, highest priority first.
func (r *Repo) ApplicableLists(ctx context.Context, tenantID, branchID uint64, group string, now time.Time) ([]PriceList, error) {
	var out []PriceList
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+listCols+` FROM price_lists
		 WHERE tenant_id=? AND is_active=1
		   AND (branch_id IS NULL OR branch_id=?)
		   AND (customer_group IS NULL OR customer_group=?)
		   AND (valid_from IS NULL OR valid_from<=?)
		   AND (valid_to IS NULL OR valid_to>=?)
		 ORDER BY priority DESC, id`, tenantID, branchID, group, now, now)
	return out, err
}

// TierFor finds the best tier for (product, variant, qty) within a list:
// the largest min_qty not exceeding qty.
func (r *Repo) TierFor(ctx context.Context, q db.Querier, listID, productID, variantID uint64, qty string) (PriceListItem, error) {
	var it PriceListItem
	err := q.GetContext(ctx, &it,
		`SELECT * FROM price_list_items
		 WHERE list_id=? AND product_id=? AND variant_id=? AND min_qty<=?
		 ORDER BY min_qty DESC LIMIT 1`, listID, productID, variantID, qty)
	return it, err
}

// ---- promotions ------------------------------------------------------------------

const promoCols = `id, tenant_id, public_id, name, kind, config, starts_at, ends_at,
	days_mask, priority, is_active, created_at, updated_at`

// CreatePromotion inserts a promotion and targets inside q's transaction.
func (r *Repo) CreatePromotion(ctx context.Context, q db.Querier, p *Promotion) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO promotions
		 (tenant_id, public_id, name, kind, config, starts_at, ends_at, days_mask, priority, is_active)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.TenantID, p.PublicID, p.Name, p.Kind, p.Config, p.StartsAt, p.EndsAt,
		p.DaysMask, p.Priority, p.IsActive)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = db.U64(id)
	return r.ReplaceTargets(ctx, q, p.ID, p.Targets)
}

// ReplaceTargets rewrites a promotion's targets inside q's transaction.
func (r *Repo) ReplaceTargets(ctx context.Context, q db.Querier, promoID uint64, ts []PromotionTarget) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM promotion_targets WHERE promotion_id=?`, promoID); err != nil {
		return err
	}
	for _, t := range ts {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO promotion_targets (promotion_id, product_id, category_id) VALUES (?,?,?)`,
			promoID, t.ProductID, t.CategoryID); err != nil {
			return err
		}
	}
	return nil
}

// PromotionByPublicID loads a promotion with its targets.
func (r *Repo) PromotionByPublicID(ctx context.Context, tenantID uint64, publicID string) (Promotion, error) {
	var p Promotion
	err := r.db.GetContext(ctx, &p,
		`SELECT `+promoCols+` FROM promotions WHERE tenant_id=? AND public_id=?`, tenantID, publicID)
	if err != nil {
		return p, err
	}
	p.Targets, err = r.targets(ctx, r.db, p.ID)
	return p, err
}

// Promotions lists all promotions.
func (r *Repo) Promotions(ctx context.Context, tenantID uint64) ([]Promotion, error) {
	var out []Promotion
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+promoCols+` FROM promotions WHERE tenant_id=? ORDER BY priority DESC, name`, tenantID)
	for i := range out {
		out[i].Targets, err = r.targets(ctx, r.db, out[i].ID)
		if err != nil {
			return out, err
		}
	}
	return out, err
}

// ActivePromotions returns promotions usable at now (the engine re-checks
// windows/days; this filters only is_active).
func (r *Repo) ActivePromotions(ctx context.Context, tenantID uint64) ([]Promotion, error) {
	var out []Promotion
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+promoCols+` FROM promotions WHERE tenant_id=? AND is_active=1`, tenantID)
	for i := range out {
		out[i].Targets, err = r.targets(ctx, r.db, out[i].ID)
		if err != nil {
			return out, err
		}
	}
	return out, err
}

func (r *Repo) targets(ctx context.Context, q db.Querier, promoID uint64) ([]PromotionTarget, error) {
	var out []PromotionTarget
	err := q.SelectContext(ctx, &out,
		`SELECT t.*, COALESCE(p.public_id,'') AS product_public_id,
			COALESCE(c.public_id,'') AS category_public_id
		 FROM promotion_targets t
		 LEFT JOIN products p ON p.id=t.product_id
		 LEFT JOIN categories c ON c.id=t.category_id
		 WHERE t.promotion_id=?`, promoID)
	return out, err
}

// UpdatePromotion updates head fields inside q's transaction.
func (r *Repo) UpdatePromotion(ctx context.Context, q db.Querier, p *Promotion) error {
	res, err := q.ExecContext(ctx,
		`UPDATE promotions SET name=?, kind=?, config=?, starts_at=?, ends_at=?,
		 days_mask=?, priority=?, is_active=? WHERE id=?`,
		p.Name, p.Kind, p.Config, p.StartsAt, p.EndsAt, p.DaysMask, p.Priority, p.IsActive, p.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeletePromotion removes a promotion (targets cascade).
func (r *Repo) DeletePromotion(ctx context.Context, tenantID uint64, id uint64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM promotions WHERE tenant_id=? AND id=?`, tenantID, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- history -----------------------------------------------------------------------

// InsertHistory writes price-history rows inside q's transaction.
func (r *Repo) InsertHistory(ctx context.Context, q db.Querier, rows []HistoryRow) error {
	for _, h := range rows {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO price_history
			 (tenant_id, product_id, variant_id, source, ref_public_id, old_price, new_price, actor_id)
			 VALUES (?,?,?,?,?,?,?,?)`,
			h.TenantID, h.ProductID, h.VariantID, h.Source, h.RefPublicID,
			h.OldPrice, h.NewPrice, h.ActorID); err != nil {
			return err
		}
	}
	return nil
}

// History lists price changes for a product (variant 0 = base).
func (r *Repo) History(ctx context.Context, tenantID, productID uint64, pg httpx.Page) ([]HistoryRow, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM price_history WHERE tenant_id=? AND product_id=?`, tenantID, productID); err != nil {
		return nil, 0, err
	}
	var out []HistoryRow
	err := r.db.SelectContext(ctx, &out,
		`SELECT h.*, p.name AS product_name
		 FROM price_history h JOIN products p ON p.id=h.product_id
		 WHERE h.tenant_id=? AND h.product_id=?
		 ORDER BY h.id DESC LIMIT ? OFFSET ?`, tenantID, productID, pg.PerPage, pg.Offset())
	return out, total, err
}
