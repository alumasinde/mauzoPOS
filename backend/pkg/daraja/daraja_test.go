package daraja_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"mauzo/pkg/daraja"
)

// stub serves a scripted Daraja: oauth token plus keyed POST responses.
func stub(t *testing.T, posts map[string]any) (*httptest.Server, *int32) {
	t.Helper()
	var tokens int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/oauth/v1/generate" {
			atomic.AddInt32(&tokens, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-123", "expires_in": "3599",
			})
			return
		}
		for path, resp := range posts {
			if r.URL.Path == path {
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		}
		http.Error(w, `{"errorMessage":"unknown path"}`, http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv, &tokens
}

func TestTokenCached(t *testing.T) {
	srv, tokens := stub(t, map[string]any{})
	c := daraja.New(srv.URL, "key", "secret", nil)
	ctx := context.Background()
	if _, err := c.Token(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Token(ctx); err != nil {
		t.Fatal(err)
	}
	if got := *tokens; got != 1 {
		t.Fatalf("expected 1 token fetch, got %d", got)
	}
}

func TestSTKPushPayload(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3599"})
		case "/mpesa/stkpush/v1/processrequest":
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"MerchantRequestID": "m1", "CheckoutRequestID": "ws_CO_1",
				"ResponseCode": "0", "ResponseDescription": "queued",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := daraja.New(srv.URL, "key", "secret", nil)
	resp, err := c.STKPush(context.Background(), daraja.STKPushRequest{
		ShortCode: "600999", Passkey: "pass123", TxType: daraja.TypeTill,
		Amount: 1500, Payer: "254712345678", CallbackURL: "https://x.test/cb",
		AccountRef: "SALE", TransactionRef: "till sale",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.CheckoutRequestID != "ws_CO_1" {
		t.Fatalf("checkout id = %q", resp.CheckoutRequestID)
	}
	if body["TransactionType"] != daraja.TypeTill {
		t.Fatalf("tx type = %v", body["TransactionType"])
	}
	if body["BusinessShortCode"] != "600999" || body["PartyB"] != "600999" {
		t.Fatalf("shortcode fields: %v %v", body["BusinessShortCode"], body["PartyB"])
	}
	pw, _ := body["Password"].(string)
	ts, _ := body["Timestamp"].(string)
	if len(ts) != 14 {
		t.Fatalf("timestamp format = %q", ts)
	}
	want := base64.StdEncoding.EncodeToString([]byte("600999" + "pass123" + ts))
	if pw != want {
		t.Fatalf("password = %q, want base64(shortcode+passkey+timestamp)", pw)
	}
}

func TestSTKPushRejected(t *testing.T) {
	srv, _ := stub(t, map[string]any{
		"/mpesa/stkpush/v1/processrequest": map[string]any{
			"ResponseCode": "1", "ResponseDescription": "bad credentials",
		},
	})
	c := daraja.New(srv.URL, "key", "secret", nil)
	_, err := c.STKPush(context.Background(), daraja.STKPushRequest{ShortCode: "1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterC2BV2(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/oauth/v1/generate" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3599"})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"ResponseCode": "0", "ResponseDescription": "ok"})
	}))
	defer srv.Close()
	c := daraja.New(srv.URL, "k", "s", nil)
	if _, err := c.RegisterC2B(context.Background(), "600999", "Completed", "https://x/c", "https://x/v"); err != nil {
		t.Fatal(err)
	}
	if body["ResponseType"] != "Completed" {
		t.Fatalf("ResponseType must be sentence-case Completed, got %v", body["ResponseType"])
	}
}

func TestCallbackMeta(t *testing.T) {
	raw := `{"Body":{"stkCallback":{"MerchantRequestID":"m","CheckoutRequestID":"ws",
	  "ResultCode":0,"ResultDesc":"ok","CallbackMetadata":{"Item":[
	    {"Name":"Amount","Value":1500},{"Name":"MpesaReceiptNumber","Value":"QKR7ABC"},
	    {"Name":"TransactionDate","Value":20250101120000},{"Name":"PhoneNumber","Value":254712345678}]}}}}`
	var cb daraja.STKCallback
	if err := json.Unmarshal([]byte(raw), &cb); err != nil {
		t.Fatal(err)
	}
	if v, ok := cb.Meta("MpesaReceiptNumber"); !ok || v != "QKR7ABC" {
		t.Fatalf("receipt meta = %q %v", v, ok)
	}
	if v, ok := cb.Meta("PhoneNumber"); !ok || v != "254712345678" {
		t.Fatalf("phone meta = %q %v", v, ok)
	}
	if cb.TransDate().IsZero() {
		t.Fatal("TransDate did not parse")
	}
}

func TestC2BPaymentFields(t *testing.T) {
	raw := `{"TransactionType":"Pay Bill","TransID":"QKR7ABC","TransTime":"20250101120000",
	  "TransAmount":"1500.00","BusinessShortCode":"600999","BillRefNumber":"acct",
	  "MSISDN":"254712345678","FirstName":"JANE","MiddleName":"","LastName":"DOE"}`
	var p daraja.C2BPayment
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	if p.Payer() != "JANE DOE" {
		t.Fatalf("payer = %q", p.Payer())
	}
	if p.Time().IsZero() {
		t.Fatal("TransTime did not parse")
	}
}
