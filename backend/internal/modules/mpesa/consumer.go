package mpesa

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// ConsumeInTx implements sales.MpesaConsumer — claims a confirmed transaction
// for a sale inside the sale's own DB transaction.
func (s *Service) ConsumeInTx(ctx context.Context, tx *sqlx.Tx, tenantID uint64,
	transID string, amount decimal.Decimal, saleID uint64) error {
	t, err := ConsumeTx(ctx, tx, tenantID, transID, amount, saleID)
	if err != nil {
		return err
	}
	LogEvent(ctx, tx, tenantID, &t.ID, EvAllocation, transID,
		"consumed at sale completion", nil)
	return nil
}

// ReleaseInTx implements sales.MpesaConsumer — frees allocations on void.
func (s *Service) ReleaseInTx(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	return ReleaseTx(ctx, tx, saleID)
}
