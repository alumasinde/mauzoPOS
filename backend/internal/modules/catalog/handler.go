package catalog

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/identity"
)

// Handler serves /t/catalog product endpoints (CRUD, lookup, import/export,
// labels, images).
type Handler struct {
	svc *Service
	tax *TaxonomyService
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, tax *TaxonomyService) *Handler {
	return &Handler{svc: svc, tax: tax}
}

// List handles GET /t/catalog/products.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, ok := identity.ActorFrom(r.Context())
	if !ok || a.TenantID == 0 {
		httpx.WriteError(w, r, httpx.ErrForbidden)
		return
	}
	page := httpx.ParsePage(r)
	f := ListFilter{Query: page.Query}
	if status := r.URL.Query().Get("status"); status == StatusArchived {
		f.Status = StatusArchived
	} else {
		f.Status = StatusActive
	}
	if cat := r.URL.Query().Get("category_id"); cat != "" {
		c, err := h.tax.CategoryByPublic(r.Context(), a.TenantID, cat)
		if err != nil {
			httpx.WriteError(w, r, ErrNotFound)
			return
		}
		f.CategoryID = c
	}
	rows, total, err := h.svc.List(r.Context(), a.TenantID, f, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]ProductResponse, len(rows))
	for i, row := range rows {
		out[i] = ToListResponse(row)
	}
	httpx.List(w, out, page.Meta(total))
}

// Create handles POST /t/catalog/products.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req ProductRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.svc.Create(r.Context(), tenantID, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ToProductDetail(p))
}

// Get handles GET /t/catalog/products/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	p, err := h.svc.Get(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToProductDetail(p))
}

// Update handles PATCH /t/catalog/products/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var req ProductRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.svc.Update(r.Context(), tenantID, chi.URLParam(r, "id"), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, ToProductDetail(p))
}

// Archive handles DELETE /t/catalog/products/{id}.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	if err := h.svc.Archive(r.Context(), tenantID, chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Lookup handles GET /t/catalog/lookup?code=.
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(map[string]any{"code": "required"}))
		return
	}
	res, err := h.svc.Lookup(r.Context(), tenantID, code)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, res)
}

// Labels handles GET /t/catalog/labels?products=a,b&copies=n.
func (h *Handler) Labels(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	pubs := []string{}
	for _, part := range strings.Split(r.URL.Query().Get("products"), ",") {
		if part = strings.TrimSpace(part); part != "" {
			pubs = append(pubs, part)
		}
	}
	if len(pubs) == 0 {
		httpx.WriteError(w, r, httpx.ErrValidation.WithDetails(map[string]any{"products": "required"}))
		return
	}
	copies, _ := strconv.Atoi(r.URL.Query().Get("copies"))
	labels, err := h.svc.Labels(r.Context(), tenantID, pubs, copies)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, labels)
}

// xlsxMIME is the Office Open XML spreadsheet content type.
const xlsxMIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Import handles POST /t/catalog/products/import. The body may be text/csv or
// an .xlsx workbook (detected by Content-Type, else file extension via
// ?format=xlsx).
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	defer func() { _ = r.Body.Close() }()
	body := io.LimitReader(r.Body, 8<<20)
	var res ImportResult
	var err error
	if isXLSX(r) {
		res, err = h.svc.ImportXLSX(r.Context(), tenantID, body)
	} else {
		res, err = h.svc.ImportCSV(r.Context(), tenantID, body)
	}
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.OK(w, res)
}

func isXLSX(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, xlsxMIME) || strings.EqualFold(r.URL.Query().Get("format"), "xlsx")
}

// Export handles GET /t/catalog/products/export?format=csv|xlsx.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	var buf bytes.Buffer
	var err error
	if strings.EqualFold(r.URL.Query().Get("format"), "xlsx") {
		err = h.svc.ExportXLSX(r.Context(), tenantID, &buf)
	} else {
		err = h.svc.ExportCSV(r.Context(), tenantID, &buf)
	}
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("format"), "xlsx") {
		w.Header().Set("Content-Type", xlsxMIME)
		w.Header().Set("Content-Disposition", `attachment; filename="products.xlsx"`)
	} else {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="products.csv"`)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// UploadImage handles POST /t/catalog/products/{id}/images (multipart file field "image").
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes)
	//nolint:gosec // body bounded by MaxBytesReader above; maxMemory only sizes the in-memory spool
	if err := r.ParseMultipartForm(maxImageBytes); err != nil {
		httpx.WriteError(w, r, ErrBadImage)
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		httpx.WriteError(w, r, ErrBadImage)
		return
	}
	defer func() { _ = file.Close() }()
	sniff := make([]byte, 512)
	n, _ := io.ReadFull(file, sniff)
	img, err := h.svc.AddImage(r.Context(), tenantID, chi.URLParam(r, "id"),
		io.MultiReader(bytes.NewReader(sniff[:n]), file), sniff[:n])
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.Created(w, ImageResponse{ID: img.ID,
		URL: "/api/t/catalog/images/" + strconv.FormatUint(img.ID, 10), Sort: img.Sort})
}

// ServeImage handles GET /t/catalog/images/{id}.
func (h *Handler) ServeImage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, r, ErrNotFound)
		return
	}
	rc, contentType, err := h.svc.OpenImage(r.Context(), tenantID, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer func() { _ = rc.Close() }()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = io.Copy(w, rc)
}

// DeleteImage handles DELETE /t/catalog/images/{id}.
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantOf(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, r, ErrNotFound)
		return
	}
	if err := h.svc.DeleteImage(r.Context(), tenantID, id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}
