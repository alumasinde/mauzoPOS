package pricing

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/catalog"
	"mauzo/pkg/pricecalc"
)

// Categories resolves category public ids for promotion targets.
type Categories interface {
	CategoryByPublic(ctx context.Context, tenantID uint64, publicID string) (uint64, error)
}

var promoKinds = []string{string(pricecalc.KindPercent), string(pricecalc.KindAmount), string(pricecalc.KindBXGY)}

// PromoService implements promotion CRUD and the sale-time engine adapter.
type PromoService struct {
	conn       *sqlx.DB
	repo       *Repo
	products   Products
	categories Categories
}

// NewPromoService wires the promotion service.
func NewPromoService(conn *sqlx.DB, repo *Repo, products Products, cats Categories) *PromoService {
	return &PromoService{conn: conn, repo: repo, products: products, categories: cats}
}

// PromotionRequest is the create/update payload.
type PromotionRequest struct {
	Name     string           `json:"name"`
	Kind     string           `json:"kind"` // percent | amount | bxgy
	Config   pricecalc.Config `json:"config"`
	StartsAt *string          `json:"starts_at"`
	EndsAt   *string          `json:"ends_at"`
	DaysMask *uint8           `json:"days_mask"`
	Priority int              `json:"priority"`
	IsActive *bool            `json:"is_active"`
	Targets  []TargetRequest  `json:"targets"` // empty = whole catalog
}

// TargetRequest scopes a promotion to products and/or categories.
type TargetRequest struct {
	ProductID  string `json:"product_id"`
	CategoryID string `json:"category_id"`
}

// Promotions lists all promotions with targets.
func (s *PromoService) Promotions(ctx context.Context, tenantID uint64) ([]Promotion, error) {
	return s.repo.Promotions(ctx, tenantID)
}

// GetPromotion loads one promotion.
func (s *PromoService) GetPromotion(ctx context.Context, tenantID uint64, publicID string) (Promotion, error) {
	p, err := s.repo.PromotionByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return p, ErrPromoNotFound
	}
	return p, err
}

// CreatePromotion validates and persists a promotion.
func (s *PromoService) CreatePromotion(ctx context.Context, actor identity.Actor, req PromotionRequest) (Promotion, error) {
	p, err := s.build(ctx, actor.TenantID, req)
	if err != nil {
		return p, err
	}
	p.PublicID = security.NewPublicID()
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		return s.repo.CreatePromotion(ctx, tx, &p)
	})
	if err != nil {
		return p, err
	}
	return s.GetPromotion(ctx, actor.TenantID, p.PublicID)
}

// UpdatePromotion rewrites a promotion.
func (s *PromoService) UpdatePromotion(ctx context.Context, actor identity.Actor, publicID string, req PromotionRequest) (Promotion, error) {
	existing, err := s.GetPromotion(ctx, actor.TenantID, publicID)
	if err != nil {
		return existing, err
	}
	p, err := s.build(ctx, actor.TenantID, req)
	if err != nil {
		return p, err
	}
	p.ID = existing.ID
	p.PublicID = existing.PublicID
	err = db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error {
		if err := s.repo.UpdatePromotion(ctx, tx, &p); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, tx, p.ID, p.Targets)
	})
	if err != nil {
		return p, err
	}
	return s.GetPromotion(ctx, actor.TenantID, publicID)
}

// DeletePromotion removes a promotion.
func (s *PromoService) DeletePromotion(ctx context.Context, tenantID uint64, publicID string) error {
	p, err := s.GetPromotion(ctx, tenantID, publicID)
	if err != nil {
		return err
	}
	if err := s.repo.DeletePromotion(ctx, tenantID, p.ID); err != nil {
		if db.IsNotFound(err) {
			return ErrPromoNotFound
		}
		return err
	}
	return nil
}

