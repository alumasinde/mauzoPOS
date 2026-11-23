package barcode

import "github.com/shopspring/decimal"

// EmbeddedKind selects what a scale label's value field encodes.
type EmbeddedKind string

// Value fields a weighted barcode can carry.
const (
	EmbeddedPrice  EmbeddedKind = "price"  // value = price in minor units (÷100)
	EmbeddedWeight EmbeddedKind = "weight" // value = weight in grams (÷1000 kg)
)

// ScaleConfig describes a tenant's scale-label format: a GS1 variable-measure
// EAN-13 of the form PP IIIII VVVVV C, where PP is a configured prefix, IIIII
// the item code (matched to product SKU or barcode), VVVVV the embedded price
// or weight, and C the standard check digit.
type ScaleConfig struct {
	Prefixes []string // e.g. {"20", "21"}; 20–29 are reserved in-store prefixes
	Embedded EmbeddedKind
}

// Scale is a decoded scale label.
type Scale struct {
	Item  string          `json:"item"`  // 5-digit item/PLU code
	Value decimal.Decimal `json:"value"` // price in KES or weight in kg
}

// DefaultScale is the common Kenyan scale configuration: prefixes 20–29 with a
// price field. Tenants can override via module config later.
func DefaultScale() ScaleConfig {
	return ScaleConfig{Prefixes: []string{"20", "21", "22", "23", "24", "25", "26", "27", "28", "29"},
		Embedded: EmbeddedPrice}
}

// ParseScale decodes code as a scale label per cfg. It returns ok=false when
// the code is not a valid, correctly-prefixed scale barcode, so callers can
// fall through to a normal barcode lookup.
func ParseScale(code string, cfg ScaleConfig) (Scale, bool) {
	if Classify(code) != TypeEAN13 || !Valid(code) {
		return Scale{}, false
	}
	prefix := code[:2]
	found := false
	for _, p := range cfg.Prefixes {
		if p == prefix {
			found = true
			break
		}
	}
	if !found {
		return Scale{}, false
	}
	raw, err := decimal.NewFromString(code[7:12])
	if err != nil {
		return Scale{}, false
	}
	divisor := int32(-2) // price: minor units → KES
	if cfg.Embedded == EmbeddedWeight {
		divisor = -3 // grams → kg
	}
	return Scale{Item: code[2:7], Value: raw.Shift(divisor)}, true
}

// IsScalePrefix reports whether code would be treated as a scale label under
// cfg — used to reserve those prefixes so real products cannot collide.
func IsScalePrefix(code string, cfg ScaleConfig) bool {
	if len(code) < 2 {
		return false
	}
	for _, p := range cfg.Prefixes {
		if code[:2] == p && len(code) == 13 {
			return true
		}
	}
	return false
}
