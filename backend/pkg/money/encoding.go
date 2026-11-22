package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// MarshalJSON encodes as a string ("110.00") so JavaScript never sees a float.
func (m Money) MarshalJSON() ([]byte, error) { return json.Marshal(m.d.String()) }

// UnmarshalJSON accepts "110.00" or 110.00.
func (m *Money) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*m = Zero
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("money: invalid JSON amount %s", b)
	}
	m.d = d
	return nil
}

// Value stores the amount as a DECIMAL string.
func (m Money) Value() (driver.Value, error) { return m.d.String(), nil }

// Scan reads DECIMAL columns (returned as []byte/string by the MySQL driver).
func (m *Money) Scan(src any) error {
	var d decimal.Decimal
	if err := d.Scan(src); err != nil {
		return fmt.Errorf("money: scan: %w", err)
	}
	m.d = d
	return nil
}

// Format renders a display string like "KSh 1,234.50". Display only — never parse it back.
func (m Money) Format(symbol string) string {
	s := m.Abs().String()
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	sign := ""
	if m.IsNegative() && !m.Round(HalfUp).IsZero() {
		sign = "-"
	}
	if symbol != "" {
		symbol += " "
	}
	return sign + symbol + b.String() + "." + frac
}