func (s *PromoService) build(ctx context.Context, tenantID uint64, req PromotionRequest) (Promotion, error) {
	p := Promotion{TenantID: tenantID, Name: req.Name, Kind: req.Kind,
		DaysMask: req.DaysMask, Priority: req.Priority, IsActive: true}
	if p.Name == "" {
		return p, httpx.ErrValidation.WithDetails(map[string]any{"name": "required"})
	}
	if !slices.Contains(promoKinds, p.Kind) {
		return p, ErrBadKind
	}
	cfg, err := json.Marshal(req.Config)
	if err != nil {
		return p, err
	}
	p.Config = cfg
	var errT error
	if p.StartsAt, errT = parseTime(req.StartsAt); errT != nil {
		return p, httpx.ErrValidation.WithDetails(map[string]any{"starts_at": "invalid timestamp"})
	}
	if p.EndsAt, errT = parseTime(req.EndsAt); errT != nil {
		return p, httpx.ErrValidation.WithDetails(map[string]any{"ends_at": "invalid timestamp"})
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
	fields := map[string]any{}
	for i, t := range req.Targets {
		pref := "targets." + strconv.Itoa(i) + "."
		var row PromotionTarget
		if t.ProductID != "" {
			prod, err := s.products.Get(ctx, tenantID, t.ProductID)
			if err != nil {
				fields[pref+"product_id"] = "invalid"
				continue
			}
			row.ProductID = &prod.ID
		}
		if t.CategoryID != "" {
			id, err := s.categories.CategoryByPublic(ctx, tenantID, t.CategoryID)
			if err != nil {
				fields[pref+"category_id"] = "invalid"
				continue
			}
			row.CategoryID = &id
		}
		p.Targets = append(p.Targets, row)
	}
	if len(fields) > 0 {
		return p, httpx.ErrValidation.WithDetails(fields)
	}
	return p, nil
}

// ActivePromotions adapts persisted promotions to engine rules.
func (s *PromoService) ActivePromotions(ctx context.Context, tenantID uint64) ([]pricecalc.Promotion, error) {
	rows, err := s.repo.ActivePromotions(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]pricecalc.Promotion, 0, len(rows))
	for _, r := range rows {
		var cfg pricecalc.Config
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			continue // corrupt row — skip rather than break the till
		}
		pr := pricecalc.Promotion{
			ID: r.ID, Name: r.Name, Kind: pricecalc.Kind(r.Kind), Config: cfg,
			StartsAt: r.StartsAt, EndsAt: r.EndsAt, DaysMask: r.DaysMask, Priority: r.Priority,
			Products: map[uint64]bool{}, Categories: map[uint64]bool{},
		}
		for _, t := range r.Targets {
			if t.ProductID != nil {
				pr.Products[*t.ProductID] = true
			}
			if t.CategoryID != nil {
				pr.Categories[*t.CategoryID] = true
			}
		}
		out = append(out, pr)
	}
	return out, nil
}

// ---- quote ---------------------------------------------------------------------------

// QuoteLine is one line to price.
type QuoteLine struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Qty       string `json:"qty"`
}

// QuoteRequest prices a basket for a branch + customer group.
type QuoteRequest struct {
	BranchID      string      `json:"branch_id"`
	CustomerGroup string      `json:"customer_group"`
	Lines         []QuoteLine `json:"lines"`
}

// QuotedLine is one priced line.
type QuotedLine struct {
	ProductID   string `json:"product_id"`
	VariantID   string `json:"variant_id,omitempty"`
	ProductName string `json:"product_name"`
	Qty         string `json:"qty"`
	UnitPrice   string `json:"unit_price"` // after list pricing, before promos
	Source      string `json:"source"`     // base | price_list
	PriceList   string `json:"price_list,omitempty"`
	Discount    string `json:"discount"`
	Promotion   string `json:"promotion,omitempty"`
	FinalPrice  string `json:"final_unit_price"`
	LineTotal   string `json:"line_total"`
}

// QuoteResponse is the priced basket.
type QuoteResponse struct {
	Subtotal string       `json:"subtotal"`
	Discount string       `json:"discount"`
	Total    string       `json:"total"`
	Lines    []QuotedLine `json:"lines"`
}

