package sales

import (
	"context"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/customers"
	"mauzo/pkg/money"
	"mauzo/pkg/pricecalc"
)

// resolveLines loads products, resolves variants and computes the unit price:
// catalog/list price unless the line carries an authorized override.
func (s *Service) resolveLines(ctx context.Context, in CompleteInput, authz *CashierAuthz) ([]line, error) {
	lines := make([]line, 0, len(in.Lines))
	fields := map[string]any{}
	for i, li := range in.Lines {
		pref := "lines." + strconv.Itoa(i) + "."
		p, err := s.catalog.Get(ctx, in.TenantID, li.ProductID)
		if err != nil {
			fields[pref+"product_id"] = "invalid"
			continue
		}
		var variantID uint64
		var vName, vSKU string
		if li.VariantID != "" {
			found := false
			for _, v := range p.Variants {
				if v.PublicID == li.VariantID {
					variantID, vName, found = v.ID, v.Name, true
					if v.SKU != nil {
						vSKU = *v.SKU
					}
				}
			}
			if !found {
				fields[pref+"variant_id"] = "invalid"
				continue
			}
		}
		qty, err := decimal.NewFromString(li.Qty)
		if err != nil || qty.LessThanOrEqual(decimal.Zero) {
			fields[pref+"qty"] = "must be a positive number"
			continue
		}
		l := line{product: p, variantID: variantID, variantPub: li.VariantID, qty: qty, name: p.Name, sku: p.SKU}
		if vName != "" {
			l.name = p.Name + " · " + vName
		}
		if vSKU != "" {
			l.sku = vSKU
		}
		if li.PriceOverride != nil && *li.PriceOverride != "" {
			if !authz.Perms["pos.price.override"] && in.SupervisorID == nil {
				return nil, ErrPriceOverride
			}
			ov, err := decimal.NewFromString(*li.PriceOverride)
			if err != nil || ov.LessThan(decimal.Zero) {
				fields[pref+"price_override"] = "invalid"
				continue
			}
			l.unit, l.override = ov, true
		} else {
			// List price → variant price → base price, in that order.
			rp, err := s.pricer.Resolve(ctx, in.TenantID, in.BranchID, in.CustomerGroup, p, variantID, qty)
			if err != nil {
				return nil, err
			}
			l.unit = rp.UnitPrice
		}
		lines = append(lines, l)
	}
	if len(fields) > 0 {
		return nil, httpx.ErrValidation.WithDetails(fields)
	}
	if len(lines) == 0 {
		return nil, ErrNoLines
	}
	return lines, nil
}

// applyPromos runs the shared promotions engine over the resolved lines at
// the sale's sold_at (offline sales keep their original discount context).
func (s *Service) applyPromos(ctx context.Context, tenantID uint64, at time.Time, lines []line) error {
	promos, err := s.promos.ActivePromotions(ctx, tenantID)
	if err != nil {
		return err
	}
	pcLines := make([]pricecalc.Line, len(lines))
	for i, l := range lines {
		var cat uint64
		if l.product.CategoryID != nil {
			cat = *l.product.CategoryID
		}
		pcLines[i] = pricecalc.Line{ProductID: l.product.ID, VariantID: l.variantID,
			CategoryID: cat, Qty: l.qty, UnitPrice: l.unit}
	}
	tot := pricecalc.Apply(pcLines, promos, at)
	for _, ld := range tot.Lines {
		lines[ld.LineIndex].discount = lines[ld.LineIndex].discount.Add(ld.Amount)
		lines[ld.LineIndex].promo = ld.Name
	}
	return nil
}

// applyManualDiscounts adds the cashier's line discounts, enforcing the
// role's max_discount_percent against the line's undiscounted value unless a
// supervisor authorized the sale.
func applyManualDiscounts(in []LineInput, lines []line, authz *CashierAuthz, supervised bool) error {
	maxPct := decimal.NewFromInt(-1)
	if authz.MaxDiscount != nil {
		if d, err := decimal.NewFromString(*authz.MaxDiscount); err == nil {
			maxPct = d
		}
	}
	for i, li := range in {
		if i >= len(lines) || li.Discount == "" || li.Discount == "0" {
			continue
		}
		d, err := decimal.NewFromString(li.Discount)
		if err != nil || d.LessThan(decimal.Zero) {
			return httpx.ErrValidation.WithDetails(map[string]any{
				"lines." + strconv.Itoa(i) + ".discount": "invalid",
			})
		}
		gross := lines[i].qty.Mul(lines[i].unit)
		if d.GreaterThan(gross) {
			return httpx.ErrValidation.WithDetails(map[string]any{
				"lines." + strconv.Itoa(i) + ".discount": "exceeds the line value",
			})
		}
		overLimit := maxPct.Sign() >= 0 && d.Mul(decimal.NewFromInt(100)).GreaterThan(gross.Mul(maxPct))
		if overLimit && !supervised && !authz.Perms["pos.discount.override"] {
			return ErrDiscountOver
		}
		if !overLimit && !supervised && !authz.Perms["pos.discount.apply"] && !authz.Perms["pos.discount.override"] {
			return ErrDiscountOver
		}
		lines[i].discount = lines[i].discount.Add(d)
	}
	return nil
}

