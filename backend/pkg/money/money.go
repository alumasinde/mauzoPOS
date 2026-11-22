// Package money provides decimal-safe monetary amounts. It is pure Go with no
// app dependencies so both the API and the offline POS (desktop) use the exact
// same arithmetic and rounding. Never use float64 for money.
package money

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Scale is the number of minor-unit digits stored for amounts (KES cents).
const Scale int32 = 2

// Money is an immutable amount. Intermediate results (e.g. qty × unit price)
// keep full precision until Round is called; values entering storage or
// receipts must be rounded explicitly with a documented Mode.
type Money struct {
	d decimal.Decimal
}

// Zero is the zero amount.
var Zero = Money{}

// New wraps a decimal amount.
func New(d decimal.Decimal) Money { return Money{d: d} }

// FromCents builds an amount from integer minor units (12345 → 123.45).
func FromCents(cents int64) Money { return Money{d: decimal.New(cents, -Scale)} }

// Parse parses a decimal string such as "110.00" or "-3.5".
func Parse(s string) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Zero, fmt.Errorf("money: invalid amount %q: %w", s, err)
	}
	return Money{d: d}, nil
}

// MustParse is for constants and tests only.
func MustParse(s string) Money {
	m, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return m
}

// Decimal returns the underlying decimal value.
func (m Money) Decimal() decimal.Decimal { return m.d }

// Add returns m + o.
func (m Money) Add(o Money) Money { return Money{d: m.d.Add(o.d)} }

// Sub returns m - o.
func (m Money) Sub(o Money) Money { return Money{d: m.d.Sub(o.d)} }

// Neg returns -m.
func (m Money) Neg() Money { return Money{d: m.d.Neg()} }

// Abs returns |m|.
func (m Money) Abs() Money { return Money{d: m.d.Abs()} }

// Mul multiplies by a quantity, weight or rate. The result is NOT rounded.
func (m Money) Mul(f decimal.Decimal) Money { return Money{d: m.d.Mul(f)} }

// Div divides by f with 16 digits of precision. The result is NOT rounded.
func (m Money) Div(f decimal.Decimal) Money { return Money{d: m.d.DivRound(f, 16)} }

// Cmp returns -1, 0 or 1 comparing m to o.
func (m Money) Cmp(o Money) int { return m.d.Cmp(o.d) }

// Equal reports whether m equals o.
func (m Money) Equal(o Money) bool { return m.d.Equal(o.d) }

// GreaterThan reports whether m > o.
func (m Money) GreaterThan(o Money) bool { return m.d.GreaterThan(o.d) }

// LessThan reports whether m < o.
func (m Money) LessThan(o Money) bool { return m.d.LessThan(o.d) }

// IsZero reports whether m is zero.
func (m Money) IsZero() bool { return m.d.IsZero() }

// IsNegative reports whether m < 0.
func (m Money) IsNegative() bool { return m.d.IsNegative() }

// IsPositive reports whether m > 0.
func (m Money) IsPositive() bool { return m.d.IsPositive() }

// Cents returns the amount in minor units, rounding half-up first.
func (m Money) Cents() int64 { return m.Round(HalfUp).d.Shift(Scale).IntPart() }

// String returns a fixed two-decimal representation, e.g. "750.34".
func (m Money) String() string { return m.d.StringFixed(Scale) }

// Sum adds amounts.
func Sum(ms ...Money) Money {
	total := Zero
	for _, m := range ms {
		total = total.Add(m)
	}
	return total
}
