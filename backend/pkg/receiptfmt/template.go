package receiptfmt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SchemaVersion is the template format this package writes. Older versions
// are upgraded by Parse; a newer version is rejected (update the POS).
const SchemaVersion = 1

// Template is a receipt layout: paper size plus an ordered list of blocks.
type Template struct {
	Version int `json:"version"`
	// Paper is the roll width in mm: 58 or 80.
	Paper int `json:"paper"`
	// Columns overrides the characters per line (0 = 32 for 58mm, 48 for
	// 80mm). Some 80mm printers use 42.
	Columns int     `json:"columns,omitempty"`
	Blocks  []Block `json:"blocks"`
}

// Block types.
const (
	BlockLogo       = "logo"        // tenant logo (raster)
	BlockText       = "text"        // free text with {{variables}}
	BlockDivider    = "divider"     // a full-width rule
	BlockFields     = "fields"      // label … value rows (receipt no, date, cashier)
	BlockItems      = "items"       // the sold lines
	BlockTotals     = "totals"      // subtotal, discount, VAT, TOTAL
	BlockPayments   = "payments"    // tendered methods, change
	BlockTaxSummary = "tax_summary" // VAT per rate
	BlockFiscal     = "fiscal"      // eTIMS stamp (hidden until the sale is signed)
	BlockQR         = "qr"          // QR code of Data
	BlockBarcode    = "barcode"     // Code 128 of Data (e.g. the receipt number)
	BlockFeed       = "feed"        // blank lines
)

// Block is one element of a template. Which fields apply depends on Type.
type Block struct {
	Type string `json:"type"`
	// If shows the block only when the condition holds: a flag
	// ("has.customer"), a variable that must be non-empty ("business.kra_pin"),
	// or either negated with "!".
	If string `json:"if,omitempty"`

	Text  string `json:"text,omitempty"`  // text; "\n" splits lines
	Align string `json:"align,omitempty"` // left | center | right
	Bold  bool   `json:"bold,omitempty"`
	Size  string `json:"size,omitempty"` // normal | tall | wide | double

	Char   string  `json:"char,omitempty"`   // divider character (default "-")
	Fields []Field `json:"fields,omitempty"` // fields block
	Lines  int     `json:"lines,omitempty"`  // feed

	Data string `json:"data,omitempty"` // qr / barcode payload, with {{variables}}
	// QRSize is the module size 1–16 (default 6).
	QRSize int `json:"qr_size,omitempty"`

	// Items block.
	Layout    string `json:"layout,omitempty"` // detailed (name, then qty × price) | compact (one line)
	ShowSKU   bool   `json:"show_sku,omitempty"`
	ShowPromo bool   `json:"show_promo,omitempty"`

	// Totals block.
	ShowItemCount bool `json:"show_item_count,omitempty"`
}

// Field is one label/value row; Value may use {{variables}} and the row is
// skipped when Value renders empty.
type Field struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Parse decodes and validates a template, upgrading older schema versions.
func Parse(raw []byte) (Template, error) {
	var t Template
	if len(raw) > maxTemplate {
		return t, fmt.Errorf("template: larger than %d KB", maxTemplate>>10)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, fmt.Errorf("template: %w", err)
	}
	if t.Version == 0 {
		t.Version = SchemaVersion
	}
	if t.Version > SchemaVersion {
		return t, fmt.Errorf("template: version %d is newer than this software supports (%d)", t.Version, SchemaVersion)
	}
	return t, t.Validate()
}

// MustParse parses a built-in template, panicking on error.
func MustParse(raw []byte) Template {
	t, err := Parse(raw)
	if err != nil {
		panic(err)
	}
	return t
}

// Width is the characters per line.
func (t Template) Width() int {
	if t.Columns > 0 {
		return t.Columns
	}
	if t.Paper == 58 {
		return 32
	}
	return 48
}

// Dots is the printable width in printer dots (203 dpi heads).
func (t Template) Dots() int {
	if t.Paper == 58 {
		return 384
	}
	return 576
}

var varRe = regexp.MustCompile(`\{\{\s*([a-z0-9_.]+)\s*\}\}`)

// ValidationError lists every problem found, by block number (1-based).
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "template: " + strings.Join(e.Problems, "; ")
}

// Limits.
const (
	maxBlocks   = 80
	maxText     = 600
	maxData     = 400
	maxFields   = 20
	maxFeed     = 10
	minColumns  = 24
	maxColumns  = 64
	maxTemplate = 64 << 10
)

// Validate checks a template's structure and variables.
func (t Template) Validate() error {
	var p []string
	if t.Paper != 58 && t.Paper != 80 {
		p = append(p, "paper must be 58 or 80")
	}
	if t.Columns != 0 && (t.Columns < minColumns || t.Columns > maxColumns) {
		p = append(p, fmt.Sprintf("columns must be between %d and %d", minColumns, maxColumns))
	}
	if len(t.Blocks) == 0 {
		p = append(p, "add at least one block")
	}
	if len(t.Blocks) > maxBlocks {
		p = append(p, fmt.Sprintf("at most %d blocks", maxBlocks))
	}
	for i, b := range t.Blocks {
		for _, msg := range b.problems() {
			p = append(p, fmt.Sprintf("block %d (%s): %s", i+1, b.Type, msg))
		}
	}
	if len(p) > 0 {
		return &ValidationError{Problems: p}
	}
	return nil
}

