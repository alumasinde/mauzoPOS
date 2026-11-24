package mpesa_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/mpesa"
	_ "mauzo/internal/modules/registerall"
	"mauzo/internal/modules/sales"
	"mauzo/internal/testutil"
	"mauzo/pkg/daraja"
)

type fixture struct {
	conn      *sqlx.DB
	svc       *mpesa.Service
	salesSvc  *sales.Service
	repo      *mpesa.Repo
	tenantID  uint64
	branchID  uint64
	cashierID uint64
	ctx       context.Context
	stub      *httptest.Server
	posts     map[string]any
	receiptNo int
}

// newFixture builds tenant + branch + cashier, wires the services, and points
// the Daraja client factory at a stub server driven by f.posts.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	conn := testutil.DB(t)
	testutil.Truncate(t, conn,
		"mpesa_reversals", "mpesa_events", "mpesa_transactions", "mpesa_credentials", "mpesa_branch_tills",
		"sale_payments", "sale_items", "sales", "tenant_counters",
		"user_branches", "users", "role_permissions", "roles",
		"branches", "tenants")
	ctx := context.Background()
	f := &fixture{conn: conn, ctx: ctx, posts: map[string]any{}}

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, status, business_type)
		 VALUES (?, 'MPesa Shop', 'mpesashop', 'active', 'retail')`, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if err := conn.GetContext(ctx, &f.tenantID,
		`SELECT id FROM tenants WHERE slug='mpesashop'`); err != nil {
		t.Fatal(err)
	}
	branchSvc := branches.NewService(conn, branches.NewRepo(conn))
	b, err := branchSvc.Create(ctx, f.tenantID, branches.CreateRequest{Code: "MAIN", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	f.branchID, err = branchSvc.BranchByPublicID(ctx, f.tenantID, b.PublicID)
	if err != nil {
		t.Fatal(err)
	}

	// Cashier with the mpesa + pos perms.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO roles (tenant_id, public_id, code, name, is_system)
		 VALUES (?, ?, 'cashier', 'Cashier', 0)`, f.tenantID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	var roleID uint64
	if err := conn.GetContext(ctx, &roleID,
		`SELECT id FROM roles WHERE tenant_id=? AND code='cashier'`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	perms := []string{
		"pos.sale.create", "mpesa.transaction.view", "mpesa.transaction.allocate",
		"mpesa.transaction.reallocate", "mpesa.config.manage",
	}
	testutil.EnsurePermissions(t, conn, perms...)
	for _, p := range perms {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)`, roleID, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO users (tenant_id, role_id, public_id, name, email, pin_hash, password_hash, status)
		 VALUES (?, ?, ?, 'Till Cashier', 'c@x.co', '', '', 'active')`,
		f.tenantID, roleID, security.NewPublicID()); err != nil {
		t.Fatal(err)
	}
	if err := conn.GetContext(ctx, &f.cashierID,
		`SELECT id FROM users WHERE tenant_id=? AND name='Till Cashier'`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO user_branches (tenant_id, user_id, branch_id) VALUES (?,?,?)`,
		f.tenantID, f.cashierID, f.branchID); err != nil {
		t.Fatal(err)
	}

	// sales service — only repo-backed methods are exercised here.
	f.salesSvc = sales.NewService(conn, sales.NewRepo(conn), nil, nil, nil, nil, nil)

	f.repo = mpesa.NewRepo(conn)
	svc, err := mpesa.NewService(conn, f.repo, "test-cred-key", "https://pos.example", nil)
	if err != nil {
		t.Fatal(err)
	}
	f.svc = svc
	mpesaSvc := svc // BindSales dep — satisfies mpesa.SalesDep
	mpesaSvc.BindSales(f.salesSvc)
	f.salesSvc.BindMpesa(mpesaSvc)

	// Daraja stub — oauth plus whatever /mpesa/* paths the test installs.
	f.stub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/oauth/v1/generate" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": "3599"})
			return
		}
		if resp, ok := f.posts[r.URL.Path]; ok {
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, `{"errorMessage":"unexpected path `+r.URL.Path+`"}`, http.StatusNotFound)
	}))
	t.Cleanup(f.stub.Close)
	svc.SetClientFactory(func(_, key, secret string) *daraja.Client {
		return daraja.New(f.stub.URL, key, secret, nil)
	})
	return f
}

// saveCreds installs sandbox credentials via the public save path.
func (f *fixture) saveCreds(t *testing.T) {
	t.Helper()
	if err := f.svc.SaveConfig(f.ctx, f.tenantID, mpesa.ConfigInput{
		Env: "sandbox", ShortCode: "600999", ShortCodeKind: "till",
		ConsumerKey: "ck", ConsumerSecret: "cs", Passkey: "pk",
		InitiatorName: "op", SecurityCredential: "cred",
	}, f.cashierID); err != nil {
		t.Fatal(err)
	}
}

// sale inserts a completed sale + one mpesa payment line.
func (f *fixture) sale(t *testing.T, amount string, ref string) (saleID uint64, salePub string) {
	t.Helper()
	salePub = security.NewPublicID()
	f.receiptNo++
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO sales (tenant_id, branch_id, cashier_id, public_id, receipt_no,
		   status, source, subtotal, discount, tax, total, paid, sold_at)
		 VALUES (?,?,?,?,?,'completed','pos',?,0,0,?,?,NOW(3))`,
		f.tenantID, f.branchID, f.cashierID, salePub, f.receiptNo, amount, amount, amount); err != nil {
		t.Fatal(err)
	}
	if err := f.conn.GetContext(f.ctx, &saleID,
		`SELECT id FROM sales WHERE public_id=?`, salePub); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`INSERT INTO sale_payments (tenant_id, sale_id, method, amount, reference, status)
		 VALUES (?,?,'mpesa',?,?,'confirmed')`, f.tenantID, saleID, amount, ref); err != nil {
		t.Fatal(err)
	}
	return saleID, salePub
}

