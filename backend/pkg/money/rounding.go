package money

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Mode is a rounding rule. Tenants/tax configs store the mode name so rounding
// is configurable, never hardcoded at call sites.
type Mode string

// Supported rounding modes.
const (
	HalfUp   Mode = "half_up"   // 0.125 → 0.13, -0.125 → -0.13 (away from zero)
	HalfEven Mode = "half_even" // banker's rounding: 0.125 → 0.12, 0.135 → 0.14
	Up       Mode = "up"        // away from zero
	Down     Mode = "down"      // toward zero (truncate)
)

// ParseMode parses a stored rounding mode name.
func ParseMode(s string) (Mode, error) {
	switch m := Mode(s); m {
	case HalfUp, HalfEven, Up, Down:
		return m, nil
	}
	return "", fmt.Errorf("money: unknown rounding mode %q", s)
}

// Round rounds to Scale (cents) using mode.
func (m Money) Round(mode Mode) Money { return m.RoundTo(Scale, mode) }

// RoundTo rounds to the given number of decimal places.
func (m Money) RoundTo(places int32, mode Mode) Money {
	return Money{d: roundDecimal(m.d, places, mode)}
}

// RoundToIncrement rounds to a cash increment such as 0.05, 0.50 or 1.00.
// Used for cash-tender rounding; the difference must be recorded on the sale.
func (m Money) RoundToIncrement(inc Money, mode Mode) Money {
	if inc.IsZero() {
		return m.Round(mode)
	}
	units := m.d.Div(inc.d)
	return Money{d: roundDecimal(units, 0, mode).Mul(inc.d)}.Round(mode)
}

func roundDecimal(d decimal.Decimal, places int32, mode Mode) decimal.Decimal {
	switch mode {
	case HalfEven:
		return d.RoundBank(places)
	case Up:
		if d.IsNegative() {
			return d.RoundFloor(places)
		}
		return d.RoundCeil(places)
	case Down:
		return d.Truncate(places)
	default: // HalfUp — shopspring's Round is half away from zero
		return d.Round(places)
	}
}
