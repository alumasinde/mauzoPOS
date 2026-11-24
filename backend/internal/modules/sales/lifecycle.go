package sales

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/inventory"
)

// Void cancels a completed sale: payments voided, stock restored, audit
// written — all in one transaction. Requires the sales.sale.void permission
// (enforced by route middleware) and a reason.
func (s *Service) Void(ctx context.Context, tenantID uint64, publicID, reason string, actorID uint64, actorType string) error {
	sale, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return ErrSaleNotFound
	}
	if sale.Status != StatusCompleted {
		return ErrNotVoidable
	}
	items, err := s.repo.Items(ctx, sale.ID)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if err := s.repo.MarkVoided(ctx, tx, sale.ID, actorID, reason); err != nil {
			return err
		}
		if err := s.repo.VoidPayments(ctx, tx, sale.ID); err != nil {
			return err
		}
		// Free any M-Pesa transactions this sale claimed so they can be
		// re-allocated (the money itself stays — reversal is a separate op).
		if err := s.releaseMpesa(ctx, tx, sale.ID); err != nil {
			return err
		}
		stockItems := restockItems(items)
		if len(stockItems) > 0 {
			if err := s.stock.PostStockInTx(ctx, tx, tenantID, actorID, sale.BranchID,
				"void", "sale", sale.PublicID, "", "sale voided", stockItems, +1); err != nil {
				return err
			}
		}
		return s.repo.InsertAudit(ctx, tx, tenantID, sale.BranchID, actorID, actorType,
			"sale.voided", sale.PublicID, map[string]any{
				"receipt_no": sale.ReceiptNo, "total": sale.Total.String(), "reason": reason,
			})
	})
}

// returnLine is a validated return row mid-pipeline.
type returnLine struct {
	item   SaleItem
	qty    decimal.Decimal
	amount decimal.Decimal
}

