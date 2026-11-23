package receipts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
	"mauzo/internal/modules/sales"
	"mauzo/pkg/receiptfmt"
)

// SaleReader loads sales for back-office receipt views.
type SaleReader interface {
	Get(ctx context.Context, tenantID uint64, publicID string) (sales.SaleDetail, error)
}

// Handler serves /t/receipts.
type Handler struct {
	svc   *Service
	sales SaleReader
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, sr SaleReader) *Handler { return &Handler{svc: svc, sales: sr} }

func actor(r *http.Request) (identity.Actor, bool) {
	a, ok := identity.ActorFrom(r.Context())
	return a, ok && a.TenantID != 0
}

// Meta handles GET /t/receipts/meta.
func (h *Handler) Meta(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	logo, err := h.svc.Logo(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := MetaResponse{SchemaVersion: receiptfmt.SchemaVersion, HasLogo: logo != nil}
	for _, b := range receiptfmt.Builtins {
		out.Builtins = append(out.Builtins, BuiltinResponse{Key: b.Key, Name: b.Name,
			Description: b.Description, Body: b.Raw})
	}
	for _, v := range receiptfmt.Variables {
		out.Variables = append(out.Variables, NamedDoc{v.Name, v.Description})
	}
	for _, c := range receiptfmt.Conditions {
		out.Conditions = append(out.Conditions, NamedDoc{c.Name, c.Description})
	}
	httpx.OK(w, out)
}

// List handles GET /t/receipts/templates.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Templates(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]TemplateResponse, len(rows))
	for i, t := range rows {
		out[i] = toTemplate(t)
	}
	httpx.OK(w, out)
}

// Get handles GET /t/receipts/templates/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	t, err := h.svc.Template(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toTemplate(t))
}

// Create handles POST /t/receipts/templates.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.CreateTemplate(r.Context(), a.TenantID, a.ID, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, toTemplate(t))
}

// Update handles PUT /t/receipts/templates/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in UpdateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.UpdateTemplate(r.Context(), a.TenantID, a.ID, chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, toTemplate(t))
}

// Archive handles DELETE /t/receipts/templates/{id}.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.svc.ArchiveTemplate(r.Context(), a.TenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Versions handles GET /t/receipts/templates/{id}/versions.
func (h *Handler) Versions(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Versions(r.Context(), a.TenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]VersionResponse, len(rows))
	for i, v := range rows {
		out[i] = VersionResponse{Version: v.Version, PaperWidth: v.PaperWidth, Author: v.AuthorName,
			Body: json.RawMessage(v.Body), CreatedAt: v.CreatedAt}
	}
	httpx.OK(w, out)
}

// Assignments handles GET /t/receipts/assignments.
func (h *Handler) Assignments(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	rows, err := h.svc.Assignments(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]AssignmentResponse, len(rows))
	for i, x := range rows {
		out[i] = AssignmentResponse{BranchID: x.BranchPublicID, BranchName: x.BranchName,
			TemplateID: x.TemplatePublicID, TemplateName: x.TemplateName}
	}
	httpx.OK(w, out)
}

// Assign handles PUT /t/receipts/assignments.
func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in AssignRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if in.BranchID != "" {
		id, err := h.svc.BranchID(r.Context(), a.TenantID, in.BranchID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if !a.CanAccessBranch(id) {
			httpx.WriteError(w, r, httpx.ErrForbidden)
			return
		}
	} else if !a.Access.AllBranches {
		httpx.WriteError(w, r, httpx.ErrForbidden) // "all branches" needs all-branch access
		return
	}
	if err := h.svc.Assign(r.Context(), a.TenantID, a.ID, in.BranchID, in.TemplateID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.Assignments(w, r)
}

// Preview handles POST /t/receipts/preview.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	var in PreviewInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	branchID, err := h.svc.BranchID(r.Context(), a.TenantID, in.BranchID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var sd *sales.SaleDetail
	if in.Sale != "" {
		d, err := h.sale(r, a, in.Sale)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		sd = &d
	}
	out, logo, err := h.svc.Preview(r.Context(), a.TenantID, in.Body, branchID, sd)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, PreviewResponse{Output: out, HasLogo: logo != nil})
}

// LogoPNG handles GET /t/receipts/logo — the stored 1-bit logo as PNG.
func (h *Handler) LogoPNG(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	logo, err := h.svc.Logo(r.Context(), a.TenantID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if logo == nil {
		httpx.WriteError(w, r, ErrNoLogo)
		return
	}
	png, err := logo.PNG()
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// UploadLogo handles PUT /t/receipts/logo (multipart "file" or a raw image body).
func (h *Handler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLogoUpload+64<<10)
	var data []byte
	var err error
	if f, _, ferr := r.FormFile("file"); ferr == nil {
		defer func() { _ = f.Close() }()
		data, err = io.ReadAll(io.LimitReader(f, maxLogoUpload+1))
	} else {
		data, err = io.ReadAll(io.LimitReader(r.Body, maxLogoUpload+1))
	}
	if err != nil {
		httpx.WriteError(w, r, ErrBadImage)
		return
	}
	logo, err := h.svc.SetLogo(r.Context(), a.TenantID, a.ID, data)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]int{"width": logo.Width, "height": logo.Height})
}

// DeleteLogo handles DELETE /t/receipts/logo.
func (h *Handler) DeleteLogo(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	if err := h.svc.DeleteLogo(r.Context(), a.TenantID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// sale loads a sale the actor's branch access allows.
func (h *Handler) sale(r *http.Request, a identity.Actor, publicID string) (sales.SaleDetail, error) {
	if h.sales == nil {
		return sales.SaleDetail{}, httpx.ErrNotFound
	}
	d, err := h.sales.Get(r.Context(), a.TenantID, publicID)
	if err != nil {
		return d, err
	}
	if !a.CanAccessBranch(d.Sale.BranchID) {
		return d, httpx.ErrNotFound
	}
	return d, nil
}

// SaleReceipt handles GET /t/receipts/sales/{id}: the original receipt and
// every copy issued.
func (h *Handler) SaleReceipt(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	d, err := h.sale(r, a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rc, err := h.svc.ForSale(r.Context(), d, a.ID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.writeReceipt(w, r, rc, d.Sale.TenantID, d.Sale.ID)
}

// Reprint handles POST /t/receipts/sales/{id}/reprint (pos.receipt.reprint).
func (h *Handler) Reprint(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	d, err := h.sale(r, a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	uid := a.ID
	rc, err := h.svc.Reprint(r.Context(), d, ReprintInput{Channel: ChannelBackoffice, UserID: &uid, ActorType: "user"})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.writeReceipt(w, r, rc, d.Sale.TenantID, d.Sale.ID)
}

// Invoice handles GET /t/receipts/sales/{id}/invoice: the data for the A4
// invoice / PDF view.
func (h *Handler) Invoice(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(r)
	if !ok {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	d, err := h.sale(r, a, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	doc, err := h.svc.InvoiceDoc(r.Context(), d)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, map[string]any{"doc": doc, "status": d.Sale.Status})
}

func (h *Handler) writeReceipt(w http.ResponseWriter, r *http.Request, rc Receipt, tenantID, saleID uint64) {
	copies, err := h.svc.Copies(r.Context(), tenantID, saleID)
	if err != nil && !errors.Is(err, ErrNotIssued) {
		httpx.WriteError(w, r, err)
		return
	}
	out := ReceiptResponse{Output: rc.Output, HasLogo: rc.Logo != nil, Issued: toIssued(rc.Issued)}
	for _, c := range copies {
		out.Copies = append(out.Copies, toIssued(c))
	}
	httpx.OK(w, out)
}
