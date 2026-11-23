package catalog

import (
	"strconv"
)

// ToCategoryResponse maps a Category to its API shape.
func ToCategoryResponse(c Category, pubs map[uint64]string) CategoryResponse {
	var parent *string
	if c.ParentID != nil {
		if pub, ok := pubs[*c.ParentID]; ok {
			parent = &pub
		}
	}
	return CategoryResponse{ID: c.PublicID, ParentID: parent, Name: c.Name, Sort: c.Sort, ProductCount: c.ProductCount, CreatedAt: c.CreatedAt}
}

// ToBrandResponse maps a Brand to its API shape.
func ToBrandResponse(b Brand) BrandResponse {
	return BrandResponse{ID: b.PublicID, Name: b.Name, ProductCount: b.ProductCount}
}

// ToUnitResponse maps a Unit to its API shape. bases maps id→code.
func ToUnitResponse(u Unit, bases map[uint64]string) UnitResponse {
	var base *string
	if u.BaseUnitID != nil {
		if code, ok := bases[*u.BaseUnitID]; ok {
			base = &code
		}
	}
	var factor *string
	if u.Factor != nil {
		f := u.Factor.String()
		factor = &f
	}
	return UnitResponse{ID: strconv.FormatUint(u.ID, 10), Code: u.Code, Name: u.Name,
		Kind: u.Kind, BaseCode: base, Factor: factor, BuiltIn: u.TenantID == nil}
}

// ToProductResponse maps a Product to its list API shape. Ref pubs are filled
// by the service when present (category/brand public ids, unit code).
func ToProductResponse(p Product, catPub, brandPub, unitCode *string) ProductResponse {
	return ProductResponse{
		ID: p.PublicID, SKU: p.SKU, Name: p.Name, Kind: p.Kind,
		CategoryID: catPub, BrandID: brandPub, UnitCode: unitCode,
		SellBy: p.SellBy, Price: p.Price.String(), Cost: decString(p.Cost),
		VatRate: decString(p.VatRate), Deposit: decString(p.Deposit),
		TrackStock: p.TrackStock, Attributes: attrsMap(p.Attributes),
		Status: p.Status, UpdatedAt: p.UpdatedAt,
	}
}

// ToProductDetail maps a hydrated Product to its detail API shape.
func ToProductDetail(p Product) ProductDetail {
	vs := make([]VariantResponse, len(p.Variants))
	byID := map[uint64]string{}
	for i, v := range p.Variants {
		vs[i] = ToVariantResponse(v)
		byID[v.ID] = v.PublicID
	}
	bs := make([]BarcodeResponse, len(p.Barcodes))
	for i, b := range p.Barcodes {
		var vpub *string
		if b.VariantID != nil {
			if pub, ok := byID[*b.VariantID]; ok {
				vpub = &pub
			}
		}
		bs[i] = BarcodeResponse{Code: b.Code, VariantID: vpub, PackQty: b.PackQty.String()}
	}
	cs := make([]ComponentResponse, len(p.Components))
	for i, c := range p.Components {
		cs[i] = ComponentResponse{ProductID: c.ComponentPublicID, SKU: c.ComponentSKU,
			Name: c.ComponentName, Qty: c.Qty.String()}
	}
	imgs := make([]ImageResponse, len(p.Images))
	for i, img := range p.Images {
		imgs[i] = ImageResponse{ID: img.ID, URL: "/api/v1/t/catalog/images/" + strconv.FormatUint(img.ID, 10), Sort: img.Sort}
	}
	return ProductDetail{
		ProductResponse: ToProductResponse(p, p.CategoryPublicID, p.BrandPublicID, p.UnitCode),
		Variants:        vs, Barcodes: bs, Components: cs, Images: imgs,
	}
}

// ToListResponse maps a joined ListRow to its API shape.
func ToListResponse(r ListRow) ProductResponse {
	resp := ToProductResponse(r.Product, r.CategoryPublicID, r.BrandPublicID, r.UnitCode)
	resp.CategoryName = r.CategoryName
	resp.BrandName = r.BrandName
	resp.LowStock = r.LowStock
	if r.ImageID != nil {
		u := "/api/v1/t/catalog/images/" + strconv.FormatUint(*r.ImageID, 10)
		resp.ImageURL = &u
	}
	if r.Stock != nil {
		s := r.Stock.String()
		resp.Stock = &s
	}
	return resp
}

// ToVariantResponse maps a Variant to its API shape.
func ToVariantResponse(v Variant) VariantResponse {
	return VariantResponse{ID: v.PublicID, SKU: v.SKU, Name: v.Name,
		Price: decString(v.Price), Attributes: attrsMap(v.Attributes)}
}
