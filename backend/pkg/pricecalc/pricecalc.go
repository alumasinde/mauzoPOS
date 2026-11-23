// Package pricecalc evaluates promotions against cart lines. It is pure and
// storage-free: the pricing module loads promotions and lines, the engine
// decides which single promotion wins each line (the largest discount — POS
// convention, promos never stack on one line).
package pricecalc

import (
	"time"

	"github.com/shopspring/decimal"
)

var hundred = decimal.NewFromInt(100)

// Kind selects how a promotion discounts matching lines.
type Kind string

// Promotion kinds.
const (
	KindPercent Kind = "percent" // percent off every unit
	KindAmount  Kind = "amount"  // fixed amount off every unit
	KindBXGY    Kind = "bxgy"    // buy N get M at get_percent off
)

// Line is one sellable row (a product or variant at a quantity).
type Line struct {
	ProductID  uint64
	VariantID  uint64 // 0 = base product
	CategoryID uint64 // 0 = uncategorised
	Qty        decimal.Decimal
	UnitPrice  decimal.Decimal // resolved unit price before promotions
}

// Config carries the kind-specific numbers. Fields not used by a kind are
// ignored.
type Config struct {
	Percent    decimal.Decimal // KindPercent: e.g. 10 = 10% off
	Amount     decimal.Decimal // KindAmount: off per unit, clamped at the unit price
	Buy        decimal.Decimal // KindBXGY: units that must be bought…
	Get        decimal.Decimal // …to get this many…
	GetPercent decimal.Decimal // …at this percent off (100 = free)
}

// Promotion is one active rule with its resolved target sets. Empty Products
// and Categories means the promotion targets the whole catalog.
type Promotion struct {
	ID         uint64
	Name       string
	Kind       Kind
	Config     Config
	StartsAt   *time.Time
	EndsAt     *time.Time
	DaysMask   *uint8 // bit0=Mon … bit6=Sun; nil = every day
	Priority   int    // higher = preferred on ties
	Products   map[uint64]bool
	Categories map[uint64]bool
}

// LineDiscount is the winning discount for one line.
type LineDiscount struct {
	LineIndex   int
	PromotionID uint64
	Name        string
	Amount      decimal.Decimal // total discount for the line (qty-level)
	UnitPrice   decimal.Decimal // effective unit price after the discount
}

// Totals is the priced result of a set of lines.
type Totals struct {
	Subtotal decimal.Decimal // Σ qty × unit price
	Discount decimal.Decimal // Σ line discounts
	Total    decimal.Decimal // Subtotal − Discount
	Lines    []LineDiscount
}

// Apply prices lines under promos active at now. Each line keeps at most one
// promotion — the one giving the biggest absolute discount (ties: higher
// Priority, then lower promotion id for determinism).
func Apply(lines []Line, promos []Promotion, now time.Time) Totals {
	var t Totals
	for i, l := range lines {
		t.Subtotal = t.Subtotal.Add(l.Qty.Mul(l.UnitPrice))
		var best LineDiscount
		var bestPrio int
		found := false
		for _, p := range promos {
			if !p.live(now) || !p.targets(l) {
				continue
			}
			amount := p.discount(l)
			if amount.LessThanOrEqual(decimal.Zero) {
				continue
			}
			better := amount.GreaterThan(best.Amount) ||
				(amount.Equal(best.Amount) && found &&
					(p.Priority > bestPrio || (p.Priority == bestPrio && p.ID < best.PromotionID)))
			if !better {
				continue
			}
			best = LineDiscount{LineIndex: i, PromotionID: p.ID, Name: p.Name, Amount: amount}
			bestPrio = p.Priority
			found = true
		}
		if found {
			best.UnitPrice = l.UnitPrice.Sub(best.Amount.Div(l.Qty))
			t.Discount = t.Discount.Add(best.Amount)
			t.Lines = append(t.Lines, best)
		}
	}
	t.Total = t.Subtotal.Sub(t.Discount)
	return t
}

// live reports whether the promotion is in force at now.
func (p Promotion) live(now time.Time) bool {
	if p.StartsAt != nil && now.Before(*p.StartsAt) {
		return false
	}
	if p.EndsAt != nil && now.After(*p.EndsAt) {
		return false
	}
	if p.DaysMask != nil {
		// Go Weekday: Sunday=0 … Saturday=6; our mask: bit0=Mon … bit6=Sun.
		bit := (int(now.Weekday()) + 6) % 7
		if *p.DaysMask&(1<<bit) == 0 {
			return false
		}
	}
	return true
}

// targets reports whether the promotion covers the line. A promotion with no
// targets covers everything; product targets cover all of a product's
// variants (targets are product/category — there is no per-variant targeting).
func (p Promotion) targets(l Line) bool {
	if len(p.Products) == 0 && len(p.Categories) == 0 {
		return true
	}
	return p.Products[l.ProductID] || (l.CategoryID != 0 && p.Categories[l.CategoryID])
}

// discount computes the line-level discount the promotion grants.
func (p Promotion) discount(l Line) decimal.Decimal {
	switch p.Kind {
	case KindPercent:
		return l.Qty.Mul(l.UnitPrice).Mul(p.Config.Percent).Div(hundred)
	case KindAmount:
		per := decimal.Min(p.Config.Amount, l.UnitPrice)
		return l.Qty.Mul(per)
	case KindBXGY:
		bundle := p.Config.Buy.Add(p.Config.Get)
		if bundle.LessThanOrEqual(decimal.Zero) || l.Qty.LessThan(bundle) {
			return decimal.Zero
		}
		sets := l.Qty.Div(bundle).Floor()
		return sets.Mul(p.Config.Get).Mul(l.UnitPrice).Mul(p.Config.GetPercent).Div(hundred)
	}
	return decimal.Zero
}
