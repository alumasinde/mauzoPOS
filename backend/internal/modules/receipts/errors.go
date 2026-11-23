package receipts

import "mauzo/internal/core/httpx"

// Errors.
var (
	ErrTemplateNotFound = httpx.NewError(404, "template_not_found", "Receipt template not found.")
	ErrTemplateInUse    = httpx.NewError(409, "template_in_use",
		"This template is assigned to a branch. Assign another one first.")
	ErrUnknownBuiltin = httpx.NewError(422, "unknown_builtin", "Unknown starting layout.")
	ErrNoLogo         = httpx.NewError(404, "no_logo", "No receipt logo has been uploaded.")
	ErrBadImage       = httpx.NewError(422, "bad_image", "Upload a PNG, JPEG or GIF image (max 2 MB).")
	ErrBranchNotFound = httpx.NewError(404, "branch_not_found", "Branch not found.")
	ErrNotIssued      = httpx.NewError(404, "receipt_not_issued", "No receipt has been issued for this sale yet.")
	ErrStaleVersion   = httpx.NewError(409, "stale_version",
		"Someone else saved this template since you opened it. Reload to see their changes.")
)
