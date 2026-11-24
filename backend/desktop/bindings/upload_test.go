package bindings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A sale the server refuses (4xx) is parked for a person; a server error
// (5xx) stays queued and is retried. Queued sales carry the offline flag.
func TestUploadParksRefusedSales(t *testing.T) {
	var status atomic.Int32
	status.Store(422)
	var sawOffline atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["offline"] == true {
			sawOffline.Store(true)
		}
		w.WriteHeader(int(status.Load()))
		_, _ = w.Write([]byte(`{"error":{"code":"x","message":"Product is archived."}}`))
	}))
	defer srv.Close()

	app, _ := newPrintApp(t, "http://127.0.0.1:1", false) // offline: sale is queued
	if res, err := app.Pay(cashPay, cart, "", "", "", ""); err != nil || !res.Offline {
		t.Fatalf("queue: %+v %v", res, err)
	}
	app.settings.APIBaseURL = srv.URL

	status.Store(500)
	if n, _ := app.SyncPending(); n != 0 {
		t.Fatal("uploaded on 500")
	}
	if failed, _ := app.FailedSales(); len(failed) != 0 {
		t.Fatal("a server error parked the sale")
	}

	status.Store(422)
	_, _ = app.SyncPending()
	failed, err := app.FailedSales()
	if err != nil || len(failed) != 1 || !strings.Contains(failed[0].Error, "archived") {
		t.Fatalf("parked: %+v %v", failed, err)
	}
	if !sawOffline.Load() {
		t.Fatal("upload did not carry the offline flag")
	}
	if n, _ := app.SyncPending(); n != 0 {
		t.Fatal("parked sale retried automatically")
	}
	if st := app.Status(); st.FailedSales != 1 {
		t.Fatalf("status failed_sales = %d", st.FailedSales)
	}

	status.Store(200)
	if _, err := app.RetryFailedSale(failed[0].ID); err == nil {
		t.Log("retry after fix accepted")
	}
}

func TestPaymentMethodsFromSettings(t *testing.T) {
	app, _ := newPrintApp(t, "http://127.0.0.1:1", false)
	if got := strings.Join(app.PaymentMethods(), ","); got != "cash,mpesa,card" {
		t.Fatalf("defaults %s", got)
	}
	if _, err := app.db.ExecContext(app.ctx,
		`INSERT INTO settings (key, value) VALUES ('payments.methods', '["mpesa","cash"]')`); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(app.PaymentMethods(), ","); got != "cash,mpesa" {
		t.Fatalf("from settings %s", got)
	}
}