// c2bBody builds a confirmation payload.
func c2bBody(transID, amount, phone, ref string) []byte {
	b, _ := json.Marshal(map[string]any{
		"TransactionType": "Customer Buy Goods", "TransID": transID, "TransTime": "20250101120000",
		"TransAmount": amount, "BusinessShortCode": "600999", "BillRefNumber": ref,
		"MSISDN": phone, "FirstName": "JANE", "LastName": "DOE",
	})
	return b
}

func stkBody(checkout string, code int, receipt string) []byte {
	items := []any{}
	if code == 0 {
		items = []any{
			map[string]any{"Name": "Amount", "Value": 1500},
			map[string]any{"Name": "MpesaReceiptNumber", "Value": receipt},
			map[string]any{"Name": "TransactionDate", "Value": 20250101120000},
			map[string]any{"Name": "PhoneNumber", "Value": 254712345678},
		}
	}
	b, _ := json.Marshal(map[string]any{"Body": map[string]any{"stkCallback": map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": checkout,
		"ResultCode": code, "ResultDesc": "result",
		"CallbackMetadata": map[string]any{"Item": items},
	}}})
	return b
}

// --- config -----------------------------------------------------------------

func TestSaveConfigMasksAndPreserves(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)

	v, err := f.svc.Config(f.ctx, f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if v.ShortCode != "600999" || v.ShortCodeKind != "till" {
		t.Fatalf("config = %+v", v)
	}
	if v.ConsumerSecret == "cs" || v.Passkey == "pk" {
		t.Fatal("secrets leaked in config view")
	}
	if !v.HasPasskey || !v.HasConsumerSecret {
		t.Fatal("has_* flags wrong")
	}
	if v.STKCallbackURL == "" || v.C2BConfirmationURL == "" {
		t.Fatal("callback URLs not generated")
	}

	// Re-save with masked values — stored secrets must survive.
	if err := f.svc.SaveConfig(f.ctx, f.tenantID, mpesa.ConfigInput{
		Env: "sandbox", ShortCode: "600999", ShortCodeKind: "till",
		ConsumerKey: v.ConsumerKey, ConsumerSecret: v.ConsumerSecret, Passkey: v.Passkey,
	}, f.cashierID); err != nil {
		t.Fatal(err)
	}
	// The push still works → passkey decrypted correctly.
	f.posts["/mpesa/stkpush/v1/processrequest"] = map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": "ws1",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	if _, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712345678", "SALE", "test", decimal.NewFromInt(100)); err != nil {
		t.Fatalf("push after masked re-save failed: %v", err)
	}
}

