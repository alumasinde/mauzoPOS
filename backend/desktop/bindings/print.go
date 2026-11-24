package bindings

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/desktop/localdb"
	"mauzo/desktop/printing"
	"mauzo/pkg/receiptfmt"
)

// Auto-print overrides on a terminal ("default" follows the business policy).
const (
	autoDefault = "default"
	autoAlways  = "always"
	autoNever   = "never"
)

// printState caches the printer probe and the last receipt of this session.
type printState struct {
	mu        sync.Mutex
	probedAt  time.Time
	probe     string
	last      *receiptfmt.Output // original receipt of the last sale
	lastCash  bool
	lastShown bool   // already printed (further copies must be audited reprints)
	lastName  string // file name for a PDF copy
}

// printerConfig reads the terminal's printer from local meta. A printer name
// saved before Phase 8 (no kind) is treated as an installed Windows printer.
func (a *App) printerConfig() printing.Config {
	if a.db == nil {
		return printing.Config{Kind: printing.KindNone}
	}
	c := printing.Config{}
	c.Kind, _ = a.db.Meta(a.ctx, "printer_kind")
	c.Name, _ = a.db.Meta(a.ctx, "printer_name")
	c.Host, _ = a.db.Meta(a.ctx, "printer_host")
	if p, _ := a.db.Meta(a.ctx, "printer_port"); p != "" {
		c.Port, _ = strconv.Atoi(p)
	}
	if c.Kind == "" && c.Name != "" {
		c.Kind = printing.KindWindows
	}
	if c.Kind == "" {
		c.Kind = printing.KindNone
	}
	return c
}

// printerStatus is the top-bar state, probing at most every 15 s.
func (a *App) printerStatus() string {
	a.print.mu.Lock()
	defer a.print.mu.Unlock()
	if time.Since(a.print.probedAt) < 15*time.Second && a.print.probe != "" {
		return a.print.probe
	}
	a.print.probe = printing.Probe(a.printerConfig())
	a.print.probedAt = time.Now()
	return a.print.probe
}

func (a *App) resetProbe() {
	a.print.mu.Lock()
	a.print.probedAt = time.Time{}
	a.print.mu.Unlock()
}

// kit returns the synced print kit, or a builtin-only kit before first sync.
func (a *App) kit() receiptfmt.Kit {
	var k receiptfmt.Kit
	if a.db != nil {
		if raw, _ := a.db.Meta(a.ctx, "receipt_kit"); raw != "" {
			_ = json.Unmarshal([]byte(raw), &k)
		}
	}
	if k.Policy.Copies < 1 {
		k.Policy.Copies = 1
	}
	return k
}

// autoPrint decides whether a completed sale prints by itself.
func (a *App) autoPrint(k receiptfmt.Kit) bool {
	mode := autoDefault
	if a.db != nil {
		if v, _ := a.db.Meta(a.ctx, "printer_auto"); v != "" {
			mode = v
		}
	}
	switch mode {
	case autoAlways:
		return true
	case autoNever:
		return false
	}
	return k.Policy.AutoPrint
}

// printOutput sends copies of a receipt, kicking the drawer on the first one
// when asked. A till without a receipt printer (or set to a Windows PDF/XPS
// printer, which cannot take ESC/POS) gets one PDF, saved and opened; the
// path is returned.
func (a *App) printOutput(out receiptfmt.Output, name string, copies int, drawer bool) (string, error) {
	cfg := a.printerConfig()
	if !cfg.Configured() {
		return "", printing.ErrNotConfigured
	}
	if cfg.SavesPDF() {
		path, err := printing.SavePDF(name, receiptfmt.EncodePDF(out))
		if err != nil {
			return "", err
		}
		_ = printing.Open(path) // saved is enough; a missing viewer is not an error
		return path, nil
	}
	k := a.kit()
	if copies < 1 {
		copies = 1
	}
	for i := range copies {
		job := receiptfmt.EncodeESCPOS(out, receiptfmt.PrintOptions{
			Logo: k.Logo, Cut: true, OpenDrawer: drawer && i == 0,
		})
		if err := printing.Send(cfg, job); err != nil {
			a.resetProbe()
			return "", err
		}
	}
	return "", nil
}

