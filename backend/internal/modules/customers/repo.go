package customers

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/jmoiron/sqlx"
)

// Repo stores customers.
type Repo struct{ conn *sqlx.DB }

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo { return &Repo{conn: conn} }

const cols = `c.id, c.tenant_id, c.public_id, c.code, c.name, c.phone, c.email,
	c.kra_pin, c.credit_limit, c.notes, c.status, c.created_at, c.updated_at,
	(SELECT COALESCE(SUM(p.amount),0) FROM sale_payments p
	 JOIN sales s ON s.id = p.sale_id
	 WHERE p.tenant_id = c.tenant_id AND p.method = 'credit' AND p.status = 'confirmed'
	   AND s.customer_id = c.id AND s.status <> 'voided') AS credit_balance`

// List returns customers, optionally filtered by a name/phone/code search.
func (r *Repo) List(ctx context.Context, tenantID uint64, search string, activeOnly bool) ([]Customer, error) {
	q := `SELECT ` + cols + ` FROM customers c WHERE c.tenant_id = ?`
	args := []any{tenantID}
	if activeOnly {
		q += ` AND c.status = 'active'`
	}
	if search != "" {
		q += ` AND (c.name LIKE ? OR c.phone LIKE ? OR c.code LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY c.name`
	out := []Customer{}
	return out, sqlx.SelectContext(ctx, r.conn, &out, q, args...)
}

// ByPublicID loads one customer.
func (r *Repo) ByPublicID(ctx context.Context, tenantID uint64, publicID string) (Customer, error) {
	var c Customer
	err := r.conn.GetContext(ctx, &c,
		`SELECT `+cols+` FROM customers c WHERE c.tenant_id = ? AND c.public_id = ?`, tenantID, publicID)
	return c, err
}

// Create inserts a customer.
func (r *Repo) Create(ctx context.Context, c *Customer) error {
	res, err := r.conn.ExecContext(ctx,
		`INSERT INTO customers (tenant_id, public_id, code, name, phone, email, kra_pin,
		 credit_limit, notes, status) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		c.TenantID, c.PublicID, c.Code, c.Name, c.Phone, c.Email, c.KRAPin,
		c.CreditLimit, c.Notes, c.Status)
	if err != nil {
		return err
	}
	c.ID, err = lastID(res)
	return err
}

// Update edits the mutable fields.
func (r *Repo) Update(ctx context.Context, c *Customer) error {
	_, err := r.conn.ExecContext(ctx,
		`UPDATE customers SET code=?, name=?, phone=?, email=?, kra_pin=?, credit_limit=?,
		 notes=?, status=? WHERE tenant_id=? AND id=?`,
		c.Code, c.Name, c.Phone, c.Email, c.KRAPin, c.CreditLimit,
		c.Notes, c.Status, c.TenantID, c.ID)
	return err
}

// OutstandingCredit is a customer's current confirmed-credit balance.
func (r *Repo) OutstandingCredit(ctx context.Context, tenantID, customerID uint64) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.conn.GetContext(ctx, &v,
		`SELECT COALESCE(SUM(p.amount),0) FROM sale_payments p
		 JOIN sales s ON s.id = p.sale_id
		 WHERE p.tenant_id = ? AND p.method = 'credit' AND p.status = 'confirmed'
		   AND s.customer_id = ? AND s.status <> 'voided'`, tenantID, customerID)
	return v, err
}

func lastID(res interface{ LastInsertId() (int64, error) }) (uint64, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id < 0 {
		return 0, nil
	}
	return uint64(id), nil //nolint:gosec // LastInsertId is non-negative on success
}
