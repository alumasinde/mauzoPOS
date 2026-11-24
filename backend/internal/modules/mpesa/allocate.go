package mpesa

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/internal/modules/sales"
)

// suggestionWindow is how far back we look for sales a collection might fund.
const suggestionWindow = 14 * 24 * time.Hour

// SalesDep is what the mpesa module needs from sales — one direction only
// (sales never imports mpesa; it consumes the MpesaConsumer interface).
type SalesDep interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (sales.SaleDetail, error)
	LookupBrief(ctx context.Context, tenantID, saleID uint64) (sales.SaleBrief, bool)
	MpesaPayments(ctx context.Context, tenantID uint64, since time.Time) ([]sales.MpesaCandidate, error)
	Authz(ctx context.Context, tenantID, userID uint64) (*sales.CashierAuthz, error)
	MarkMpesaVerified(ctx context.Context, tenantID, saleID uint64, transID string) error
}

// BindSales wires the sales dependency after construction (container order).
func (s *Service) BindSales(dep SalesDep) { s.sales = dep }

// verifyAllocation enforces the rules from the design: a collection can only
// fund a sale when the money lines up — never on amount alone.
//   - txn is a confirmed success, not already allocated
//   - sale exists, completed (or partially returned), same tenant
//   - branch matches when the txn carries one
//   - the sale has an mpesa payment line whose received amount (tendered,
//     else applied) equals this transaction, and whose reference is empty,
//     already cites this transaction, or is an unverified offline entry
func (s *Service) verifyAllocation(ctx context.Context, tenantID uint64,
	t *Transaction, salePublicID string) (*sales.SaleDetail, error) {
	if t.Status != StSuccess || t.TransID == nil {
		return nil, ErrNotConfirmed
	}
	if t.AllocatedSaleID != nil {
		return nil, ErrAlreadyAllocated
	}
	detail, err := s.sales.Get(ctx, tenantID, salePublicID)
	if err != nil {
		return nil, ErrNoTransaction
	}
	if detail.Sale.Status != sales.StatusCompleted && detail.Sale.Status != sales.StatusPartiallyReturned {
		return nil, ErrSaleNotPayable
	}
	if t.BranchID != nil && *t.BranchID != detail.Sale.BranchID {
		return nil, ErrBranchMismatch
	}
	if !anyLineMatches(detail.Payments, t) {
		return nil, ErrAmountMismatch
	}
	return &detail, nil
}

// Allocate links a confirmed collection to a sale after verification.
func (s *Service) Allocate(ctx context.Context, tenantID, actorID uint64,
	txnPublicID, salePublicID string) error {
	t, err := s.repo.ByPublic(ctx, tenantID, txnPublicID)
	if err != nil || t == nil {
		return ErrNoTransaction
	}
	detail, err := s.verifyAllocation(ctx, tenantID, t, salePublicID)
	if err != nil {
		return err
	}
	if err := s.repo.Allocate(ctx, t.ID, detail.Sale.ID, actorID); err != nil {
		return err
	}
	if err := s.sales.MarkMpesaVerified(ctx, tenantID, detail.Sale.ID, deref(t.TransID)); err != nil {
		return err
	}
	LogEvent(ctx, s.conn, tenantID, &t.ID, EvAllocation, deref(t.TransID),
		fmt.Sprintf("allocated to sale #%d by %d", detail.Sale.ReceiptNo, actorID), nil)
	if s.audit != nil {
		s.audit.Log(ctx, auditEntry(tenantID, actorID, "mpesa.txn.allocated", t.PublicID))
	}
	return nil
}

