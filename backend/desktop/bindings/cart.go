package bindings

import (
	"encoding/json"
	"errors"

	"mauzo/desktop/localdb"
	"mauzo/internal/core/security"
)

// CartLine is one row the UI sends for pricing.
type CartLine struct {
	ProductID     uint64 `json:"product_id"`
	VariantID     uint64 `json:"variant_id"`
	CategoryID    uint64 `json:"category_id"`
	Name          string `json:"name"`
	SKU           string `json:"sku"`
	Unit          string `json:"unit"`
	Qty           string `json:"qty"`
	PriceOverride string `json:"price_override"`
	Discount      string `json:"discount"` // cashier line discount (absolute)
}

// Quote prices the whole cart (tiers + promos + VAT) against the local mirror.
func (a *App) Quote(lines []localdb.QuoteLine) (localdb.Quote, error) {
	if a.db == nil {
		return localdb.Quote{}, errors.New("local database not ready")
	}
	return a.db.QuoteCart(a.ctx, lines)
}

// Held is one parked cart shown in the recall list.
type Held struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Cart   string `json:"cart"`
	HeldBy string `json:"held_by"`
	HeldAt string `json:"held_at"`
	Source string `json:"source"` // server | local
}

// Hold parks the cart — server-side when online (visible to other tills),
// always mirrored locally so offline holds still work.
func (a *App) Hold(label, cartJSON string) (Held, error) {
	if a.db == nil {
		return Held{}, errors.New("local database not ready")
	}
	cashier := ""
	if s := a.CurrentSession(); s != nil {
		cashier = s.Name
	}
	var out struct {
		ID string `json:"id"`
	}
	err := a.request("POST", "/api/v1/pos/held", map[string]any{
		"user_id": a.cashierID(), "label": label, "cart": json.RawMessage(cartJSON),
	}, &out)
	if err == nil {
		_ = a.db.HoldCart(a.ctx, out.ID, label, cartJSON, cashier, true)
		return Held{ID: out.ID, Label: label, Cart: cartJSON, HeldBy: cashier, Source: "server"}, nil
	}
	if !errors.Is(err, ErrOffline) {
		return Held{}, err
	}
	id := security.NewPublicID()
	if err := a.db.HoldCart(a.ctx, id, label, cartJSON, cashier, false); err != nil {
		return Held{}, err
	}
	return Held{ID: id, Label: label, Cart: cartJSON, HeldBy: cashier, Source: "local"}, nil
}

// HeldList returns parked carts (server list when online, local otherwise).
func (a *App) HeldList() ([]Held, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	var server []struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Cart   string `json:"cart"`
		HeldBy string `json:"held_by"`
		HeldAt string `json:"held_at"`
	}
	err := a.request("GET", "/api/v1/pos/held", nil, &server)
	if err == nil {
		out := make([]Held, 0, len(server))
		for _, h := range server {
			out = append(out, Held{ID: h.ID, Label: h.Label, Cart: h.Cart,
				HeldBy: h.HeldBy, HeldAt: h.HeldAt, Source: "server"})
		}
		return out, nil
	}
	if !errors.Is(err, ErrOffline) {
		return nil, err
	}
	locals, err := a.db.HeldCarts(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Held, 0, len(locals))
	for _, h := range locals {
		out = append(out, Held{ID: h.ID, Label: h.Label, Cart: h.Cart,
			HeldBy: h.HeldBy, HeldAt: h.CreatedAt, Source: "local"})
	}
	return out, nil
}

// RecallHeld returns a parked cart and removes it from the held list.
func (a *App) RecallHeld(id string) (Held, error) {
	if a.db == nil {
		return Held{}, errors.New("local database not ready")
	}
	var out struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Cart  string `json:"cart"`
	}
	err := a.request("POST", "/api/v1/pos/held/"+id+"/recall", nil, &out)
	if err == nil {
		_ = a.db.DeleteHeldCart(a.ctx, id)
		return Held{ID: out.ID, Label: out.Label, Cart: out.Cart, Source: "server"}, nil
	}
	if !errors.Is(err, ErrOffline) {
		return Held{}, err
	}
	locals, err := a.db.HeldCarts(a.ctx)
	if err != nil {
		return Held{}, err
	}
	for _, h := range locals {
		if h.ID == id {
			_ = a.db.DeleteHeldCart(a.ctx, id)
			return Held{ID: h.ID, Label: h.Label, Cart: h.Cart, Source: "local"}, nil
		}
	}
	return Held{}, errors.New("held sale not found")
}

// CancelHeld deletes a parked cart.
func (a *App) CancelHeld(id string) error {
	if a.db == nil {
		return errors.New("local database not ready")
	}
	err := a.request("DELETE", "/api/v1/pos/held/"+id, nil, nil)
	if err != nil && !errors.Is(err, ErrOffline) {
		return err
	}
	return a.db.DeleteHeldCart(a.ctx, id)
}

// PayResult is what the UI shows after tendering.
type PayResult struct {
	SaleID  string `json:"sale_id"`
	Total   string `json:"total"`
	Method  string `json:"method"`
	Offline bool   `json:"offline"`
}

// Pay completes the sale locally. The pending_sales row is the source for the
// Phase 7 upload pipeline — checkout works fully offline.
func (a *App) Pay(method, cartJSON, totalsJSON string) (PayResult, error) {
	if a.db == nil {
		return PayResult{}, errors.New("local database not ready")
	}
	id := security.NewPublicID()
	cashier := a.cashierID()
	if err := a.db.RecordPendingSale(a.ctx, id, cartJSON, totalsJSON, method, cashier); err != nil {
		return PayResult{}, err
	}
	return PayResult{SaleID: id, Method: method, Offline: !a.online()}, nil
}

func (a *App) cashierID() string {
	if s := a.CurrentSession(); s != nil {
		return s.CashierID
	}
	return ""
}
