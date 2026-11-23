package pricing

import (
	"context"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/catalog"
)

// Products resolves catalog products to internal ids, variants, categories
// and base prices.
type Products interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (catalog.Product, error)
}

// Branches resolves branch public ids within a tenant.
type Branches interface {
	BranchByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

// Service implements price lists, resolution and history.
type Service struct {
	conn     *sqlx.DB
	repo     *Repo
	products Products
	branches Branches
}

// NewService wires the pricing service.
func NewService(conn *sqlx.DB, repo *Repo, products Products, branches Branches) *Service {
	return &Service{conn: conn, repo: repo, products: products, branches: branches}
}

// ---- price lists ----------------------------------------------------------------

// ListRequest is the create/update payload.
type ListRequest struct {
	Name          string            `json:"name"`
	BranchID      *string           `json:"branch_id"` // null = all branches
	CustomerGroup *string           `json:"customer_group"`
	Priority      int               `json:"priority"`
	IsActive      *bool             `json:"is_active"`
	ValidFrom     *string           `json:"valid_from"` // RFC3339
	ValidTo       *string           `json:"valid_to"`
	Items         []ListItemRequest `json:"items"`
}

// ListItemRequest is one tier row.
type ListItemRequest struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	MinQty    string `json:"min_qty"`
	Price     string `json:"price"`
}

// Lists returns all price lists (heads only).
func (s *Service) Lists(ctx context.Context, tenantID uint64) ([]PriceList, error) {
	return s.repo.Lists(ctx, tenantID)
}

// GetList loads one list with items.
func (s *Service) GetList(ctx context.Context, tenantID uint64, publicID string) (PriceList, error) {
	l, err := s.repo.ListByPublicID(ctx, s.conn, tenantID, publicID)
	if db.IsNotFound(err) {
		return l, ErrListNotFound
	}
	return l, err
}

// CreateList validates and persists a list.
func (s *Service) CreateList(ctx context.Context, actor identity.Actor, req ListRequest) (PriceList, error) {
	l, err := s.buildList(ctx, actor, req)
	if err != nil {
		return l, err
	}
	l.PublicID = security.NewPublicID()
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		return s.repo.CreateList(ctx, tx, &l)
	})
	if err != nil {
		if db.IsDuplicate(err) {
			return l, ErrDuplicateName
		}
		return l, err
	}
	return s.GetList(ctx, actor.TenantID, l.PublicID)
}

// UpdateList rewrites a list head + items.
func (s *Service) UpdateList(ctx context.Context, actor identity.Actor, publicID string, req ListRequest) (PriceList, error) {
	existing, err := s.GetList(ctx, actor.TenantID, publicID)
	if err != nil {
		return existing, err
	}
	l, err := s.buildList(ctx, actor, req)
	if err != nil {
		return l, err
	}
	l.ID = existing.ID
	l.PublicID = existing.PublicID
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if err := s.repo.UpdateList(ctx, tx, &l); err != nil {
			return err
		}
		return s.repo.ReplaceItems(ctx, tx, l.ID, l.Items)
	})
	if err != nil {
		if db.IsDuplicate(err) {
			return l, ErrDuplicateName
		}
		return l, err
	}
	return s.GetList(ctx, actor.TenantID, publicID)
}

