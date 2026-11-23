package sales

import (
	"context"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/inventory"
)

// LineInput is one line the client submits. Prices are recomputed
// server-side; override/discounts are applied only within the acting user's
// permissions (a supervisor grant is required beyond role limits).
type LineInput struct {
	ProductID     string  `json:"product_id"` // public id
	VariantID     string  `json:"variant_id"` // public id, optional
	Qty           string  `json:"qty"`
	PriceOverride *string `json:"price_override"` // final unit price — needs pos.price.override
	Discount      string  `json:"discount"`       // manual absolute discount
}

// PaymentInput is one tendered payment (multiple rows = split payment).
type PaymentInput struct {
	Method    string  `json:"method"`
	Amount    string  `json:"amount"`              // applied to the sale
	Tendered  *string `json:"tendered,omitempty"`  // cash only
	Reference string  `json:"reference,omitempty"` // card/bank
}

// CompleteInput is everything needed to record a sale.
type CompleteInput struct {
	TenantID      uint64
	BranchID      uint64
	TerminalID    *uint64
	CashierID     *uint64 // user who rang it (device sign-in or actor)
	CustomerID    string  // public id, optional
	CustomerGroup string

	IdempotencyKey string
	Source         string
	SoldAt         time.Time
	Lines          []LineInput
	OrderDiscount  string
	Payments       []PaymentInput

	// Authorization: the acting user's role perms/limits are loaded by the
	// service. SupervisorID marks elevated operations authorized by a
	// supervisor grant (POS) or left nil.
	ActorID      uint64
	ActorType    string
	SupervisorID *uint64
	DiscountNote string
}

// line is a resolved line mid-pipeline.
type line struct {
	product    catalog.Product
	variantID  uint64
	variantPub string
	name       string
	sku        string
	qty        decimal.Decimal
	unit       decimal.Decimal // after list price / override, before discounts
	promo      string
	override   bool
	discount   decimal.Decimal // promo + manual + order share
	total      decimal.Decimal // qty*unit - discount (2dp)
	tax        decimal.Decimal
	taxRate    *decimal.Decimal
}

// Complete records a sale atomically: lines + payments + stock decrement +
// audit row in one transaction. Idempotent on IdempotencyKey — a retried
// upload returns the original sale with deduplicated=true.
func (s *Service) Complete(ctx context.Context, in CompleteInput) (SaleDetail, bool, error) {
	if in.IdempotencyKey != "" {
		if dup, err := s.repo.ByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey); err != nil {
			return SaleDetail{}, false, err
		} else if dup != nil {
			d, err := s.detail(ctx, *dup)
			return d, true, err
		}
	}
	if len(in.Lines) == 0 {
		return SaleDetail{}, false, ErrNoLines
	}
	if len(in.Payments) == 0 {
		return SaleDetail{}, false, ErrNoPayments
	}
	if in.Source == "" {
		in.Source = SourcePOS
	}
	if in.SoldAt.IsZero() {
		in.SoldAt = time.Now()
	}
	// The acting user's permissions and role limits govern overrides and
	// discounts server-side — the PIN screen is not the security boundary.
	authz, err := s.cashierAuthz(ctx, in)
	if err != nil {
		return SaleDetail{}, false, err
	}

	lines, err := s.resolveLines(ctx, in, authz)
	if err != nil {
		return SaleDetail{}, false, err
	}
	if err := s.applyPromos(ctx, in.TenantID, in.SoldAt, lines); err != nil {
		return SaleDetail{}, false, err
	}
	if err := applyManualDiscounts(in.Lines, lines, authz, in.SupervisorID != nil); err != nil {
		return SaleDetail{}, false, err
	}
	subtotal, discount, tax, total, err := finalizeLines(in.OrderDiscount, lines, authz, in.SupervisorID != nil)
	if err != nil {
		return SaleDetail{}, false, err
	}

	customer, err := s.resolveCustomer(ctx, in)
	if err != nil {
		return SaleDetail{}, false, err
	}
	var customerID *uint64
	if customer != nil {
		customerID = &customer.ID
	}
	payments, change, err := s.validatePayments(ctx, in.TenantID, customer, total, in.Payments)
	if err != nil {
		return SaleDetail{}, false, err
	}

	sale := Sale{
		TenantID: in.TenantID, BranchID: in.BranchID, TerminalID: in.TerminalID,
		CashierID: in.CashierID, CustomerID: customerID,
		PublicID: security.NewPublicID(), Status: StatusCompleted, Source: in.Source,
		Currency: "KES", Subtotal: subtotal, Discount: discount, Tax: tax, Total: total,
		Paid: total, ChangeDue: change, DiscountNote: in.DiscountNote, SoldAt: in.SoldAt,
	}
	if in.IdempotencyKey != "" {
		sale.IdemKey = &in.IdempotencyKey
	}

	if err := s.insertSale(ctx, in, &sale, lines, payments); err != nil {
		if errors.Is(err, ErrDuplicateSale) && in.IdempotencyKey != "" {
			// Concurrent retry won the insert — return its sale.
			dup, derr := s.repo.ByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey)
			if derr == nil && dup != nil {
				d, derr := s.detail(ctx, *dup)
				return d, true, derr
			}
		}
		return SaleDetail{}, false, err
	}
	d, err := s.detail(ctx, sale)
	return d, false, err
}

