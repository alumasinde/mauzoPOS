package pricecalc

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func lines() []Line {
	return []Line{
		{ProductID: 1, CategoryID: 10, Qty: d("2"), UnitPrice: d("100")},
		{ProductID: 2, Qty: d("6"), UnitPrice: d("50")},
		{ProductID: 3, CategoryID: 20, Qty: d("1"), UnitPrice: d("200")},
	}
}

func TestPercentAndAmountDiscounts(t *testing.T) {
	tot := Apply(lines(), []Promotion{
		{Name: "10% off cat10", Kind: KindPercent, Config: Config{Percent: d("10")}, Categories: map[uint64]bool{10: true}},
		{Name: "KES 5 off p2", Kind: KindAmount, Config: Config{Amount: d("5")}, Products: map[uint64]bool{2: true}},
	}, time.Now())
	if !tot.Subtotal.Equal(d("700")) {
		t.Fatalf("subtotal %s", tot.Subtotal)
	}
	// line0: 2×100×10% = 20; line1: 6×5 = 30
	if !tot.Discount.Equal(d("50")) || !tot.Total.Equal(d("650")) {
		t.Fatalf("discount %s total %s", tot.Discount, tot.Total)
	}
	if tot.Lines[0].UnitPrice.String() != "90" {
		t.Fatalf("effective unit price %s", tot.Lines[0].UnitPrice)
	}
}

func TestBXGY(t *testing.T) {
	// buy 2 get 1 free; qty 6 → two full bundles → 2 free units.
	tot := Apply([]Line{{ProductID: 1, Qty: d("6"), UnitPrice: d("50")}}, []Promotion{
		{Name: "b2g1", Kind: KindBXGY, Config: Config{Buy: d("2"), Get: d("1"), GetPercent: d("100")}},
	}, time.Now())
	if !tot.Discount.Equal(d("100")) {
		t.Fatalf("bxgy discount %s", tot.Discount)
	}
	// qty 5 → one bundle only → 1 free unit.
	tot = Apply([]Line{{ProductID: 1, Qty: d("5"), UnitPrice: d("50")}}, []Promotion{
		{Name: "b2g1", Kind: KindBXGY, Config: Config{Buy: d("2"), Get: d("1"), GetPercent: d("100")}},
	}, time.Now())
	if !tot.Discount.Equal(d("50")) {
		t.Fatalf("partial bundle discount %s", tot.Discount)
	}
}

func TestBestPerLineWins(t *testing.T) {
	tot := Apply([]Line{{ProductID: 1, Qty: d("1"), UnitPrice: d("100")}}, []Promotion{
		{Name: "5%", Kind: KindPercent, Config: Config{Percent: d("5")}},
		{Name: "20%", Kind: KindPercent, Config: Config{Percent: d("20")}},
	}, time.Now())
	if len(tot.Lines) != 1 || !tot.Discount.Equal(d("20")) || tot.Lines[0].Name != "20%" {
		t.Fatalf("expected only the 20%% promo, got %+v", tot.Lines)
	}
	// Equal discount → higher priority wins.
	tot = Apply([]Line{{ProductID: 1, Qty: d("1"), UnitPrice: d("100")}}, []Promotion{
		{ID: 1, Name: "low", Kind: KindPercent, Config: Config{Percent: d("10")}, Priority: 1},
		{ID: 2, Name: "high", Kind: KindPercent, Config: Config{Percent: d("10")}, Priority: 9},
	}, time.Now())
	if tot.Lines[0].Name != "high" {
		t.Fatalf("priority tie-break failed: %+v", tot.Lines[0])
	}
}

func TestWindowsAndDays(t *testing.T) {
	mon := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) // a Monday
	sun := mon.AddDate(0, 0, 6)
	monOnly := uint8(1 << 0) // bit0 = Monday
	past := mon.Add(-time.Hour)
	future := mon.AddDate(0, 0, 8) // still in the future on Sunday too

	promos := []Promotion{
		{Name: "monday only", Kind: KindPercent, Config: Config{Percent: d("10")}, DaysMask: &monOnly},
		{Name: "already ended", Kind: KindPercent, Config: Config{Percent: d("10")}, EndsAt: &past},
		{Name: "not started", Kind: KindPercent, Config: Config{Percent: d("10")}, StartsAt: &future},
	}
	line := []Line{{ProductID: 1, Qty: d("1"), UnitPrice: d("100")}}

	if got := Apply(line, promos, mon); !got.Discount.Equal(d("10")) {
		t.Fatalf("monday: only the monday promo should fire, got %s", got.Discount)
	}
	if got := Apply(line, promos, sun); !got.Discount.IsZero() {
		t.Fatalf("sunday: nothing should fire, got %s", got.Discount)
	}
}

func TestUntargetedCoversAll(t *testing.T) {
	tot := Apply(lines(), []Promotion{
		{Name: "sitewide 5%", Kind: KindPercent, Config: Config{Percent: d("5")}},
	}, time.Now())
	if len(tot.Lines) != 3 || !tot.Discount.Equal(d("35")) {
		t.Fatalf("sitewide should hit all 3 lines: %+v", tot)
	}
}

func TestAmountClampedAtPrice(t *testing.T) {
	tot := Apply([]Line{{ProductID: 1, Qty: d("2"), UnitPrice: d("30")}}, []Promotion{
		{Name: "50 off", Kind: KindAmount, Config: Config{Amount: d("50")}},
	}, time.Now())
	if !tot.Total.IsZero() {
		t.Fatalf("discount should clamp at line value, total %s", tot.Total)
	}
}