// --- STK lifecycle -------------------------------------------------------------

func TestSTKInitiateCallbackSuccess(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.posts["/mpesa/stkpush/v1/processrequest"] = map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": "ws1",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	txn, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712 345 678", "SALE", "test", decimal.NewFromInt(1500))
	if err != nil {
		t.Fatal(err)
	}
	if txn.Status != mpesa.StPending || txn.Phone != "254712345678" {
		t.Fatalf("txn = %+v", txn)
	}
	f.svc.HandleSTKCallback(f.ctx, f.tenantID, stkBody("ws1", 0, "QKR7ABC"))

	got, err := f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != mpesa.StSuccess || got.TransID == nil || *got.TransID != "QKR7ABC" {
		t.Fatalf("after callback: %+v", got)
	}
	if got.Phone != "254712345678" || got.TransTime == nil {
		t.Fatalf("metadata not merged: %+v", got)
	}
	// Duplicate delivery is a no-op.
	f.svc.HandleSTKCallback(f.ctx, f.tenantID, stkBody("ws1", 0, "QKR7ABC"))
	var n int
	if err := f.conn.GetContext(f.ctx, &n,
		`SELECT COUNT(*) FROM mpesa_events WHERE txn_id=?`, got.ID); err != nil {
		t.Fatal(err)
	}
	if n != 3 { // initiated + 2 callbacks (second logged but not reprocessed)
		t.Fatalf("events = %d", n)
	}
}

func TestSTKCallbackCancelled(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.posts["/mpesa/stkpush/v1/processrequest"] = map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": "ws2",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	txn, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712345678", "S", "", decimal.NewFromInt(50))
	if err != nil {
		t.Fatal(err)
	}
	f.svc.HandleSTKCallback(f.ctx, f.tenantID, stkBody("ws2", 1032, ""))
	got, _ := f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.Status != mpesa.StCancelled || got.TransID != nil {
		t.Fatalf("cancelled txn: %+v", got)
	}
}

func TestSTKStatusQueryFallback(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.posts["/mpesa/stkpush/v1/processrequest"] = map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": "ws3",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	f.posts["/mpesa/stkpushquery/v1/query"] = map[string]any{
		"ResponseCode": "0", "ResultCode": "0", "ResultDesc": "The service request is processed successfully.",
	}
	txn, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712345678", "S", "", decimal.NewFromInt(10))
	if err != nil {
		t.Fatal(err)
	}
	// Force the row old enough for the query window.
	if _, err := f.conn.ExecContext(f.ctx,
		`UPDATE mpesa_transactions SET created_at = UTC_TIMESTAMP(3) - INTERVAL 60 SECOND WHERE id=?`,
		txn.ID); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.STKStatus(f.ctx, f.tenantID, txn.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != mpesa.StSuccess {
		t.Fatalf("status after fallback: %s", got.Status)
	}
}

func TestInitiateSTKValidation(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	if _, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"1234", "S", "", decimal.NewFromInt(10)); err == nil {
		t.Fatal("bad phone accepted")
	}
	if _, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712345678", "S", "", decimal.NewFromFloat(0.5)); err == nil {
		t.Fatal("fractional amount accepted")
	}
}

// --- C2B feed ------------------------------------------------------------------

