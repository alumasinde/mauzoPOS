package sales

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// MpesaConsumer is the seam the mpesa module fills. sales declares it (never
// imports mpesa) so a completed sale can atomically claim the confirmed
// transaction its payment reference cites, inside the same DB transaction.
type MpesaConsumer interface {
	// ConsumeInTx marks a confirmed M-Pesa transaction as funding this sale.
	// Fails (rolling the whole sale back) when the transaction is missing,
	// unconfirmed, already allocated, or a different amount.
	ConsumeInTx(ctx context.Context, tx *sqlx.Tx, tenantID uint64,
		transID string, amount decimal.Decimal, saleID uint64) error
	// ReleaseInTx frees transactions allocated to a sale (void path).
	ReleaseInTx(ctx context.Context, tx *sqlx.Tx, saleID uint64) error
}

// BindMpesa wires the consumer after construction (the mpesa module registers
// itself; when it is disabled the payment line still records with its
// reference but nothing is consumed).
func (s *Service) BindMpesa(c MpesaConsumer) { s.mpesa = c }

// consumeMpesa claims the transactions cited by mpesa payment lines, inside
// the caller's transaction.
func (s *Service) consumeMpesa(ctx context.Context, tx *sqlx.Tx, tenantID uint64,
	payments []SalePayment, saleID uint64) error {
	if s.mpesa == nil {
		return nil
	}
	for _, p := range payments {
		if p.Method != "mpesa" || p.Reference == "" || !p.Amount.IsPositive() {
			continue
		}
		if err := s.mpesa.ConsumeInTx(ctx, tx, tenantID, p.Reference, p.Amount, saleID); err != nil {
			return err
		}
	}
	return nil
}

// releaseMpesa frees allocations when a sale is voided.
func (s *Service) releaseMpesa(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	if s.mpesa == nil {
		return nil
	}
	return s.mpesa.ReleaseInTx(ctx, tx, saleID)
}

// MpesaCandidate is a sale carrying mpesa tender — the reconciliation screen
// matches unallocated collections against these.
type MpesaCandidate struct {
	SaleID      uint64          `db:"sale_id" json:"sale_id"`
	PublicID    string          `db:"public_id" json:"public_id"`
	ReceiptNo   uint64          `db:"receipt_no" json:"receipt_no"`
	BranchID    uint64          `db:"branch_id" json:"branch_id"`
	BranchName  string          `db:"branch_name" json:"branch_name"`
	CashierName string          `db:"cashier_name" json:"cashier_name"`
	Total       decimal.Decimal `db:"total" json:"total"`
	MpesaAmount decimal.Decimal `db:"mpesa_amount" json:"mpesa_amount"`
	Reference   string          `db:"reference" json:"reference"`
	SoldAt      time.Time       `db:"sold_at" json:"sold_at"`
	Allocated   bool            `db:"allocated" json:"allocated"`
}

// MpesaPayments lists recent completed sales that took mpesa tender, flagged
// when a transaction already claims them.
func (s *Service) MpesaPayments(ctx context.Context, tenantID uint64, since time.Time) ([]MpesaCandidate, error) {
	return s.repo.MpesaCandidates(ctx, tenantID, since)
}

// SaleBrief is a minimal sale reference for cross-module display.
type SaleBrief struct {
	PublicID  string `db:"public_id" json:"public_id"`
	ReceiptNo uint64 `db:"receipt_no" json:"receipt_no"`
}

// LookupBrief resolves an internal sale id to its public reference.
func (s *Service) LookupBrief(ctx context.Context, tenantID, saleID uint64) (SaleBrief, bool) {
	b, err := s.repo.SaleBrief(ctx, tenantID, saleID)
	if err != nil || b == nil {
		return SaleBrief{}, false
	}
	return *b, true
}
