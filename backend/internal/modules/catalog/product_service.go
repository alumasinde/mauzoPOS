package catalog

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/core/security"
	"mauzo/pkg/barcode"
)

// PriceWatcher records a price change (the pricing module plugs in for
// price_history). Called inside the update transaction.
type PriceWatcher func(ctx context.Context, q db.Querier, tenantID, productID, variantID uint64,
	oldPrice *decimal.Decimal, newPrice decimal.Decimal, actorID uint64) error

// Service owns product business rules: validation, child rows (variants,
// barcodes, components, images) and the POS lookup.
type Service struct {
	db      *sqlx.DB
	repo    *ProductRepo
	tax     *TaxonomyRepo
	scale   barcode.ScaleConfig
	images  ImageStore
	publics map[uint64]string // public-id cache for child rows (per service)
	prices  PriceWatcher      // optional price-change hook
}

// WatchPrices registers the price-change hook (pricing module).
func (s *Service) WatchPrices(w PriceWatcher) { s.prices = w }

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *ProductRepo, tax *TaxonomyRepo, images ImageStore) *Service {
	return &Service{db: conn, repo: repo, tax: tax,
		scale: barcode.DefaultScale(), images: images, publics: map[uint64]string{}}
}

// List returns a page of products with taxonomy names resolved.
func (s *Service) List(ctx context.Context, tenantID uint64, f ListFilter, p httpx.Page) ([]ListRow, int, error) {
	return s.repo.List(ctx, tenantID, f, p)
}

// Get loads a product with variants, barcodes, components and images.
func (s *Service) Get(ctx context.Context, tenantID uint64, publicID string) (Product, error) {
	p, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	if err := s.hydrate(ctx, &p); err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *Service) hydrate(ctx context.Context, p *Product) error {
	var err error
	if p.Variants, err = s.repo.VariantsOf(ctx, p.ID); err != nil {
		return err
	}
	if p.Barcodes, err = s.repo.BarcodesOf(ctx, p.ID); err != nil {
		return err
	}
	if p.Components, err = s.repo.ComponentsOf(ctx, p.ID); err != nil {
		return err
	}
	if p.Images, err = s.repo.ImagesOf(ctx, p.ID); err != nil {
		return err
	}
	// Resolve taxonomy refs for the response.
	if p.CategoryID != nil {
		if c, err := s.tax.CategoryByID(ctx, p.TenantID, *p.CategoryID); err == nil {
			p.CategoryPublicID = &c.PublicID
		}
	}
	if p.BrandID != nil {
		if b, err := s.tax.BrandByID(ctx, p.TenantID, *p.BrandID); err == nil {
			p.BrandPublicID = &b.PublicID
		}
	}
	if p.UnitID != nil {
		if u, err := s.tax.UnitByID(ctx, p.TenantID, *p.UnitID); err == nil {
			p.UnitCode = &u.Code
		}
	}
	return nil
}

// Create validates and inserts a product with its child rows in one tx.
func (s *Service) Create(ctx context.Context, tenantID uint64, req ProductRequest) (Product, error) {
	p, vs, bs, cs, err := s.build(ctx, tenantID, 0, req)
	if err != nil {
		return Product{}, err
	}
	p.TenantID = tenantID
	p.PublicID = security.NewPublicID()
	p.Status = StatusActive
	err = db.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := s.repo.Create(ctx, tx, &p); err != nil {
			return err
		}
		if err := s.repo.ReplaceVariants(ctx, tx, tenantID, p.ID, vs); err != nil {
			return err
		}
		// Barcode requests may name variants by name (variants have no ids yet).
		byName := map[string]uint64{}
		for _, v := range vs {
			byName[v.Name] = v.ID
		}
		for i := range bs {
			if key := variantKey(req.Barcodes[i]); key != "" {
				if id, ok := byName[key]; ok {
					bs[i].VariantID = &id
				}
			}
		}
		if err := s.repo.ReplaceBarcodes(ctx, tx, tenantID, p.ID, bs); err != nil {
			return err
		}
		return s.repo.ReplaceComponents(ctx, tx, p.ID, cs)
	})
	if err != nil {
		return Product{}, mapWriteErr(err)
	}
	return s.Get(ctx, tenantID, p.PublicID)
}

