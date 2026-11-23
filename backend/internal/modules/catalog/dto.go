package catalog

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// CategoryResponse is the API shape of a category.
type CategoryResponse struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id"`
	Name      string    `json:"name"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

// CategoryRequest creates or updates a category. ParentID is a public id.
type CategoryRequest struct {
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
	Sort     int     `json:"sort"`
}

// BrandResponse is the API shape of a brand.
type BrandResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BrandRequest creates or renames a brand.
type BrandRequest struct {
	Name string `json:"name"`
}

// UnitResponse is the API shape of a unit of measure.
type UnitResponse struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	BaseCode *string `json:"base_code"`
	Factor   *string `json:"factor"` // qty × factor = qty in base unit
	BuiltIn  bool    `json:"built_in"`
}

// UnitRequest creates a tenant unit. For non-base units BaseCode selects the
// kind's base unit and Factor is the conversion multiplier.
type UnitRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	BaseCode string `json:"base_code"` // "" marks the unit as its kind's base
	Factor   string `json:"factor"`    // decimal string; "" → 1 for base units
}

// VariantRequest is one variant inside a product write.
type VariantRequest struct {
	SKU        string            `json:"sku"`
	Name       string            `json:"name"`
	Price      *string           `json:"price"`
	Attributes map[string]string `json:"attributes"`
}

// BarcodeRequest is one barcode inside a product write.
type BarcodeRequest struct {
	Code      string  `json:"code"`
	VariantID *string `json:"variant_id"` // variant public id, or variant name during create
	PackQty   *string `json:"pack_qty"`   // decimal string, default "1"
}

// ComponentRequest is one composite ingredient inside a product write.
type ComponentRequest struct {
	ProductID string `json:"product_id"` // public id of the component product
	Qty       string `json:"qty"`
}

// ProductRequest creates or replaces a product. Variants/barcodes/components
// are fully replaced on update.
type ProductRequest struct {
	SKU         string             `json:"sku"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Kind        string             `json:"kind"`
	CategoryID  *string            `json:"category_id"` // public id
	BrandID     *string            `json:"brand_id"`    // public id
	UnitCode    string             `json:"unit_code"`
	SellBy      string             `json:"sell_by"`
	Price       string             `json:"price"`
	Cost        string             `json:"cost"`
	VatRate     string             `json:"vat_rate"`
	Deposit     string             `json:"deposit"`
	TrackStock  *bool              `json:"track_stock"`
	Attributes  map[string]string  `json:"attributes"`
	Variants    []VariantRequest   `json:"variants"`
	Barcodes    []BarcodeRequest   `json:"barcodes"`
	Components  []ComponentRequest `json:"components"`
}

// VariantResponse is the API shape of a variant.
type VariantResponse struct {
	ID         string            `json:"id"`
	SKU        *string           `json:"sku"`
	Name       string            `json:"name"`
	Price      *string           `json:"price"`
	Attributes map[string]string `json:"attributes"`
}

// BarcodeResponse is the API shape of a barcode assignment.
type BarcodeResponse struct {
	Code      string  `json:"code"`
	VariantID *string `json:"variant_id"`
	PackQty   string  `json:"pack_qty"`
}

// ComponentResponse is the API shape of a composite ingredient.
type ComponentResponse struct {
	ProductID string `json:"product_id"` // public id
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Qty       string `json:"qty"`
}

// ImageResponse is the API shape of a product image.
type ImageResponse struct {
	ID   uint64 `json:"id"`
	URL  string `json:"url"`
	Sort int    `json:"sort"`
}

// ProductResponse is a product in list context (no child rows).
type ProductResponse struct {
	ID         string            `json:"id"`
	SKU        string            `json:"sku"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	CategoryID *string           `json:"category_id"`
	BrandID    *string           `json:"brand_id"`
	UnitCode   *string           `json:"unit_code"`
	SellBy     string            `json:"sell_by"`
	Price      string            `json:"price"`
	Cost       *string           `json:"cost"`
	VatRate    *string           `json:"vat_rate"`
	Deposit    *string           `json:"deposit"`
	TrackStock bool              `json:"track_stock"`
	Attributes map[string]string `json:"attributes"`
	Status     string            `json:"status"`
	// Display-only names resolved for list responses.
	CategoryName *string   `json:"category_name"`
	BrandName    *string   `json:"brand_name"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProductDetail extends ProductResponse with child rows (single fetch).
type ProductDetail struct {
	ProductResponse
	Variants   []VariantResponse   `json:"variants"`
	Barcodes   []BarcodeResponse   `json:"barcodes"`
	Components []ComponentResponse `json:"components"`
	Images     []ImageResponse     `json:"images"`
}

// LookupResult answers a POS barcode scan: which product/variant and how many.
type LookupResult struct {
	Source   string           `json:"source"` // barcode | scale | sku
	Product  ProductDetail    `json:"product"`
	Variant  *VariantResponse `json:"variant"`
	Quantity string           `json:"quantity"` // units (pack size or measured weight)
	Price    string           `json:"price"`    // resolved unit price for this scan
	Scale    *ScaleInfo       `json:"scale,omitempty"`
}

// ScaleInfo carries the decoded scale-label fields when Source is "scale".
type ScaleInfo struct {
	ItemCode string `json:"item_code"`
	Embedded string `json:"embedded"` // price | weight
	Value    string `json:"value"`
}

// ImportReport summarizes a CSV import.
type ImportReport struct {
	Created int        `json:"created"`
	Updated int        `json:"updated"`
	Errors  []RowError `json:"errors"`
}

// RowError is one failed import row.
type RowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// LabelResponse is one printable shelf label with its Code 128 pattern.
type LabelResponse struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Price     string `json:"price"`
	Barcode   string `json:"barcode"`
	Pattern   []bool `json:"pattern"`
}

// helper: decimal string pointers
func decString(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

func attrsMap(raw *json.RawMessage) map[string]string {
	if raw == nil {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(*raw, &m); err != nil {
		return nil
	}
	return m
}
