package mpesa

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var errDup = errors.New("duplicate key")

// Repo is the mpesa persistence layer.
type Repo struct{ db *sqlx.DB }

// NewRepo builds the repository.
func NewRepo(db *sqlx.DB) *Repo { return &Repo{db: db} }

// --- credentials -------------------------------------------------------------

// Credentials loads the tenant's Daraja config row (nil when unconfigured).
func (r *Repo) Credentials(ctx context.Context, tenantID uint64) (*Credentials, error) {
	var c Credentials
	err := r.db.GetContext(ctx, &c, `SELECT * FROM mpesa_credentials WHERE tenant_id = ?`, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

// UpsertCredentials writes the sealed credential row, keyed on tenant_id.
func (r *Repo) UpsertCredentials(ctx context.Context, c *Credentials) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mpesa_credentials
		   (tenant_id, env, short_code, short_code_kind, consumer_key_enc, consumer_secret_enc,
		    passkey_enc, initiator_name, security_credential_enc, callback_token)
		 VALUES (?,?,?,?,?,?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE
		   env=VALUES(env), short_code=VALUES(short_code), short_code_kind=VALUES(short_code_kind),
		   consumer_key_enc=VALUES(consumer_key_enc), consumer_secret_enc=VALUES(consumer_secret_enc),
		   passkey_enc=VALUES(passkey_enc), initiator_name=VALUES(initiator_name),
		   security_credential_enc=VALUES(security_credential_enc)`,
		c.TenantID, c.Env, c.ShortCode, c.ShortCodeKind, c.ConsumerKeyEnc, c.ConsumerSecretEnc,
		c.PasskeyEnc, c.InitiatorName, c.SecurityCredentialEnc, c.CallbackToken)
	return err
}

// SetC2BRegistered stamps when the C2B URLs were last registered on Daraja.
func (r *Repo) SetC2BRegistered(ctx context.Context, tenantID uint64, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_credentials SET c2b_registered_at = ? WHERE tenant_id = ?`, at, tenantID)
	return err
}

// TenantByToken resolves a callback path secret to its tenant.
func (r *Repo) TenantByToken(ctx context.Context, token string) (uint64, error) {
	var id uint64
	err := r.db.GetContext(ctx, &id,
		`SELECT tenant_id FROM mpesa_credentials WHERE callback_token = ?`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// --- transactions ------------------------------------------------------------

// InsertTransaction records a new feed row. trans_id and checkout ids stay
// NULL until known so the dedupe unique keys never collide on empty values.
func (r *Repo) InsertTransaction(ctx context.Context, t *Transaction) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO mpesa_transactions
		   (tenant_id, public_id, branch_id, kind, status, trans_id, checkout_request_id,
		    merchant_request_id, phone, payer_name, amount, account_ref, short_code,
		    result_code, result_desc, allocated_sale_id, allocated_by, allocated_at,
		    trans_time, initiated_by, raw)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.TenantID, t.PublicID, t.BranchID, t.Kind, t.Status, t.TransID, t.CheckoutRequestID,
		t.MerchantRequestID, t.Phone, t.PayerName, t.Amount, t.AccountRef, t.ShortCode,
		t.ResultCode, t.ResultDesc, t.AllocatedSaleID, t.AllocatedBy, t.AllocatedAt,
		t.TransTime, t.InitiatedBy, t.Raw)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return errDup
		}
		return err
	}
	id, _ := res.LastInsertId()
	t.ID = uint64(id) //nolint:gosec // AUTO_INCREMENT ids are positive
	return nil
}

// UpdateOutcome writes a terminal/interim result onto a transaction.
func (r *Repo) UpdateOutcome(ctx context.Context, id uint64, status string, transID *string,
	code *int, desc string, payer, phone string, transTime *time.Time, raw []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_transactions SET status=?, trans_id=COALESCE(trans_id, ?),
		   result_code=?, result_desc=?, payer_name=CASE WHEN ?='' THEN payer_name ELSE ? END,
		   phone=CASE WHEN ?='' THEN phone ELSE ? END,
		   trans_time=COALESCE(trans_time, ?), raw=COALESCE(?, raw)
		 WHERE id=?`,
		status, transID, code, desc, payer, payer, phone, phone, transTime, raw, id)
	return err
}

// UpdateStatus moves a transaction to a new lifecycle state.
func (r *Repo) UpdateStatus(ctx context.Context, id uint64, status, desc string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_transactions SET status=?, result_desc=CASE WHEN ?='' THEN result_desc ELSE ? END
		 WHERE id=?`, status, desc, desc, id)
	return err
}

