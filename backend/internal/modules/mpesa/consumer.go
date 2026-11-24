package mpesa

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// ConsumeInTx implements sales.MpesaConsumer — claims a confirmed transaction
// for a sale inside the sale's own DB transaction. received is what the
// customer sent; it must equal the transaction amount exactly.
//
// lenient is set for sales a till completed offline: if the code cannot be
// matched (unknown, unconfirmed, already used, other amount) the sale is
// kept with the payment unverified and nothing is consumed, so the goods
// already handed over stay on record and reconciliation resolves it.
func (s *Service) ConsumeInTx(ctx context.Context, tx *sqlx.Tx, tenantID uint64,
	transID string, received decimal.Decimal, saleID uint64, lenient bool) (bool, error) {
	t, err := ConsumeTx(ctx, tx, tenantID, transID, received, saleID)
	if err != nil {
		if lenient && isVerificationErr(err) {
			LogEvent(ctx, tx, tenantID, nil, EvAllocation, transID,
				"offline sale recorded with unverified code: "+err.Error(), nil)
			return false, nil
		}
		return false, err
	}
	LogEvent(ctx, tx, tenantID, &t.ID, EvAllocation, transID,
		"consumed at sale completion", nil)
	return true, nil
}

func isVerificationErr(err error) bool {
	return errors.Is(err, ErrNoTransaction) || errors.Is(err, ErrNotConfirmed) ||
		errors.Is(err, ErrAlreadyAllocated) || errors.Is(err, ErrAmountMismatch)
}

// ReleaseInTx implements sales.MpesaConsumer — frees allocations on void.
func (s *Service) ReleaseInTx(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	return ReleaseTx(ctx, tx, saleID)
}