func TestC2BConfirmInsertsAndDedupes(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	body := c2bBody("QKR7C2B", "1500.00", "254712345678", "acct1")
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, body)

	got, err := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7C2B")
	if err != nil || got == nil {
		t.Fatal("c2b not stored")
	}
	if got.Status != mpesa.StSuccess || got.Kind != mpesa.KindC2B ||
		got.PayerName != "JANE DOE" || got.Amount.String() != "1500" {
		t.Fatalf("c2b txn: %+v", got)
	}
	// Same TransID again — no second row.
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, body)
	var n int
	if err := f.conn.GetContext(f.ctx, &n,
		`SELECT COUNT(*) FROM mpesa_transactions WHERE trans_id='QKR7C2B'`); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("dedupe failed: %d rows", n)
	}
}

func TestC2BMergesSTKRow(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.posts["/mpesa/stkpush/v1/processrequest"] = map[string]any{
		"MerchantRequestID": "m1", "CheckoutRequestID": "ws4",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	txn, err := f.svc.InitiateSTK(f.ctx, f.tenantID, f.branchID, f.cashierID,
		"0712345678", "S", "", decimal.NewFromInt(800))
	if err != nil {
		t.Fatal(err)
	}
	// Daraja fires C2B for STK payments too — the pending row must merge, not
	// duplicate. But the C2B carries the same TransID only when the STK
	// callback already stamped it; without it the confirmation is its own row.
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7M1", "800.00", "254712345678", "acct"))
	var n int
	if err := f.conn.GetContext(f.ctx, &n,
		`SELECT COUNT(*) FROM mpesa_transactions WHERE tenant_id=?`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if n != 2 { // STK row + C2B row — distinct because no shared trans_id
		t.Fatalf("rows = %d", n)
	}
	_ = txn
}

// --- allocation ----------------------------------------------------------------

func TestAllocateHappyPath(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7AL", "1200.00", "254700000001", ""))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7AL")
	_, salePub := f.sale(t, "1200", "")

	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn.PublicID, salePub); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.AllocatedSaleID == nil || got.AllocatedBy == nil || *got.AllocatedBy != f.cashierID {
		t.Fatalf("allocation: %+v", got)
	}
	// Second allocation is refused.
	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn.PublicID, salePub); err == nil {
		t.Fatal("double allocation allowed")
	}
}

func TestAllocateAmountMismatch(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7MM", "999.00", "254700000002", ""))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7MM")
	_, salePub := f.sale(t, "1200", "")
	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn.PublicID, salePub); err == nil {
		t.Fatal("amount mismatch allocated")
	}
}

func TestAllocateRefCited(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7RF", "700.00", "254700000003", ""))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7RF")
	// Payment line already cites this code — verification passes.
	_, salePub := f.sale(t, "700", "QKR7RF")
	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn.PublicID, salePub); err != nil {
		t.Fatal(err)
	}
	// A line citing a different code must not match.
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7RF2", "700.00", "254700000004", ""))
	txn2, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7RF2")
	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn2.PublicID, salePub); err == nil {
		t.Fatal("allocated to a sale whose line cites another code")
	}
}

// --- consume inside a tx --------------------------------------------------------

func TestConsumeAndReleaseInTx(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7CS", "500.00", "254700000005", ""))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7CS")
	saleID, _ := f.sale(t, "500", "QKR7CS")
	amt, _ := decimal.NewFromString("500")

	// Consume inside a committed tx — row becomes allocated.
	err := db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		_, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "QKR7CS", amt, saleID, false)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.AllocatedSaleID == nil || *got.AllocatedSaleID != saleID {
		t.Fatalf("consume: %+v", got)
	}
	// Rollback frees nothing when the tx fails.
	err = db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		if _, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "QKR7CS", amt, saleID+1, false); err == nil {
			return fmt.Errorf("expected already-allocated error")
		}
		return fmt.Errorf("forced rollback")
	})
	if err == nil || err.Error() != "forced rollback" {
		t.Fatalf("rollback tx: %v", err)
	}
	// Release frees the row again (void path).
	err = db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		return f.svc.ReleaseInTx(f.ctx, tx, saleID)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ = f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.AllocatedSaleID != nil {
		t.Fatal("release did not free the transaction")
	}
	// Wrong amount inside a tx fails cleanly.
	err = db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		_, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "QKR7CS", decimal.NewFromInt(999), saleID, false)
		return err
	})
	if err == nil {
		t.Fatal("amount mismatch consumed")
	}
}