// problems lists what is wrong with one block.
func (b Block) problems() []string {
	var p []string
	add := func(f string, a ...any) { p = append(p, fmt.Sprintf(f, a...)) }
	vars := func(s string) {
		for _, m := range varRe.FindAllStringSubmatch(s, -1) {
			if !KnownVariable(m[1]) {
				add("unknown variable {{%s}}", m[1])
			}
		}
	}
	if b.If != "" && !KnownCondition(strings.TrimPrefix(b.If, "!")) {
		add("unknown condition %q", b.If)
	}
	if !oneOf(b.Align, "", "left", "center", "right") {
		add("align must be left, center or right")
	}
	if !oneOf(b.Size, "", "normal", "tall", "wide", "double") {
		add("size must be normal, tall, wide or double")
	}
	switch b.Type {
	case BlockLogo, BlockTotals, BlockPayments, BlockTaxSummary, BlockFiscal:
	case BlockText:
		if strings.TrimSpace(b.Text) == "" {
			add("text is empty")
		}
		if len(b.Text) > maxText {
			add("text longer than %d characters", maxText)
		}
		vars(b.Text)
	case BlockDivider:
		if len([]rune(b.Char)) > 1 {
			add("divider char must be one character")
		}
	case BlockFields:
		if len(b.Fields) == 0 || len(b.Fields) > maxFields {
			add("between 1 and %d fields", maxFields)
		}
		for _, f := range b.Fields {
			if len(f.Label) > 40 || len(f.Value) > 200 {
				add("field %q is too long", f.Label)
			}
			vars(f.Label)
			vars(f.Value)
		}
	case BlockItems:
		if !oneOf(b.Layout, "", "detailed", "compact") {
			add("layout must be detailed or compact")
		}
	case BlockQR, BlockBarcode:
		if strings.TrimSpace(b.Data) == "" {
			add("data is empty")
		}
		if len(b.Data) > maxData {
			add("data longer than %d characters", maxData)
		}
		if b.QRSize < 0 || b.QRSize > 16 {
			add("qr_size must be 1–16")
		}
		vars(b.Data)
	case BlockFeed:
		if b.Lines < 1 || b.Lines > maxFeed {
			add("lines must be 1–%d", maxFeed)
		}
	default:
		add("unknown block type")
	}
	return p
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// Variables documents every {{variable}} (for the designer's picker).
var Variables = []struct{ Name, Description string }{
	{"business.name", "Business name"},
	{"business.address", "Business address"},
	{"business.phone", "Business phone"},
	{"business.email", "Business email"},
	{"business.kra_pin", "KRA PIN"},
	{"branch.name", "Branch name"},
	{"branch.code", "Branch code"},
	{"branch.phone", "Branch phone"},
	{"branch.location", "Branch location"},
	{"sale.receipt_no", "Receipt number (offline: reference)"},
	{"sale.ref", "Sale reference"},
	{"sale.date", "Date (dd/mm/yyyy)"},
	{"sale.time", "Time (hh:mm)"},
	{"sale.datetime", "Date and time"},
	{"sale.cashier", "Cashier"},
	{"sale.terminal", "Till / terminal"},
	{"sale.customer", "Customer name"},
	{"sale.customer_phone", "Customer phone"},
	{"sale.customer_pin", "Customer KRA PIN"},
	{"sale.subtotal", "Subtotal"},
	{"sale.discount", "Discount"},
	{"sale.tax", "VAT"},
	{"sale.total", "Total"},
	{"sale.paid", "Amount paid"},
	{"sale.change", "Change"},
	{"sale.item_count", "Number of items"},
	{"sale.currency", "Currency code"},
	{"footer", "Footer text from Settings"},
	{"fiscal.invoice_no", "eTIMS invoice number"},
	{"fiscal.cu_serial", "eTIMS control unit serial"},
	{"fiscal.signature", "eTIMS signature"},
	{"fiscal.internal_data", "eTIMS internal data"},
	{"fiscal.qr_data", "eTIMS verification link"},
	{"fiscal.signed_at", "eTIMS signing time"},
	{"extra.mpesa_till", "M-Pesa till number (Settings)"},
	{"extra.mpesa_paybill", "M-Pesa paybill (Settings)"},
	{"extra.mpesa_account", "M-Pesa account number (Settings)"},
}

// Conditions documents the boolean flags usable in "if".
var Conditions = []struct{ Name, Description string }{
	{"has.customer", "A customer is attached"},
	{"has.discount", "The sale has a discount"},
	{"has.tax", "The sale has VAT"},
	{"has.change", "Change was given"},
	{"has.cash", "Paid (partly) in cash"},
	{"has.card", "Paid (partly) by card"},
	{"has.mpesa", "Paid (partly) by M-Pesa"},
	{"has.bank", "Paid (partly) by bank transfer"},
	{"has.credit", "Paid (partly) on account"},
	{"has.fiscal", "eTIMS-signed"},
	{"has.logo", "A logo is uploaded"},
	{"is.reprint", "This is a reprint"},
	{"is.offline", "Provisional (offline) receipt"},
	{"is.voided", "The sale was voided"},
	{"is.vat", "Business is VAT registered"},
}

var known = func() map[string]bool {
	m := map[string]bool{}
	for _, v := range Variables {
		m[v.Name] = true
	}
	return m
}()

var knownCond = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range Conditions {
		m[c.Name] = true
	}
	return m
}()

// KnownVariable reports whether name is a documented variable. Any
// extra.<key> is accepted so new receipt settings need no code change here.
func KnownVariable(name string) bool {
	return known[name] || (strings.HasPrefix(name, "extra.") && len(name) > len("extra."))
}

// KnownCondition reports whether name can appear in a block's "if" (flags and
// non-empty variables).
func KnownCondition(name string) bool {
	return knownCond[name] || KnownVariable(name)
}
