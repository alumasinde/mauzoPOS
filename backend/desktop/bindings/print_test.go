package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mauzo/desktop/localdb"
	"mauzo/desktop/settings"
	"mauzo/pkg/receiptfmt"
)

// newPrintApp builds an App with a synced local mirror (one product, a
// minimart print kit) and a file "printer" so jobs can be inspected.
func newPrintApp(t *testing.T, apiURL string, autoPrint bool) (*App, string) {
	t.Helper()
	ctx := context.Background()
	db, err := localdb.Open(ctx, filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	b, _ := receiptfmt.BuiltinByKey("minimart")
	kit := receiptfmt.Kit{BuiltinKey: "minimart", Template: b.Raw,
		Business: receiptfmt.Business{Name: "Kona Mini Mart"},
		Branch:   receiptfmt.Branch{Name: "Ruaka", Phone: "0700 000 000"},
		Extra:    map[string]string{"mpesa_till": "987654"},
		Policy:   receiptfmt.Policy{AutoPrint: autoPrint, Copies: 1, OpenDrawer: true}}
	kitJSON, _ := json.Marshal(kit)
	bundle := `{"server_time":"2026-11-23T10:00:00Z","cursor":"2026-11-23T10:00:00Z",
		"products":[{"id":10,"public_id":"p1","sku":"SUG1","name":"Kabras Sugar 1kg","kind":"standard",
		"category_id":0,"unit_code":"pc","sell_by":"unit","price":"180.00","vat_rate":"16","track_stock":true,
		"updated_at":"2026-11-23T10:00:00Z"}],"receipt":` + string(kitJSON) + `}`
	var bd localdb.Bundle
	if err := json.Unmarshal([]byte(bundle), &bd); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplySync(ctx, bd); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "printer.bin")
	for k, v := range map[string]string{"printer_kind": "file", "printer_name": out, "terminal_name": "Till 2"} {
		if err := db.SetMeta(ctx, k, v); err != nil {
			t.Fatal(err)
		}
	}
	app := NewApp("test", settings.Settings{APIBaseURL: apiURL, HTTPTimeout: 2 * time.Second})
	app.db = db
	app.ctx = ctx
	app.token = "device-token"
	app.session = &Session{CashierID: "u1", Name: "Wanjiku"}
	return app, out
}

func printed(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const cart = `[{"product_id":10,"name":"Kabras Sugar 1kg","sku":"SUG1","qty":"2"}]`
const cashPay = `[{"method":"cash","amount":"360.00","tendered":"500.00"}]`

// Offline: the sale is queued and a PROVISIONAL receipt prints from the kit,
// opening the drawer for cash.
func TestOfflineSalePrintsProvisionalReceipt(t *testing.T) {
	app, out := newPrintApp(t, "http://127.0.0.1:1", true)
	res, err := app.Pay(cashPay, cart, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Offline || !res.Printed || res.PrintError != "" {
		t.Fatalf("result %+v", res)
	}
	job := printed(t, out)
	for _, want := range []string{"PROVISIONAL - OFFLINE", "Kona Mini Mart", "2 x Kabras Sugar 1kg", "360.00",
		"CHANGE", "140.00", "Till No: 987654", "Wanjiku"} {
		if !strings.Contains(job, want) {
			t.Errorf("missing %q in job:\n%s", want, job)
		}
	}
	if !bytes.Contains([]byte(job), []byte{0x1B, 'p', 0, 25, 250}) {
		t.Error("cash sale did not kick the drawer")
	}
	if err := app.PrintLast(); err == nil {
		t.Fatal("PrintLast after auto-print must refuse (copies are audited reprints)")
	}
}

// Online: the server's issued snapshot is what prints; with auto-print off
// the cashier prints it once with PrintLast; reprints go through the API.
func TestOnlineSalePrintsServerSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := receiptfmt.Output{Paper: 58, Width: 32, Lines: []receiptfmt.Line{
			{Kind: receiptfmt.LineText, Text: "SERVER RECEIPT 77"}}}
		switch {
		case r.URL.Path == "/api/v1/pos/sales":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"sale":       map[string]any{"id": "s1", "receipt_no": 77, "total": "360.00", "change_due": "140.00"},
				"receipt_no": 77, "receipt": map[string]any{"id": "r1", "output": snap},
			}})
		case strings.HasSuffix(r.URL.Path, "/reprint"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id": "r2", "copy_no": 1, "output": receiptfmt.Mark(snap, receiptfmt.ReprintBanner(1)),
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	app, out := newPrintApp(t, srv.URL, false)
	res, err := app.Pay(cashPay, cart, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.ReceiptNo != 77 || res.Printed || !res.CanPrint {
		t.Fatalf("auto-print off: %+v", res)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("printed although auto-print is off")
	}
	if err := app.PrintLast(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(printed(t, out), "SERVER RECEIPT 77") {
		t.Fatal("server snapshot not printed")
	}
	if err := app.PrintLast(); err == nil {
		t.Fatal("second PrintLast must refuse")
	}
	rp, err := app.Reprint("77", "")
	if err != nil || !rp.Printed || rp.CopyNo != 1 {
		t.Fatalf("reprint %+v %v", rp, err)
	}
	if !strings.Contains(printed(t, out), "*** REPRINT 1 ***") {
		t.Fatal("reprint banner missing")
	}
}

func TestPrinterSettingsRoundTrip(t *testing.T) {
	app, _ := newPrintApp(t, "http://127.0.0.1:1", true)
	err := app.SaveSettings(TerminalSettings{PrinterKind: "network", PrinterHost: "192.168.1.50",
		PrinterPort: 9100, AutoPrint: "never", AutoLockSeconds: 120})
	if err != nil {
		t.Fatal(err)
	}
	s, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.PrinterKind != "network" || s.PrinterHost != "192.168.1.50" || s.AutoPrint != "never" || s.PaperWidth != 58 {
		t.Fatalf("settings %+v", s)
	}
	if err := app.SaveSettings(TerminalSettings{PrinterKind: "laser"}); err == nil {
		t.Fatal("unknown printer type accepted")
	}
}
