package mpesa

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// tillFeedWindow is how far back a till lists unclaimed payments: long enough
// for "I paid a few minutes ago", short enough that an old payment cannot be
// passed off as today's. Older ones are reconciled in the back office.
const tillFeedWindow = 3 * time.Hour

// BranchTill maps an M-Pesa short code (till / store number) to a branch.
type BranchTill struct {
	ShortCode      string `db:"short_code" json:"short_code"`
	BranchID       uint64 `db:"branch_id" json:"-"`
	BranchPublicID string `db:"branch_public_id" json:"branch_id"`
	BranchName     string `db:"branch_name" json:"branch_name"`
}

// BranchForShortCode returns the branch a short code belongs to (nil = the
// whole business).
func (r *Repo) BranchForShortCode(ctx context.Context, tenantID uint64, code string) (*uint64, error) {
	var id uint64
	err := r.db.GetContext(ctx, &id,
		`SELECT branch_id FROM mpesa_branch_tills WHERE tenant_id = ? AND short_code = ?`,
		tenantID, strings.TrimSpace(code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// BranchTills lists the mappings.
func (r *Repo) BranchTills(ctx context.Context, tenantID uint64) ([]BranchTill, error) {
	out := []BranchTill{}
	err := r.db.SelectContext(ctx, &out,
		`SELECT m.short_code, m.branch_id, b.public_id AS branch_public_id, b.name AS branch_name
		 FROM mpesa_branch_tills m JOIN branches b ON b.id = m.branch_id
		 WHERE m.tenant_id = ? ORDER BY b.name, m.short_code`, tenantID)
	return out, err
}

// ReplaceBranchTills swaps the whole mapping in one transaction.
func (r *Repo) ReplaceBranchTills(ctx context.Context, tenantID uint64, rows []BranchTill) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM mpesa_branch_tills WHERE tenant_id = ?`, tenantID); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO mpesa_branch_tills (tenant_id, short_code, branch_id) VALUES (?, ?, ?)`,
			tenantID, row.ShortCode, row.BranchID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// BranchIDByPublicID resolves a branch of the tenant (0 when unknown).
func (r *Repo) BranchIDByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	var id uint64
	err := r.db.GetContext(ctx, &id, `SELECT id FROM branches WHERE tenant_id = ? AND public_id = ?`,
		tenantID, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// UnallocatedForBranch lists recent unclaimed collections a till may use:
// paid to this branch's short code, or to an unmapped (business-wide) one.
func (r *Repo) UnallocatedForBranch(ctx context.Context, tenantID, branchID uint64, since time.Time) ([]Transaction, error) {
	var rows []Transaction
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM mpesa_transactions
		 WHERE tenant_id = ? AND status = 'success' AND allocated_sale_id IS NULL AND created_at >= ?
		   AND (branch_id IS NULL OR branch_id = ?)
		 ORDER BY created_at DESC LIMIT 50`, tenantID, since, branchID)
	return rows, err
}

// BranchTillInput is one mapping in PUT /t/mpesa/branch-tills.
type BranchTillInput struct {
	ShortCode string `json:"short_code"`
	BranchID  string `json:"branch_id"`
}

// SetBranchTills validates and stores the short code → branch mapping.
func (s *Service) SetBranchTills(ctx context.Context, tenantID uint64, in []BranchTillInput) ([]BranchTill, error) {
	seen := map[string]bool{}
	rows := make([]BranchTill, 0, len(in))
	for _, x := range in {
		code := strings.TrimSpace(x.ShortCode)
		if code == "" || len(code) > 20 || !digits(code) {
			return nil, ErrInvalidShortCode
		}
		if seen[code] {
			return nil, ErrInvalidShortCode
		}
		seen[code] = true
		id, err := s.repo.BranchIDByPublicID(ctx, tenantID, x.BranchID)
		if err != nil {
			return nil, err
		}
		if id == 0 {
			return nil, ErrBranchMismatch
		}
		rows = append(rows, BranchTill{ShortCode: code, BranchID: id})
	}
	if err := s.repo.ReplaceBranchTills(ctx, tenantID, rows); err != nil {
		return nil, err
	}
	return s.repo.BranchTills(ctx, tenantID)
}

// BranchTills lists the mapping.
func (s *Service) BranchTills(ctx context.Context, tenantID uint64) ([]BranchTill, error) {
	return s.repo.BranchTills(ctx, tenantID)
}

// FeedForTill returns this branch's recent unclaimed payments, phone numbers
// masked (tills only need enough to confirm with the customer).
func (s *Service) FeedForTill(ctx context.Context, tenantID, branchID uint64) ([]Transaction, error) {
	rows, err := s.repo.UnallocatedForBranch(ctx, tenantID, branchID, time.Now().Add(-tillFeedWindow))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i] = tillView(rows[i])
	}
	return rows, nil
}

// tillView strips what a till does not need and masks the phone number.
func tillView(t Transaction) Transaction {
	t.Raw = nil
	t.Phone = MaskPhone(t.Phone)
	return t
}

// MaskPhone keeps the first 4 and last 3 digits: 0712•••678.
func MaskPhone(p string) string {
	n := NormalizePhone(p)
	if n == "" {
		if len(p) > 6 {
			return p[:3] + "•••" + p[len(p)-3:]
		}
		return p
	}
	local := "0" + strings.TrimPrefix(n, "254")
	if len(local) < 8 {
		return local
	}
	return local[:4] + "•••" + local[len(local)-3:]
}

func digits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}
