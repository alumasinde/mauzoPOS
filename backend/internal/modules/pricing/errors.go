package pricing

import (
	"net/http"

	"mauzo/internal/core/httpx"
)

// Pricing errors.
var (
	ErrListNotFound  = httpx.NewError(http.StatusNotFound, "price_list_not_found", "Price list not found.")
	ErrPromoNotFound = httpx.NewError(http.StatusNotFound, "promotion_not_found", "Promotion not found.")
	ErrBadKind       = httpx.NewError(http.StatusUnprocessableEntity, "invalid_kind", "Promotion kind must be percent, amount or bxgy.")
	ErrNoItems       = httpx.NewError(http.StatusUnprocessableEntity, "no_items", "At least one item is required.")
	ErrBranchAccess  = httpx.NewError(http.StatusForbidden, "branch_access_denied", "You cannot access that branch.")
	ErrDuplicateName = httpx.NewError(http.StatusConflict, "name_taken", "A price list with that name already exists.")
)