// receiptName names a PDF copy of a sale's receipt.
func receiptName(res *PayResult) string {
	if res.ReceiptNo > 0 {
		return "receipt-" + strconv.FormatUint(res.ReceiptNo, 10)
	}
	id := res.SaleID
	if len(id) > 8 {
		id = id[len(id)-8:]
	}
	return "provisional-" + id
}

// afterSale remembers the sale's receipt and prints it when the policy says
// so. It never fails the sale: a printer problem is reported in the result.
func (a *App) afterSale(res *PayResult, out *receiptfmt.Output, hasCash bool) {
	if out == nil {
		return
	}
	a.print.mu.Lock()
	a.print.last, a.print.lastCash, a.print.lastShown = out, hasCash, false
	a.print.lastName = receiptName(res)
	a.print.mu.Unlock()
	res.CanPrint = true
	k := a.kit()
	if !a.autoPrint(k) {
		return
	}
	file, err := a.printOutput(*out, receiptName(res), k.Policy.Copies, hasCash && k.Policy.OpenDrawer)
	if err != nil {
		res.PrintError = err.Error()
		return
	}
	res.Printed, res.ReceiptFile = true, file
	a.print.mu.Lock()
	a.print.lastShown = true
	a.print.mu.Unlock()
}

// PrintLast prints the original receipt of the last sale when it has not
// been printed yet (auto-print off, or the printer was down). Once printed,
// further copies are reprints and go through Reprint (audited). It returns
// the saved file when the till prints to PDF.
func (a *App) PrintLast() (string, error) {
	a.print.mu.Lock()
	out, cash, shown, name := a.print.last, a.print.lastCash, a.print.lastShown, a.print.lastName
	a.print.mu.Unlock()
	if out == nil {
		return "", errors.New("no receipt to print")
	}
	if shown {
		return "", errors.New("already printed - use Reprint for another copy")
	}
	k := a.kit()
	file, err := a.printOutput(*out, name, k.Policy.Copies, cash && k.Policy.OpenDrawer)
	if err != nil {
		return "", err
	}
	a.print.mu.Lock()
	a.print.lastShown = true
	a.print.mu.Unlock()
	return file, nil
}

// ReprintResult is what the UI shows after a reprint.
type ReprintResult struct {
	CopyNo  int    `json:"copy_no"`
	Printed bool   `json:"printed"`
	Error   string `json:"error,omitempty"`
	File    string `json:"file,omitempty"` // saved PDF, when printing to PDF
}

// Reprint asks the server for an audited copy of a receipt of this branch
// and prints it. grant is a supervisor grant when the cashier lacks
// pos.receipt.reprint. Reprints need the server (offline they are refused so
// every copy is accounted for).
func (a *App) Reprint(ref, grant string) (ReprintResult, error) {
	var out struct {
		ID     string            `json:"id"`
		CopyNo int               `json:"copy_no"`
		Output receiptfmt.Output `json:"output"`
	}
	err := a.request("POST", "/api/v1/pos/sales/"+urlQuery(ref)+"/reprint",
		map[string]string{"cashier_id": a.cashierID(), "supervisor_grant": grant}, &out)
	if errors.Is(err, ErrOffline) {
		return ReprintResult{}, errors.New("reprints need a connection to the server")
	}
	if err != nil {
		return ReprintResult{}, err
	}
	res := ReprintResult{CopyNo: out.CopyNo}
	name := "receipt-" + ref + "-copy-" + strconv.Itoa(out.CopyNo)
	if file, perr := a.printOutput(out.Output, name, 1, false); perr != nil {
		res.Error = perr.Error()
	} else {
		res.Printed, res.File = true, file
	}
	return res, nil
}

