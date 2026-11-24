package shifts

import "mauzo/internal/core/httpx"

// Public errors for the shifts module.
var (
	ErrShiftNotFound = httpx.NewError(404, "shift_not_found", "The shift was not found.")
	ErrShiftOpen     = httpx.NewError(409, "shift_open", "This till already has an open shift.")
	ErrNoShift       = httpx.NewError(409, "no_shift", "There is no open shift on this till.")
	ErrShiftClosed   = httpx.NewError(409, "shift_closed", "That shift is already closed.")
	ErrBadFloat      = httpx.NewError(422, "bad_float", "Float must be a non-negative amount.")
	ErrBadAmount     = httpx.NewError(422, "bad_amount", "Amount must be greater than zero.")
	ErrBadDirection  = httpx.NewError(422, "bad_direction", "Direction must be 'in' or 'out'.")
	ErrCashOutLimit  = httpx.NewError(422, "cash_out_limit", "Cash out exceeds your role limit — a supervisor must do it.")
	ErrNotYourShift  = httpx.NewError(403, "not_your_shift", "That shift belongs to another cashier.")
)
