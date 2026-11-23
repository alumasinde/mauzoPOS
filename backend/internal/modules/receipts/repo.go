package receipts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the receipts data access layer.
type Repo struct {
	q db.Querier
}

// NewRepo builds a Repo.
func NewRepo(q db.Querier) *Repo { return &Repo{q: q} }

func (r *Repo) with(q db.Querier) *Repo { return &Repo{q: q} }

const tplCols = `t.id, t.tenant_id, t.public_id, t.name, t.paper_width, t.body, t.version,
	t.based_on, t.archived_at, t.created_by, t.updated_by, t.created_at, t.updated_at`

// Templates lists live (not archived) templates, newest first.
func (r *Repo) Templates(ctx context.Context, tenantID uint64) ([]Template, error) {
	out := []Template{}
	err := r.q.SelectContext(ctx, &out, `SELECT `+tplCols+` FROM receipt_templates t
		WHERE t.tenant_id = ? AND t.archived_at IS NULL ORDER BY t.updated_at DESC`, tenantID)
	return out, err
}

// Template loads one template by public id.
func (r *Repo) Template(ctx context.Context, tenantID uint64, publicID string) (Template, error) {
	var t Template
	err := r.q.GetContext(ctx, &t, `SELECT `+tplCols+` FROM receipt_templates t
		WHERE t.tenant_id = ? AND t.public_id = ? AND t.archived_at IS NULL`, tenantID, publicID)
	return t, notFound(err, ErrTemplateNotFound)
}

// TemplateByID loads one template by id (any state).
func (r *Repo) TemplateByID(ctx context.Context, tenantID, id uint64) (Template, error) {
	var t Template
	err := r.q.GetContext(ctx, &t, `SELECT `+tplCols+` FROM receipt_templates t
		WHERE t.tenant_id = ? AND t.id = ?`, tenantID, id)
	return t, notFound(err, ErrTemplateNotFound)
}

// LockTemplate loads a template FOR UPDATE (version bumps).
func (r *Repo) LockTemplate(ctx context.Context, tenantID uint64, publicID string) (Template, error) {
	var t Template
	err := r.q.GetContext(ctx, &t, `SELECT `+tplCols+` FROM receipt_templates t
		WHERE t.tenant_id = ? AND t.public_id = ? AND t.archived_at IS NULL FOR UPDATE`, tenantID, publicID)
	return t, notFound(err, ErrTemplateNotFound)
}

// InsertTemplate creates a template and its version 1.
func (r *Repo) InsertTemplate(ctx context.Context, t *Template) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO receipt_templates
		(tenant_id, public_id, name, paper_width, body, version, based_on, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)`,
		t.TenantID, t.PublicID, t.Name, t.PaperWidth, t.Body, t.BasedOn, t.CreatedBy, t.CreatedBy)
	if err != nil {
		return err
	}
	id, err := lastID(res)
	if err != nil {
		return err
	}
	t.ID, t.Version = id, 1
	return r.InsertVersion(ctx, t)
}

// UpdateTemplate stores a new version of an existing template.
func (r *Repo) UpdateTemplate(ctx context.Context, t *Template) error {
	if _, err := r.q.ExecContext(ctx, `UPDATE receipt_templates
		SET name = ?, paper_width = ?, body = ?, version = ?, updated_by = ?
		WHERE id = ? AND tenant_id = ?`,
		t.Name, t.PaperWidth, t.Body, t.Version, t.UpdatedBy, t.ID, t.TenantID); err != nil {
		return err
	}
	return r.InsertVersion(ctx, t)
}

// RenameTemplate changes only the name (no new layout version).
func (r *Repo) RenameTemplate(ctx context.Context, tenantID, id uint64, name string, by uint64) error {
	_, err := r.q.ExecContext(ctx, `UPDATE receipt_templates SET name = ?, updated_by = ?
		WHERE id = ? AND tenant_id = ?`, name, by, id, tenantID)
	return err
}

// InsertVersion appends the immutable history row for t.Version.
func (r *Repo) InsertVersion(ctx context.Context, t *Template) error {
	by := t.UpdatedBy
	if by == nil {
		by = t.CreatedBy
	}
	_, err := r.q.ExecContext(ctx, `INSERT INTO receipt_template_versions
		(template_id, version, tenant_id, paper_width, body, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Version, t.TenantID, t.PaperWidth, t.Body, by)
	return err
}

// Versions lists a template's history, newest first.
func (r *Repo) Versions(ctx context.Context, tenantID, templateID uint64) ([]Version, error) {
	out := []Version{}
	err := r.q.SelectContext(ctx, &out, `SELECT v.template_id, v.version, v.paper_width, v.body,
		v.created_by, COALESCE(u.name, '') AS author_name, v.created_at
		FROM receipt_template_versions v LEFT JOIN users u ON u.id = v.created_by
		WHERE v.tenant_id = ? AND v.template_id = ? ORDER BY v.version DESC LIMIT 100`, tenantID, templateID)
	return out, err
}

