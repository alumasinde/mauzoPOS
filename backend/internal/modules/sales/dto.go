package sales

import "time"

// SaleResponse is the list-row API view.
type SaleResponse struct {
	ID         string     `json:"id"`
	ReceiptNo  uint64     `json:"receipt_no"`
	Status     string     `json:"status"`
	Source     string     `json:"source"`
	Branch     string     `json:"branch"`
	Terminal   string     `json:"terminal"`
	Cashier    string     `json:"cashier"`
	Customer   string     `json:"customer"`
	Subtotal   string     `json:"subtotal"`
	Discount   string     `json:"discount"`
	Tax        string     `json:"tax"`
	Total      string     `json:"total"`
	Paid       string     `json:"paid"`
	ChangeDue  string     `json:"change_due"`
	SoldAt     time.Time  `json:"sold_at"`
	VoidedAt   *time.Time `json:"voided_at,omitempty"`
	VoidReason string     `json:"void_reason,omitempty"`
}

// ItemResponse is one sale line.
type ItemResponse struct {
	LineNo        int    `json:"line_no"`
	Name          string `json:"name"`
	SKU           string `json:"sku"`
	Qty           string `json:"qty"`
	UnitPrice     string `json:"unit_price"`
	Discount      string `json:"discount"`
	Tax           string `json:"tax"`
	Total         string `json:"total"`
	ReturnedQty   string `json:"returned_qty"`
	PriceOverride bool   `json:"price_override"`
	Promotion     string `json:"promotion,omitempty"`
}

// PaymentResponse is one tendered row.
type PaymentResponse struct {
	Method    string `json:"method"`
	Amount    string `json:"amount"`
	Tendered  string `json:"tendered,omitempty"`
	ChangeDue string `json:"change_due,omitempty"`
	Reference string `json:"reference,omitempty"`
	Status    string `json:"status"`
}

// ReturnResponse is a refund header.
type ReturnResponse struct {
	ID           string    `json:"id"`
	ReturnNo     uint64    `json:"return_no"`
	ReceiptNo    uint64    `json:"receipt_no,omitempty"`
	Reason       string    `json:"reason"`
	RefundMethod string    `json:"refund_method"`
	RefundAmount string    `json:"refund_amount"`
	Status       string    `json:"status"`
	Cashier      string    `json:"cashier,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DetailResponse is the full sale.
type DetailResponse struct {
	SaleResponse
	Items    []ItemResponse    `json:"items"`
	Payments []PaymentResponse `json:"payments"`
	Returns  []ReturnResponse  `json:"returns"`
}

// ToResponse maps a sale for the list.
func ToResponse(s Sale) SaleResponse {
	return SaleResponse{
		ID: s.PublicID, ReceiptNo: s.ReceiptNo, Status: s.Status, Source: s.Source,
		Branch: s.BranchName, Terminal: s.TerminalName, Cashier: s.CashierName,
		Customer: s.CustomerName,
		Subtotal: s.Subtotal.String(), Discount: s.Discount.String(), Tax: s.Tax.String(),
		Total: s.Total.String(), Paid: s.Paid.String(), ChangeDue: s.ChangeDue.String(),
		SoldAt: s.SoldAt, VoidedAt: s.VoidedAt, VoidReason: s.VoidReason,
	}
}

// ToDetail maps a full sale.
func ToDetail(d SaleDetail) DetailResponse {
	out := DetailResponse{SaleResponse: ToResponse(d.Sale)}
	for _, it := range d.Items {
		out.Items = append(out.Items, ItemResponse{
			LineNo: it.LineNo, Name: it.Name, SKU: it.SKU, Qty: it.Qty.String(),
			UnitPrice: it.UnitPrice.String(), Discount: it.Discount.String(),
			Tax: it.Tax.String(), Total: it.Total.String(),
			ReturnedQty: it.ReturnedQty.String(), PriceOverride: it.PriceOverride,
			Promotion: it.Promotion,
		})
	}
	for _, p := range d.Payments {
		pr := PaymentResponse{Method: p.Method, Amount: p.Amount.String(),
			Reference: p.Reference, Status: p.Status}
		if p.Tendered != nil {
			pr.Tendered = p.Tendered.String()
		}
		if p.ChangeDue != nil {
			pr.ChangeDue = p.ChangeDue.String()
		}
		out.Payments = append(out.Payments, pr)
	}
	for _, r := range d.Returns {
		out.Returns = append(out.Returns, ToReturnResponse(r))
	}
	return out
}

// ToReturnResponse maps a return for the API.
func ToReturnResponse(r SaleReturn) ReturnResponse {
	return ReturnResponse{
		ID: r.PublicID, ReturnNo: r.ReturnNo, ReceiptNo: r.ReceiptNo, Reason: r.Reason,
		RefundMethod: r.RefundMethod, RefundAmount: r.RefundAmount.String(),
		Status: r.Status, Cashier: r.CashierName, CreatedAt: r.CreatedAt,
	}
}

// HeldResponse is the back-office view of a parked cart.
type HeldResponse struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	HeldBy   string    `json:"held_by"`
	Terminal string    `json:"terminal"`
	Branch   string    `json:"branch"`
	Cart     string    `json:"cart"`
	HeldAt   time.Time `json:"held_at"`
}

// ToHeldResponse maps a parked cart.
func ToHeldResponse(h HeldCart) HeldResponse {
	return HeldResponse{
		ID: h.PublicID, Label: h.Label, HeldBy: h.HeldBy, Terminal: h.TerminalName,
		Branch: h.BranchName, Cart: h.Cart, HeldAt: h.CreatedAt,
	}
}
