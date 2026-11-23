package receipts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/pkg/receiptfmt"
)

// Service owns templates, assignments, the logo and issued receipts.
type Service struct {
	conn *sqlx.DB
	repo *Repo
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo) *Service {
	return &Service{conn: conn, repo: repo}
}

func (s *Service) inTx(ctx context.Context, fn func(r *Repo) error) error {
	return db.WithTx(ctx, s.conn, func(tx *sqlx.Tx) error { return fn(s.repo.with(tx)) })
}

// ---- templates ----------------------------------------------------------------

// Templates lists the tenant's layouts.
func (s *Service) Templates(ctx context.Context, tenantID uint64) ([]Template, error) {
	return s.repo.Templates(ctx, tenantID)
}

// Template returns one layout.
func (s *Service) Template(ctx context.Context, tenantID uint64, publicID string) (Template, error) {
	return s.repo.Template(ctx, tenantID, publicID)
}

// CreateInput starts a template from a builtin, another template, or a body.
type CreateInput struct {
	Name     string          `json:"name"`
	Builtin  string          `json:"builtin,omitempty"`   // builtin key
	CopyFrom string          `json:"copy_from,omitempty"` // template public id
	Body     json.RawMessage `json:"body,omitempty"`
}

// CreateTemplate saves a new layout (version 1).
func (s *Service) CreateTemplate(ctx context.Context, tenantID, userID uint64, in CreateInput) (Template, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 120 {
		return Template{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "1–120 characters"})
	}
	body, based := in.Body, ""
	switch {
	case in.Builtin != "":
		b, ok := receiptfmt.BuiltinByKey(in.Builtin)
		if !ok {
			return Template{}, ErrUnknownBuiltin
		}
		body, based = b.Raw, b.Key
	case in.CopyFrom != "":
		src, err := s.repo.Template(ctx, tenantID, in.CopyFrom)
		if err != nil {
			return Template{}, err
		}
		body, based = src.Body, src.BasedOn
	}
	tpl, canon, err := parseBody(body)
	if err != nil {
		return Template{}, err
	}
	t := Template{
		TenantID: tenantID, PublicID: security.NewPublicID(), Name: name,
		PaperWidth: tpl.Paper, Body: canon, BasedOn: based, CreatedBy: &userID,
	}
	if err := s.inTx(ctx, func(r *Repo) error { return r.InsertTemplate(ctx, &t) }); err != nil {
		return Template{}, err
	}
	return t, nil
}

// UpdateInput saves a new layout version. Version is the version the editor
// loaded; a mismatch means someone else saved in between.
type UpdateInput struct {
	Name    string          `json:"name"`
	Body    json.RawMessage `json:"body"`
	Version int             `json:"version"`
}

// UpdateTemplate stores a new version (or only renames when the body is
// unchanged).
func (s *Service) UpdateTemplate(ctx context.Context, tenantID, userID uint64, publicID string, in UpdateInput) (Template, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 120 {
		return Template{}, httpx.ErrValidation.WithDetails(map[string]any{"name": "1–120 characters"})
	}
	tpl, canon, err := parseBody(in.Body)
	if err != nil {
		return Template{}, err
	}
	var out Template
	err = s.inTx(ctx, func(r *Repo) error {
		t, err := r.LockTemplate(ctx, tenantID, publicID)
		if err != nil {
			return err
		}
		if in.Version != 0 && in.Version != t.Version {
			return ErrStaleVersion
		}
		if sameJSON(t.Body, canon) {
			if t.Name != name {
				if err := r.RenameTemplate(ctx, tenantID, t.ID, name, userID); err != nil {
					return err
				}
				t.Name = name
			}
			out = t
			return nil
		}
		t.Name, t.PaperWidth, t.Body = name, tpl.Paper, canon
		t.Version++
		t.UpdatedBy = &userID
		out = t
		return r.UpdateTemplate(ctx, &t)
	})
	return out, err
}

// ArchiveTemplate removes a layout from the library unless a branch uses it.
func (s *Service) ArchiveTemplate(ctx context.Context, tenantID uint64, publicID string) error {
	t, err := s.repo.Template(ctx, tenantID, publicID)
	if err != nil {
		return err
	}
	used, err := s.repo.TemplateAssigned(ctx, tenantID, t.ID)
	if err != nil {
		return err
	}
	if used {
		return ErrTemplateInUse
	}
	return s.repo.ArchiveTemplate(ctx, tenantID, t.ID)
}

// Versions lists a template's saved versions.
func (s *Service) Versions(ctx context.Context, tenantID uint64, publicID string) ([]Version, error) {
	t, err := s.repo.Template(ctx, tenantID, publicID)
	if err != nil {
		return nil, err
	}
	return s.repo.Versions(ctx, tenantID, t.ID)
}

