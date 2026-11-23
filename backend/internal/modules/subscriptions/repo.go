package subscriptions

import (
	"context"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Repo is the plans + tenant_subscriptions store.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{db: conn} }

const planCols = `id, public_id, name, description, price_kes, billing_interval,
	modules, max_branches, max_users, is_active, sort, created_at, updated_at`

// ListPlans returns plans; activeOnly filters to purchasable ones.
func (r *Repo) ListPlans(ctx context.Context, activeOnly bool) ([]Plan, error) {
	var out []Plan
	q := `SELECT ` + planCols + ` FROM plans`
	if activeOnly {
		q += ` WHERE is_active = 1`
	}
	err := r.db.SelectContext(ctx, &out, q+` ORDER BY sort, price_kes`)
	return out, err
}

// PlanByPublicID loads one plan.
func (r *Repo) PlanByPublicID(ctx context.Context, publicID string) (Plan, error) {
	var p Plan
	err := r.db.GetContext(ctx, &p, `SELECT `+planCols+` FROM plans WHERE public_id = ?`, publicID)
	return p, err
}

// CreatePlan inserts and hydrates a plan.
func (r *Repo) CreatePlan(ctx context.Context, p *Plan) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO plans (public_id, name, description, price_kes, billing_interval,
		 modules, max_branches, max_users, is_active, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.PublicID, p.Name, p.Description, p.PriceKES, p.BillingInterval,
		p.ModulesJSON, p.MaxBranches, p.MaxUsers, p.IsActive, p.Sort)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = db.U64(id)
	fresh, err := r.PlanByPublicID(ctx, p.PublicID)
	if err != nil {
		return err
	}
	*p = fresh
	return nil
}

// UpdatePlan updates a plan and hydrates the row.
func (r *Repo) UpdatePlan(ctx context.Context, p *Plan) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plans SET name = ?, description = ?, price_kes = ?, billing_interval = ?,
		 modules = ?, max_branches = ?, max_users = ?, is_active = ?, sort = ? WHERE id = ?`,
		p.Name, p.Description, p.PriceKES, p.BillingInterval,
		p.ModulesJSON, p.MaxBranches, p.MaxUsers, p.IsActive, p.Sort, p.ID)
	if err != nil {
		return err
	}
	fresh, err := r.PlanByPublicID(ctx, p.PublicID)
	if err != nil {
		return err
	}
	*p = fresh
	return nil
}

// ─── tenant_subscriptions ───────────────────────────────────────────────────

const subCols = `s.id, s.tenant_id, s.plan_id, s.status, s.starts_at, s.trial_ends_at,
	s.ends_at, s.created_at, p.public_id AS plan_public_id, p.name AS plan_name`

// Current returns the tenant's newest non-cancelled subscription, or sql.ErrNoRows.
func (r *Repo) Current(ctx context.Context, tenantID uint64) (Subscription, error) {
	var s Subscription
	err := r.db.GetContext(ctx, &s,
		`SELECT `+subCols+` FROM tenant_subscriptions s JOIN plans p ON p.id = s.plan_id
		 WHERE s.tenant_id = ? AND s.status <> 'cancelled' ORDER BY s.id DESC LIMIT 1`, tenantID)
	return s, err
}

// CancelOpen marks every non-cancelled subscription for the tenant cancelled.
func (r *Repo) CancelOpen(ctx context.Context, tenantID uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenant_subscriptions SET status = 'cancelled', ends_at = NOW(3)
		 WHERE tenant_id = ? AND status <> 'cancelled'`, tenantID)
	return err
}

// InsertSubscription adds a subscription row.
func (r *Repo) InsertSubscription(ctx context.Context, s *Subscription) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, trial_ends_at)
		 VALUES (?, ?, ?, ?)`, s.TenantID, s.PlanID, s.Status, s.TrialEndsAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = db.U64(id)
	return nil
}

// CurrentPlan returns the plan of the tenant's open subscription, or sql.ErrNoRows.
func (r *Repo) CurrentPlan(ctx context.Context, tenantID uint64) (Plan, error) {
	var p Plan
	err := r.db.GetContext(ctx, &p,
		`SELECT p.id, p.public_id, p.name, p.description, p.price_kes, p.billing_interval, p.modules,
		        p.max_branches, p.max_users, p.is_active, p.sort, p.created_at, p.updated_at
		 FROM tenant_subscriptions s JOIN plans p ON p.id = s.plan_id
		 WHERE s.tenant_id = ? AND s.status <> 'cancelled' ORDER BY s.id DESC LIMIT 1`, tenantID)
	return p, err
}
