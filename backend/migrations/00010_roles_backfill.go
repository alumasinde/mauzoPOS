package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/security"
)

// 00010 creates the system roles for every existing tenant, moves users from
// the interim users.role label onto users.role_id, then drops the label.
// Role permissions are filled at boot by the template sync (roles module),
// because the permission registry is code-defined.
func init() {
	goose.AddNamedMigrationNoTxContext("00010_roles_backfill.go", upRolesBackfill, downRolesBackfill)
}

func upRolesBackfill(ctx context.Context, db *sql.DB) error {
	if err := EnsureSystemRolesAll(ctx, db); err != nil {
		return err
	}
	stmts := []string{
		// Interim labels map 1:1 onto template codes.
		`UPDATE users u JOIN roles r ON r.tenant_id = u.tenant_id AND r.code = u.role
		 SET u.role_id = r.id WHERE u.role_id IS NULL`,
		// Anything unexpected falls back to the least-privileged role.
		`UPDATE users u JOIN roles r ON r.tenant_id = u.tenant_id AND r.code = 'cashier'
		 SET u.role_id = r.id WHERE u.role_id IS NULL`,
		`ALTER TABLE users MODIFY role_id BIGINT UNSIGNED NOT NULL`,
		`ALTER TABLE users ADD CONSTRAINT fk_users_role FOREIGN KEY (tenant_id, role_id) REFERENCES roles (tenant_id, id)`,
		`ALTER TABLE users DROP COLUMN role`,
	}
	return execAll(ctx, db, stmts)
}

func downRolesBackfill(ctx context.Context, db *sql.DB) error {
	return execAll(ctx, db, []string{
		`ALTER TABLE users ADD COLUMN role VARCHAR(30) NOT NULL DEFAULT 'cashier' AFTER pin_hash`,
		`UPDATE users u JOIN roles r ON r.id = u.role_id SET u.role = LEFT(r.code, 30)`,
		`ALTER TABLE users DROP FOREIGN KEY fk_users_role`,
		`ALTER TABLE users MODIFY role_id BIGINT UNSIGNED NULL`,
		`DELETE FROM roles`,
	})
}

// EnsureSystemRolesAll inserts any missing template role for every tenant.
// Idempotent; also used by the seed command.
func EnsureSystemRolesAll(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT id FROM tenants`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, tid := range ids {
		for _, t := range authz.RoleTemplates {
			if _, err := db.ExecContext(ctx,
				`INSERT INTO roles (tenant_id, public_id, code, name, description, is_system, is_locked,
				   max_discount_percent, max_refund_amount, max_cash_out_amount)
				 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE id = id`,
				tid, security.NewPublicID(), t.Code, t.Name, t.Description, t.Locked,
				decStr(t.MaxDiscountPercent), decStr(t.MaxRefundAmount), decStr(t.MaxCashOutAmount),
			); err != nil {
				return fmt.Errorf("role %s for tenant %d: %w", t.Code, tid, err)
			}
		}
	}
	return nil
}

func execAll(ctx context.Context, db *sql.DB, stmts []string) error {
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("%w\n%s", err, s)
		}
	}
	return nil
}

type stringer interface{ String() string }

func decStr[T stringer](d *T) any {
	if d == nil {
		return nil
	}
	return (*d).String()
}