// parseBody validates a template and returns its canonical JSON (re-encoded so
// stored bodies are uniform and free of unknown fields).
func parseBody(raw json.RawMessage) (receiptfmt.Template, []byte, error) {
	if len(raw) == 0 {
		return receiptfmt.Template{}, nil, httpx.ErrValidation.WithDetails(map[string]any{"body": "required"})
	}
	t, err := receiptfmt.Parse(raw)
	if err != nil {
		var ve *receiptfmt.ValidationError
		if errors.As(err, &ve) {
			return t, nil, httpx.ErrValidation.WithDetails(map[string]any{"body": ve.Problems})
		}
		return t, nil, httpx.ErrValidation.WithDetails(map[string]any{"body": []string{err.Error()}})
	}
	canon, err := json.Marshal(t)
	return t, canon, err
}

func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	ax, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return string(ax) == string(by)
}

// ---- assignments ------------------------------------------------------------

// Assignments lists which layout each branch prints.
func (s *Service) Assignments(ctx context.Context, tenantID uint64) ([]Assignment, error) {
	return s.repo.Assignments(ctx, tenantID)
}

// Assign sets (templatePub != "") or clears a branch's layout. branchPub ""
// means "all branches".
func (s *Service) Assign(ctx context.Context, tenantID, userID uint64, branchPub, templatePub string) error {
	var branchID uint64
	if branchPub != "" {
		id, err := s.repo.BranchIDByPublicID(ctx, tenantID, branchPub)
		if err != nil {
			return err
		}
		branchID = id
	}
	if templatePub == "" {
		return s.repo.DeleteAssignment(ctx, tenantID, branchID)
	}
	t, err := s.repo.Template(ctx, tenantID, templatePub)
	if err != nil {
		return err
	}
	return s.repo.SetAssignment(ctx, tenantID, branchID, t.ID, userID)
}

// resolved is the layout a branch prints right now.
type resolved struct {
	tpl        receiptfmt.Template
	raw        []byte
	templateID *uint64
	publicID   string
	version    *int
	builtinKey string
}

// resolve picks the branch's layout: its own assignment, then the
// all-branches assignment, then the builtin of the business profile.
func (s *Service) resolve(ctx context.Context, tenantID, branchID uint64, set settingsView) (resolved, error) {
	id, ok, err := s.repo.AssignedTemplateID(ctx, tenantID, branchID)
	if err != nil {
		return resolved{}, err
	}
	if ok {
		t, err := s.repo.TemplateByID(ctx, tenantID, id)
		if err != nil {
			return resolved{}, err
		}
		if tpl, err := receiptfmt.Parse(t.Body); err == nil {
			v := t.Version
			return resolved{tpl: tpl, raw: t.Body, templateID: &t.ID, publicID: t.PublicID, version: &v}, nil
		}
		// A stored body that no longer parses must not stop the till: fall
		// through to the builtin.
	}
	b := receiptfmt.DefaultFor(set.profileTemplate, set.paperWidth)
	return resolved{tpl: b.Template, raw: b.Raw, builtinKey: b.Key}, nil
}

// ---- logo ---------------------------------------------------------------------

const maxLogoUpload = 2 << 20

// SetLogo converts an uploaded image to a 1-bit raster and stores it.
func (s *Service) SetLogo(ctx context.Context, tenantID, userID uint64, data []byte) (receiptfmt.Raster, error) {
	if len(data) == 0 || len(data) > maxLogoUpload {
		return receiptfmt.Raster{}, ErrBadImage
	}
	r, err := receiptfmt.LogoFromImage(data, receiptfmt.LogoMaxWidth, receiptfmt.LogoMaxHeight)
	if err != nil {
		return receiptfmt.Raster{}, ErrBadImage
	}
	return r, s.repo.SetLogo(ctx, tenantID, Logo{Width: r.Width, Height: r.Height, Bits: r.Bits}, userID)
}

// Logo returns the stored logo raster.
func (s *Service) Logo(ctx context.Context, tenantID uint64) (*receiptfmt.Raster, error) {
	l, err := s.repo.Logo(ctx, tenantID)
	if errors.Is(err, ErrNoLogo) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r := receiptfmt.Raster{Width: l.Width, Height: l.Height, Bits: l.Bits}
	if !r.Valid() {
		return nil, nil
	}
	return &r, nil
}

// DeleteLogo removes the logo.
func (s *Service) DeleteLogo(ctx context.Context, tenantID uint64) error {
	return s.repo.DeleteLogo(ctx, tenantID)
}
