// Package catalog owns the tenant's sellable data: categories, brands, units
// of measure, products with variants/barcodes/components and images, plus the
// barcode lookup the POS uses at scan time. Stock on shelves is Phase 5;
// this module never stores quantities.
package catalog

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Product kinds.
const (
	KindStandard  = "standard"
	KindService   = "service"
	KindPackaged  = "packaged"
	KindComposite = "composite"
)

// Sell modes.
const (
	SellByUnit   = "unit"
	SellByWeight = "weight"
)

// Product statuses.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Unit kinds.
const (
	UnitKindCount  = "count"
	UnitKindMass   = "mass"
	UnitKindVolume = "volume"
	UnitKindLength = "length"
	UnitKindTime   = "time"
)

// Category is a node in the tenant's category tree; roots are departments.
type Category struct {
	ID        uint64    `db:"id"`
	TenantID  uint64    `db:"tenant_id"`
	PublicID  string    `db:"public_id"`
	ParentID  *uint64   `db:"parent_id"`
	Name      string    `db:"name"`
	Sort      int       `db:"sort"`
	CreatedAt time.Time `db:"created_at"`
	// ProductCount is populated by ListCategories only.
	ProductCount int `db:"product_count"`
}

// Brand is a flat product brand label.
type Brand struct {
	ID        uint64    `db:"id"`
	TenantID  uint64    `db:"tenant_id"`
	PublicID  string    `db:"public_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	// ProductCount is populated by ListBrands only.
	ProductCount int `db:"product_count"`
}

// Unit is a unit of measure. TenantID nil marks a shared built-in.
// Factor-to-base converts inside a Kind: qty*Factor is the base-unit amount.
type Unit struct {
	ID         uint64           `db:"id"`
	TenantID   *uint64          `db:"tenant_id"`
	Code       string           `db:"code"`
	Name       string           `db:"name"`
	Kind       string           `db:"kind"`
	BaseUnitID *uint64          `db:"base_unit_id"`
	Factor     *decimal.Decimal `db:"factor"`
}

// Product is one sellable thing (or a service).
type Product struct {
	ID          uint64           `db:"id"`
	TenantID    uint64           `db:"tenant_id"`
	PublicID    string           `db:"public_id"`
	SKU         string           `db:"sku"`
	Name        string           `db:"name"`
	Description *string          `db:"description"`
	Kind        string           `db:"kind"`
	CategoryID  *uint64          `db:"category_id"`
	BrandID     *uint64          `db:"brand_id"`
	UnitID      *uint64          `db:"unit_id"`
	SellBy      string           `db:"sell_by"`
	Price       decimal.Decimal  `db:"price"`
	Cost        *decimal.Decimal `db:"cost"`
	VatRate     *decimal.Decimal `db:"vat_rate"`
	Deposit     *decimal.Decimal `db:"deposit"`
	TrackStock  bool             `db:"track_stock"`
	Attributes  *json.RawMessage `db:"attributes"`
	Status      string           `db:"status"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`

	// Hydrated on single-product reads only.
	Variants   []Variant   `db:"-"`
	Barcodes   []Barcode   `db:"-"`
	Components []Component `db:"-"`
	Images     []Image     `db:"-"`

	// Resolved references for responses (list join or hydration).
	CategoryPublicID *string `db:"-"`
	BrandPublicID    *string `db:"-"`
	UnitCode         *string `db:"-"`
}

// Variant is a purchasable option of a product (size, colour, pack).
type Variant struct {
	ID         uint64           `db:"id"`
	TenantID   uint64           `db:"tenant_id"`
	ProductID  uint64           `db:"product_id"`
	PublicID   string           `db:"public_id"`
	SKU        *string          `db:"sku"`
	Name       string           `db:"name"`
	Price      *decimal.Decimal `db:"price"`
	Attributes *json.RawMessage `db:"attributes"`
}

// Barcode is one scannable code; PackQty>1 sells multiple units per scan.
type Barcode struct {
	ID        uint64          `db:"id"`
	TenantID  uint64          `db:"tenant_id"`
	ProductID uint64          `db:"product_id"`
	VariantID *uint64         `db:"variant_id"`
	Code      string          `db:"barcode"`
	PackQty   decimal.Decimal `db:"pack_qty"`
}

// Component is one ingredient row of a composite product.
type Component struct {
	ProductID   uint64          `db:"product_id"`
	ComponentID uint64          `db:"component_id"`
	Qty         decimal.Decimal `db:"qty"`

	// Hydrated for responses.
	ComponentPublicID string `db:"-"`
	ComponentSKU      string `db:"-"`
	ComponentName     string `db:"-"`
}

// Image is a stored product photo; Path is an ImageStore key, not a URL.
type Image struct {
	ID        uint64 `db:"id"`
	ProductID uint64 `db:"product_id"`
	Path      string `db:"path"`
	Sort      int    `db:"sort"`
}
