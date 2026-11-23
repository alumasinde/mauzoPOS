package pos

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/modules/receipts"
	"mauzo/internal/modules/sales"
	"mauzo/pkg/receiptfmt"
)

// Receipts is the receipt surface the device endpoints use.
type Receipts interface {
	Issue(ctx context.Context, sd sales.SaleDetail, in receipts.IssueInput) (receipts.Receipt, error)
	Reprint(ctx context.Context, sd sales.SaleDetail, in receipts.ReprintInput) (receipts.Receipt, error)
	Kit(ctx context.Context, tenantID, branchID uint64) (receiptfmt.Kit, error)
}

// BindReceipts attaches the receipts service.
func (s *Service) BindReceipts(v Receipts) { s.receipts = v }

// permReprint is checked against the cashier (or a supervisor grant).
const permReprint = PermReceiptReprint

// ErrSupervisorRequired asks the till to collect a supervisor PIN for the
// permission in details.permission.
var ErrSupervisorRequired = httpx.NewError(403, "supervisor_required",
	"A supervisor must authorise this reprint.").WithDetails(map[string]any{"permission": permReprint})

// ReceiptView is the printable receipt returned to a terminal.
type ReceiptView struct {
	ID     string            `json:"id"`
	CopyNo int               `json:"copy_no"`
	Output receiptfmt.Output `json:"output"`
}

// issueReceipt snapshots the original receipt of a just-completed sale. A
// failure never fails the sale (it is committed): the till falls back to
// rendering locally from its kit, and the back office issues it on demand.
func (s *Service) issueReceipt(ctx context.Context, d Device, sd sales.SaleDetail, cashierID uint64) *ReceiptView {
	if s.receipts == nil {
		return nil
	}
	term := d.TerminalID
	rc, err := s.receipts.Issue(ctx, sd, receipts.IssueInput{
		Channel: receipts.ChannelPOS, TerminalID: &term, UserID: &cashierID,
	})
	if err != nil {
		slog.Default().Warn("pos: issue receipt", "sale", sd.Sale.PublicID, "err", err)
		return nil
	}
	return &ReceiptView{ID: rc.Issued.PublicID, CopyNo: rc.Issued.CopyNo, Output: rc.Output}
}

// ReprintRequest is POST /pos/sales/{ref}/reprint.
type ReprintRequest struct {
	CashierID       string `json:"cashier_id"`
	SupervisorGrant string `json:"supervisor_grant"`
}

// Reprint handles POST /pos/sales/{ref}/reprint — an audited copy of a
// receipt from this branch. The cashier needs pos.receipt.reprint or a
// supervisor grant for it.
func (h *Handler) Reprint(w http.ResponseWriter, r *http.Request) {
	d, _ := DeviceFrom(r.Context())
	var req ReprintRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if h.svc.sales == nil || h.svc.receipts == nil {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	cashierID, err := h.svc.userByPublicID(r.Context(), d.TenantID, req.CashierID)
	if err != nil {
		httpx.WriteError(w, r, sales.ErrCashierNotFound)
		return
	}
	authz, err := h.svc.sales.Authz(r.Context(), d.TenantID, cashierID)
	if err != nil {
		httpx.WriteError(w, r, sales.ErrCashierNotFound)
		return
	}
	var supervisorID *uint64
	if !authz.Perms[permReprint] {
		supervisorID, err = h.svc.ConsumeGrant(d, req.SupervisorGrant, permReprint)
		if err != nil || supervisorID == nil {
			httpx.WriteError(w, r, ErrSupervisorRequired)
			return
		}
	}
	sd, err := h.svc.sales.GetByReceipt(r.Context(), d.TenantID, 0, chi.URLParam(r, "ref"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if sd.Sale.BranchID != d.BranchID {
		httpx.WriteError(w, r, httpx.ErrNotFound)
		return
	}
	term := d.TerminalID
	rc, err := h.svc.receipts.Reprint(r.Context(), sd, receipts.ReprintInput{
		Channel: receipts.ChannelPOS, TerminalID: &term, UserID: &cashierID,
		SupervisorID: supervisorID, ActorType: "user",
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ReceiptView{ID: rc.Issued.PublicID, CopyNo: rc.Issued.CopyNo, Output: rc.Output})
}