// finalizeLines allocates the order discount across lines, computes totals
// and the inclusive VAT component per line.
func finalizeLines(orderDiscount string, lines []line, authz *CashierAuthz, supervised bool) (
	subtotal, discount, tax, total decimal.Decimal, err error) {
	subtotal = decimal.Zero
	weights := make([]decimal.Decimal, len(lines))
	for i, l := range lines {
		gross := l.qty.Mul(l.unit)
		subtotal = subtotal.Add(gross)
		weights[i] = gross.Sub(l.discount)
	}
	if orderDiscount != "" && orderDiscount != "0" {
		od, derr := decimal.NewFromString(orderDiscount)
		if derr != nil || od.LessThan(decimal.Zero) {
			return subtotal, discount, tax, total, httpx.ErrValidation.WithDetails(map[string]any{"order_discount": "invalid"})
		}
		if od.Sign() > 0 {
			maxPct := decimal.NewFromInt(-1)
			if authz.MaxDiscount != nil {
				if d, perr := decimal.NewFromString(*authz.MaxDiscount); perr == nil {
					maxPct = d
				}
			}
			overLimit := maxPct.Sign() >= 0 && od.Mul(decimal.NewFromInt(100)).GreaterThan(subtotal.Mul(maxPct))
			if overLimit && !supervised && !authz.Perms["pos.discount.override"] {
				return subtotal, discount, tax, total, ErrDiscountOver
			}
			if !supervised && !authz.Perms["pos.discount.apply"] && !authz.Perms["pos.discount.override"] {
				return subtotal, discount, tax, total, ErrDiscountOver
			}
			shares, aerr := money.Allocate(money.New(od), weights)
			if aerr != nil {
				return subtotal, discount, tax, total, aerr
			}
			for i := range lines {
				lines[i].discount = lines[i].discount.Add(shares[i].Decimal())
			}
		}
	}
	discount, tax, total = decimal.Zero, decimal.Zero, decimal.Zero
	for i := range lines {
		l := &lines[i]
		gross := l.qty.Mul(l.unit)
		if l.discount.GreaterThan(gross) {
			l.discount = gross // promo + manual can never make a line negative
		}
		l.total = gross.Sub(l.discount).Round(2)
		l.taxRate = l.product.VatRate
		if l.taxRate != nil && l.taxRate.Sign() > 0 {
			// VAT-inclusive pricing: tax = total × rate/(100+rate).
			l.tax = l.total.Mul(*l.taxRate).Div(decimal.NewFromInt(100).Add(*l.taxRate)).Round(2)
		}
		discount = discount.Add(l.discount)
		tax = tax.Add(l.tax)
		total = total.Add(l.total)
	}
	return subtotal.Round(2), discount, tax, total, nil
}

// resolveCustomer loads the optional customer account.
func (s *Service) resolveCustomer(ctx context.Context, in CompleteInput) (*customers.Customer, error) {
	if in.CustomerID == "" {
		return nil, nil
	}
	c, err := s.customers.Get(ctx, in.TenantID, in.CustomerID)
	if err != nil {
		return nil, httpx.ErrValidation.WithDetails(map[string]any{"customer_id": "invalid"})
	}
	if c.Status != customers.StatusActive {
		return nil, httpx.ErrValidation.WithDetails(map[string]any{"customer_id": "archived"})
	}
	return &c, nil
}

// validatePayments checks method rules, sums payments against the total and
// computes change. Split payments are multiple rows summing to the total.
func (s *Service) validatePayments(ctx context.Context, tenantID uint64, customer *customers.Customer,
	total decimal.Decimal, in []PaymentInput) ([]SalePayment, decimal.Decimal, error) {
	out := make([]SalePayment, 0, len(in))
	sum := decimal.Zero
	change := decimal.Zero
	fields := map[string]any{}
	credit := decimal.Zero
	for i, pi := range in {
		spec, ok := methods[pi.Method]
		if !ok {
			return nil, change, ErrUnknownMethod
		}
		amt, err := decimal.NewFromString(pi.Amount)
		if err != nil || amt.LessThanOrEqual(decimal.Zero) {
			fields["payments."+strconv.Itoa(i)+".amount"] = "invalid"
			continue
		}
		p := SalePayment{TenantID: tenantID, Method: pi.Method, Amount: amt, Status: PayConfirmed}
		if spec.RequiresReference && pi.Reference == "" {
			return nil, change, ErrNeedReference
		}
		p.Reference = pi.Reference
		if spec.Tendered {
			tendered := amt
			if pi.Tendered != nil && *pi.Tendered != "" {
				t, terr := decimal.NewFromString(*pi.Tendered)
				if terr != nil {
					fields["payments."+strconv.Itoa(i)+".tendered"] = "invalid"
					continue
				}
				tendered = t
			}
			if tendered.LessThan(amt) {
				return nil, change, ErrUnderTendered
			}
			cd := tendered.Sub(amt)
			p.Tendered, p.ChangeDue = &tendered, &cd
			change = change.Add(cd)
		}
		if spec.RequiresCustomer {
			if customer == nil {
				return nil, change, ErrNeedCustomer
			}
			credit = credit.Add(amt)
		}
		sum = sum.Add(amt)
		out = append(out, p)
	}
	if len(fields) > 0 {
		return nil, change, httpx.ErrValidation.WithDetails(fields)
	}
	if sum.LessThan(total) {
		return nil, change, ErrUnderpaid
	}
	if sum.GreaterThan(total) {
		return nil, change, ErrOverpaid
	}
	if credit.Sign() > 0 && customer.CreditLimit != nil {
		out5, err := s.customers.OutstandingCredit(ctx, tenantID, customer.ID)
		if err != nil {
			return nil, change, err
		}
		if out5.Add(credit).GreaterThan(*customer.CreditLimit) {
			return nil, change, ErrCreditExceeded
		}
	}
	return out, change.Round(2), nil
}
