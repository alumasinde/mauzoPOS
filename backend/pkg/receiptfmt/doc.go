// Package receiptfmt turns a receipt template (versioned JSON) and the data of
// one sale into a printable layout, and that layout into printer bytes.
//
// The same code runs on the API (previews, issued-receipt snapshots, back-office
// reprints, A4 invoices) and inside the Windows POS (local printing, including
// provisional receipts while offline), so a receipt looks identical wherever it
// is produced. It has no database or network dependencies.
//
//	template + Doc ──Render──▶ Output (lines) ──EncodeESCPOS──▶ bytes for the printer
//	                                        └──Text / JSON────▶ previews, snapshots
package receiptfmt

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Doc is everything a receipt can show. Money and quantities are decimal
// strings ("1200.00", "1.5"); the renderer formats them.
type Doc struct {
	Business Business  `json:"business"`
	Branch   Branch    `json:"branch"`
	Sale     Sale      `json:"sale"`
	Items    []Item    `json:"items"`
	Payments []Payment `json:"payments"`
	// Taxes is the VAT breakdown; computed from Items when empty.
	Taxes  []TaxLine `json:"taxes,omitempty"`
	Fiscal *Fiscal   `json:"fiscal,omitempty"`
	// Footer is the tenant's receipt.footer setting.
	Footer string `json:"footer,omitempty"`
	// Extra holds tenant receipt settings exposed as {{extra.<key>}}, e.g.
	// mpesa_till, mpesa_paybill, mpesa_account.
	Extra map[string]string `json:"extra,omitempty"`
	Flags Flags             `json:"flags"`
}

