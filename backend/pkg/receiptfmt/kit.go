package receiptfmt

import "encoding/json"

// Kit is everything a POS terminal needs to print without the server: the
// branch's layout, the receipt header, the logo and the print policy. The
// API builds it; the terminal stores it with its sync bundle.
type Kit struct {
	TemplateID      string          `json:"template_id,omitempty"` // empty = builtin
	TemplateVersion int             `json:"template_version,omitempty"`
	BuiltinKey      string          `json:"builtin_key,omitempty"`
	Template        json.RawMessage `json:"template"`

	Business         Business          `json:"business"`
	Branch           Branch            `json:"branch"`
	Footer           string            `json:"footer,omitempty"`
	Extra            map[string]string `json:"extra,omitempty"`
	PricesIncludeVAT bool              `json:"prices_include_vat,omitempty"`
	Logo             *Raster           `json:"logo,omitempty"`
	Policy           Policy            `json:"policy"`
}

// Policy is the tenant's printing preferences.
type Policy struct {
	AutoPrint  bool `json:"auto_print"`  // print when a sale completes
	Copies     int  `json:"copies"`      // 1–3
	OpenDrawer bool `json:"open_drawer"` // kick the drawer on cash sales
}

// Base returns a Doc pre-filled with the kit's header fields; the caller adds
// the sale, items and payments.
func (k Kit) Base() Doc {
	return Doc{
		Business: k.Business, Branch: k.Branch, Footer: k.Footer, Extra: k.Extra,
		Sale:  Sale{PricesIncludeVAT: k.PricesIncludeVAT},
		Flags: Flags{HasLogo: k.Logo != nil && k.Logo.Valid()},
	}
}

// ParsedTemplate parses the kit's layout, falling back to the supermarket
// builtin when it is missing or unreadable (a till must always print).
func (k Kit) ParsedTemplate() Template {
	if len(k.Template) > 0 {
		if t, err := Parse(k.Template); err == nil {
			return t
		}
	}
	return DefaultFor(k.BuiltinKey, 80).Template
}
