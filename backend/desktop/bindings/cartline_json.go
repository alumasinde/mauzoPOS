package bindings

import (
	"encoding/json"
	"strconv"
	"strings"
)

// UnmarshalJSON accepts the cart exactly as the checkout screen stores it:
// qty as a number (2) or a string ("2"), and the cashier discount under
// either "discount" or "line_discount". Held carts saved by earlier
// versions keep working.
func (c *CartLine) UnmarshalJSON(data []byte) error {
	var raw struct {
		ProductID     uint64          `json:"product_id"`
		VariantID     uint64          `json:"variant_id"`
		CategoryID    uint64          `json:"category_id"`
		Name          string          `json:"name"`
		SKU           string          `json:"sku"`
		Unit          string          `json:"unit"`
		Qty           json.RawMessage `json:"qty"`
		PriceOverride json.RawMessage `json:"price_override"`
		Discount      json.RawMessage `json:"discount"`
		LineDiscount  json.RawMessage `json:"line_discount"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = CartLine{ProductID: raw.ProductID, VariantID: raw.VariantID, CategoryID: raw.CategoryID,
		Name: raw.Name, SKU: raw.SKU, Unit: raw.Unit}
	var err error
	if c.Qty, err = numText(raw.Qty); err != nil {
		return err
	}
	if c.PriceOverride, err = numText(raw.PriceOverride); err != nil {
		return err
	}
	if c.Discount, err = numText(raw.Discount); err != nil {
		return err
	}
	if c.Discount == "" {
		if c.Discount, err = numText(raw.LineDiscount); err != nil {
			return err
		}
	}
	if c.Discount == "0" {
		c.Discount = ""
	}
	return nil
}

// numText reads a JSON number or string (or null) as decimal text.
func numText(v json.RawMessage) (string, error) {
	s := strings.TrimSpace(string(v))
	if s == "" || s == "null" {
		return "", nil
	}
	if s[0] == '"' {
		var out string
		err := json.Unmarshal(v, &out)
		return strings.TrimSpace(out), err
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(f, 'f', -1, 64), nil
}