// cashierAuthz loads the acting user's permissions and role limits.
func (s *Service) cashierAuthz(ctx context.Context, in CompleteInput) (*CashierAuthz, error) {
	id := in.ActorID
	if in.CashierID != nil {
		id = *in.CashierID
	}
	a, err := s.repo.CashierAuthz(ctx, in.TenantID, id)
	if err != nil {
		return nil, ErrCashierNotFound
	}
	return a, nil
}

// insertSale is the transactional half of Complete.
func (s *Service) insertSale(ctx context.Context, in CompleteInput, sale *Sale,
	lines []line, payments []SalePayment) error {
	return db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		no, err := s.repo.NextCounter(ctx, tx, sale.TenantID, counterReceipt)
		if err != nil {
			return err
		}
		sale.ReceiptNo = no
		if err := s.repo.InsertSale(ctx, tx, sale); err != nil {
			if isDup(err) {
				return ErrDuplicateSale
			}
			return err
		}
		stockItems := make([]inventory.LineRequest, 0, len(lines))
		for i, l := range lines {
			it := SaleItem{
				TenantID: sale.TenantID, SaleID: sale.ID, LineNo: i + 1,
				ProductID: l.product.ID, Name: l.name, SKU: l.sku, Qty: l.qty,
				UnitPrice: l.unit, Discount: l.discount, TaxRate: l.taxRate, Tax: l.tax,
				Total: l.total, PriceOverride: l.override, Promotion: l.promo,
			}
			if l.variantID != 0 {
				it.VariantID = &l.variantID
			}
			if err := s.repo.InsertItem(ctx, tx, &it); err != nil {
				return err
			}
			if l.product.TrackStock && l.product.Kind != catalog.KindService {
				req := inventory.LineRequest{ProductID: l.product.PublicID, Qty: l.qty.String()}
				if l.variantPub != "" {
					req.VariantID = l.variantPub
				}
				stockItems = append(stockItems, req)
			}
		}
		for i := range payments {
			payments[i].SaleID = sale.ID
			if err := s.repo.InsertPayment(ctx, tx, &payments[i]); err != nil {
				return err
			}
		}
		if len(stockItems) > 0 {
			if err := s.stock.PostStockInTx(ctx, tx, sale.TenantID, in.ActorID, sale.BranchID,
				"sale", "sale", sale.PublicID, "", "", stockItems, -1); err != nil {
				return err
			}
		}
		return s.repo.InsertAudit(ctx, tx, sale.TenantID, sale.BranchID, in.ActorID,
			in.ActorType, "sale.completed", sale.PublicID, map[string]any{
				"receipt_no": sale.ReceiptNo, "total": sale.Total.String(),
				"payments": len(payments), "source": sale.Source,
				"supervisor_id": in.SupervisorID,
			})
	})
}
