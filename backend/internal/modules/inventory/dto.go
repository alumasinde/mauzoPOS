package inventory

import "time"

// LocationResponse is a stock location.
type LocationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	IsDefault bool   `json:"is_default"`
}

func locResponse(l Location) LocationResponse {
	return LocationResponse{ID: l.PublicID, Name: l.Name, Kind: l.Kind, IsDefault: l.IsDefault}
}

// BalanceResponse is one stock-on-hand row.
type BalanceResponse struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	VariantID   string `json:"variant_id,omitempty"`
	VariantName string `json:"variant_name,omitempty"`
	LocationID  string `json:"location_id"`
	Location    string `json:"location_name"`
	BranchID    string `json:"branch_id"`
	Branch      string `json:"branch_name"`
	BatchID     uint64 `json:"batch_id,omitempty"`
	Qty         string `json:"qty"`
	AvgCost     string `json:"avg_cost"`
}

func balResponse(b Balance) BalanceResponse {
	return BalanceResponse{
		ProductID: b.ProductPublicID, ProductName: b.ProductName, SKU: b.ProductSKU,
		VariantID: b.VariantPublicID, VariantName: b.VariantName,
		LocationID: b.LocationPublicID, Location: b.LocationName,
		BranchID: b.BranchPublicID, Branch: b.BranchName,
		BatchID: b.BatchID, Qty: b.Qty.String(), AvgCost: b.AvgCost.String(),
	}
}

// MovementResponse is one ledger row.
type MovementResponse struct {
	ID        uint64    `json:"id"`
	Product   string    `json:"product"`
	SKU       string    `json:"sku"`
	Variant   string    `json:"variant_name,omitempty"`
	Location  string    `json:"location_name"`
	Branch    string    `json:"branch_name"`
	Qty       string    `json:"qty"`
	Kind      string    `json:"kind"`
	RefType   *string   `json:"ref_type,omitempty"`
	RefID     *string   `json:"ref_id,omitempty"`
	Reason    *string   `json:"reason,omitempty"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func movResponse(m Movement) MovementResponse {
	return MovementResponse{
		ID: m.ID, Product: m.ProductName, SKU: m.ProductSKU, Variant: m.VariantName,
		Location: m.LocationName, Branch: m.BranchName, Qty: m.Qty.String(),
		Kind: m.Kind, RefType: m.RefType, RefID: m.RefPublicID, Reason: m.Reason,
		Note: m.Note, CreatedAt: m.CreatedAt,
	}
}

// DocItemResponse is a shared line shape for adjustments/transfers/counts.
type DocItemResponse struct {
	ID          string  `json:"id,omitempty"` // public id (transfers/counts)
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	VariantID   string  `json:"variant_id,omitempty"`
	VariantName string  `json:"variant_name,omitempty"`
	LocationID  string  `json:"location_id,omitempty"`
	Location    string  `json:"location_name,omitempty"`
	Qty         string  `json:"qty,omitempty"`
	QtySent     string  `json:"qty_sent,omitempty"`
	QtyReceived *string `json:"qty_received,omitempty"`
	Expected    string  `json:"expected_qty,omitempty"`
	Counted     *string `json:"counted_qty,omitempty"`
}

// AdjustmentResponse is an adjustment document.
type AdjustmentResponse struct {
	ID        string            `json:"id"`
	BranchID  string            `json:"branch_id"`
	Branch    string            `json:"branch_name"`
	Status    string            `json:"status"`
	Reason    string            `json:"reason"`
	Note      *string           `json:"note,omitempty"`
	Items     []DocItemResponse `json:"items"`
	PostedAt  *time.Time        `json:"posted_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func adjResponse(a Adjustment) AdjustmentResponse {
	out := AdjustmentResponse{
		ID: a.PublicID, BranchID: a.BranchPublicID, Status: a.Status, Branch: a.BranchName, Reason: a.Reason,
		Note: a.Note, PostedAt: a.PostedAt, CreatedAt: a.CreatedAt,
	}
	for _, it := range a.Items {
		out.Items = append(out.Items, DocItemResponse{
			ProductID: it.ProductPublicID, ProductName: it.ProductName,
			VariantID: it.VariantPublicID, VariantName: it.VariantName,
			LocationID: it.LocationPublicID, Location: it.LocationName, Qty: it.Qty.String(),
		})
	}
	return out
}