// TouchQueried stamps the last STK status-query time (fallback pacing).
func (r *Repo) TouchQueried(ctx context.Context, id uint64, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_transactions SET last_queried_at=? WHERE id=?`, at, id)
	return err
}

// ByID loads a transaction by internal id (nil when absent).
func (r *Repo) ByID(ctx context.Context, tenantID, id uint64) (*Transaction, error) {
	var t Transaction
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM mpesa_transactions WHERE tenant_id=? AND id=?`, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

// ByPublic loads a transaction by public id (nil when absent).
func (r *Repo) ByPublic(ctx context.Context, tenantID uint64, publicID string) (*Transaction, error) {
	var t Transaction
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM mpesa_transactions WHERE tenant_id=? AND public_id=?`, tenantID, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

// ByTransID loads a transaction by the Daraja TransID (nil when absent).
func (r *Repo) ByTransID(ctx context.Context, tenantID uint64, transID string) (*Transaction, error) {
	var t Transaction
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM mpesa_transactions WHERE tenant_id=? AND trans_id=?`, tenantID, transID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

// ByCheckout loads an STK transaction by CheckoutRequestID (nil when absent).
func (r *Repo) ByCheckout(ctx context.Context, tenantID uint64, checkout string) (*Transaction, error) {
	var t Transaction
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM mpesa_transactions WHERE tenant_id=? AND checkout_request_id=?`, tenantID, checkout)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

// List returns feed rows newest-first plus the filtered total.
func (r *Repo) List(ctx context.Context, tenantID uint64, f ListFilter, page, per int) ([]Transaction, int, error) {
	where, args := []string{"tenant_id = ?"}, []any{tenantID}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, f.Kind)
	}
	if f.Allocated == "yes" {
		where = append(where, "allocated_sale_id IS NOT NULL")
	}
	if f.Allocated == "no" {
		where = append(where, "allocated_sale_id IS NULL AND status IN ('success','reversal_pending')")
	}
	if f.BranchID != 0 {
		where = append(where, "(branch_id = ? OR branch_id IS NULL)")
		args = append(args, f.BranchID)
	}
	if f.From != "" {
		where = append(where, "created_at >= ?")
		args = append(args, f.From+" 00:00:00")
	}
	if f.To != "" {
		where = append(where, "created_at <= ?")
		args = append(args, f.To+" 23:59:59.999")
	}
	if f.Query != "" {
		where = append(where, "(phone LIKE ? OR trans_id LIKE ? OR account_ref LIKE ? OR payer_name LIKE ?)")
		q := "%" + f.Query + "%"
		args = append(args, q, q, q, q)
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM mpesa_transactions WHERE `+w, args...); err != nil {
		return nil, 0, err
	}
	var rows []Transaction
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM mpesa_transactions WHERE `+w+
			` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, per, (page-1)*per)...)
	return rows, total, err
}

// Unallocated returns successful transactions awaiting allocation — the till
// feed and the reconciliation queue read this.
func (r *Repo) Unallocated(ctx context.Context, tenantID uint64, since time.Time) ([]Transaction, error) {
	var rows []Transaction
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM mpesa_transactions
		 WHERE tenant_id=? AND status='success' AND allocated_sale_id IS NULL AND created_at >= ?
		 ORDER BY created_at DESC`, tenantID, since)
	return rows, err
}

// Summary aggregates the dashboard/page stat numbers.
func (r *Repo) Summary(ctx context.Context, tenantID uint64) (Summary, error) {
	var s Summary
	err := r.db.GetContext(ctx, &s, `SELECT
		COALESCE(SUM(CASE WHEN status='success' AND DATE(created_at)=CURDATE() THEN amount END),0) AS collected_today,
		COALESCE(SUM(CASE WHEN status='success' AND DATE(created_at)=CURDATE() THEN 1 ELSE 0 END),0) AS count_today,
		COALESCE(SUM(CASE WHEN status='success' AND allocated_sale_id IS NULL THEN 1 ELSE 0 END),0) AS unallocated,
		COALESCE(SUM(CASE WHEN status IN ('initiated','pending') THEN 1 ELSE 0 END),0) AS pending,
		COALESCE(SUM(CASE WHEN status IN ('failed','cancelled','timeout') AND created_at >= CURDATE()-INTERVAL 7 DAY THEN 1 ELSE 0 END),0) AS failed_7d,
		COALESCE(SUM(CASE WHEN status='reversed' THEN 1 ELSE 0 END),0) AS reversed
	  FROM mpesa_transactions WHERE tenant_id=?`, tenantID)
	return s, err
}

// --- allocation (tx-bound so sale completion stays atomic) -------------------

// ConsumeTx marks a confirmed transaction as funding a sale, inside the
// sale's own transaction. Returns the row for the caller to attach. Fails unless the
// txn is successful, unallocated, un-reversed, and the amount matches.
func ConsumeTx(ctx context.Context, tx *sqlx.Tx, tenantID uint64, transID string,
	amount decimal.Decimal, saleID uint64) (*Transaction, error) {
	var t Transaction
	err := tx.GetContext(ctx, &t,
		`SELECT * FROM mpesa_transactions WHERE tenant_id=? AND trans_id=? FOR UPDATE`, tenantID, transID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoTransaction
	}
	if err != nil {
		return nil, err
	}
	if t.Status != StSuccess {
		return nil, ErrNotConfirmed
	}
	if t.AllocatedSaleID != nil {
		return nil, ErrAlreadyAllocated
	}
	if !t.Amount.Equal(amount) {
		return nil, ErrAmountMismatch
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE mpesa_transactions SET allocated_sale_id=?, allocated_at=UTC_TIMESTAMP(3)
		 WHERE id=? AND allocated_sale_id IS NULL`, saleID, t.ID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrAlreadyAllocated
	}
	t.AllocatedSaleID = &saleID
	return &t, nil
}