// --- reversal ----------------------------------------------------------------

func TestReversalFlow(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7RV", "300.00", "254700000006", ""))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "QKR7RV")

	f.posts["/mpesa/reversal/v1/request"] = map[string]any{
		"OriginatorConversationID": "oc1", "ConversationID": "c1",
		"ResponseCode": "0", "ResponseDescription": "queued",
	}
	if err := f.svc.RequestReversal(f.ctx, f.tenantID, txn.ID, f.cashierID, "wrong till"); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.Status != mpesa.StRevPending {
		t.Fatalf("status = %s", got.Status)
	}
	// Daraja's async result marks it reversed.
	res, _ := json.Marshal(map[string]any{"Result": map[string]any{
		"ResultType": 0, "ResultCode": 0, "ResultDesc": "reversed",
		"OriginatorConversationID": "oc1", "ConversationID": "c1", "TransactionID": "QKR7RV",
	}})
	f.svc.HandleReversalResult(f.ctx, f.tenantID, res)
	got, _ = f.repo.ByPublic(f.ctx, f.tenantID, txn.PublicID)
	if got.Status != mpesa.StReversed {
		t.Fatalf("status after result = %s", got.Status)
	}
	var vs string
	if err := f.conn.GetContext(f.ctx, &vs,
		`SELECT status FROM mpesa_reversals WHERE originator_conversation_id='oc1'`); err != nil {
		t.Fatal(err)
	}
	if vs != "reversed" {
		t.Fatalf("reversal row = %s", vs)
	}
}

// --- token + phone ---------------------------------------------------------------

func TestCallbackTokenResolvesTenant(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	var token string
	if err := f.conn.GetContext(f.ctx, &token,
		`SELECT callback_token FROM mpesa_credentials WHERE tenant_id=?`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	if len(token) < 32 {
		t.Fatalf("weak callback token: %q", token)
	}
	id, err := f.repo.TenantByToken(f.ctx, token)
	if err != nil || id != f.tenantID {
		t.Fatalf("token resolved to %d", id)
	}
	if id, _ := f.repo.TenantByToken(f.ctx, "bogus"); id != 0 {
		t.Fatal("bogus token resolved")
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"0712345678":    "254712345678",
		"+254712345678": "254712345678",
		"254712345678":  "254712345678",
		"0112345678":    "254112345678",
		"0712 345 678":  "254712345678",
		"1234":          "",
		"0799":          "",
		"254812345678":  "", // non-safaricom prefix
	}
	for in, want := range cases {
		if got := mpesa.NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- suggestions ----------------------------------------------------------------

func TestSuggestionsStrongAndWeak(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7SG", "250.00", "254700000007", ""))
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bBody("QKR7SG2", "250.00", "254700000008", ""))
	// One sale cites the code (strong), another just matches on amount (weak).
	f.sale(t, "250", "QKR7SG")
	f.sale(t, "250", "")

	sug, err := f.svc.Suggestions(f.ctx, f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	byTxn := map[string]string{}
	for _, s := range sug {
		if s.Txn.TransID != nil {
			key := *s.Txn.TransID + "/" + s.Candidate.Reference
			byTxn[key] = s.Strength
		}
	}
	if byTxn["QKR7SG/QKR7SG"] != "strong" {
		t.Fatalf("strong match missing: %v", byTxn)
	}
	if byTxn["QKR7SG2/"] != "weak" && byTxn["QKR7SG/"] != "weak" {
		t.Fatalf("weak match missing: %v", byTxn)
	}
}
