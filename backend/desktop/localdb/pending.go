package localdb

import "context"

// pending_sales.uploaded states.
const (
	PendingQueued   = 0 // waiting to upload
	PendingUploaded = 1
	PendingFailed   = 2 // the server refused it: needs a person, never retried blindly
)

// FailedSale is an offline sale the server refused on upload.
type FailedSale struct {
	ID        string `db:"id" json:"id"`
	Payload   string `db:"cart" json:"payload"`
	CashierID string `db:"cashier_id" json:"cashier_id"`
	CreatedAt string `db:"created_at" json:"created_at"`
	Error     string `db:"last_error" json:"error"`
}

// MarkSaleFailed parks a sale the server refused, with its reason.
func (d *DB) MarkSaleFailed(ctx context.Context, id, reason string) error {
	_, err := d.ExecContext(ctx, `UPDATE pending_sales SET uploaded = ?, last_error = ? WHERE id = ?`,
		PendingFailed, reason, id)
	return err
}

// FailedSales lists parked sales, oldest first.
func (d *DB) FailedSales(ctx context.Context) ([]FailedSale, error) {
	out := []FailedSale{}
	return out, d.SelectContext(ctx, &out,
		`SELECT id, cart, cashier_id, created_at, last_error FROM pending_sales
		 WHERE uploaded = ? ORDER BY created_at`, PendingFailed)
}

// RequeueSale puts a parked sale back in the upload queue (after a manager
// fixed the cause, e.g. a product re-activated).
func (d *DB) RequeueSale(ctx context.Context, id string) error {
	_, err := d.ExecContext(ctx, `UPDATE pending_sales SET uploaded = ?, last_error = '' WHERE id = ? AND uploaded = ?`,
		PendingQueued, id, PendingFailed)
	return err
}

// FailedCount is the status-bar warning count.
func (d *DB) FailedCount(ctx context.Context) (int, error) {
	var n int
	err := d.GetContext(ctx, &n, `SELECT COUNT(*) FROM pending_sales WHERE uploaded = ?`, PendingFailed)
	return n, err
}
