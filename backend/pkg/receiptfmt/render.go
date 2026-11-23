package receiptfmt

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"mauzo/pkg/barcode"
)

// Output is a laid-out receipt: what every printer, preview and snapshot
// consumes. Text lines are already wrapped and column-aligned to Width.
type Output struct {
	Paper int    `json:"paper"`
	Width int    `json:"width"` // characters per line at normal size
	Lines []Line `json:"lines"`
}

// Line kinds.
const (
	LineText    = "text"
	LineLogo    = "logo"
	LineQR      = "qr"
	LineBarcode = "barcode"
	LineFeed    = "feed"
)

// Line is one printable row.
type Line struct {
	Kind  string `json:"k"`
	Text  string `json:"t,omitempty"`
	Align string `json:"a,omitempty"` // only for text the renderer did not pad
	Bold  bool   `json:"b,omitempty"`
	Size  string `json:"s,omitempty"` // tall | wide | double
	Data  string `json:"d,omitempty"` // qr / barcode payload
	QR    int    `json:"q,omitempty"` // qr module size
	N     int    `json:"n,omitempty"` // feed lines
	// Bars is the Code 128 pattern ("1" = bar) for previews; printers encode
	// barcodes natively from Data.
	Bars string `json:"bars,omitempty"`
}

// Text returns the receipt as plain text (tests, logs, e-mail bodies).
func (o Output) Text() string {
	var b strings.Builder
	for _, l := range o.Lines {
		switch l.Kind {
		case LineText:
			b.WriteString(strings.TrimRight(l.Text, " "))
		case LineLogo:
			b.WriteString(center("[logo]", o.Width))
		case LineQR:
			b.WriteString(center("[QR "+l.Data+"]", o.Width))
		case LineBarcode:
			b.WriteString(center("|||| "+l.Data+" ||||", o.Width))
		case LineFeed:
			b.WriteString(strings.Repeat("\n", max(l.N-1, 0)))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Render lays out doc with t. t must be valid (Parse/Validate).
func Render(t Template, d Doc) Output {
	r := renderer{t: t, d: d, w: t.Width(), vars: d.vars(), flags: d.flags()}
	r.banners()
	for _, b := range t.Blocks {
		if !r.cond(b.If) {
			continue
		}
		r.block(b)
	}
	r.banners() // repeated at the bottom so a torn receipt still shows it
	return Output{Paper: t.Paper, Width: r.w, Lines: r.out}
}

type renderer struct {
	t     Template
	d     Doc
	w     int
	vars  map[string]string
	flags map[string]bool
	out   []Line
}

func (r *renderer) cond(c string) bool {
	if c == "" {
		return true
	}
	neg := strings.HasPrefix(c, "!")
	c = strings.TrimPrefix(c, "!")
	ok, isFlag := r.flags[c]
	if !isFlag {
		ok = strings.TrimSpace(r.vars[c]) != ""
	}
	return ok != neg
}

// allVarsEmpty reports whether s uses variables and every one is empty.
func (r *renderer) allVarsEmpty(s string) bool {
	ms := varRe.FindAllStringSubmatch(s, -1)
	if len(ms) == 0 {
		return false
	}
	for _, m := range ms {
		if strings.TrimSpace(r.vars[m[1]]) != "" {
			return false
		}
	}
	return true
}

func (r *renderer) expand(s string) string {
	return varRe.ReplaceAllStringFunc(s, func(m string) string {
		return r.vars[varRe.FindStringSubmatch(m)[1]]
	})
}

// banners mark copies that must never pass for an original.
func (r *renderer) banners() {
	f := r.d.Flags
	var msgs []string
	if f.Sample {
		msgs = append(msgs, "SAMPLE - NOT A RECEIPT")
	}
	if f.Voided {
		msgs = append(msgs, "*** VOIDED ***")
	}
	if f.Offline {
		msgs = append(msgs, "PROVISIONAL - OFFLINE")
	}
	if f.Reprint {
		s := "*** REPRINT ***"
		if f.CopyNo > 0 {
			s = "*** REPRINT " + itoa(f.CopyNo) + " ***"
		}
		msgs = append(msgs, s)
	}
	for _, m := range msgs {
		r.text(center(m, r.w), true, "")
	}
}

func (r *renderer) text(s string, bold bool, size string) {
	r.out = append(r.out, Line{Kind: LineText, Text: s, Bold: bold, Size: size})
}

// width is the characters available at a size (wide fonts halve it).
func (r *renderer) width(size string) int {
	if size == "wide" || size == "double" {
		return r.w / 2
	}
	return r.w
}

func (r *renderer) block(b Block) {
	switch b.Type {
	case BlockLogo:
		if r.d.Flags.HasLogo {
			r.out = append(r.out, Line{Kind: LineLogo, Align: or(b.Align, "center")})
		}
	case BlockText:
		r.textBlock(b)
	case BlockDivider:
		r.divider(b)
	case BlockFields:
		r.fields(b)
	case BlockItems:
		r.items(b)
	case BlockTotals:
		r.totals(b)
	case BlockPayments:
		r.payments()
	case BlockTaxSummary:
		r.taxSummary()
	case BlockFiscal:
		r.fiscal()
	case BlockQR:
		r.qr(b)
	case BlockBarcode:
		r.barcode(b)
	case BlockFeed:
		r.out = append(r.out, Line{Kind: LineFeed, N: b.Lines})
	}
}

func (r *renderer) textBlock(b Block) {
	size := b.Size
	if size == "normal" {
		size = ""
	}
	w := r.width(size)
	for _, src := range strings.Split(b.Text, "\n") {
		// A line whose variables are all empty ("Tel: {{branch.phone}}"
		// without a phone) is dropped rather than printed half-filled.
		if r.allVarsEmpty(src) {
			continue
		}
		for _, ln := range wrap(r.expand(src), w) {
			r.text(align(ln, w, b.Align), b.Bold, size)
		}
	}
}

func (r *renderer) divider(b Block) {
	// Two rules in a row (a hidden block between them) print as one.
	if n := len(r.out); n > 0 && r.out[n-1].Kind == LineText && isRule(r.out[n-1].Text) {
		return
	}
	r.text(strings.Repeat(or(b.Char, "-"), r.w), false, "")
}

func (r *renderer) fields(b Block) {
	for _, f := range b.Fields {
		v := strings.TrimSpace(r.expand(f.Value))
		if v == "" {
			continue
		}
		for _, ln := range pair(r.expand(f.Label), v, r.w) {
			r.text(ln, b.Bold, "")
		}
	}
}

func (r *renderer) qr(b Block) {
	data := strings.TrimSpace(r.expand(b.Data))
	if data == "" {
		return
	}
	n := b.QRSize
	if n == 0 {
		n = 6
	}
	r.out = append(r.out, Line{Kind: LineQR, Data: data, QR: n, Align: or(b.Align, "center")})
}

func (r *renderer) barcode(b Block) {
	data := strings.TrimSpace(r.expand(b.Data))
	if data == "" {
		return
	}
	bars, err := barcode.Code128(data)
	if err != nil {
		r.text(center(data, r.w), false, "") // non-printable: fall back to text
		return
	}
	var sb strings.Builder
	for _, on := range bars {
		if on {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	r.out = append(r.out, Line{Kind: LineBarcode, Data: data, Bars: sb.String(), Align: or(b.Align, "center")})
}

func (r *renderer) items(b Block) {
	for _, it := range r.d.Items {
		name := it.Name
		if b.ShowSKU && it.SKU != "" {
			name += " (" + it.SKU + ")"
		}
		total := Money(it.Total)
		if b.Layout == "compact" {
			// "2 x Sugar 1kg ............ 360.00"
			label := Qty(it.Qty) + " x " + name
			for _, ln := range pair(label, total, r.w) {
				r.text(ln, false, "")
			}
		} else {
			for _, ln := range wrap(name, r.w) {
				r.text(ln, false, "")
			}
			detail := "  " + Qty(it.Qty) + " x " + Money(it.UnitPrice)
			r.text(pairOne(detail, total, r.w), false, "")
		}
		if d := dec(it.Discount); d.IsPositive() {
			r.text(pairOne("  Discount", "-"+Money(it.Discount), r.w), false, "")
		}
		if b.ShowPromo && it.Promotion != "" {
			for _, ln := range wrap("  * "+it.Promotion, r.w) {
				r.text(ln, false, "")
			}
		}
	}
}

func (r *renderer) totals(b Block) {
	s := r.d.Sale
	if b.ShowItemCount {
		r.text(pairOne("Items", r.vars["sale.item_count"], r.w), false, "")
	}
	discount := dec(s.Discount)
	if discount.IsPositive() || !dec(s.Subtotal).Equal(dec(s.Total)) {
		r.text(pairOne("Subtotal", Money(s.Subtotal), r.w), false, "")
	}
	if discount.IsPositive() {
		label := "Discount"
		if s.DiscountNote != "" {
			label += " (" + s.DiscountNote + ")"
		}
		for _, ln := range pair(label, "-"+Money(s.Discount), r.w) {
			r.text(ln, false, "")
		}
	}
	if dec(s.Tax).IsPositive() {
		label := "VAT"
		if s.PricesIncludeVAT {
			label = "VAT (incl.)"
		}
		r.text(pairOne(label, Money(s.Tax), r.w), false, "")
	}
	// TOTAL in double size: half the columns.
	hw := r.w / 2
	r.text(pairOne("TOTAL", Money(s.Total), hw), true, "double")
}

func (r *renderer) payments() {
	for _, p := range r.d.Payments {
		r.text(pairOne(MethodLabel(p.Method), Money(p.Amount), r.w), false, "")
		if t := dec(p.Tendered); t.IsPositive() && !t.Equal(dec(p.Amount)) {
			r.text(pairOne("  Tendered", Money(p.Tendered), r.w), false, "")
		}
		if p.Reference != "" {
			for _, ln := range pair("  Ref", p.Reference, r.w) {
				r.text(ln, false, "")
			}
		}
	}
	if dec(r.d.Sale.Change).IsPositive() {
		r.text(pairOne("CHANGE", Money(r.d.Sale.Change), r.w), true, "")
	}
}

func (r *renderer) taxSummary() {
	lines := r.d.Taxes
	if len(lines) == 0 {
		lines = TaxBreakdown(r.d.Items, r.d.Sale.PricesIncludeVAT)
	}
	if len(lines) == 0 {
		return
	}
	cw := r.w / 3
	row := func(a, b, c string) string {
		return padRight(a, r.w-2*cw) + padLeft(b, cw) + padLeft(c, cw)
	}
	r.text(row("VAT rate", "Taxable", "VAT"), true, "")
	for _, t := range lines {
		r.text(row(Qty(t.Rate)+"%", Money(t.Taxable), Money(t.Tax)), false, "")
	}
}

func (r *renderer) fiscal() {
	f := r.d.Fiscal
	if f == nil {
		return
	}
	rows := [][2]string{
		{"CU Serial", f.CUSerial}, {"CU Invoice No", f.InvoiceNo},
		{"Internal data", f.Internal}, {"Signature", f.Signature}, {"Signed", f.SignedAt},
	}
	r.text(center("eTIMS", r.w), true, "")
	for _, kv := range rows {
		if kv[1] == "" {
			continue
		}
		for _, ln := range pair(kv[0], kv[1], r.w) {
			r.text(ln, false, "")
		}
	}
	if f.QRData != "" {
		r.out = append(r.out, Line{Kind: LineQR, Data: f.QRData, QR: 5, Align: "center"})
	}
}

// TaxBreakdown groups VAT by rate. Taxable is the net amount: line total
// minus its VAT when prices include VAT, the line total otherwise.
func TaxBreakdown(items []Item, inclusive bool) []TaxLine {
	type acc struct{ taxable, tax decimal.Decimal }
	by := map[string]*acc{}
	for _, it := range items {
		if it.TaxRate == nil {
			continue
		}
		rate := dec(*it.TaxRate).String()
		a := by[rate]
		if a == nil {
			a = &acc{}
			by[rate] = a
		}
		total, tax := dec(it.Total), dec(it.Tax)
		if inclusive {
			total = total.Sub(tax)
		}
		a.taxable = a.taxable.Add(total)
		a.tax = a.tax.Add(tax)
	}
	rates := make([]string, 0, len(by))
	for k := range by {
		rates = append(rates, k)
	}
	sort.Slice(rates, func(i, j int) bool { return dec(rates[i]).GreaterThan(dec(rates[j])) })
	out := make([]TaxLine, 0, len(rates))
	for _, k := range rates {
		out = append(out, TaxLine{Rate: k, Taxable: by[k].taxable.StringFixed(2), Tax: by[k].tax.StringFixed(2)})
	}
	return out
}

func itoa(n int) string { return decimal.NewFromInt(int64(n)).String() }

// isRule reports whether a laid-out line is a divider (one repeated symbol).
func isRule(s string) bool {
	if len(s) < 8 {
		return false
	}
	c := s[0]
	if c != '-' && c != '=' && c != '*' && c != '_' && c != '.' && c != '~' && c != '#' {
		return false
	}
	return strings.Count(s, string(c)) == len(s)
}

// Mark adds a banner (e.g. "*** REPRINT 2 ***") to the top and bottom of an
// already issued receipt without touching its content.
func Mark(o Output, msg string) Output {
	w := o.Width
	if w <= 0 {
		w = 48
	}
	banner := Line{Kind: LineText, Text: center(msg, w), Bold: true}
	lines := make([]Line, 0, len(o.Lines)+2)
	lines = append(lines, banner)
	lines = append(lines, o.Lines...)
	lines = append(lines, banner)
	o.Lines = lines
	return o
}

// ReprintBanner is the banner for copy n of a receipt.
func ReprintBanner(n int) string {
	if n <= 0 {
		return "*** REPRINT ***"
	}
	return "*** REPRINT " + itoa(n) + " ***"
}
