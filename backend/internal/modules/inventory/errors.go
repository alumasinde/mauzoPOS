package inventory

import (
	"mauzo/internal/core/httpx"
)

// Inventory errors.
var (
	ErrLocationNotFound = httpx.NewError(404, "location_not_found", "Location not found.")
	ErrLocationInUse    = httpx.NewError(409, "location_in_use", "Location has stock or movements and cannot be deleted.")
	ErrDocNotFound      = httpx.NewError(404, "document_not_found", "Document not found.")
	ErrBadState         = httpx.NewError(409, "bad_state", "The document is not in a state that allows this action.")
	ErrBadReason        = httpx.NewError(422, "bad_reason", "Unknown reason code.")
	ErrBadQty           = httpx.NewError(422, "bad_qty", "Quantities must be non-zero; sends/receives must be positive.")
	ErrSameBranch       = httpx.NewError(422, "same_branch", "Transfers need two different branches.")
	ErrNoItems          = httpx.NewError(422, "no_items", "The document needs at least one line.")
	ErrNotStocked       = httpx.NewError(422, "not_stocked", "This product is a service or is not stock-tracked.")
	ErrBranchAccess     = httpx.NewError(403, "branch_denied", "You do not have access to this branch.")
	ErrUncounted        = httpx.NewError(422, "uncounted_lines", "All count lines need a counted quantity before posting.")
	ErrSelfApprove      = httpx.NewError(403, "self_approve", "You cannot approve a document you created.")
	ErrBatchNotFound    = httpx.NewError(404, "batch_not_found", "Batch not found.")
)
