package receipts

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"mauzo/internal/modules/sales"
	"mauzo/pkg/receiptfmt"
)

// Settings read for receipts (all optional).
var receiptSettingKeys = []string{
	"receipt.footer", "receipt.paper_width", "profile.receipt_template",
	"receipt.mpesa_till", "receipt.mpesa_paybill", "receipt.mpesa_account",
	"receipt.auto_print", "receipt.copies", "receipt.open_drawer",
}

// settingsView is the decoded receipt settings.
type settingsView struct {
	footer          string
	paperWidth      int
	profileTemplate string
	extra           map[string]string
	policy          receiptfmt.Policy
}

func (s *Service) settings(ctx context.Context, tenantID uint64) (settingsView, error) {
	raw, err := s.repo.Settings(ctx, tenantID, receiptSettingKeys...)
	if err != nil {
		return settingsView{}, err
	}
	str := func(k string) string {
		var v string
		_ = json.Unmarshal(raw[k], &v)
		return strings.TrimSpace(v)
	}
	num := func(k string) int {
		var v float64
		_ = json.Unmarshal(raw[k], &v)
		return int(v)
	}
	boolean := func(k string, def bool) bool {
		if _, ok := raw[k]; !ok {
			return def
		}
		var v bool
		_ = json.Unmarshal(raw[k], &v)
		return v
	}
	v := settingsView{
		footer: str("receipt.footer"), paperWidth: num("receipt.paper_width"),
		profileTemplate: str("profile.receipt_template"), extra: map[string]string{},
		policy: receiptfmt.Policy{
			AutoPrint: boolean("receipt.auto_print", true), Copies: num("receipt.copies"),
			OpenDrawer: boolean("receipt.open_drawer", true),
		},
	}
	if v.policy.Copies < 1 || v.policy.Copies > 3 {
		v.policy.Copies = 1
	}
	for _, k := range []string{"mpesa_till", "mpesa_paybill", "mpesa_account"} {
		if val := str("receipt." + k); val != "" {
			v.extra[k] = val
		}
	}
	return v, nil
}

// baseDoc is the header part of a receipt for a branch.
func (s *Service) baseDoc(ctx context.Context, tenantID, branchID uint64, set settingsView) (receiptfmt.Doc, *receiptfmt.Raster, error) {
	h, err := s.repo.Header(ctx, tenantID, branchID)
	if err != nil {
		return receiptfmt.Doc{}, nil, err
	}
	logo, err := s.Logo(ctx, tenantID)
	if err != nil {
		return receiptfmt.Doc{}, nil, err
	}
	d := receiptfmt.Doc{
		Business: receiptfmt.Business{Name: h.BusinessName, Address: h.Address, Phone: h.Phone,
			Email: h.Email, KRAPin: h.KRAPin, VATRegistered: h.VATRegistered},
		Branch: receiptfmt.Branch{Name: h.BranchName, Code: h.BranchCode,
			Phone: firstNonEmpty(h.BranchPhone, h.Phone), Location: firstNonEmpty(h.Location, h.Address)},
		Footer: set.footer, Extra: set.extra,
		Flags: receiptfmt.Flags{HasLogo: logo != nil},
	}
	return d, logo, nil
}

// fillSale copies a completed sale into the receipt document.
func fillSale(d *receiptfmt.Doc, sd sales.SaleDetail) {
	sale := sd.Sale
	d.Sale = receiptfmt.Sale{
		ReceiptNo: strconv.FormatUint(sale.ReceiptNo, 10), Ref: sale.PublicID, SoldAt: sale.SoldAt,
		Cashier: sale.CashierName, Terminal: sale.TerminalName, Customer: sale.CustomerName,
		Currency: sale.Currency, Subtotal: sale.Subtotal.StringFixed(2), Discount: sale.Discount.StringFixed(2),
		DiscountNote: sale.DiscountNote, Tax: sale.Tax.StringFixed(2), Total: sale.Total.StringFixed(2),
		Paid: sale.Paid.StringFixed(2), Change: sale.ChangeDue.StringFixed(2),
		// The sales pipeline extracts VAT from tax-inclusive prices.
		PricesIncludeVAT: true,
	}
	d.Items = make([]receiptfmt.Item, 0, len(sd.Items))
	for _, it := range sd.Items {
		item := receiptfmt.Item{
			Name: it.Name, SKU: it.SKU, Qty: it.Qty.String(), UnitPrice: it.UnitPrice.StringFixed(2),
			Discount: it.Discount.StringFixed(2), Total: it.Total.StringFixed(2), Tax: it.Tax.StringFixed(2),
			Promotion: it.Promotion,
		}
		if it.TaxRate != nil {
			r := it.TaxRate.String()
			item.TaxRate = &r
		}
		d.Items = append(d.Items, item)
	}
	d.Payments = d.Payments[:0]
	// Every tender is shown as it was taken, even if later voided or
	// refunded: the receipt documents the original transaction.
	for _, p := range sd.Payments {
		pay := receiptfmt.Payment{Method: p.Method, Amount: p.Amount.StringFixed(2), Reference: p.Reference}
		if p.Tendered != nil {
			pay.Tendered = p.Tendered.StringFixed(2)
		}
		if p.ChangeDue != nil {
			pay.Change = p.ChangeDue.StringFixed(2)
		}
		d.Payments = append(d.Payments, pay)
	}
	d.Flags.Voided = sale.Status == sales.StatusVoided
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
