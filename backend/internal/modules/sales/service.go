package sales

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/catalog"
	"mauzo/internal/modules/customers"
	"mauzo/internal/modules/inventory"
	"mauzo/internal/modules/pricing"
	"mauzo/pkg/pricecalc"
)

// Dependencies — interfaces keep the module testable and the wiring explicit.

// Catalog resolves products by public id (with variants).
type Catalog interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (catalog.Product, error)
}

// Pricer resolves the branch/list unit price for a product at qty.
type Pricer interface {
	Resolve(ctx context.Context, tenantID, branchID uint64, group string,
		p catalog.Product, variantID uint64, qty decimal.Decimal) (pricing.ResolvedPrice, error)
}

// Promos returns promotions live at sale time.
type Promos interface {
	ActivePromotions(ctx context.Context, tenantID uint64) ([]pricecalc.Promotion, error)
}

// Stock posts inventory movements inside the caller's transaction.
type Stock interface {
	PostStockInTx(ctx context.Context, tx *sqlx.Tx, tenantID, actorID, branchID uint64,
		kind, refType, refID, reason, note string, items []inventory.LineRequest, sign int) error
}

// CustomerLookup resolves customers and their outstanding credit.
type CustomerLookup interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (customers.Customer, error)
	OutstandingCredit(ctx context.Context, tenantID, customerID uint64) (decimal.Decimal, error)
}

// Service owns sale business rules.
type Service struct {
	conn      *sqlx.DB
	repo      *Repo
	catalog   Catalog
	pricer    Pricer
	promos    Promos
	stock     Stock
	customers CustomerLookup
	mpesa     MpesaConsumer
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo, c Catalog, p Pricer, pr Promos, st Stock, cu CustomerLookup) *Service {
	return &Service{conn: conn, repo: repo, catalog: c, pricer: p, promos: pr, stock: st, customers: cu}
}

// Get returns a sale with items, payments and returns.
func (s *Service) Get(ctx context.Context, tenantID uint64, publicID string) (SaleDetail, error) {
	sale, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if err != nil {
		return SaleDetail{}, ErrSaleNotFound
	}
	return s.detail(ctx, sale)
}

// GetByReceipt returns a sale by receipt number or public id — the POS
// transaction lookup (device calls are already branch-scoped).
func (s *Service) GetByReceipt(ctx context.Context, tenantID, branchID uint64, ref string) (SaleDetail, error) {
	sale, err := s.repo.ByReceipt(ctx, tenantID, branchID, ref)
	if err != nil {
		return SaleDetail{}, ErrSaleNotFound
	}
	return s.detail(ctx, sale)
}

func (s *Service) detail(ctx context.Context, sale Sale) (SaleDetail, error) {
	items, err := s.repo.Items(ctx, sale.ID)
	if err != nil {
		return SaleDetail{}, err
	}
	payments, err := s.repo.Payments(ctx, sale.ID)
	if err != nil {
		return SaleDetail{}, err
	}
	returns, err := s.repo.Returns(ctx, sale.ID)
	if err != nil {
		return SaleDetail{}, err
	}
	return SaleDetail{Sale: sale, Items: items, Payments: payments, Returns: returns}, nil
}

// SaleDetail is a sale plus its child rows.
type SaleDetail struct {
	Sale     Sale
	Items    []SaleItem
	Payments []SalePayment
	Returns  []SaleReturn
}

// List pages sales under a filter (branch scope applied by the handler).
func (s *Service) List(ctx context.Context, tenantID uint64, f ListFilter, pg httpx.Page) ([]Sale, int, error) {
	return s.repo.List(ctx, tenantID, f, pg)
}

// ListReturns pages the returns register.
func (s *Service) ListReturns(ctx context.Context, tenantID uint64, pg httpx.Page) ([]SaleReturn, int, error) {
	return s.repo.ListReturns(ctx, tenantID, pg)
}

// Held lists parked carts across the tenant's tills.
func (s *Service) Held(ctx context.Context, tenantID uint64) ([]HeldCart, error) {
	return s.repo.Held(ctx, tenantID)
}

// Authz loads a user's role permissions and limits — used by the POS sale
// endpoint to re-check pos.sale.create server-side.
func (s *Service) Authz(ctx context.Context, tenantID, userID uint64) (*CashierAuthz, error) {
	return s.repo.CashierAuthz(ctx, tenantID, userID)
}

// Summarize returns totals for a filter window.
func (s *Service) Summarize(ctx context.Context, tenantID uint64, f ListFilter) (Summary, error) {
	return s.repo.Summarize(ctx, tenantID, f)
}

// Daily returns per-day totals for a filter window.
func (s *Service) Daily(ctx context.Context, tenantID uint64, f ListFilter) ([]DailyPoint, error) {
	return s.repo.Daily(ctx, tenantID, f)
}

// MethodTotals splits payments by method for a filter window.
func (s *Service) MethodTotals(ctx context.Context, tenantID uint64, f ListFilter) ([]MethodSplit, error) {
	return s.repo.MethodTotals(ctx, tenantID, f)
}
