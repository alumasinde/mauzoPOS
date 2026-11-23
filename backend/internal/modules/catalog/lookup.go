package catalog

import (
	"context"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/db"
	"mauzo/pkg/barcode"
)

// Lookup resolves a scanned code for the POS. Order: exact barcode match →
// scale (weighted) label decode → SKU match. Quantity is the number of sell
// units the scan represents (pack size, measured weight, or 1).
func (s *Service) Lookup(ctx context.Context, tenantID uint64, code string) (LookupResult, error) {
	// 1. Exact barcode.
	if bc, err := s.repo.BarcodeByCode(ctx, tenantID, code); err == nil {
		return s.lookupResult(ctx, tenantID, bc.ProductID, bc.VariantID, bc.PackQty, nil, "barcode")
	} else if !db.IsNotFound(err) {
		return LookupResult{}, err
	}

	// 2. Scale label (EAN-13 with in-store prefix carrying price or weight).
	if sc, ok := barcode.ParseScale(code, s.scale); ok {
		res, err := s.lookupScale(ctx, tenantID, sc)
		if err != nil {
			return LookupResult{}, err
		}
		res.Scale = &ScaleInfo{ItemCode: sc.Item, Embedded: string(s.scale.Embedded), Value: sc.Value.String()}
		return res, nil
	}

	// 3. SKU.
	if p, err := s.repo.BySKU(ctx, tenantID, code); err == nil {
		return s.lookupResult(ctx, tenantID, p.ID, nil, decimal.NewFromInt(1), nil, "sku")
	} else if !db.IsNotFound(err) {
		return LookupResult{}, err
	}
	return LookupResult{}, ErrNotFound
}

func (s *Service) lookupScale(ctx context.Context, tenantID uint64, sc barcode.Scale) (LookupResult, error) {
	// The 5-digit item code maps to a product barcode or SKU.
	bc, err := s.repo.BarcodeByCode(ctx, tenantID, sc.Item)
	if err != nil && !db.IsNotFound(err) {
		return LookupResult{}, err
	}
	if err == nil {
		if s.scale.Embedded == barcode.EmbeddedWeight {
			return s.lookupResult(ctx, tenantID, bc.ProductID, bc.VariantID, sc.Value, nil, "scale")
		}
		return s.lookupResult(ctx, tenantID, bc.ProductID, bc.VariantID, decimal.NewFromInt(1), &sc.Value, "scale")
	}
	p, err := s.repo.BySKU(ctx, tenantID, sc.Item)
	if db.IsNotFound(err) {
		return LookupResult{}, ErrNotFound
	}
	if err != nil {
		return LookupResult{}, err
	}
	if s.scale.Embedded == barcode.EmbeddedWeight {
		return s.lookupResult(ctx, tenantID, p.ID, nil, sc.Value, nil, "scale")
	}
	return s.lookupResult(ctx, tenantID, p.ID, nil, decimal.NewFromInt(1), &sc.Value, "scale")
}

func (s *Service) lookupResult(ctx context.Context, tenantID, productID uint64, variantID *uint64, qty decimal.Decimal, priceOverride *decimal.Decimal, source string) (LookupResult, error) {
	p, err := s.repo.ByID(ctx, tenantID, productID)
	if db.IsNotFound(err) {
		return LookupResult{}, ErrNotFound
	}
	if err != nil {
		return LookupResult{}, err
	}
	if p.Status != StatusActive {
		return LookupResult{}, ErrNotFound
	}
	if err := s.hydrate(ctx, &p); err != nil {
		return LookupResult{}, err
	}
	price := p.Price
	var vr *VariantResponse
	if variantID != nil {
		for _, v := range p.Variants {
			if v.ID == *variantID {
				if v.Price != nil {
					price = *v.Price
				}
				resp := ToVariantResponse(v)
				vr = &resp
				break
			}
		}
	}
	if priceOverride != nil {
		price = *priceOverride
	}
	return LookupResult{
		Source:   source,
		Product:  ToProductDetail(p),
		Variant:  vr,
		Quantity: qty.String(),
		Price:    price.String(),
	}, nil
}