// resolveReturnLines validates the requested lines against the sale, sums the
// refund and enforces the acting user's refund limit (supervisor bypasses).
func (s *Service) resolveReturnLines(ctx context.Context, sale Sale, in ReturnInput) ([]returnLine, decimal.Decimal, error) {
	items, err := s.repo.Items(ctx, sale.ID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	byLine := map[int]SaleItem{}
	for _, it := range items {
		byLine[it.LineNo] = it
	}
	authz, err := s.repo.CashierAuthz(ctx, in.TenantID, in.ActorID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	rlist := make([]returnLine, 0, len(in.Lines))
	refund := decimal.Zero
	fields := map[string]any{}
	for i, li := range in.Lines {
		it, ok := byLine[li.LineNo]
		if !ok {
			fields["lines."+strconv.Itoa(i)+".line_no"] = "not on this sale"
			continue
		}
		q, qerr := decimal.NewFromString(li.Qty)
		if qerr != nil || q.LessThanOrEqual(decimal.Zero) {
			fields["lines."+strconv.Itoa(i)+".qty"] = "must be a positive number"
			continue
		}
		if q.GreaterThan(it.Qty.Sub(it.ReturnedQty)) {
			return nil, decimal.Zero, ErrOverReturn
		}
		// Per-unit effective price = line total / qty; refund qty × that.
		amt := it.Total.Div(it.Qty).Mul(q).Round(2)
		refund = refund.Add(amt)
		rlist = append(rlist, returnLine{item: it, qty: q, amount: amt})
	}
	if len(fields) > 0 {
		return nil, decimal.Zero, httpx.ErrValidation.WithDetails(fields)
	}
	if authz.MaxRefund != nil && in.SupervisorID == nil {
		if lim, perr := decimal.NewFromString(*authz.MaxRefund); perr == nil && refund.GreaterThan(lim) {
			return nil, decimal.Zero, ErrRefundTooLarge
		}
	}
	return rlist, refund, nil
}

// restockItems builds inbound lines for the unsold remainder of each tracked
// line.
func restockItems(items []SaleItem) []inventory.LineRequest {
	out := make([]inventory.LineRequest, 0, len(items))
	for _, it := range items {
		if !it.TrackStock {
			continue
		}
		qty := it.Qty.Sub(it.ReturnedQty)
		if !qty.IsPositive() {
			continue
		}
		out = append(out, stockLineFor(it, qty))
	}
	return out
}

// stockLineFor rebuilds a stock posting line for a sale item.
func stockLineFor(it SaleItem, qty decimal.Decimal) inventory.LineRequest {
	return inventory.LineRequest{
		ProductID: it.ProductPublicID, VariantID: it.VariantPublicID, Qty: qty.String(),
	}
}

// ReturnLineInput is one returned line, addressed by its sale line_no.
type ReturnLineInput struct {
	LineNo  int    `json:"line_no"`
	Qty     string `json:"qty"`
	Restock bool   `json:"restock"`
}

// ReturnInput processes a refund against a sale.
type ReturnInput struct {
	TenantID     uint64
	SalePublicID string
	Reason       string
	RefundMethod string
	Lines        []ReturnLineInput
	ActorID      uint64
	ActorType    string
	SupervisorID *uint64
}

// CreateReturn records a refund: return header + lines, returned_qty bumps,
// restock movements, refund payment row and audit — one transaction.
func (s *Service) CreateReturn(ctx context.Context, in ReturnInput) (SaleReturn, error) {
	sale, err := s.repo.ByPublicID(ctx, in.TenantID, in.SalePublicID)
	if err != nil {
		return SaleReturn{}, ErrSaleNotFound
	}
	if sale.Status != StatusCompleted && sale.Status != StatusPartiallyReturned {
		return SaleReturn{}, ErrNotReturnable
	}
	if len(in.Lines) == 0 {
		return SaleReturn{}, httpx.ErrValidation.WithDetails(map[string]any{"lines": "required"})
	}
	if _, ok := methods[in.RefundMethod]; !ok {
		return SaleReturn{}, ErrUnknownMethod
	}
	rlist, refund, err := s.resolveReturnLines(ctx, sale, in)
	if err != nil {
		return SaleReturn{}, err
	}

	rt := SaleReturn{
		TenantID: in.TenantID, SaleID: sale.ID, PublicID: security.NewPublicID(),
		Reason: in.Reason, RefundMethod: in.RefundMethod, RefundAmount: refund,
		Status: "completed", SupervisorID: in.SupervisorID, CreatedBy: &in.ActorID,
	}
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		no, cerr := s.repo.NextCounter(ctx, tx, in.TenantID, counterReturn)
		if cerr != nil {
			return cerr
		}
		rt.ReturnNo = no
		if cerr := s.repo.InsertReturn(ctx, tx, &rt); cerr != nil {
			return cerr
		}
		stockItems := make([]inventory.LineRequest, 0, len(rlist))
		for i, r := range rlist {
			ri := SaleReturnItem{
				TenantID: in.TenantID, ReturnID: rt.ID, SaleItemID: r.item.ID,
				Qty: r.qty, Restock: in.Lines[i].Restock, Amount: r.amount,
			}
			if cerr := s.repo.InsertReturnItem(ctx, tx, &ri); cerr != nil {
				return cerr
			}
			if cerr := s.repo.AddReturnedQty(ctx, tx, r.item.ID, r.qty.String()); cerr != nil {
				return cerr
			}
			if in.Lines[i].Restock && r.item.TrackStock {
				stockItems = append(stockItems, stockLineFor(r.item, r.qty))
			}
		}
		if len(stockItems) > 0 {
			if cerr := s.stock.PostStockInTx(ctx, tx, in.TenantID, in.ActorID, sale.BranchID,
				"return", "sale_return", rt.PublicID, "", in.Reason, stockItems, +1); cerr != nil {
				return cerr
			}
		}
		// Refund outflow recorded as a negative payment so method summaries net.
		neg := refund.Neg()
		if cerr := s.repo.InsertPayment(ctx, tx, &SalePayment{
			TenantID: in.TenantID, SaleID: sale.ID, Method: in.RefundMethod,
			Amount: neg, Status: PayRefunded, Reference: "return:" + rt.PublicID, CreatedBy: &in.ActorID,
		}); cerr != nil {
			return cerr
		}
		if cerr := s.repo.RefreshReturnStatus(ctx, tx, sale.ID); cerr != nil {
			return cerr
		}
		return s.repo.InsertAudit(ctx, tx, in.TenantID, sale.BranchID, in.ActorID, in.ActorType,
			"sale.returned", rt.PublicID, map[string]any{
				"sale": sale.PublicID, "receipt_no": sale.ReceiptNo,
				"refund": refund.String(), "method": in.RefundMethod,
				"supervisor_id": in.SupervisorID,
			})
	})
	return rt, err
}
