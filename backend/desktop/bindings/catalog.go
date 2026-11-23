package bindings

import (
	"errors"

	"mauzo/desktop/localdb"
	"mauzo/pkg/barcode"

	"github.com/shopspring/decimal"
)

// ScannedItem is the result of one scan/search pick: what to add to the cart.
type ScannedItem struct {
	Product       localdb.Product `json:"product"`
	Qty           string          `json:"qty"`            // decimal; pack qty or embedded weight
	PriceOverride string          `json:"price_override"` // embedded price wins when set
	Unknown       bool            `json:"unknown"`        // code not found — UI shows the not-found panel
}

// Categories lists the grid filter tiles.
func (a *App) Categories() ([]localdb.Category, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	return a.db.Categories(a.ctx)
}

// Grid lists products for a category tile (0 = all).
func (a *App) Grid(categoryID uint64) ([]localdb.Product, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	return a.db.Grid(a.ctx, categoryID, 60)
}

// SearchProducts matches name/SKU substrings (products + variants).
func (a *App) SearchProducts(q string) ([]localdb.Product, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	return a.db.Search(a.ctx, q, 24)
}

// Scan resolves one scanned code. Order: scale label → exact barcode → item
// code → unknown. Weighted labels carry qty (kg) or a price override.
func (a *App) Scan(code string) (ScannedItem, error) {
	if a.db == nil {
		return ScannedItem{}, errors.New("local database not ready")
	}
	item, err := a.scanLocal(code)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, localdb.ErrNotFound) {
		return item, err
	}
	return ScannedItem{Unknown: true}, nil
}

func (a *App) scanLocal(code string) (ScannedItem, error) {
	// Variable-measure label (e.g. 20PPPPPVVVVVC): resolve the item code.
	if sc, ok := barcode.ParseScale(code, a.scale); ok {
		p, err := a.db.ByItemCode(a.ctx, sc.Item)
		if err != nil {
			return ScannedItem{}, localdb.ErrNotFound
		}
		item := ScannedItem{Product: p, Qty: "1"}
		if a.scale.Embedded == barcode.EmbeddedWeight {
			item.Qty = sc.Value.String()
		} else {
			item.PriceOverride = sc.Value.StringFixed(2)
		}
		return item, nil
	}
	p, packQty, err := a.db.ByBarcode(a.ctx, code)
	if err != nil {
		return ScannedItem{}, err
	}
	qty := packQty
	if _, derr := decimal.NewFromString(packQty); derr != nil {
		qty = "1"
	}
	return ScannedItem{Product: p, Qty: qty}, nil
}

// PriceCheck is F6: resolved price + VAT for a code, without touching the cart.
func (a *App) PriceCheck(code string) (map[string]any, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	item, err := a.scanLocal(code)
	if err != nil {
		if errors.Is(err, localdb.ErrNotFound) {
			return map[string]any{"unknown": true, "code": code}, nil
		}
		return nil, err
	}
	p := item.Product
	return map[string]any{
		"unknown": false, "name": p.Name, "sku": p.SKU,
		"price": p.Price, "unit": p.UnitCode, "sell_by": p.SellBy,
	}, nil
}
