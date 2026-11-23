package localdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"mauzo/internal/core/security"
)

// Cashier is a synced sign-in candidate.
type Cashier struct {
	ID          string   `db:"id" json:"id"`
	Name        string   `db:"name" json:"name"`
	RoleName    string   `db:"role_name" json:"role_name"`
	PINHash     string   `db:"pin_hash" json:"-"`
	Permissions []string `db:"-" json:"permissions"`
}

// Cashiers lists synced sign-in candidates.
func (d *DB) Cashiers(ctx context.Context) ([]Cashier, error) {
	var rows []struct {
		ID       string `db:"id"`
		Name     string `db:"name"`
		RoleName string `db:"role_name"`
		Perms    string `db:"permissions"`
	}
	if err := d.SelectContext(ctx, &rows,
		`SELECT id, name, role_name, permissions FROM cashiers ORDER BY name`); err != nil {
		return nil, err
	}
	out := make([]Cashier, 0, len(rows))
	for _, r := range rows {
		c := Cashier{ID: r.ID, Name: r.Name, RoleName: r.RoleName}
		_ = json.Unmarshal([]byte(r.Perms), &c.Permissions)
		out = append(out, c)
	}
	return out, nil
}

// VerifyPIN checks a PIN against the synced argon2 hash — offline sign-in.
func (d *DB) VerifyPIN(ctx context.Context, cashierID, pin string) (Cashier, error) {
	var row struct {
		ID    string `db:"id"`
		Name  string `db:"name"`
		Role  string `db:"role_name"`
		Hash  string `db:"pin_hash"`
		Perms string `db:"permissions"`
	}
	err := d.GetContext(ctx, &row,
		`SELECT id, name, role_name, pin_hash, permissions FROM cashiers WHERE id = ?`, cashierID)
	if errors.Is(err, sql.ErrNoRows) {
		return Cashier{}, ErrNotFound
	}
	if err != nil {
		return Cashier{}, err
	}
	if err := security.VerifyPassword(pin, row.Hash); err != nil {
		return Cashier{}, errors.New("bad_pin")
	}
	c := Cashier{ID: row.ID, Name: row.Name, RoleName: row.Role}
	_ = json.Unmarshal([]byte(row.Perms), &c.Permissions)
	return c, nil
}

// AuthorizePIN finds any cashier holding perm whose PIN matches — the offline
// supervisor-override check.
func (d *DB) AuthorizePIN(ctx context.Context, perm, pin string) (string, error) {
	cashiers, err := d.Cashiers(ctx)
	if err != nil {
		return "", err
	}
	for _, c := range cashiers {
		ok := false
		for _, p := range c.Permissions {
			if p == perm {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		var hash string
		if err := d.GetContext(ctx, &hash,
			`SELECT pin_hash FROM cashiers WHERE id = ?`, c.ID); err != nil {
			continue
		}
		if security.VerifyPassword(pin, hash) == nil {
			return c.Name, nil
		}
	}
	return "", errors.New("bad_pin")
}

// HeldCart is a locally parked cart.
type HeldCart struct {
	ID        string `db:"id" json:"id"`
	Label     string `db:"label" json:"label"`
	Cart      string `db:"cart" json:"cart"`
	HeldBy    string `db:"held_by" json:"held_by"`
	CreatedAt string `db:"created_at" json:"created_at"`
	Synced    bool   `db:"synced" json:"synced"`
}

// HoldCart parks a cart locally.
func (d *DB) HoldCart(ctx context.Context, id, label, cart, heldBy string, synced bool) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO held_carts (id, label, cart, held_by, created_at, synced) VALUES (?,?,?,?,?,?)`,
		id, label, cart, heldBy, time.Now().UTC().Format(time.RFC3339), synced)
	return err
}

// HeldCarts lists parked carts (local + synced-server copies).
func (d *DB) HeldCarts(ctx context.Context) ([]HeldCart, error) {
	var out []HeldCart
	err := d.SelectContext(ctx, &out,
		`SELECT id, label, cart, held_by, created_at, synced FROM held_carts ORDER BY created_at DESC`)
	return out, err
}

// DeleteHeldCart removes a parked cart.
func (d *DB) DeleteHeldCart(ctx context.Context, id string) error {
	_, err := d.ExecContext(ctx, `DELETE FROM held_carts WHERE id = ?`, id)
	return err
}

// RecordPendingSale stores a completed sale for Phase 7 upload.
func (d *DB) RecordPendingSale(ctx context.Context, id, cart, totals, payment, cashierID string) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO pending_sales (id, cart, totals, payment, cashier_id, created_at) VALUES (?,?,?,?,?,?)`,
		id, cart, totals, payment, cashierID, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Customer is a synced customer account for the credit-payment picker.
type Customer struct {
	ID            uint64  `db:"id" json:"id"`
	PublicID      string  `db:"public_id" json:"public_id"`
	Name          string  `db:"name" json:"name"`
	Phone         string  `db:"phone" json:"phone"`
	Code          string  `db:"code" json:"code"`
	CreditLimit   *string `db:"credit_limit" json:"credit_limit"`
	CreditBalance string  `db:"credit_balance" json:"credit_balance"`
}

// Customers lists active customer accounts.
func (d *DB) Customers(ctx context.Context, search string, limit int) ([]Customer, error) {
	q := `SELECT * FROM customers`
	args := []any{}
	if search != "" {
		q += ` WHERE name LIKE ? OR phone LIKE ? OR code LIKE ?`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY name LIMIT ?`
	args = append(args, limit)
	out := []Customer{}
	return out, d.SelectContext(ctx, &out, q, args...)
}

// PendingSale is a queued offline sale awaiting upload.
type PendingSale struct {
	ID        string `db:"id" json:"id"`
	Payload   string `db:"cart" json:"cart"`
	Totals    string `db:"totals" json:"totals"`
	Payment   string `db:"payment" json:"payment"`
	CashierID string `db:"cashier_id" json:"cashier_id"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

// PendingSales lists unuploaded sales oldest-first.
func (d *DB) PendingSales(ctx context.Context) ([]PendingSale, error) {
	out := []PendingSale{}
	return out, d.SelectContext(ctx, &out,
		`SELECT id, cart, totals, payment, cashier_id, created_at FROM pending_sales
		 WHERE uploaded = 0 ORDER BY created_at`)
}

// MarkSaleUploaded clears a pending sale after a successful POST.
func (d *DB) MarkSaleUploaded(ctx context.Context, id string) error {
	_, err := d.ExecContext(ctx, `UPDATE pending_sales SET uploaded = 1 WHERE id = ?`, id)
	return err
}

// PendingCount is the status-bar badge.
func (d *DB) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := d.GetContext(ctx, &n, `SELECT COUNT(*) FROM pending_sales WHERE uploaded = 0`)
	return n, err
}
