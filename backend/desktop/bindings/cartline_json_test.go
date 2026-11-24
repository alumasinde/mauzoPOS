package bindings

import (
	"encoding/json"
	"testing"
)

// The checkout screen sends qty as a number and the discount as
// line_discount — this used to fail every payment with "bad cart".
func TestCartLineAcceptsScreenShape(t *testing.T) {
	var lines []CartLine
	in := `[{"product_id":10,"variant_id":0,"category_id":2,"name":"Sugar","sku":"S1","unit":"pc",
	        "qty":2,"price_override":"","line_discount":"20"},
	        {"product_id":11,"qty":"1.5","discount":"0"}]`
	if err := json.Unmarshal([]byte(in), &lines); err != nil {
		t.Fatal(err)
	}
	if lines[0].Qty != "2" || lines[0].Discount != "20" || lines[0].PriceOverride != "" {
		t.Fatalf("line 0 %+v", lines[0])
	}
	if lines[1].Qty != "1.5" || lines[1].Discount != "" {
		t.Fatalf("line 1 %+v", lines[1])
	}
}