// TransferResponse is a transfer document.
type TransferResponse struct {
	ID         string            `json:"id"`
	From       string            `json:"from_branch"`
	FromName   string            `json:"from_name"`
	To         string            `json:"to_branch"`
	ToName     string            `json:"to_name"`
	Status     string            `json:"status"`
	Note       *string           `json:"note,omitempty"`
	Items      []DocItemResponse `json:"items"`
	SentAt     *time.Time        `json:"sent_at,omitempty"`
	ReceivedAt *time.Time        `json:"received_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

func trResponse(t Transfer) TransferResponse {
	out := TransferResponse{
		ID: t.PublicID, From: t.FromPublicID, FromName: t.FromName, To: t.ToPublicID, ToName: t.ToName, Status: t.Status, Note: t.Note,
		SentAt: t.SentAt, ReceivedAt: t.ReceivedAt, CreatedAt: t.CreatedAt,
	}
	for _, it := range t.Items {
		var recv *string
		if it.QtyReceived != nil {
			s := it.QtyReceived.String()
			recv = &s
		}
		out.Items = append(out.Items, DocItemResponse{
			ID: it.PublicID, ProductID: it.ProductPublicID, ProductName: it.ProductName,
			VariantID: it.VariantPublicID, VariantName: it.VariantName,
			QtySent: it.QtySent.String(), QtyReceived: recv,
		})
	}
	return out
}

// CountResponse is a stocktake document.
type CountResponse struct {
	ID        string            `json:"id"`
	BranchID  string            `json:"branch_id"`
	Branch    string            `json:"branch_name"`
	Status    string            `json:"status"`
	Location  *string           `json:"location_id,omitempty"`
	Note      *string           `json:"note,omitempty"`
	Items     []DocItemResponse `json:"items"`
	PostedAt  *time.Time        `json:"posted_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func cntResponse(c Count) CountResponse {
	var loc *string
	if c.LocationPublicID != "" {
		loc = &c.LocationPublicID
	}
	out := CountResponse{
		ID: c.PublicID, BranchID: c.BranchPublicID, Branch: c.BranchName, Status: c.Status, Location: loc,
		Note: c.Note, PostedAt: c.PostedAt, CreatedAt: c.CreatedAt,
	}
	for _, it := range c.Items {
		var counted *string
		if it.CountedQty != nil {
			s := it.CountedQty.String()
			counted = &s
		}
		out.Items = append(out.Items, DocItemResponse{
			ID: it.PublicID, ProductID: it.ProductPublicID, ProductName: it.ProductName,
			VariantID: it.VariantPublicID, VariantName: it.VariantName,
			LocationID: it.LocationPublicID, Location: it.LocationName,
			Expected: it.ExpectedQty.String(), Counted: counted,
		})
	}
	return out
}

// BatchResponse is a product batch.
type BatchResponse struct {
	ID         uint64     `json:"id"`
	BatchNo    string     `json:"batch_no"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`
	ReceivedAt time.Time  `json:"received_at"`
}

// ExpiringResponse is an expiry alert row.
type ExpiringResponse struct {
	BatchNo     string     `json:"batch_no"`
	ProductName string     `json:"product_name"`
	Branch      string     `json:"branch_name"`
	Qty         string     `json:"qty"`
	ExpiryDate  *time.Time `json:"expiry_date"`
}

// ReorderResponse is a reorder level + current stock.
type ReorderResponse struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	VariantID   string `json:"variant_id,omitempty"`
	VariantName string `json:"variant_name,omitempty"`
	MinQty      string `json:"min_qty"`
	TargetQty   string `json:"target_qty"`
	Qty         string `json:"qty"`
}

// ValuationResponse is stock value for a branch.
type ValuationResponse struct {
	BranchID uint64 `json:"branch_id"`
	Branch   string `json:"branch_name"`
	SKUs     int    `json:"sku_count"`
	Units    string `json:"units"`
	Value    string `json:"value"`
}