// ArchiveTemplate hides a template from the library.
func (r *Repo) ArchiveTemplate(ctx context.Context, tenantID, id uint64) error {
	_, err := r.q.ExecContext(ctx, `UPDATE receipt_templates SET archived_at = CURRENT_TIMESTAMP(3)
		WHERE id = ? AND tenant_id = ?`, id, tenantID)
	return err
}

// Assignments lists every assignment with names.
func (r *Repo) Assignments(ctx context.Context, tenantID uint64) ([]Assignment, error) {
	out := []Assignment{}
	err := r.q.SelectContext(ctx, &out, `SELECT a.branch_id, COALESCE(b.public_id, '') AS branch_public_id,
		COALESCE(b.name, '') AS branch_name, a.template_id, t.public_id AS template_public_id,
		t.name AS template_name
		FROM receipt_assignments a
		JOIN receipt_templates t ON t.id = a.template_id
		LEFT JOIN branches b ON b.id = a.branch_id AND b.tenant_id = a.tenant_id
		WHERE a.tenant_id = ? ORDER BY a.branch_id`, tenantID)
	return out, err
}

// AssignedTemplateID returns the template for branchID, falling back to the
// all-branches row (0). ok is false when neither exists.
func (r *Repo) AssignedTemplateID(ctx context.Context, tenantID, branchID uint64) (uint64, bool, error) {
	var id uint64
	err := r.q.GetContext(ctx, &id, `SELECT template_id FROM receipt_assignments
		WHERE tenant_id = ? AND branch_id IN (?, 0) ORDER BY branch_id DESC LIMIT 1`, tenantID, branchID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

// SetAssignment upserts a branch assignment.
func (r *Repo) SetAssignment(ctx context.Context, tenantID, branchID, templateID, by uint64) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO receipt_assignments (tenant_id, branch_id, template_id, updated_by)
		VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE template_id = VALUES(template_id), updated_by = VALUES(updated_by)`,
		tenantID, branchID, templateID, by)
	return err
}

// DeleteAssignment removes a branch assignment.
func (r *Repo) DeleteAssignment(ctx context.Context, tenantID, branchID uint64) error {
	_, err := r.q.ExecContext(ctx, `DELETE FROM receipt_assignments WHERE tenant_id = ? AND branch_id = ?`,
		tenantID, branchID)
	return err
}

// TemplateAssigned reports whether any branch uses the template.
func (r *Repo) TemplateAssigned(ctx context.Context, tenantID, templateID uint64) (bool, error) {
	var n int
	err := r.q.GetContext(ctx, &n, `SELECT COUNT(*) FROM receipt_assignments
		WHERE tenant_id = ? AND template_id = ?`, tenantID, templateID)
	return n > 0, err
}

// Logo returns the tenant logo (ErrNoLogo when none).
func (r *Repo) Logo(ctx context.Context, tenantID uint64) (Logo, error) {
	var l Logo
	err := r.q.GetContext(ctx, &l, `SELECT width, height, bits, updated_at FROM receipt_logos
		WHERE tenant_id = ?`, tenantID)
	return l, notFound(err, ErrNoLogo)
}

// SetLogo upserts the tenant logo.
func (r *Repo) SetLogo(ctx context.Context, tenantID uint64, l Logo, by uint64) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO receipt_logos (tenant_id, width, height, bits, updated_by)
		VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE width = VALUES(width), height = VALUES(height),
		bits = VALUES(bits), updated_by = VALUES(updated_by)`, tenantID, l.Width, l.Height, l.Bits, by)
	return err
}

// DeleteLogo removes the tenant logo.
func (r *Repo) DeleteLogo(ctx context.Context, tenantID uint64) error {
	_, err := r.q.ExecContext(ctx, `DELETE FROM receipt_logos WHERE tenant_id = ?`, tenantID)
	return err
}

// Header loads the business and branch details printed on receipts.
func (r *Repo) Header(ctx context.Context, tenantID, branchID uint64) (Header, error) {
	var h Header
	err := r.q.GetContext(ctx, &h, `SELECT t.name AS business_name, COALESCE(t.address, '') AS address,
		COALESCE(t.phone, '') AS phone, COALESCE(t.email, '') AS email, COALESCE(t.kra_pin, '') AS kra_pin,
		t.vat_registered, COALESCE(b.name, '') AS branch_name, COALESCE(b.code, '') AS branch_code,
		COALESCE(b.phone, '') AS branch_phone, COALESCE(b.location, '') AS location
		FROM tenants t LEFT JOIN branches b ON b.id = ? AND b.tenant_id = t.id
		WHERE t.id = ?`, branchID, tenantID)
	return h, err
}

// Settings returns tenant settings by key as raw JSON (missing keys absent).
func (r *Repo) Settings(ctx context.Context, tenantID uint64, keys ...string) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	if len(keys) == 0 {
		return out, nil
	}
	query, args, err := sqlx.In("SELECT `key`, value FROM tenant_settings WHERE tenant_id = ? AND `key` IN (?)",
		tenantID, keys)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Key   string `db:"key"`
		Value []byte `db:"value"`
	}
	if err := r.q.SelectContext(ctx, &rows, r.rebind(query), args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Key] = json.RawMessage(row.Value)
	}
	return out, nil
}

