package catalog

import (
	"context"

	"mauzo/pkg/barcode"
)

// maxLabels caps one print job.
const maxLabels = 500

// Labels builds printable shelf-label rows for the requested products. Each
// label carries its Code 128 module pattern so the UI can render a scannable
// barcode without a font or library. copies applies per product.
func (s *Service) Labels(ctx context.Context, tenantID uint64, productPubs []string, copies int) ([]LabelResponse, error) {
	if copies < 1 {
		copies = 1
	}
	if copies > 100 {
		copies = 100
	}
	out := []LabelResponse{}
	for _, pub := range productPubs {
		p, err := s.repo.ByPublicID(ctx, tenantID, pub)
		if err != nil || p.Status != StatusActive {
			continue // unknown or archived ids are skipped, not fatal for a print job
		}
		code := p.SKU
		if bcs, err := s.repo.BarcodesOf(ctx, p.ID); err == nil && len(bcs) > 0 {
			code = bcs[0].Code // prefer a scannable barcode over the SKU
		}
		pat, err := barcode.Code128(code)
		if err != nil {
			continue // unencodable codes (non-ASCII) are skipped
		}
		for i := 0; i < copies; i++ {
			out = append(out, LabelResponse{
				ProductID: p.PublicID,
				Name:      p.Name,
				Price:     p.Price.String(),
				Barcode:   code,
				Pattern:   pat,
			})
		}
		if len(out) >= maxLabels {
			return out[:maxLabels], nil
		}
	}
	return out, nil
}
