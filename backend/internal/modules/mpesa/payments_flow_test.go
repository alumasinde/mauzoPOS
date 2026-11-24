package mpesa_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/modules/branches"
	"mauzo/internal/modules/mpesa"
	"mauzo/internal/modules/sales"
)

func c2bTo(shortCode, transID, amount string) []byte {
	b, _ := json.Marshal(map[string]any{
		"TransactionType": "Customer Buy Goods", "TransID": transID, "TransTime": "20250101120000",
		"TransAmount": amount, "BusinessShortCode": shortCode, "MSISDN": "254712345678",
		"FirstName": "JANE", "LastName": "DOE",
	})
	return b
}

// A payment to a branch's till number reaches only that branch's tills; an
// unmapped number is visible everywhere; phone numbers are masked.
func TestFeedIsScopedToBranchAndMasked(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	other, err := branches.NewService(f.conn, branches.NewRepo(f.conn)).Create(f.ctx, f.tenantID,
		branches.CreateRequest{Code: "WEST", Name: "Westlands"})
	if err != nil {
		t.Fatal(err)
	}
	var mainPub string
	if err := f.conn.GetContext(f.ctx, &mainPub, `SELECT public_id FROM branches WHERE id = ?`, f.branchID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SetBranchTills(f.ctx, f.tenantID, []mpesa.BranchTillInput{
		{ShortCode: "111111", BranchID: mainPub}, {ShortCode: "222222", BranchID: other.PublicID},
	}); err != nil {
		t.Fatal(err)
	}
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("111111", "MAIN000001", "100.00"))
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("222222", "WEST000001", "200.00"))
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("999999", "ANYW000001", "300.00"))

	rows, err := f.svc.FeedForTill(f.ctx, f.tenantID, f.branchID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range rows {
		got[*r.TransID] = true
		if !strings.Contains(r.Phone, "•••") {
			t.Errorf("phone not masked: %q", r.Phone)
		}
	}
	if !got["MAIN000001"] || !got["ANYW000001"] || got["WEST000001"] {
		t.Fatalf("main branch feed = %v", got)
	}
	if _, err := f.svc.SetBranchTills(f.ctx, f.tenantID, []mpesa.BranchTillInput{
		{ShortCode: "12a", BranchID: mainPub}}); err == nil {
		t.Fatal("non-digit short code accepted")
	}
}

// Customers round up: 1,100 sent for a 1,090 bill. The sale applies 1,090,
// records 1,100 received and 10 change, and consumes the transaction.
func TestOverpaymentBecomesChange(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("600999", "OVER000001", "1100.00"))
	saleID, _ := f.sale(t, "1090", "OVER000001")
	err := db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		ok, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "OVER000001", decimal.NewFromInt(1100), saleID, false)
		if err == nil && !ok {
			t.Error("not verified")
		}
		return err
	})
	if err != nil {
		t.Fatalf("overpayment consume: %v", err)
	}
	// Received less than the transaction is still a mismatch.
	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("600999", "OVER000002", "500.00"))
	err = db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		_, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "OVER000002", decimal.NewFromInt(450), saleID, false)
		return err
	})
	if err == nil {
		t.Fatal("received amount different from the transaction was accepted")
	}
}

// An offline till's typed code that does not match is recorded as
// unverified (the sale stands); allocating the real payment verifies it.
func TestOfflineUnverifiedThenAllocated(t *testing.T) {
	f := newFixture(t)
	f.saveCreds(t)
	saleID, salePub := f.sale(t, "700", "")
	err := db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		ok, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "TYPO000001", decimal.NewFromInt(700), saleID, true)
		if ok {
			t.Error("unknown code reported verified")
		}
		return err
	})
	if err != nil {
		t.Fatalf("lenient consume must not fail: %v", err)
	}
	if _, err := f.conn.ExecContext(f.ctx,
		`UPDATE sale_payments SET status = ?, reference = 'TYPO000001' WHERE sale_id = ?`,
		sales.PayUnverified, saleID); err != nil {
		t.Fatal(err)
	}
	// Online, the same mismatch is refused.
	err = db.WithTx(f.ctx, f.conn, func(tx *sqlx.Tx) error {
		_, err := f.svc.ConsumeInTx(f.ctx, tx, f.tenantID, "TYPO000001", decimal.NewFromInt(700), saleID, false)
		return err
	})
	if err == nil {
		t.Fatal("online sale accepted an unknown code")
	}

	f.svc.HandleC2BConfirmation(f.ctx, f.tenantID, c2bTo("600999", "REAL000001", "700.00"))
	txn, _ := f.repo.ByTransID(f.ctx, f.tenantID, "REAL000001")
	sugg, err := f.svc.Suggestions(f.ctx, f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sugg) == 0 || sugg[0].Strength != "unverified" {
		t.Fatalf("suggestions %+v", sugg)
	}
	if err := f.svc.Allocate(f.ctx, f.tenantID, f.cashierID, txn.PublicID, salePub); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	var status, ref string
	if err := f.conn.QueryRowxContext(f.ctx,
		`SELECT status, reference FROM sale_payments WHERE sale_id = ?`, saleID).Scan(&status, &ref); err != nil {
		t.Fatal(err)
	}
	if status != sales.PayConfirmed || ref != "REAL000001" {
		t.Fatalf("payment after allocation: %s %s", status, ref)
	}
}

func TestMaskPhone(t *testing.T) {
	for in, want := range map[string]string{
		"254712345678": "0712•••678", "0712345678": "0712•••678", "+254112345678": "0112•••678",
	} {
		if got := mpesa.MaskPhone(in); got != want {
			t.Errorf("MaskPhone(%q) = %q, want %q", in, got, want)
		}
	}
}