func (r *Repo) rebind(q string) string {
	if b, ok := r.q.(interface{ Rebind(string) string }); ok {
		return b.Rebind(q)
	}
	return q
}

// SaleNames fills display names a freshly completed sale does not carry.
func (r *Repo) SaleNames(ctx context.Context, tenantID, saleID uint64) (cashier, terminal, customer string, err error) {
	var row struct {
		Cashier  string `db:"cashier"`
		Terminal string `db:"terminal"`
		Customer string `db:"customer"`
	}
	err = r.q.GetContext(ctx, &row, `SELECT COALESCE(u.name, '') AS cashier, COALESCE(pt.name, '') AS terminal,
		COALESCE(c.name, '') AS customer
		FROM sales s
		LEFT JOIN users u ON u.id = s.cashier_id
		LEFT JOIN pos_terminals pt ON pt.id = s.terminal_id
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.tenant_id = ? AND s.id = ?`, tenantID, saleID)
	return row.Cashier, row.Terminal, row.Customer, err
}

// BranchIDByPublicID resolves a branch of the tenant.
func (r *Repo) BranchIDByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	var id uint64
	err := r.q.GetContext(ctx, &id, `SELECT id FROM branches WHERE tenant_id = ? AND public_id = ?`,
		tenantID, publicID)
	return id, notFound(err, ErrBranchNotFound)
}

const issuedCols = `i.id, i.tenant_id, i.public_id, i.sale_id, i.branch_id, i.copy_no, i.template_id,
	i.template_version, i.builtin_key, i.paper_width, i.content, i.channel, i.terminal_id, i.issued_by,
	i.supervisor_id, i.created_at, COALESCE(t.name, '') AS template_name, COALESCE(u.name, '') AS issued_by_name`

const issuedFrom = ` FROM issued_receipts i
	LEFT JOIN receipt_templates t ON t.id = i.template_id
	LEFT JOIN users u ON u.id = i.issued_by`

// Original returns the copy_no 0 snapshot of a sale (ErrNotIssued when none).
func (r *Repo) Original(ctx context.Context, tenantID, saleID uint64) (Issued, error) {
	var i Issued
	err := r.q.GetContext(ctx, &i, `SELECT `+issuedCols+issuedFrom+`
		WHERE i.tenant_id = ? AND i.sale_id = ? AND i.copy_no = 0`, tenantID, saleID)
	return i, notFound(err, ErrNotIssued)
}

// Copies lists every issued copy of a sale.
func (r *Repo) Copies(ctx context.Context, tenantID, saleID uint64) ([]Issued, error) {
	out := []Issued{}
	err := r.q.SelectContext(ctx, &out, `SELECT `+issuedCols+issuedFrom+`
		WHERE i.tenant_id = ? AND i.sale_id = ? ORDER BY i.copy_no`, tenantID, saleID)
	return out, err
}

// NextCopyNo locks the sale's receipts and returns the next reprint number.
func (r *Repo) NextCopyNo(ctx context.Context, tenantID, saleID uint64) (int, error) {
	var n sql.NullInt64
	err := r.q.GetContext(ctx, &n, `SELECT MAX(copy_no) FROM issued_receipts
		WHERE tenant_id = ? AND sale_id = ? FOR UPDATE`, tenantID, saleID)
	if err != nil {
		return 0, err
	}
	return int(n.Int64) + 1, nil
}

// InsertIssued stores a receipt snapshot.
func (r *Repo) InsertIssued(ctx context.Context, i *Issued) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO issued_receipts
		(tenant_id, public_id, sale_id, branch_id, copy_no, template_id, template_version, builtin_key,
		 paper_width, content, channel, terminal_id, issued_by, supervisor_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		i.TenantID, i.PublicID, i.SaleID, i.BranchID, i.CopyNo, i.TemplateID, i.TemplateVersion, i.BuiltinKey,
		i.PaperWidth, i.Content, i.Channel, i.TerminalID, i.IssuedBy, i.SupervisorID)
	if err != nil {
		return err
	}
	i.ID, err = lastID(res)
	return err
}

// InsertAudit writes an audit row inside the caller's transaction.
func (r *Repo) InsertAudit(ctx context.Context, tenantID, branchID uint64, actorType string, actorID,
	supervisorID *uint64, action, entityID string, data map[string]any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = r.q.ExecContext(ctx, `INSERT INTO audit_logs (tenant_id, branch_id, actor_type, actor_id,
		supervisor_id, action, entity_type, entity_id, data) VALUES (?,?,?,?,?,?,?,?,?)`,
		tenantID, branchID, actorType, actorID, supervisorID, action, "sale", entityID, payload)
	return err
}

func lastID(res sql.Result) (uint64, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id < 0 {
		return 0, errors.New("receipts: negative insert id")
	}
	return uint64(id), nil
}

func notFound(err, target error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return target
	}
	return err
}