// Update validates and rewrites a product with its child rows in one tx.
func (s *Service) Update(ctx context.Context, tenantID uint64, publicID string, req ProductRequest) (Product, error) {
	existing, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	p, vs, bs, cs, err := s.build(ctx, tenantID, existing.ID, req)
	if err != nil {
		return Product{}, err
	}
	oldVars, err := s.repo.VariantsOf(ctx, existing.ID)
	if err != nil {
		return Product{}, err
	}
	p.ID = existing.ID
	p.TenantID = tenantID
	p.PublicID = existing.PublicID
	p.Status = existing.Status
	err = db.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := s.repo.Update(ctx, tx, &p); err != nil {
			return err
		}
		if err := s.repo.ReplaceVariants(ctx, tx, tenantID, p.ID, vs); err != nil {
			return err
		}
		// Resolve variant references: public ids (existing flow) or names.
		variants, err := s.repo.VariantsOf(ctx, p.ID)
		if err != nil {
			return err
		}
		byName, byPub := map[string]uint64{}, map[string]uint64{}
		for _, v := range variants {
			byName[v.Name] = v.ID
			byPub[v.PublicID] = v.ID
		}
		for i := range bs {
			if key := variantKey(req.Barcodes[i]); key != "" {
				if id, ok := byPub[key]; ok {
					bs[i].VariantID = &id
				} else if id, ok := byName[key]; ok {
					bs[i].VariantID = &id
				}
			}
		}
		if err := s.repo.ReplaceBarcodes(ctx, tx, tenantID, p.ID, bs); err != nil {
			return err
		}
		if err := s.repo.ReplaceComponents(ctx, tx, p.ID, cs); err != nil {
			return err
		}
		return s.recordPriceChanges(ctx, tx, tenantID, existing, oldVars, p, variants)
	})
	if err != nil {
		return Product{}, mapWriteErr(err)
	}
	return s.Get(ctx, tenantID, publicID)
}

