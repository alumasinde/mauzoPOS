package catalog

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Overview powers the KPI cards on the products page. Every figure is derived
// from real rows: product counts from `products`, low stock from
// `reorder_levels` vs `stock_balances`, and the low-stock trend replayed from
// the `stock_movements` ledger.
type Overview struct {
	Total         int      `json:"total"`
	Active        int      `json:"active"`
	Archived      int      `json:"archived"`
	LowStock      int      `json:"low_stock"`
	Added30d      int      `json:"added_30d"`
	DeltaPct      *float64 `json:"delta_pct"`       // growth vs 30 days ago
	DeltaActive   *float64 `json:"delta_active"`    // active growth vs 30 days ago
	TotalTrend    []int    `json:"trend_total"`     // cumulative catalogue size, last 30 days
	ActiveTrend   []int    `json:"trend_active"`    // cumulative active products
	ArchivedTrend []int    `json:"trend_archived"`  // cumulative archived products
	LowStockTrend []int    `json:"trend_low_stock"` // products below reorder, per day
}

const overviewWindowDays = 30

// pb keys a product's balance at one branch.
type pb struct {
	product uint64
	branch  uint64
}

// Overview compiles catalogue KPIs for the tenant's products page.
func (s *Service) Overview(ctx context.Context, tenantID uint64) (Overview, error) {
	o := Overview{}
	if err := s.db.GetContext(ctx, &o.Total,
		`SELECT COUNT(*) FROM products WHERE tenant_id = ?`, tenantID); err != nil {
		return o, err
	}
	if err := s.db.GetContext(ctx, &o.Active,
		`SELECT COUNT(*) FROM products WHERE tenant_id = ? AND status = 'active'`, tenantID); err != nil {
		return o, err
	}
	o.Archived = o.Total - o.Active
	if err := s.db.GetContext(ctx, &o.Added30d,
		`SELECT COUNT(*) FROM products WHERE tenant_id = ? AND created_at >= ?`,
		tenantID, time.Now().UTC().AddDate(0, 0, -overviewWindowDays)); err != nil {
		return o, err
	}
	if base := o.Total - o.Added30d; base > 0 {
		d := float64(o.Added30d) / float64(base) * 100
		o.DeltaPct = &d
	}
	if err := s.lowStock(ctx, &o, tenantID); err != nil {
		return o, err
	}
	return o, s.statusTrends(ctx, &o, tenantID)
}

// lowStock counts products that are below a reorder level at any branch, and
// replays the same count backwards over the window. A product counts once no
// matter how many branches it is low at.
func (s *Service) lowStock(ctx context.Context, o *Overview, tenantID uint64) error {
	type balRow struct {
		BranchID  uint64          `db:"branch_id"`
		ProductID uint64          `db:"product_id"`
		Qty       decimal.Decimal `db:"qty"`
	}
	var bals []balRow
	if err := s.db.SelectContext(ctx, &bals,
		`SELECT branch_id, product_id, SUM(qty) AS qty FROM stock_balances
		 WHERE tenant_id = ? GROUP BY branch_id, product_id`, tenantID); err != nil {
		return err
	}
	type rlRow struct {
		BranchID  uint64          `db:"branch_id"`
		ProductID uint64          `db:"product_id"`
		MinQty    decimal.Decimal `db:"min_qty"`
	}
	var levels []rlRow
	if err := s.db.SelectContext(ctx, &levels,
		`SELECT branch_id, product_id, SUM(min_qty) AS min_qty FROM reorder_levels
		 WHERE tenant_id = ? AND min_qty > 0 GROUP BY branch_id, product_id`, tenantID); err != nil {
		return err
	}
	if len(levels) == 0 {
		return nil
	}
	low := func(bal map[pb]decimal.Decimal) int {
		n := 0
		seen := map[uint64]bool{}
		for _, l := range levels {
			if seen[l.ProductID] {
				continue
			}
			if bal[pb{l.ProductID, l.BranchID}].LessThan(l.MinQty) {
				seen[l.ProductID] = true
				n++
			}
		}
		return n
	}
	cur := map[pb]decimal.Decimal{}
	for _, b := range bals {
		cur[pb{b.ProductID, b.BranchID}] = b.Qty
	}
	o.LowStock = low(cur)
	return s.lowStockTrend(ctx, o, tenantID, low, cur)
}

