package money

import (
	"errors"

	"github.com/shopspring/decimal"
)

// Allocate splits m (rounded to cents) across weights so the parts always sum
// exactly to the rounded total. Leftover cents go to the parts with the
// largest remainders (ties: earliest index). Used for spreading an order
// discount across lines and for split-payment breakdowns.
func Allocate(m Money, weights []decimal.Decimal) ([]Money, error) {
	if len(weights) == 0 {
		return nil, errors.New("money: allocate needs at least one weight")
	}
	totalW := decimal.Zero
	for _, w := range weights {
		if w.IsNegative() {
			return nil, errors.New("money: allocate weights must be non-negative")
		}
		totalW = totalW.Add(w)
	}
	if totalW.IsZero() {
		return nil, errors.New("money: allocate weights sum to zero")
	}

	totalCents := m.Cents()
	sign := int64(1)
	if totalCents < 0 {
		sign, totalCents = -1, -totalCents
	}

	parts := make([]int64, len(weights))
	rema := make([]decimal.Decimal, len(weights))
	var used int64
	for i, w := range weights {
		exact := decimal.NewFromInt(totalCents).Mul(w).Div(totalW)
		parts[i] = exact.IntPart()
		rema[i] = exact.Sub(decimal.NewFromInt(parts[i]))
		used += parts[i]
	}
	for left := totalCents - used; left > 0; left-- {
		best := 0
		for i := 1; i < len(rema); i++ {
			if rema[i].GreaterThan(rema[best]) {
				best = i
			}
		}
		parts[best]++
		rema[best] = decimal.NewFromInt(-1) // consumed
	}

	out := make([]Money, len(parts))
	for i, c := range parts {
		out[i] = FromCents(sign * c)
	}
	return out, nil
}
