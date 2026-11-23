package localdb

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/pkg/pricecalc"
)

// QuoteLine is one cart row the caller wants priced.
type QuoteLine struct {
	ProductID  uint64 `json:"product_id"`
	VariantID  uint64 `json:"variant_id"`
	CategoryID uint64 `json:"category_id"`
	Qty        string `json:"qty"`
}

// PricedLine is the resolved result for one row.
type PricedLine struct {
	QuoteLine
	UnitPrice string `json:"unit_price"`
	Discount  string `json:"discount"`
	Total     string `json:"total"`
	PromoName string `json:"promo_name,omitempty"`
	Source    string `json:"source"` // base | price_list
}

// Quote is the priced cart.
type Quote struct {
	Subtotal string       `json:"subtotal"`
	Discount string       `json:"discount"`
	Vat      string       `json:"vat"`
	Total    string       `json:"total"`
	Lines    []PricedLine `json:"lines"`
}

// QuoteCart prices cart lines offline: branch price-list tiers first (same
// rule as the server — highest priority list, largest applicable min_qty),
// then synced promotions via the shared pricecalc engine. VAT is computed
// per-line from each product's rate on the discounted amount.
func (d *DB) QuoteCart(ctx context.Context, lines []QuoteLine) (Quote, error) {
	resolved := make([]pricecalc.Line, 0, len(lines))
	out := Quote{Lines: make([]PricedLine, len(lines))}
	prices := make([]decimal.Decimal, len(lines))

	for i, l := range lines {
		qty, err := decimal.NewFromString(l.Qty)
		if err != nil || qty.Sign() <= 0 {
			qty = decimal.NewFromInt(1)
		}
		price, src, err := d.resolve(ctx, l.ProductID, l.VariantID, qty)
		if err != nil {
			return out, err
		}
		prices[i] = price
		out.Lines[i].QuoteLine = l
		out.Lines[i].Source = src
		resolved = append(resolved, pricecalc.Line{
			ProductID: l.ProductID, VariantID: l.VariantID,
			CategoryID: l.CategoryID, Qty: qty, UnitPrice: price,
		})
	}

	promos, err := d.activePromos(ctx)
	if err != nil {
		return out, err
	}
	totals := pricecalc.Apply(resolved, promos, time.Now())

	sub := decimal.Zero
	vat := decimal.Zero
	byLine := map[int]pricecalc.LineDiscount{}
	for _, ld := range totals.Lines {
		byLine[ld.LineIndex] = ld
	}
	for i, l := range lines {
		qty := resolved[i].Qty
		eff := prices[i]
		disc := decimal.Zero
		if ld, ok := byLine[i]; ok {
			disc = ld.Amount
			eff = ld.UnitPrice
			out.Lines[i].PromoName = ld.Name
		}
		lineTotal := qty.Mul(eff)
		sub = sub.Add(qty.Mul(prices[i]))
		rate, err := d.vatRate(ctx, l.ProductID)
		if err != nil {
			return out, err
		}
		if rate != nil {
			vat = vat.Add(lineTotal.Mul(*rate).Div(decimal.NewFromInt(100).Add(*rate)))
		}
		out.Lines[i].UnitPrice = prices[i].StringFixed(2)
		out.Lines[i].Discount = disc.StringFixed(2)
		out.Lines[i].Total = lineTotal.StringFixed(2)
	}
	out.Subtotal = sub.StringFixed(2)
	out.Discount = totals.Discount.StringFixed(2)
	out.Vat = vat.StringFixed(2)
	out.Total = totals.Total.StringFixed(2)
	return out, nil
}

