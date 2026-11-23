package users

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

// UserBranchIDs returns the branches an employee may access (when
// all_branches is 0). Implements roles.BranchSource.
func (r *UserRepo) UserBranchIDs(ctx context.Context, tenantID, userID uint64) ([]uint64, error) {
	out := []uint64{}
	err := r.db.SelectContext(ctx, &out,
		`SELECT branch_id FROM user_branches WHERE tenant_id = ? AND user_id = ? ORDER BY branch_id`,
		tenantID, userID)
	return out, err
}

// SetUserBranches replaces an employee's branch list. Call inside InTx.
func (r *UserRepo) SetUserBranches(ctx context.Context, tenantID, userID uint64, branchIDs []uint64) error {
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM user_branches WHERE tenant_id = ? AND user_id = ?`, tenantID, userID); err != nil {
		return err
	}
	for _, b := range branchIDs {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO user_branches (tenant_id, user_id, branch_id) VALUES (?, ?, ?)`,
			tenantID, userID, b); err != nil {
			return err
		}
	}
	return nil
}

// BranchIDsForUsers returns branch ids per user for a page of employees.
func (r *UserRepo) BranchIDsForUsers(ctx context.Context, tenantID uint64, userIDs []uint64) (map[uint64][]uint64, error) {
	out := map[uint64][]uint64{}
	if len(userIDs) == 0 {
		return out, nil
	}
	q, args, err := sqlx.In(
		`SELECT user_id, branch_id FROM user_branches WHERE tenant_id = ? AND user_id IN (?) ORDER BY branch_id`,
		tenantID, userIDs)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		UserID   uint64 `db:"user_id"`
		BranchID uint64 `db:"branch_id"`
	}
	if err := r.db.SelectContext(ctx, &rows, strings.TrimSpace(q), args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserID] = append(out[row.UserID], row.BranchID)
	}
	return out, nil
}