// Business is the tenant header.
type Business struct {
	Name          string `json:"name"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	KRAPin        string `json:"kra_pin,omitempty"`
	VATRegistered bool   `json:"vat_registered,omitempty"`
}

// Branch is where the sale happened.
type Branch struct {
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Location string `json:"location,omitempty"`
}

// Sale is the transaction header.
type Sale struct {
	// ReceiptNo is the server-issued number; empty on a provisional (offline)
	// receipt, which shows Ref instead.
	ReceiptNo        string    `json:"receipt_no,omitempty"`
	Ref              string    `json:"ref,omitempty"` // sale public id / idempotency key
	SoldAt           time.Time `json:"sold_at"`
	Cashier          string    `json:"cashier,omitempty"`
	Terminal         string    `json:"terminal,omitempty"`
	Customer         string    `json:"customer,omitempty"`
	CustomerPhone    string    `json:"customer_phone,omitempty"`
	CustomerPin      string    `json:"customer_pin,omitempty"` // buyer KRA PIN (Phase 10)
	Currency         string    `json:"currency,omitempty"`
	Subtotal         string    `json:"subtotal"`
	Discount         string    `json:"discount,omitempty"`
	DiscountNote     string    `json:"discount_note,omitempty"`
	Tax              string    `json:"tax,omitempty"`
	Total            string    `json:"total"`
	Paid             string    `json:"paid,omitempty"`
	Change           string    `json:"change,omitempty"`
	PricesIncludeVAT bool      `json:"prices_include_vat,omitempty"`
}

// Item is one sold line.
type Item struct {
	Name      string  `json:"name"`
	SKU       string  `json:"sku,omitempty"`
	Qty       string  `json:"qty"`
	UnitPrice string  `json:"unit_price"`
	Discount  string  `json:"discount,omitempty"`
	Total     string  `json:"total"`
	TaxRate   *string `json:"tax_rate,omitempty"` // percent, nil = no VAT info
	Tax       string  `json:"tax,omitempty"`
	Promotion string  `json:"promotion,omitempty"`
}

// Payment is one tendered payment.
type Payment struct {
	Method    string `json:"method"` // cash | card | mpesa | bank | credit | …
	Amount    string `json:"amount"`
	Tendered  string `json:"tendered,omitempty"`
	Change    string `json:"change,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// TaxLine is one VAT rate's totals.
type TaxLine struct {
	Rate    string `json:"rate"` // percent, e.g. "16"
	Taxable string `json:"taxable"`
	Tax     string `json:"tax"`
}

// Fiscal is the KRA eTIMS stamp (Phase 10). Nil until the sale is signed.
type Fiscal struct {
	InvoiceNo string `json:"invoice_no,omitempty"`
	CUSerial  string `json:"cu_serial,omitempty"`
	Signature string `json:"signature,omitempty"`
	Internal  string `json:"internal_data,omitempty"`
	QRData    string `json:"qr_data,omitempty"`
	SignedAt  string `json:"signed_at,omitempty"`
}

// Flags mark the copy. The renderer prints them as banners the template
// cannot remove, so a reprint or provisional receipt is always recognisable.
type Flags struct {
	Reprint bool `json:"reprint,omitempty"`
	CopyNo  int  `json:"copy_no,omitempty"`
	Offline bool `json:"offline,omitempty"` // provisional: not yet accepted by the server
	Voided  bool `json:"voided,omitempty"`
	Sample  bool `json:"sample,omitempty"` // designer preview data
	HasLogo bool `json:"has_logo,omitempty"`
}

// Location is used for receipt dates (EAT).
var Location = func() *time.Location {
	if l, err := time.LoadLocation("Africa/Nairobi"); err == nil {
		return l
	}
	return time.FixedZone("EAT", 3*60*60)
}()

// MethodLabel is the printed name of a payment method code.
func MethodLabel(code string) string {
	switch code {
	case "cash":
		return "Cash"
	case "card":
		return "Card"
	case "mpesa":
		return "M-Pesa"
	case "bank":
		return "Bank transfer"
	case "credit":
		return "On account"
	}
	if code == "" {
		return "Payment"
	}
	return strings.ToUpper(code[:1]) + code[1:]
}

// vars builds the {{name}} substitution table. Every key is documented in
// Variables so the designer can list them.
func (d Doc) vars() map[string]string {
	soldAt := d.Sale.SoldAt
	if soldAt.IsZero() {
		soldAt = time.Now()
	}
	soldAt = soldAt.In(Location)
	itemCount := decimal.Zero
	for _, it := range d.Items {
		itemCount = itemCount.Add(dec(it.Qty))
	}
	receiptNo := d.Sale.ReceiptNo
	if receiptNo == "" {
		receiptNo = shortRef(d.Sale.Ref)
	}
	v := map[string]string{
		"business.name":       d.Business.Name,
		"business.address":    d.Business.Address,
		"business.phone":      d.Business.Phone,
		"business.email":      d.Business.Email,
		"business.kra_pin":    d.Business.KRAPin,
		"branch.name":         d.Branch.Name,
		"branch.code":         d.Branch.Code,
		"branch.phone":        d.Branch.Phone,
		"branch.location":     d.Branch.Location,
		"sale.receipt_no":     receiptNo,
		"sale.ref":            shortRef(d.Sale.Ref),
		"sale.date":           soldAt.Format("02/01/2006"),
		"sale.time":           soldAt.Format("15:04"),
		"sale.datetime":       soldAt.Format("02/01/2006 15:04"),
		"sale.cashier":        d.Sale.Cashier,
		"sale.terminal":       d.Sale.Terminal,
		"sale.customer":       d.Sale.Customer,
		"sale.customer_phone": d.Sale.CustomerPhone,
		"sale.customer_pin":   d.Sale.CustomerPin,
		"sale.subtotal":       Money(d.Sale.Subtotal),
		"sale.discount":       Money(d.Sale.Discount),
		"sale.tax":            Money(d.Sale.Tax),
		"sale.total":          Money(d.Sale.Total),
		"sale.paid":           Money(d.Sale.Paid),
		"sale.change":         Money(d.Sale.Change),
		"sale.item_count":     Qty(itemCount.String()),
		"sale.currency":       or(d.Sale.Currency, "KES"),
		"footer":              d.Footer,
	}
	if d.Fiscal != nil {
		v["fiscal.invoice_no"] = d.Fiscal.InvoiceNo
		v["fiscal.cu_serial"] = d.Fiscal.CUSerial
		v["fiscal.signature"] = d.Fiscal.Signature
		v["fiscal.internal_data"] = d.Fiscal.Internal
		v["fiscal.qr_data"] = d.Fiscal.QRData
		v["fiscal.signed_at"] = d.Fiscal.SignedAt
	}
	for k, val := range d.Extra {
		v["extra."+k] = val
	}
	return v
}

// flags are boolean conditions usable in a block's "if".
func (d Doc) flags() map[string]bool {
	f := map[string]bool{
		"has.customer": d.Sale.Customer != "",
		"has.discount": dec(d.Sale.Discount).IsPositive(),
		"has.tax":      dec(d.Sale.Tax).IsPositive(),
		"has.change":   dec(d.Sale.Change).IsPositive(),
		"has.fiscal":   d.Fiscal != nil,
		"has.logo":     d.Flags.HasLogo,
		"is.reprint":   d.Flags.Reprint,
		"is.offline":   d.Flags.Offline,
		"is.voided":    d.Flags.Voided,
		"is.vat":       d.Business.VATRegistered,
	}
	for _, p := range d.Payments {
		f["has."+p.Method] = true
	}
	return f
}

// shortRef keeps the tail of a long public id — enough to find the sale.
func shortRef(ref string) string {
	if len(ref) > 10 {
		return ref[len(ref)-10:]
	}
	return ref
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return d
}