// TestPrint prints a sample receipt with this branch's layout.
func (a *App) TestPrint() error {
	k := a.kit()
	doc := receiptfmt.SampleDoc()
	base := k.Base()
	if base.Business.Name != "" {
		doc.Business, doc.Branch, doc.Footer = base.Business, base.Branch, base.Footer
	}
	if len(base.Extra) > 0 {
		doc.Extra = base.Extra
	}
	doc.Flags.HasLogo = base.Flags.HasLogo
	_, err := a.printOutput(receiptfmt.Render(k.ParsedTemplate(), doc), "test-print", 1, false)
	return err
}

// OpenDrawer pulses the cash drawer through the receipt printer.
func (a *App) OpenDrawer() error {
	cfg := a.printerConfig()
	if !cfg.Configured() {
		return printing.ErrNotConfigured
	}
	if cfg.SavesPDF() {
		return errors.New("no cash drawer: receipts are saved as PDF on this till")
	}
	return printing.Send(cfg, []byte{0x1B, '@', 0x1B, 'p', 0, 25, 250})
}

// OpenReceiptFile reopens a saved PDF receipt. Only files inside the
// receipts folder can be opened, so the WebView cannot launch anything else.
func (a *App) OpenReceiptFile(path string) error {
	dir, err := printing.ReceiptsDir()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(dir, filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) ||
		!strings.EqualFold(filepath.Ext(path), ".pdf") {
		return errors.New("not a saved receipt")
	}
	return printing.Open(path)
}

// ListPrinters returns the printers installed in Windows.
func (a *App) ListPrinters() ([]string, error) { return printing.List() }

// ---- receipt documents built on the till --------------------------------------

// saleDetailWire is the server's sale detail (sales.DetailResponse).
type saleDetailWire struct {
	ID        string    `json:"id"`
	ReceiptNo uint64    `json:"receipt_no"`
	Status    string    `json:"status"`
	Terminal  string    `json:"terminal"`
	Cashier   string    `json:"cashier"`
	Customer  string    `json:"customer"`
	Subtotal  string    `json:"subtotal"`
	Discount  string    `json:"discount"`
	Tax       string    `json:"tax"`
	Total     string    `json:"total"`
	Paid      string    `json:"paid"`
	ChangeDue string    `json:"change_due"`
	SoldAt    time.Time `json:"sold_at"`
	Items     []struct {
		Name      string `json:"name"`
		SKU       string `json:"sku"`
		Qty       string `json:"qty"`
		UnitPrice string `json:"unit_price"`
		Discount  string `json:"discount"`
		Tax       string `json:"tax"`
		Total     string `json:"total"`
		Promotion string `json:"promotion"`
	} `json:"items"`
	Payments []struct {
		Method    string `json:"method"`
		Amount    string `json:"amount"`
		Tendered  string `json:"tendered"`
		ChangeDue string `json:"change_due"`
		Reference string `json:"reference"`
	} `json:"payments"`
}

// receiptFromDetail renders a server-confirmed sale locally — used only when
// the server committed the sale but could not return its receipt snapshot.
func (a *App) receiptFromDetail(d saleDetailWire) receiptfmt.Output {
	k := a.kit()
	doc := k.Base()
	doc.Sale = receiptfmt.Sale{
		ReceiptNo: strconv.FormatUint(d.ReceiptNo, 10), Ref: d.ID, SoldAt: d.SoldAt,
		Cashier: d.Cashier, Terminal: d.Terminal, Customer: d.Customer,
		Subtotal: d.Subtotal, Discount: d.Discount, Tax: d.Tax, Total: d.Total, Paid: d.Paid,
		Change: d.ChangeDue, PricesIncludeVAT: true,
	}
	for _, it := range d.Items {
		doc.Items = append(doc.Items, receiptfmt.Item{Name: it.Name, SKU: it.SKU, Qty: it.Qty,
			UnitPrice: it.UnitPrice, Discount: it.Discount, Tax: it.Tax, Total: it.Total, Promotion: it.Promotion})
	}
	for _, p := range d.Payments {
		doc.Payments = append(doc.Payments, receiptfmt.Payment{Method: p.Method, Amount: p.Amount,
			Tendered: p.Tendered, Change: p.ChangeDue, Reference: p.Reference})
	}
	return receiptfmt.Render(k.ParsedTemplate(), doc)
}

