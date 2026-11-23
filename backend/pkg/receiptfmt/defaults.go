package receiptfmt

import (
	"embed"
	"time"
)

//go:embed defaults/*.json
var defaultFS embed.FS

// Builtin is a shipped starting layout.
type Builtin struct {
	Key         string
	Name        string
	Description string
	Raw         []byte
	Template    Template
}

// Builtins are the default layouts, in display order. Every tenant can copy
// one into the designer; a branch with no assigned template prints the
// builtin chosen by its business profile.
var Builtins = func() []Builtin {
	defs := []Builtin{
		{Key: "supermarket", Name: "Supermarket (80 mm)",
			Description: "Logo, full item lines, VAT summary and a scannable receipt barcode."},
		{Key: "minimart", Name: "Mini-mart (58 mm, M-Pesa)",
			Description: "Compact one-line items and your Lipa na M-Pesa till or paybill."},
		{Key: "restaurant", Name: "Restaurant (80 mm, tax invoice)",
			Description: "Tax-invoice heading, customer PIN and the eTIMS block once signed."},
	}
	for i := range defs {
		raw, err := defaultFS.ReadFile("defaults/" + defs[i].Key + ".json")
		if err != nil {
			panic(err)
		}
		defs[i].Raw = raw
		defs[i].Template = MustParse(raw)
	}
	return defs
}()

// BuiltinByKey returns a shipped layout.
func BuiltinByKey(key string) (Builtin, bool) {
	for _, b := range Builtins {
		if b.Key == key {
			return b, true
		}
	}
	return Builtin{}, false
}

// DefaultFor picks the builtin for a business profile's receipt_template
// code, falling back on paper width (58 mm → minimart, else supermarket).
func DefaultFor(profileKey string, paper int) Builtin {
	if b, ok := BuiltinByKey(profileKey); ok {
		return b
	}
	if paper == 58 {
		b, _ := BuiltinByKey("minimart")
		return b
	}
	b, _ := BuiltinByKey("supermarket")
	return b
}

// SampleDoc is realistic preview data for the designer. It is flagged as a
// sample so it can never be mistaken for a real receipt.
func SampleDoc() Doc {
	r16 := "16"
	r0 := "0"
	return Doc{
		Business: Business{Name: "Umoja Supermarket", Address: "Moi Avenue, Nairobi", Phone: "0712 345 678",
			KRAPin: "P051234567X", VATRegistered: true},
		Branch: Branch{Name: "Main Store", Code: "MAIN", Phone: "0712 345 678", Location: "Moi Avenue, Nairobi"},
		Sale: Sale{
			ReceiptNo: "1047", Ref: "01JSAMPLE00000000000001047",
			SoldAt:  time.Date(2026, 11, 23, 16, 32, 0, 0, Location),
			Cashier: "Mary W.", Terminal: "Till 1", Customer: "John Kamau", CustomerPhone: "0722 000 111",
			Currency: "KES", Subtotal: "1290.00", Discount: "40.00", DiscountNote: "Loyalty", Tax: "115.87",
			Total: "1250.00", Paid: "1250.00", Change: "50.00", PricesIncludeVAT: true,
		},
		Items: []Item{
			{Name: "Kabras Sugar 2kg", SKU: "SUG-2KG", Qty: "2", UnitPrice: "320.00", Total: "640.00", TaxRate: &r16, Tax: "88.28"},
			{Name: "Brookside Milk 500ml", SKU: "MLK-500", Qty: "3", UnitPrice: "65.00", Total: "195.00", TaxRate: &r0, Tax: "0.00",
				Promotion: "3 for 195"},
			{Name: "Jogoo Maize Flour 2kg", SKU: "UNG-2KG", Qty: "1", UnitPrice: "215.00", Total: "215.00", TaxRate: &r0, Tax: "0.00"},
			{Name: "Fresh Fri Cooking Oil 1L", SKU: "OIL-1L", Qty: "1", UnitPrice: "240.00", Discount: "40.00", Total: "200.00",
				TaxRate: &r16, Tax: "27.59"},
		},
		Payments: []Payment{
			{Method: "mpesa", Amount: "800.00", Reference: "SKL4H7Q2XZ"},
			{Method: "cash", Amount: "450.00", Tendered: "500.00", Change: "50.00"},
		},
		Footer: "Prices include VAT where applicable.",
		Extra:  map[string]string{"mpesa_till": "123456"},
		Flags:  Flags{Sample: true},
	}
}