// DeleteList removes a list.
func (s *Service) DeleteList(ctx context.Context, tenantID uint64, publicID string) error {
	l, err := s.GetList(ctx, tenantID, publicID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteList(ctx, tenantID, l.ID); err != nil {
		if db.IsNotFound(err) {
			return ErrListNotFound
		}
		return err
	}
	return nil
}

// buildList resolves the request into a model (products/variants/branches to
// internal ids, decimals parsed).
func (s *Service) buildList(ctx context.Context, actor identity.Actor, req ListRequest) (PriceList, error) {
	l := PriceList{TenantID: actor.TenantID, Name: req.Name,
		CustomerGroup: req.CustomerGroup, Priority: req.Priority, IsActive: true}
	if l.Name == "" {
		return l, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	if req.IsActive != nil {
		l.IsActive = *req.IsActive
	}
	var err error
	if req.BranchID != nil && *req.BranchID != "" {
		var id uint64
		if id, err = s.branches.BranchByPublicID(ctx, actor.TenantID, *req.BranchID); err != nil {
			return l, httpx.ErrValidation.WithDetails(map[string]any{"branch_id": "invalid"})
		}
		l.BranchID = &id
	}
	if l.ValidFrom, err = parseTime(req.ValidFrom); err != nil {
		return l, httpx.ErrValidation.WithDetails(map[string]any{"valid_from": "invalid timestamp"})
	}
	if l.ValidTo, err = parseTime(req.ValidTo); err != nil {
		return l, httpx.ErrValidation.WithDetails(map[string]any{"valid_to": "invalid timestamp"})
	}
	items, err := s.buildItems(ctx, actor.TenantID, req.Items)
	l.Items = items
	return l, err
}

func (s *Service) buildItems(ctx context.Context, tenantID uint64, in []ListItemRequest) ([]PriceListItem, error) {
	fields := map[string]any{}
	var out []PriceListItem
	for i, it := range in {
		pref := "items." + strconv.Itoa(i) + "."
		p, err := s.products.Get(ctx, tenantID, it.ProductID)
		if err != nil {
			fields[pref+"product_id"] = "invalid"
			continue
		}
		row := PriceListItem{ProductID: p.ID}
		if it.VariantID != "" {
			found := false
			for _, v := range p.Variants {
				if v.PublicID == it.VariantID {
					row.VariantID, found = v.ID, true
				}
			}
			if !found {
				fields[pref+"variant_id"] = "invalid"
				continue
			}
		}
		row.MinQty = decimal.NewFromInt(1)
		if it.MinQty != "" {
			if row.MinQty, err = decimal.NewFromString(it.MinQty); err != nil || row.MinQty.LessThanOrEqual(decimal.Zero) {
				fields[pref+"min_qty"] = "must be a positive number"
				continue
			}
		}
		if row.Price, err = decimal.NewFromString(it.Price); err != nil || row.Price.IsNegative() {
			fields[pref+"price"] = "must be a non-negative number"
			continue
		}
		out = append(out, row)
	}
	if len(fields) > 0 {
		return nil, httpx.ErrValidation.WithDetails(fields)
	}
	return out, nil
}

// ---- price resolution --------------------------------------------------------------

// ResolvedPrice is the outcome of price resolution for one line.
type ResolvedPrice struct {
	UnitPrice decimal.Decimal
	Source    string // "base" | "price_list"
	ListID    uint64
	ListName  string
}

// Resolve returns the unit price for (product, variant) at qty in a branch
// for a customer group: the highest-priority applicable price list tier wins,
// else the catalog base price.
func (s *Service) Resolve(ctx context.Context, tenantID, branchID uint64,
	group string, p catalog.Product, variantID uint64, qty decimal.Decimal) (ResolvedPrice, error) {
	lists, err := s.repo.ApplicableLists(ctx, tenantID, branchID, group, time.Now())
	if err != nil {
		return ResolvedPrice{}, err
	}
	for _, l := range lists {
		it, err := s.repo.TierFor(ctx, s.conn, l.ID, p.ID, variantID, qty.String())
		if err == nil {
			return ResolvedPrice{UnitPrice: it.Price, Source: HistoryPriceList,
				ListID: l.ID, ListName: l.Name}, nil
		}
		if !db.IsNotFound(err) {
			return ResolvedPrice{}, err
		}
	}
	base := p.Price
	if variantID != 0 {
		for _, v := range p.Variants {
			if v.ID == variantID && v.Price != nil {
				base = *v.Price
			}
		}
	}
	return ResolvedPrice{UnitPrice: base, Source: HistoryBase}, nil
}

// ---- history -------------------------------------------------------------------------

// RecordPrice is the catalog hook: called inside the product-update
// transaction whenever a base or variant price changes. oldPrice is nil for
// a first-time variant override.
func (s *Service) RecordPrice(ctx context.Context, q db.Querier, tenantID, productID, variantID uint64,
	oldPrice *decimal.Decimal, newPrice decimal.Decimal, actorID uint64) error {
	var actor *uint64
	if actorID != 0 {
		actor = &actorID
	}
	return s.repo.InsertHistory(ctx, q, []HistoryRow{{
		TenantID: tenantID, ProductID: productID, VariantID: variantID,
		Source: HistoryBase, OldPrice: oldPrice, NewPrice: newPrice, ActorID: actor,
	}})
}

// History returns price-change rows for a product.
func (s *Service) History(ctx context.Context, tenantID uint64, productPub string, pg httpx.Page) ([]HistoryRow, int, error) {
	p, err := s.products.Get(ctx, tenantID, productPub)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.History(ctx, tenantID, p.ID, pg)
}

// ---- helpers ---------------------------------------------------------------------------

func parseTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