// recordPriceChanges invokes the price watcher for base/variant prices that
// changed in this update. Variants match old↔new by name (rows are
// replaced wholesale, so identity survives as the business key).
func (s *Service) recordPriceChanges(ctx context.Context, q db.Querier, tenantID uint64,
	existing Product, oldVars []Variant, p Product, newVars []Variant) error {
	if s.prices == nil {
		return nil
	}
	var actorID uint64
	if a, ok := identity.ActorFrom(ctx); ok {
		actorID = a.ID
	}
	if !existing.Price.Equal(p.Price) {
		old := existing.Price
		if err := s.prices(ctx, q, tenantID, p.ID, 0, &old, p.Price, actorID); err != nil {
			return err
		}
	}
	oldByName := map[string]Variant{}
	for _, v := range oldVars {
		oldByName[v.Name] = v
	}
	for _, nv := range newVars {
		ov, ok := oldByName[nv.Name]
		if !ok {
			if nv.Price != nil {
				if err := s.prices(ctx, q, tenantID, p.ID, nv.ID, nil, *nv.Price, actorID); err != nil {
					return err
				}
			}
			continue
		}
		switch {
		case ov.Price == nil && nv.Price == nil:
		case ov.Price == nil || nv.Price == nil || !ov.Price.Equal(*nv.Price):
			if nv.Price != nil {
				if err := s.prices(ctx, q, tenantID, p.ID, nv.ID, ov.Price, *nv.Price, actorID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Archive soft-deletes a product.
func (s *Service) Archive(ctx context.Context, tenantID uint64, publicID string) error {
	p, err := s.repo.ByPublicID(ctx, tenantID, publicID)
	if db.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.repo.Archive(ctx, p.ID)
}

// build normalises and validates a ProductRequest into model rows. selfID is
// the product being updated (0 on create) so composite checks can exempt it.
func (s *Service) build(ctx context.Context, tenantID, selfID uint64, req ProductRequest) (Product, []Variant, []Barcode, []Component, error) {
	p, err := s.buildScalars(ctx, tenantID, req)
	if err != nil {
		return Product{}, nil, nil, nil, err
	}
	vs, err := s.buildVariants(req)
	if err != nil {
		return Product{}, nil, nil, nil, err
	}
	bs, err := s.buildBarcodes(ctx, tenantID, selfID, req)
	if err != nil {
		return Product{}, nil, nil, nil, err
	}
	cs, err := s.buildComponents(ctx, tenantID, selfID, p.Kind, req)
	if err != nil {
		return Product{}, nil, nil, nil, err
	}
	return p, vs, bs, cs, nil
}

// buildScalars validates the product's own columns.
func (s *Service) buildScalars(ctx context.Context, tenantID uint64, req ProductRequest) (Product, error) {
	fields := map[string]any{}
	p := Product{
		SKU:    strings.TrimSpace(req.SKU),
		Name:   strings.TrimSpace(req.Name),
		Kind:   KindStandard,
		SellBy: SellByUnit,
		Price:  decimal.Zero,
	}
	if p.SKU == "" {
		fields["sku"] = "required"
	}
	if p.Name == "" {
		fields["name"] = "required"
	}
	if req.Kind != "" {
		p.Kind = req.Kind
	}
	if !slices.Contains([]string{KindStandard, KindService, KindPackaged, KindComposite}, p.Kind) {
		fields["kind"] = "must be standard, service, packaged or composite"
	}
	if req.SellBy != "" {
		p.SellBy = req.SellBy
	}
	if p.SellBy != SellByUnit && p.SellBy != SellByWeight {
		fields["sell_by"] = "must be unit or weight"
	}
	if req.Description != "" {
		d := req.Description
		p.Description = &d
	}
	var err error
	if req.Price == "" {
		fields["price"] = "required"
	} else if p.Price, err = decimal.NewFromString(req.Price); err != nil || p.Price.IsNegative() {
		fields["price"] = "must be a non-negative number"
	}
	if p.Cost, err = optDec(req.Cost); err != nil {
		fields["cost"] = "must be a number"
	}
	if p.VatRate, err = optDec(req.VatRate); err != nil {
		fields["vat_rate"] = "must be a number"
	}
	if p.Deposit, err = optDec(req.Deposit); err != nil {
		fields["deposit"] = "must be a number"
	}
	p.TrackStock = true
	if req.TrackStock != nil {
		p.TrackStock = *req.TrackStock
	}
	if len(fields) > 0 {
		return Product{}, httpx.ErrValidation.WithDetails(fields)
	}
	if err := s.resolveRefs(ctx, tenantID, req, &p); err != nil {
		return Product{}, err
	}

	// Services never carry stock; composite/deposit stay trackable.
	if p.Kind == KindService {
		p.TrackStock = false
		if p.SellBy == SellByWeight {
			return Product{}, ErrBadUnit
		}
	}
	if req.Attributes != nil {
		raw, err := json.Marshal(req.Attributes)
		if err != nil {
			return Product{}, httpx.ErrValidation
		}
		rm := json.RawMessage(raw)
		p.Attributes = &rm
	}
	return p, nil
}

// resolveRefs maps public references onto internal ids and enforces the
// sell-mode/unit rules.
func (s *Service) resolveRefs(ctx context.Context, tenantID uint64, req ProductRequest, p *Product) error {
	// References: category, brand (tenant rows), unit (tenant or built-in).
	if req.CategoryID != nil && *req.CategoryID != "" {
		c, err := s.tax.CategoryByPublicID(ctx, tenantID, *req.CategoryID)
		if err != nil {
			return refErr(err, "category_id")
		}
		p.CategoryID = &c.ID
	}
	if req.BrandID != nil && *req.BrandID != "" {
		b, err := s.tax.BrandByPublicID(ctx, tenantID, *req.BrandID)
		if err != nil {
			return refErr(err, "brand_id")
		}
		p.BrandID = &b.ID
	}
	if req.UnitCode != "" {
		u, err := s.tax.UnitByCode(ctx, tenantID, req.UnitCode)
		if err != nil {
			return refErr(err, "unit_code")
		}
		p.UnitID = &u.ID
		if p.SellBy == SellByWeight && u.Kind != UnitKindMass {
			return ErrBadUnit
		}
	} else if p.SellBy == SellByWeight {
		return ErrBadUnit
	}
	return nil
}

func (s *Service) buildVariants(req ProductRequest) ([]Variant, error) {
	vs := make([]Variant, 0, len(req.Variants))
	seen := map[string]bool{}
	for _, vr := range req.Variants {
		name := strings.TrimSpace(vr.Name)
		if name == "" {
			return nil, httpx.ErrValidation.WithDetails(map[string]any{"variants": "name is required"})
		}
		if seen[name] {
			return nil, ErrBadVariant
		}
		seen[name] = true
		v := Variant{PublicID: security.NewPublicID(), Name: name}
		if vr.SKU != "" {
			v.SKU = &vr.SKU
		}
		if vr.Price != nil && *vr.Price != "" {
			pr, err := decimal.NewFromString(*vr.Price)
			if err != nil || pr.IsNegative() {
				return nil, ErrBadVariant
			}
			v.Price = &pr
		}
		if vr.Attributes != nil {
			raw, err := json.Marshal(vr.Attributes)
			if err != nil {
				return nil, ErrBadVariant
			}
			rm := json.RawMessage(raw)
			v.Attributes = &rm
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (s *Service) buildBarcodes(ctx context.Context, tenantID, selfID uint64, req ProductRequest) ([]Barcode, error) {
	bs := make([]Barcode, 0, len(req.Barcodes))
	seen := map[string]bool{}
	for _, br := range req.Barcodes {
		code := strings.TrimSpace(br.Code)
		if code == "" || len(code) > barcode.MaxLen {
			return nil, ErrBadBarcode
		}
		// Reject malformed EAN/UPC (digits with the wrong check digit) and
		// codes in the scale-label prefix range — those decode to products.
		if isGTIN(code) && !barcode.Valid(code) {
			return nil, ErrBadBarcode
		}
		if barcode.IsScalePrefix(code, s.scale) {
			return nil, ErrBadBarcode
		}
		if seen[code] {
			return nil, ErrBadBarcode
		}
		seen[code] = true
		b := Barcode{Code: code, PackQty: decimal.NewFromInt(1)}
		if br.PackQty != nil && *br.PackQty != "" {
			q, err := decimal.NewFromString(*br.PackQty)
			if err != nil || !q.IsPositive() {
				return nil, ErrBadBarcode
			}
			b.PackQty = q
		}
		bs = append(bs, b)
	}
	// Tenant-wide uniqueness pre-check for a friendly 409 (DB enforces it too);
	// the product's own codes are exempt on update.
	for _, b := range bs {
		row, err := s.repo.BarcodeByCode(ctx, tenantID, b.Code)
		if err == nil && row.ProductID != selfID {
			return nil, ErrBarcodeTaken
		}
		if err != nil && !db.IsNotFound(err) {
			return nil, err
		}
	}
	return bs, nil
}

func (s *Service) buildComponents(ctx context.Context, tenantID, selfID uint64, kind string, req ProductRequest) ([]Component, error) {
	if kind != KindComposite {
		if len(req.Components) > 0 {
			return nil, httpx.ErrValidation.WithDetails(map[string]any{"components": "only composite products have components"})
		}
		return nil, nil
	}
	if len(req.Components) == 0 {
		return nil, ErrBadComposite
	}
	cs := make([]Component, 0, len(req.Components))
	for _, cr := range req.Components {
		comp, err := s.repo.ByPublicID(ctx, tenantID, cr.ProductID)
		if db.IsNotFound(err) {
			return nil, ErrBadComposite
		}
		if err != nil {
			return nil, err
		}
		if comp.ID == selfID || comp.Kind == KindComposite {
			return nil, ErrBadComposite
		}
		q, err := decimal.NewFromString(cr.Qty)
		if err != nil || !q.IsPositive() {
			return nil, ErrBadComposite
		}
		cs = append(cs, Component{ComponentID: comp.ID, Qty: q})
	}
	return cs, nil
}

func optDec(s string) (*decimal.Decimal, error) {
	if s == "" {
		return nil, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func refErr(err error, field string) error {
	if db.IsNotFound(err) {
		return httpx.ErrValidation.WithDetails(map[string]any{field: "unknown reference"})
	}
	return err
}

// variantKey normalises the variant reference on a barcode request.
func variantKey(br BarcodeRequest) string {
	if br.VariantID == nil {
		return ""
	}
	return *br.VariantID
}

// isGTIN reports whether code is all digits of a checkable GS1 length.
func isGTIN(code string) bool {
	t := barcode.Classify(code)
	return t == barcode.TypeEAN8 || t == barcode.TypeEAN13 || t == barcode.TypeUPCA
}

func mapWriteErr(err error) error {
	if db.IsDuplicate(err) {
		return ErrSKUTaken
	}
	return err
}
