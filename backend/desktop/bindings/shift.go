package bindings

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/pkg/receiptfmt"
)

// ShiftStatus returns the till's open shift, or {"shift": null}.
func (a *App) ShiftStatus() (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/shift", nil, &out)
	return out, err
}

// ShiftOpen starts the drawer session with a float.
func (a *App) ShiftOpen(floatAmount string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("POST", "/api/v1/pos/shift/open", map[string]string{
		"cashier_id": a.cashierID(), "float": floatAmount,
	}, &out)
	return out, err
}

// ShiftCash records a manual cash in / cash out on the open shift.
func (a *App) ShiftCash(direction, amount, reason string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("POST", "/api/v1/pos/shift/cash", map[string]string{
		"cashier_id": a.cashierID(), "direction": direction,
		"amount": amount, "reason": reason,
	}, &out)
	return out, err
}

// ShiftDrawer audits a no-sale drawer open server-side, then kicks the
// drawer through the receipt printer. The audit is best-effort when offline —
// the till still opens.
func (a *App) ShiftDrawer() error {
	_ = a.request("POST", "/api/v1/pos/shift/drawer",
		map[string]string{"cashier_id": a.cashierID()}, nil)
	return a.OpenDrawer()
}

// ShiftXReport returns the running totals for the open shift.
func (a *App) ShiftXReport() (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/shift/report?cashier_id="+urlQuery(a.cashierID()),
		nil, &out)
	return out, err
}

// ShiftReport returns the report for a specific shift (Z reprint).
func (a *App) ShiftReport(shiftID string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("GET", "/api/v1/pos/shift/"+urlQuery(shiftID)+
		"/report?cashier_id="+urlQuery(a.cashierID()), nil, &out)
	return out, err
}

// ShiftClose ends the open shift with the blind count; returns the Z-report.
func (a *App) ShiftClose(declared, note string) (json.RawMessage, error) {
	var out json.RawMessage
	err := a.request("POST", "/api/v1/pos/shift/close", map[string]string{
		"cashier_id": a.cashierID(), "declared": declared, "note": note,
	}, &out)
	return out, err
}

// PrintShiftReport renders an X/Z report JSON to the receipt printer.
func (a *App) PrintShiftReport(report json.RawMessage) (string, error) {
	out, err := shiftReportOutput(report, a.kit())
	if err != nil {
		return "", err
	}
	return a.printOutput(*out, "shift-report", 1, false)
}

// shiftReportOutput lays a shift report out as a printable receipt.
// shiftReportPayload mirrors the /pos/shift/report response for receipt
// rendering without importing the module's types.
type shiftReportPayload struct {
	Shift struct {
		PublicID     string  `json:"id"`
		Status       string  `json:"status"`
		Float        string  `json:"float"`
		OpenedAt     string  `json:"opened_at"`
		ClosedAt     *string `json:"closed_at"`
		OpenedByName string  `json:"opened_by_name"`
		Terminal     string  `json:"terminal"`
		Branch       string  `json:"branch"`
		Declared     *string `json:"declared"`
		Expected     *string `json:"expected"`
		Variance     *string `json:"variance"`
	} `json:"shift"`
	SalesCount int    `json:"sales_count"`
	SalesTotal string `json:"sales_total"`
	Methods    []struct {
		Method string `json:"method"`
		Count  int    `json:"count"`
		Total  string `json:"total"`
	} `json:"methods"`
	CashSales  string `json:"cash_sales"`
	CashRefund string `json:"cash_refunded"`
	CashIn     string `json:"cash_in"`
	CashOut    string `json:"cash_out"`
	Expected   string `json:"expected"`
	Movements  []struct {
		Kind   string `json:"kind"`
		Amount string `json:"amount"`
		Reason string `json:"reason"`
		At     string `json:"created_at"`
		By     string `json:"created_by_name"`
	} `json:"movements"`
}

func shiftReportOutput(raw json.RawMessage, k receiptfmt.Kit) (*receiptfmt.Output, error) {
	var rep shiftReportPayload
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	width := 48
	if t := receiptTemplate(k); t != nil {
		width = t.Width()
	}
	o := &receiptfmt.Output{Paper: 80, Width: width, Lines: []receiptfmt.Line{}}
	row := func(text string, bold bool) {
		o.Lines = append(o.Lines, receiptfmt.Line{Kind: receiptfmt.LineText, Text: text, Bold: bold})
	}
	cols := func(left, right string, bold bool) {
		pad := width - len([]rune(left)) - len([]rune(right))
		if pad < 1 {
			pad = 1
		}
		row(left+strings.Repeat(" ", pad)+right, bold)
	}
	title := map[bool]string{false: "X REPORT", true: "Z REPORT"}[rep.Shift.Status == "closed"]
	row(centerText(title, width), true)
	row(centerText(k.Business.Name, width), false)
	row(centerText(rep.Shift.Branch+" - "+rep.Shift.Terminal, width), false)
	row(strings.Repeat("-", width), false)
	cols("Opened", fmt.Sprintf("%s  %s", shortTime(rep.Shift.OpenedAt), rep.Shift.OpenedByName), false)
	if rep.Shift.ClosedAt != nil {
		cols("Closed", shortTime(*rep.Shift.ClosedAt), false)
	}
	row(strings.Repeat("-", width), false)
	cols(fmt.Sprintf("Sales x%d", rep.SalesCount), money(rep.SalesTotal), true)
	for _, m := range rep.Methods {
		cols("  "+m.Method+fmt.Sprintf(" x%d", m.Count), money(m.Total), false)
	}
	row(strings.Repeat("-", width), false)
	cols("Float", money(rep.Shift.Float), false)
	cols("Cash sales", money(rep.CashSales), false)
	cols("Cash in", money(rep.CashIn), false)
	cols("Cash out", "-"+money(rep.CashOut), false)
	cols("Cash refunds", "-"+money(rep.CashRefund), false)
	row(strings.Repeat("-", width), false)
	cols("EXPECTED DRAWER", money(rep.Expected), true)
	if rep.Shift.Declared != nil {
		cols("Counted", money(*rep.Shift.Declared), false)
	}
	if rep.Shift.Variance != nil {
		cols("VARIANCE", money(*rep.Shift.Variance), true)
	}
	if len(rep.Movements) > 0 {
		row(strings.Repeat("-", width), false)
		row("Movements", true)
		for _, m := range rep.Movements {
			label := m.Kind
			if m.Reason != "" && m.Reason != "float" {
				label = m.Kind + " " + truncate(m.Reason, 14)
			}
			cols("  "+shortTime(m.At)+" "+label, money(m.Amount), false)
		}
	}
	row(strings.Repeat("-", width), false)
	row(centerText("Shift "+rep.Shift.PublicID, width), false)
	return o, nil
}

func receiptTemplate(k receiptfmt.Kit) *receiptfmt.Template {
	if len(k.Template) == 0 {
		return nil
	}
	var t receiptfmt.Template
	if err := json.Unmarshal(k.Template, &t); err != nil {
		return nil
	}
	return &t
}

func centerText(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", (width-len(s))/2) + s
}

func money(s string) string {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return s
	}
	return d.StringFixed(2)
}

func shortTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		if i := strings.IndexByte(iso, 'T'); i > 0 && len(iso) >= i+6 {
			return iso[i+1 : i+6]
		}
		return iso
	}
	return t.Local().Format("15:04")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
