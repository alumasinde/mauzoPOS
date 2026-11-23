package tenants

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
	"mauzo/internal/core/httpx"
)

// Repo is the tenants data store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo {
	return &Repo{db: conn}
}

const tenantCols = `id, public_id, name, slug, email, phone, status,
	business_type, kra_pin, vat_registered, address, onboarding_step,
	onboarded_at, suspended_at, suspend_reason, created_at, updated_at`

// Create inserts a tenant.
func (r *Repo) Create(ctx context.Context, t *Tenant) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (public_id, name, slug, email, phone, status) VALUES (?, ?, ?, ?, ?, ?)`,
		t.PublicID, t.Name, t.Slug, t.Email, t.Phone, t.Status)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = db.U64(id)
	fresh, err := r.ByID(ctx, t.ID)
	if err != nil {
		return err
	}
	*t = fresh
	return nil
}

// ByPublicID loads a tenant by its API id.
func (r *Repo) ByPublicID(ctx context.Context, publicID string) (Tenant, error) {
	var t Tenant
	err := r.db.GetContext(ctx, &t,
		`SELECT `+tenantCols+` FROM tenants WHERE public_id = ?`, publicID)
	return t, err
}

// BySlug loads a tenant by its sign-in code.
func (r *Repo) BySlug(ctx context.Context, slug string) (Tenant, error) {
	var t Tenant
	err := r.db.GetContext(ctx, &t,
		`SELECT `+tenantCols+` FROM tenants WHERE slug = ?`, slug)
	return t, err
}

// SlugExists reports whether the slug is taken.
func (r *Repo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM tenants WHERE slug = ?`, slug)
	return n > 0, err
}

// List returns a page of tenants plus the total matching count.
func (r *Repo) List(ctx context.Context, p httpx.Page) ([]Tenant, int, error) {
	where, args := tenantFilter(p)
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM tenants`+where, args...); err != nil {
		return nil, 0, err
	}
	order := "name ASC"
	if p.Sort == "-created_at" {
		order = "created_at DESC"
	}
	var out []Tenant
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+tenantCols+` FROM tenants`+where+
			fmt.Sprintf(` ORDER BY %s LIMIT ? OFFSET ?`, order),
		append(args, p.Limit(), p.Offset())...)
	return out, total, err
}

func tenantFilter(p httpx.Page) (string, []any) {
	var parts []string
	var args []any
	if p.Query != "" {
		parts = append(parts, `(name LIKE ? OR slug LIKE ?)`)
		like := "%" + p.Query + "%"
		args = append(args, like, like)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

// Update changes the mutable tenant fields and returns the fresh row.
func (r *Repo) Update(ctx context.Context, t Tenant) (Tenant, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET name = ?, email = ?, phone = ? WHERE id = ?`,
		t.Name, t.Email, t.Phone, t.ID)
	if err != nil {
		return Tenant{}, err
	}
	return r.ByID(ctx, t.ID)
}

// SetStatus flips status with the suspension metadata and returns the fresh row.
func (r *Repo) SetStatus(ctx context.Context, id uint64, status string, reason *string) (Tenant, error) {
	var suspendedAt *time.Time
	if status == StatusSuspended {
		now := time.Now()
		suspendedAt = &now
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET status = ?, suspended_at = ?, suspend_reason = ? WHERE id = ?`,
		status, suspendedAt, reason, id)
	if err != nil {
		return Tenant{}, err
	}
	return r.ByID(ctx, id)
}

// SetBusinessType writes the business_type code (idempotent profile apply).
func (r *Repo) SetBusinessType(ctx context.Context, tenantID uint64, code string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET business_type = ? WHERE id = ?`, code, tenantID)
	return err
}

// SetBusinessFields stores the step-1 business details.
func (r *Repo) SetBusinessFields(ctx context.Context, tenantID uint64, name string, kraPin, address *string, vat bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET name = ?, kra_pin = ?, address = ?, vat_registered = ? WHERE id = ?`,
		name, kraPin, address, vat, tenantID)
	return err
}

// SetOnboardingStep advances onboarding_step (never backwards).
func (r *Repo) SetOnboardingStep(ctx context.Context, tenantID uint64, step int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET onboarding_step = GREATEST(onboarding_step, ?) WHERE id = ?`,
		step, tenantID)
	return err
}

// CompleteOnboarding stamps onboarded_at.
func (r *Repo) CompleteOnboarding(ctx context.Context, tenantID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET onboarded_at = COALESCE(onboarded_at, NOW(3)) WHERE id = ?`, tenantID)
	return err
}

// ByID loads a tenant by internal id (used after writes; also by auth).
func (r *Repo) ByID(ctx context.Context, id uint64) (Tenant, error) {
	var t Tenant
	err := r.db.GetContext(ctx, &t,
		`SELECT `+tenantCols+` FROM tenants WHERE id = ?`, id)
	return t, err
}
