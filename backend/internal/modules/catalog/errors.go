package catalog

import "mauzo/internal/core/httpx"

// Catalog errors.
var (
	ErrNotFound      = httpx.NewError(404, "catalog_not_found", "The requested catalog item was not found.")
	ErrSKUTaken      = httpx.NewError(409, "sku_taken", "A product with that SKU already exists.")
	ErrBarcodeTaken  = httpx.NewError(409, "barcode_taken", "That barcode is already assigned to another product.")
	ErrNameTaken     = httpx.NewError(409, "name_taken", "That name is already in use.")
	ErrBadBarcode    = httpx.NewError(422, "invalid_barcode", "That barcode is not valid (check digit mismatch or reserved prefix).")
	ErrCategoryInUse = httpx.NewError(409, "category_in_use", "Remove sub-categories and products from the category first.")
	ErrBadVariant    = httpx.NewError(422, "invalid_variant", "A variant is invalid for this product.")
	ErrBadComposite  = httpx.NewError(422, "invalid_composite", "Composite products need at least one existing, non-composite component.")
	ErrBadUnit       = httpx.NewError(422, "invalid_unit", "The unit does not match the product's sell mode.")
	ErrImportRow     = httpx.NewError(422, "invalid_import", "The import file could not be read.")
	ErrBadImage      = httpx.NewError(422, "invalid_image", "The image must be a JPEG, PNG or WebP under 5 MB.")
)
