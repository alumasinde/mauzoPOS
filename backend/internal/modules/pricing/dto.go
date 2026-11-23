package pricing

import (
	"encoding/json"
	"time"

	"mauzo/pkg/pricecalc"
)

// ListResponse is a price list head.
type ListResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	BranchID      string         `json:"branch_id,omitempty"`
	CustomerGroup *string        `json:"customer_group,omitempty"`
	Priority      int            `json:"priority"`
	IsActive      bool           `json:"is_active"`
	ValidFrom     *time.Time     `json:"valid_from,omitempty"`
	ValidTo       *time.Time     `json:"valid_to,omitempty"`
	Items         []ItemResponse `json:"items,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// ItemResponse is one tier row.
type ItemResponse struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	VariantID   string `json:"variant_id,omitempty"`
	MinQty      string `json:"min_qty"`
	Price       string `json:"price"`
}

func listResponse(l PriceList) ListResponse {
	out := ListResponse{
		ID: l.PublicID, Name: l.Name, BranchID: l.BranchPublicID,
		CustomerGroup: l.CustomerGroup, Priority: l.Priority, IsActive: l.IsActive,
		ValidFrom: l.ValidFrom, ValidTo: l.ValidTo, CreatedAt: l.CreatedAt,
	}
	for _, it := range l.Items {
		out.Items = append(out.Items, ItemResponse{
			ProductID: it.ProductPublicID, ProductName: it.ProductName,
			VariantID: it.VariantPublicID, MinQty: it.MinQty.String(), Price: it.Price.String(),
		})
	}
	return out
}

// PromotionResponse is a persisted promotion.
type PromotionResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Kind      string           `json:"kind"`
	Config    pricecalc.Config `json:"config"`
	StartsAt  *time.Time       `json:"starts_at,omitempty"`
	EndsAt    *time.Time       `json:"ends_at,omitempty"`
	DaysMask  *uint8           `json:"days_mask,omitempty"`
	Priority  int              `json:"priority"`
	IsActive  bool             `json:"is_active"`
	Targets   []TargetResponse `json:"targets"`
	CreatedAt time.Time        `json:"created_at"`
}

// TargetResponse is one promotion target.
type TargetResponse struct {
	ProductID  string `json:"product_id,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
}

func promoResponse(p Promotion) PromotionResponse {
	out := PromotionResponse{
		ID: p.PublicID, Name: p.Name, Kind: p.Kind, StartsAt: p.StartsAt,
		EndsAt: p.EndsAt, DaysMask: p.DaysMask, Priority: p.Priority,
		IsActive: p.IsActive, CreatedAt: p.CreatedAt,
	}
	_ = json.Unmarshal(p.Config, &out.Config)
	for _, t := range p.Targets {
		out.Targets = append(out.Targets, TargetResponse{
			ProductID: t.ProductPublicID, CategoryID: t.CategoryPublicID,
		})
	}
	return out
}

// HistoryResponse is one price-change row.
type HistoryResponse struct {
	ProductName string    `json:"product_name"`
	Source      string    `json:"source"`
	RefID       *string   `json:"ref_id,omitempty"`
	OldPrice    *string   `json:"old_price,omitempty"`
	NewPrice    string    `json:"new_price"`
	CreatedAt   time.Time `json:"created_at"`
}

func historyResponse(h HistoryRow) HistoryResponse {
	out := HistoryResponse{
		ProductName: h.ProductName, Source: h.Source, RefID: h.RefPublicID,
		NewPrice: h.NewPrice.String(), CreatedAt: h.CreatedAt,
	}
	if h.OldPrice != nil {
		s := h.OldPrice.String()
		out.OldPrice = &s
	}
	return out
}