// lowStockTrend replays daily net movement per (branch, product), subtracting
// day by day backwards from the current balance to get each day's close.
func (s *Service) lowStockTrend(ctx context.Context, o *Overview, tenantID uint64,
	low func(map[pb]decimal.Decimal) int, cur map[pb]decimal.Decimal) error {
	since := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -overviewWindowDays+1)
	type deltaRow struct {
		BranchID  uint64          `db:"branch_id"`
		ProductID uint64          `db:"product_id"`
		Day       time.Time       `db:"d"`
		Qty       decimal.Decimal `db:"qty"`
	}
	var deltas []deltaRow
	if err := s.db.SelectContext(ctx, &deltas,
		`SELECT branch_id, product_id, DATE(created_at) AS d, SUM(qty) AS qty
		 FROM stock_movements WHERE tenant_id = ? AND created_at >= ?
		 GROUP BY branch_id, product_id, DATE(created_at)`, tenantID, since); err != nil {
		return err
	}
	byDay := map[string]map[pb]decimal.Decimal{}
	for _, d := range deltas {
		key := d.Day.Format("2006-01-02")
		if byDay[key] == nil {
			byDay[key] = map[pb]decimal.Decimal{}
		}
		k := pb{d.ProductID, d.BranchID}
		byDay[key][k] = byDay[key][k].Add(d.Qty)
	}
	counts := map[string]int{}
	day := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < overviewWindowDays; i++ {
		key := day.Format("2006-01-02")
		counts[key] = low(cur)
		for k, q := range byDay[key] {
			cur[k] = cur[k].Sub(q)
		}
		day = day.AddDate(0, 0, -1)
	}
	trend := make([]int, 0, overviewWindowDays)
	for i := 0; i < overviewWindowDays; i++ {
		key := day.AddDate(0, 0, i+1).Format("2006-01-02")
		trend = append(trend, counts[key])
	}
	o.LowStockTrend = trend
	return nil
}

// statusTrends replays the cumulative catalogue size per current status over
// the window — i.e. on day D, how many of today's active (or archived)
// products had been created. Archive dates are not tracked, so this is the
// honest per-status history we can derive.
func (s *Service) statusTrends(ctx context.Context, o *Overview, tenantID uint64) error {
	type row struct {
		Status string    `db:"status"`
		Day    time.Time `db:"d"`
		N      int       `db:"n"`
	}
	var rows []row
	since := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -overviewWindowDays+1)
	if err := s.db.SelectContext(ctx, &rows,
		`SELECT status, DATE(created_at) AS d, COUNT(*) AS n FROM products
		 WHERE tenant_id = ? AND created_at >= ? GROUP BY status, DATE(created_at)`,
		tenantID, since); err != nil {
		return err
	}
	addedTotal := map[string]int{}
	addedActive := map[string]int{}
	addedArchived := map[string]int{}
	addedAct30, addedArch30 := 0, 0
	for _, r := range rows {
		key := r.Day.Format("2006-01-02")
		addedTotal[key] += r.N
		if r.Status == StatusArchived {
			addedArchived[key] += r.N
			addedArch30 += r.N
		} else {
			addedActive[key] += r.N
			addedAct30 += r.N
		}
	}
	if base := o.Active - addedAct30; base > 0 {
		d := float64(addedAct30) / float64(base) * 100
		o.DeltaActive = &d
	}
	o.TotalTrend = cumulative(overviewWindowDays, since, o.Total-o.Added30d, addedTotal)
	o.ActiveTrend = cumulative(overviewWindowDays, since, o.Active-addedAct30, addedActive)
	o.ArchivedTrend = cumulative(overviewWindowDays, since, o.Archived-addedArch30, addedArchived)
	return nil
}

// cumulative walks the window from `since`, adding each day's new rows to a
// running baseline — the series ends at the current count.
func cumulative(days int, since time.Time, base int, added map[string]int) []int {
	trend := make([]int, 0, days)
	day := since
	for i := 0; i < days; i++ {
		base += added[day.Format("2006-01-02")]
		trend = append(trend, base)
		day = day.AddDate(0, 0, 1)
	}
	return trend
}