// Quote resolves list prices then applies live promotions, best-per-line.
func Quote(ctx context.Context, actor identity.Actor, svc *Service, promoSvc *PromoService, branches Branches, req QuoteRequest) (QuoteResponse, error) {
	var branchID uint64
	if req.BranchID != "" {
		id, err := branches.BranchByPublicID(ctx, actor.TenantID, req.BranchID)
		if err != nil {
			return QuoteResponse{}, httpx.ErrValidation.WithDetails(map[string]any{"branch_id": "invalid"})
		}
		if !actor.CanAccessBranch(id) {
			return QuoteResponse{}, ErrBranchAccess
		}
		branchID = id
	}
	if len(req.Lines) == 0 {
		return QuoteResponse{}, ErrNoItems
	}
	lines, meta, resolved, err := resolveQuoteLines(ctx, actor, svc, branchID, req)
	if err != nil {
		return QuoteResponse{}, err
	}
	promos, err := promoSvc.ActivePromotions(ctx, actor.TenantID)
	if err != nil {
		return QuoteResponse{}, err
	}
	tot := pricecalc.Apply(lines, promos, time.Now())
	out := QuoteResponse{Subtotal: tot.Subtotal.String(), Discount: tot.Discount.String(), Total: tot.Total.String()}
	byIdx := map[int]pricecalc.LineDiscount{}
	for _, ld := range tot.Lines {
		byIdx[ld.LineIndex] = ld
	}
	for i, l := range lines {
		out.Lines = append(out.Lines, quoteLineOut(req.Lines[i], meta[i], resolved[i], l, byIdx[i]))
	}
	return out, nil
}

// resolveQuoteLines resolves products/variants/qty and applies list pricing.
func resolveQuoteLines(ctx context.Context, actor identity.Actor, svc *Service,
	branchID uint64, req QuoteRequest) ([]pricecalc.Line, []catalog.Product, []ResolvedPrice, error) {
	fields := map[string]any{}
	lines := make([]pricecalc.Line, len(req.Lines))
	meta := make([]catalog.Product, len(req.Lines))
	resolved := make([]ResolvedPrice, len(req.Lines))
	for i, l := range req.Lines {
		pref := "lines." + strconv.Itoa(i) + "."
		p, err := svc.products.Get(ctx, actor.TenantID, l.ProductID)
		if err != nil {
			fields[pref+"product_id"] = "invalid"
			continue
		}
		var variantID uint64
		if l.VariantID != "" {
			found := false
			for _, v := range p.Variants {
				if v.PublicID == l.VariantID {
					variantID, found = v.ID, true
				}
			}
			if !found {
				fields[pref+"variant_id"] = "invalid"
				continue
			}
		}
		qty, err := decimal.NewFromString(l.Qty)
		if err != nil || qty.LessThanOrEqual(decimal.Zero) {
			fields[pref+"qty"] = "must be a positive number"
			continue
		}
		rp, err := svc.Resolve(ctx, actor.TenantID, branchID, req.CustomerGroup, p, variantID, qty)
		if err != nil {
			return nil, nil, nil, err
		}
		resolved[i], meta[i] = rp, p
		var catID uint64
		if p.CategoryID != nil {
			catID = *p.CategoryID
		}
		lines[i] = pricecalc.Line{ProductID: p.ID, VariantID: variantID,
			CategoryID: catID, Qty: qty, UnitPrice: rp.UnitPrice}
	}
	if len(fields) > 0 {
		return nil, nil, nil, httpx.ErrValidation.WithDetails(fields)
	}
	return lines, meta, resolved, nil
}

func quoteLineOut(in QuoteLine, p catalog.Product, rp ResolvedPrice,
	l pricecalc.Line, d pricecalc.LineDiscount) QuotedLine {
	ql := QuotedLine{
		ProductID: in.ProductID, VariantID: in.VariantID,
		ProductName: p.Name, Qty: l.Qty.String(), UnitPrice: l.UnitPrice.String(),
		Source: rp.Source, PriceList: rp.ListName,
		Discount: "0", FinalPrice: l.UnitPrice.String(),
	}
	final := l.UnitPrice
	if d.Amount.IsPositive() {
		ql.Discount = d.Amount.String()
		ql.Promotion = d.Name
		ql.FinalPrice = d.UnitPrice.String()
		final = d.UnitPrice
	}
	ql.LineTotal = l.Qty.Mul(final).String()
	return ql
}