// provisionalReceipt renders an offline sale from the local price quote. It
// is marked PROVISIONAL: the server prices the sale again when it uploads and
// issues the real receipt number.
func (a *App) provisionalReceipt(ref string, lines []CartLine, payments []PayMethod, orderDiscount string) (receiptfmt.Output, error) {
	ql := make([]localdb.QuoteLine, len(lines))
	for i, l := range lines {
		ql[i] = localdb.QuoteLine{ProductID: l.ProductID, VariantID: l.VariantID, CategoryID: l.CategoryID, Qty: l.Qty}
	}
	q, err := a.db.QuoteCart(a.ctx, ql)
	if err != nil {
		return receiptfmt.Output{}, err
	}
	k := a.kit()
	doc := k.Base()
	subtotal, total := decimal.Zero, decimal.Zero
	for i, l := range lines {
		qty := num(l.Qty)
		unit := num(q.Lines[i].UnitPrice)
		lineTotal := num(q.Lines[i].Total)
		if l.PriceOverride != "" {
			unit = num(l.PriceOverride)
			lineTotal = unit.Mul(qty)
		}
		lineDisc := num(l.Discount)
		lineTotal = lineTotal.Sub(lineDisc)
		subtotal = subtotal.Add(unit.Mul(qty))
		total = total.Add(lineTotal)
		doc.Items = append(doc.Items, receiptfmt.Item{
			Name: l.Name, SKU: l.SKU, Qty: l.Qty, UnitPrice: unit.StringFixed(2),
			Discount: unit.Mul(qty).Sub(lineTotal).StringFixed(2), Total: lineTotal.StringFixed(2),
			Promotion: q.Lines[i].PromoName,
		})
	}
	total = total.Sub(num(orderDiscount))
	paid, tendered, cashAmount := decimal.Zero, decimal.Zero, decimal.Zero
	for _, p := range payments {
		paid = paid.Add(num(p.Amount))
		pay := receiptfmt.Payment{Method: p.Method, Amount: p.Amount, Tendered: p.Tendered, Reference: p.Reference}
		if p.Tendered != "" {
			tendered = tendered.Add(num(p.Tendered))
			cashAmount = cashAmount.Add(num(p.Amount))
		}
		doc.Payments = append(doc.Payments, pay)
	}
	change := tendered.Sub(cashAmount)
	if change.IsNegative() {
		change = decimal.Zero
	}
	doc.Sale = receiptfmt.Sale{
		Ref: ref, SoldAt: time.Now(), Cashier: a.cashierName(),
		Subtotal: subtotal.StringFixed(2), Discount: subtotal.Sub(total).StringFixed(2),
		Tax: q.Vat, Total: total.StringFixed(2), Paid: paid.StringFixed(2), Change: change.StringFixed(2),
		PricesIncludeVAT: true,
	}
	if name, _ := a.db.Meta(a.ctx, "terminal_name"); name != "" {
		doc.Sale.Terminal = name
	}
	doc.Flags.Offline = true
	return receiptfmt.Render(k.ParsedTemplate(), doc), nil
}

func (a *App) cashierName() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return ""
	}
	return a.session.Name
}

func num(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return d
}

func hasCash(payments []PayMethod) bool {
	for _, p := range payments {
		if p.Method == "cash" {
			return true
		}
	}
	return false
}
