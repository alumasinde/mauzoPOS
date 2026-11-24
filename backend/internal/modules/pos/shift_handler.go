package pos

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/shifts"
)

// ShiftStatus handles GET /pos/shift — the till's open shift (or null).
func (h *Handler) ShiftStatus(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	if h.svc.shifts == nil {
		httpx.OK(w, map[string]any{"shift": nil})
		return
	}
	cur, err := h.svc.shifts.Current(r.Context(), d.TenantID, d.TerminalID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if cur == nil {
		httpx.OK(w, map[string]any{"shift": nil})
		return
	}
	httpx.OK(w, map[string]any{"shift": cur})
}

// ShiftOpenRequest is POST /pos/shift/open.
type ShiftOpenRequest struct {
	CashierID string `json:"cashier_id"`
	Float     string `json:"float"`
}

// ShiftOpen starts the till's drawer session (pos.shift.open).
func (h *Handler) ShiftOpen(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req ShiftOpenRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.shifts == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, PermShiftOpen)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sh, err := h.svc.shifts.Open(r.Context(), shifts.OpenInput{
		TenantID: d.TenantID, BranchID: d.BranchID, TerminalID: d.TerminalID,
		CashierID: cashierID, Float: req.Float,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, sh)
}

// ShiftCashRequest is POST /pos/shift/cash — a manual cash in/out.
type ShiftCashRequest struct {
	CashierID string `json:"cashier_id"`
	Direction string `json:"direction"` // in | out
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
}

// ShiftCash records a drop or paid-out (pos.cash.in_out — sensitive, and
// bounded by the role's max_cash_out_amount).
func (h *Handler) ShiftCash(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req ShiftCashRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.shifts == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, PermCashInOut)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var maxOut *decimal.Decimal
	if authz, aerr := h.svc.sales.Authz(r.Context(), d.TenantID, cashierID); aerr == nil &&
		authz.MaxCashOut != nil {
		if lim, perr := decimal.NewFromString(*authz.MaxCashOut); perr == nil {
			maxOut = &lim
		}
	}
	m, err := h.svc.shifts.AddMovement(r.Context(), shifts.MovementInput{
		TenantID: d.TenantID, BranchID: d.BranchID, TerminalID: d.TerminalID,
		CashierID: cashierID, Direction: req.Direction,
		Amount: req.Amount, Reason: req.Reason, MaxOut: maxOut,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, m)
}

// ShiftDrawerRequest is POST /pos/shift/drawer.
type ShiftDrawerRequest struct {
	CashierID string `json:"cashier_id"`
}

// ShiftDrawer audits a no-sale drawer kick (pos.drawer.open — sensitive).
// The physical kick happens client-side; this endpoint only records it.
func (h *Handler) ShiftDrawer(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req ShiftDrawerRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.shifts == nil {
		httpx.OK(w, map[string]any{"recorded": false})
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, PermDrawerOpen)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.shifts.DrawerOpen(r.Context(), d.TenantID, d.BranchID,
		d.TerminalID, cashierID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"recorded": true})
}

// ShiftXReport handles GET /pos/shift/report — running totals for the open
// shift (pos.report.x).
func (h *Handler) ShiftXReport(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	cashierID, err := h.cashierCan(r, d.TenantID,
		r.URL.Query().Get("cashier_id"), PermReportX)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_ = cashierID
	if h.svc.shifts == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	cur, err := h.svc.shifts.Current(r.Context(), d.TenantID, d.TerminalID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if cur == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	rep, err := h.svc.shifts.Report(r.Context(), *cur)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rep)
}

// ShiftZReport handles GET /pos/shift/{id}/report — the stored Z-report for
// a closed shift (pos.report.z), or the X for an open one.
func (h *Handler) ShiftZReport(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	cashierID, err := h.cashierCan(r, d.TenantID,
		r.URL.Query().Get("cashier_id"), PermReportZ)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_ = cashierID
	if h.svc.shifts == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	rep, err := h.svc.shifts.ReportByPublic(r.Context(), d.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rep)
}

// ShiftCloseRequest is POST /pos/shift/close — the blind count.
type ShiftCloseRequest struct {
	CashierID string `json:"cashier_id"`
	Declared  string `json:"declared"` // counted drawer cash
	Note      string `json:"note"`
}

// ShiftClose ends the open shift and returns the Z-report (pos.shift.close).
func (h *Handler) ShiftClose(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req ShiftCloseRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.shifts == nil {
		httpx.WriteError(w, r, shifts.ErrNoShift)
		return
	}
	cashierID, err := h.cashierCan(r, d.TenantID, req.CashierID, PermShiftClose)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rep, err := h.svc.shifts.Close(r.Context(), shifts.CloseInput{
		TenantID: d.TenantID, BranchID: d.BranchID, TerminalID: d.TerminalID,
		CashierID: cashierID, Declared: req.Declared, Note: req.Note,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, rep)
}