// Reallocate moves a collection between sales — a sensitive, separately-
// permissioned op used to fix mis-keyed allocations.
func (s *Service) Reallocate(ctx context.Context, tenantID, actorID uint64,
	txnPublicID, salePublicID string) error {
	t, err := s.repo.ByPublic(ctx, tenantID, txnPublicID)
	if err != nil || t == nil {
		return ErrNoTransaction
	}
	if t.Status != StSuccess && t.Status != StRevPending {
		return ErrNotConfirmed
	}
	detail, err := s.sales.Get(ctx, tenantID, salePublicID)
	if err != nil {
		return ErrNoTransaction
	}
	if detail.Sale.Status != sales.StatusCompleted && detail.Sale.Status != sales.StatusPartiallyReturned {
		return ErrSaleNotPayable
	}
	if !t.Amount.Equal(mpesaLineTotal(detail.Payments)) && !anyLineMatches(detail.Payments, t) {
		return ErrAmountMismatch
	}
	if err := s.repo.Reallocate(ctx, t.ID, detail.Sale.ID, actorID); err != nil {
		return err
	}
	LogEvent(ctx, s.conn, tenantID, &t.ID, EvAllocation, deref(t.TransID),
		fmt.Sprintf("reallocated to sale #%d by %d", detail.Sale.ReceiptNo, actorID), nil)
	if s.audit != nil {
		s.audit.Log(ctx, auditEntry(tenantID, actorID, "mpesa.txn.reallocated", t.PublicID))
	}
	return nil
}

// Suggestion pairs an unallocated collection with candidate sales.
type Suggestion struct {
	Txn       *Transaction          `json:"txn"`
	Candidate *sales.MpesaCandidate `json:"candidate"`
	Strength  string                `json:"strength"` // strong = receipt cites the ref; weak = amount-only
}

// Suggestions returns, per unallocated transaction, the sales it probably
// funded. Amount matches are surfaced as weak — a human always confirms.
func (s *Service) Suggestions(ctx context.Context, tenantID uint64) ([]Suggestion, error) {
	txns, err := s.repo.Unallocated(ctx, tenantID, time.Now().Add(-suggestionWindow))
	if err != nil {
		return nil, err
	}
	candidates, err := s.sales.MpesaPayments(ctx, tenantID, time.Now().Add(-suggestionWindow))
	if err != nil {
		return nil, err
	}
	saleIDs := make([]uint64, len(candidates))
	for i := range candidates {
		saleIDs[i] = candidates[i].SaleID
	}
	claimed, err := s.repo.AllocatedSaleIDs(ctx, tenantID, saleIDs)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		candidates[i].Allocated = claimed[candidates[i].SaleID]
	}
	out := []Suggestion{}
	for i := range txns {
		t := &txns[i]
		for j := range candidates {
			c := &candidates[j]
			if c.Allocated || !c.MpesaAmount.Equal(t.Amount) {
				continue
			}
			strength := "weak"
			if t.TransID != nil && c.Reference == *t.TransID {
				strength = "strong"
			}
			if c.Unverified && strength == "weak" {
				strength = "unverified" // offline entry awaiting its transaction
			}
			if t.BranchID != nil && *t.BranchID != c.BranchID {
				continue
			}
			out = append(out, Suggestion{Txn: t, Candidate: c, Strength: strength})
		}
	}
	return out, nil
}

// received is what the customer sent on a line: tendered when recorded
// (overpayments handed back as change), else the applied amount.
func received(p sales.SalePayment) decimal.Decimal {
	if p.Tendered != nil && p.Tendered.IsPositive() {
		return *p.Tendered
	}
	return p.Amount
}

// mpesaLineTotal sums positive mpesa tender received on a sale.
func mpesaLineTotal(payments []sales.SalePayment) decimal.Decimal {
	sum := decimal.Zero
	for _, p := range payments {
		if p.Method == "mpesa" && p.Amount.IsPositive() {
			sum = sum.Add(received(p))
		}
	}
	return sum
}

func anyLineMatches(payments []sales.SalePayment, t *Transaction) bool {
	for _, p := range payments {
		if p.Method != "mpesa" || !p.Amount.IsPositive() || !received(p).Equal(t.Amount) {
			continue
		}
		if p.Reference == "" || p.Status == sales.PayUnverified ||
			(t.TransID != nil && p.Reference == *t.TransID) {
			return true
		}
	}
	return false
}
