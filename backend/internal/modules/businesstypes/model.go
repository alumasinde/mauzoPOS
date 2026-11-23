// Package businesstypes implements business types and profile templates — the
// data side of the Business Profile Engine — plus the tenant_modules store.
package businesstypes

import (
	"encoding/json"
	"time"
)

// BusinessType is one row of business_types (supermarket, butchery, ...).
type BusinessType struct {
	ID   uint64 `db:"id"`
	Code string `db:"code"`
	Name string `db:"name"`
	Sort int    `db:"sort"`
}

// ProfileDefinition is the JSON payload a profile seeds onto a tenant.
type ProfileDefinition struct {
	Modules         []string                   `json:"modules"`
	Categories      []string                   `json:"categories"`
	Settings        map[string]json.RawMessage `json:"settings"`
	ReceiptTemplate string                     `json:"receipt_template"`
	PosLayout       map[string]any             `json:"pos_layout"`
}

// Profile is one profile template under a business type.
type Profile struct {
	ID             uint64            `db:"id"`
	PublicID       string            `db:"public_id"`
	BusinessTypeID uint64            `db:"business_type_id"`
	Name           string            `db:"name"`
	IsDefault      bool              `db:"is_default"`
	DefinitionJSON json.RawMessage   `db:"definition"`
	Definition     ProfileDefinition `db:"-"`
	TypeCode       string            `db:"type_code"` // joined
	CreatedAt      time.Time         `db:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"`
}