// resolve picks the best price-list tier, else the catalog/variant base.
func (d *DB) resolve(ctx context.Context, productID, variantID uint64, qty decimal.Decimal) (decimal.Decimal, string, error) {
	var price string
	err := d.GetContext(ctx, &price,
		`SELECT i.price FROM price_items i
		 JOIN price_lists l ON l.id = i.list_id
		 WHERE i.product_id = ? AND i.variant_id = ?
		   AND CAST(i.min_qty AS REAL) <= ?
		   AND (l.valid_from IS NULL OR l.valid_from <= ?)
		   AND (l.valid_to IS NULL OR l.valid_to >= ?)
		 ORDER BY l.priority DESC, CAST(i.min_qty AS REAL) DESC LIMIT 1`,
		productID, variantID, qty.InexactFloat64(),
		time.Now().Format(time.RFC3339Nano), time.Now().Format(time.RFC3339Nano))
	if err == nil {
		p, err := decimal.NewFromString(price)
		return p, "price_list", err
	}
	var base *string
	err = d.GetContext(ctx, &base,
		`SELECT CASE WHEN ? = 0 THEN price ELSE
			(SELECT COALESCE(v.price, p.price) FROM variants v WHERE v.id = ? AND v.product_id = p.id)
		 END FROM products p WHERE p.id = ?`, variantID, variantID, productID)
	if err != nil {
		return decimal.Zero, "", err
	}
	if base == nil {
		return decimal.Zero, "", err
	}
	p, err := decimal.NewFromString(*base)
	return p, "base", err
}

// activePromos loads live promotions with their target sets.
func (d *DB) activePromos(ctx context.Context) ([]pricecalc.Promotion, error) {
	var rows []struct {
		ID       uint64  `db:"id"`
		Name     string  `db:"name"`
		Kind     string  `db:"kind"`
		Config   string  `db:"config"`
		StartsAt *string `db:"starts_at"`
		EndsAt   *string `db:"ends_at"`
		DaysMask *int    `db:"days_mask"`
		Priority int     `db:"priority"`
	}
	if err := d.SelectContext(ctx, &rows,
		`SELECT id, name, kind, config, starts_at, ends_at, days_mask, priority FROM promotions`); err != nil {
		return nil, err
	}
	var targets []struct {
		PromotionID uint64 `db:"promotion_id"`
		ProductID   uint64 `db:"product_id"`
		CategoryID  uint64 `db:"category_id"`
	}
	if err := d.SelectContext(ctx, &targets,
		`SELECT promotion_id, product_id, category_id FROM promo_targets`); err != nil {
		return nil, err
	}
	byPromo := map[uint64][]struct {
		PromotionID uint64 `db:"promotion_id"`
		ProductID   uint64 `db:"product_id"`
		CategoryID  uint64 `db:"category_id"`
	}{}
	for _, t := range targets {
		byPromo[t.PromotionID] = append(byPromo[t.PromotionID], t)
	}
	out := make([]pricecalc.Promotion, 0, len(rows))
	for _, r := range rows {
		var cfg pricecalc.Config
		if err := json.Unmarshal([]byte(r.Config), &cfg); err != nil {
			continue
		}
		p := pricecalc.Promotion{
			ID: r.ID, Name: r.Name, Kind: pricecalc.Kind(r.Kind), Config: cfg,
			Priority: r.Priority,
			StartsAt: parseTime(r.StartsAt), EndsAt: parseTime(r.EndsAt),
			Products: map[uint64]bool{}, Categories: map[uint64]bool{},
		}
		if r.DaysMask != nil {
			m := uint8(*r.DaysMask) //nolint:gosec // days mask is a 7-bit weekday set
			p.DaysMask = &m
		}
		for _, t := range byPromo[r.ID] {
			if t.ProductID != 0 {
				p.Products[t.ProductID] = true
			}
			if t.CategoryID != 0 {
				p.Categories[t.CategoryID] = true
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func (d *DB) vatRate(ctx context.Context, productID uint64) (*decimal.Decimal, error) {
	var raw *string
	err := d.GetContext(ctx, &raw, `SELECT vat_rate FROM products WHERE id = ?`, productID)
	if err != nil || raw == nil {
		return nil, err
	}
	v, err := decimal.NewFromString(*raw)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		return nil
	}
	return &t
}