// ReleaseTx frees transactions allocated to a sale (void/full refund path),
// inside the caller's transaction.
func ReleaseTx(ctx context.Context, tx *sqlx.Tx, saleID uint64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE mpesa_transactions SET allocated_sale_id=NULL, allocated_at=NULL, allocated_by=NULL
		 WHERE allocated_sale_id=?`, saleID)
	return err
}

// AllocatedSaleIDs returns the subset of saleIDs already claimed by a txn.
func (r *Repo) AllocatedSaleIDs(ctx context.Context, tenantID uint64, saleIDs []uint64) (map[uint64]bool, error) {
	out := map[uint64]bool{}
	if len(saleIDs) == 0 {
		return out, nil
	}
	q, args, err := sqlx.In(
		`SELECT allocated_sale_id FROM mpesa_transactions
		 WHERE tenant_id=? AND allocated_sale_id IN (?)`, tenantID, saleIDs)
	if err != nil {
		return nil, err
	}
	var ids []uint64
	if err := r.db.SelectContext(ctx, &ids, r.db.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// Allocate links a feed transaction to a sale administratively.
func (r *Repo) Allocate(ctx context.Context, txnID, saleID, by uint64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_transactions SET allocated_sale_id=?, allocated_by=?, allocated_at=UTC_TIMESTAMP(3)
		 WHERE id=? AND status='success' AND allocated_sale_id IS NULL`,
		saleID, by, txnID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAlreadyAllocated
	}
	return nil
}

// Reallocate moves a transaction to a different sale (sensitive op).
func (r *Repo) Reallocate(ctx context.Context, txnID, saleID, by uint64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_transactions SET allocated_sale_id=?, allocated_by=?, allocated_at=UTC_TIMESTAMP(3)
		 WHERE id=? AND status IN ('success','reversal_pending')`,
		saleID, by, txnID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoTransaction
	}
	return nil
}

// --- events ------------------------------------------------------------------

// LogEvent records an audit row; the tx variant shares a caller's transaction.
func LogEvent(ctx context.Context, db sqlx.ExtContext, tenantID uint64, txnID *uint64,
	kind, ref, detail string, payload []byte) {
	_, _ = db.ExecContext(ctx,
		`INSERT INTO mpesa_events (tenant_id, txn_id, kind, ref, detail, payload) VALUES (?,?,?,?,?,?)`,
		tenantID, txnID, kind, ref, detail, payload)
}

// Events returns the raw-callback audit trail for a transaction.
func (r *Repo) Events(ctx context.Context, tenantID, txnID uint64) ([]Event, error) {
	var rows []Event
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM mpesa_events WHERE tenant_id=? AND txn_id=? ORDER BY id`, tenantID, txnID)
	return rows, err
}

// --- reversals ---------------------------------------------------------------

// InsertReversal records a pending reversal before the Daraja call resolves.
func (r *Repo) InsertReversal(ctx context.Context, v *Reversal) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO mpesa_reversals
		   (tenant_id, txn_id, originator_conversation_id, conversation_id, amount, reason,
		    status, requested_by)
		 VALUES (?,?,?,?,?,?, 'pending', ?)`,
		v.TenantID, v.TxnID, v.OriginatorConversationID, v.ConversationID, v.Amount, v.Reason,
		v.RequestedBy)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return errDup
		}
		return err
	}
	id, _ := res.LastInsertId()
	v.ID = uint64(id) //nolint:gosec // AUTO_INCREMENT ids are positive
	return nil
}

// ReversalByConversation resolves a reversal by its OriginatorConversationID.
func (r *Repo) ReversalByConversation(ctx context.Context, convID string) (*Reversal, error) {
	var v Reversal
	err := r.db.GetContext(ctx, &v,
		`SELECT * FROM mpesa_reversals WHERE originator_conversation_id=?`, convID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &v, err
}

// UpdateReversal writes the async reversal outcome onto the pending row.
func (r *Repo) UpdateReversal(ctx context.Context, id uint64, status string, code *int, desc string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_reversals SET status=?, result_code=?, result_desc=? WHERE id=?`,
		status, code, desc, id)
	return err
}

// --- STK rate guard -----------------------------------------------------------

// CheckAndBumpSTKCount enforces a per-day ceiling on pushes per tenant.
func (r *Repo) CheckAndBumpSTKCount(ctx context.Context, tenantID uint64, limit int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE mpesa_credentials SET
		   stk_pushes_today = IF(stk_pushes_on = CURDATE(), stk_pushes_today + 1, 1),
		   stk_pushes_on = CURDATE()
		 WHERE tenant_id = ? AND (stk_pushes_on IS NULL OR stk_pushes_on < CURDATE() OR stk_pushes_today < ?)`,
		tenantID, limit)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRateLimited
	}
	return nil
}
