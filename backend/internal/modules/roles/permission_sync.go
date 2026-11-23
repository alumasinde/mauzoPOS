package roles

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"mauzo/internal/core/authz"
)

// SyncPermissions mirrors the code registry into the permissions table:
// upserts every permission and deletes rows whose code no longer exists
// (their grants cascade away). Run at boot and by `migrate seed`.
func (s *Service) SyncPermissions(ctx context.Context, log *slog.Logger) error {
	// A binary that did not link every module would delete real grants via
	// the stale-row cleanup below.
	if !authz.IsComplete() {
		return fmt.Errorf("roles: permission registry incomplete — import modules/registerall")
	}
	all := authz.All()
	codes := make([]any, 0, len(all))
	for _, p := range all {
		var module any
		if p.Module != "" {
			module = p.Module
		}
		if _, err := s.repo.q.ExecContext(ctx,
			`INSERT INTO permissions (code, scope, module, description, is_sensitive) VALUES (?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE scope = VALUES(scope), module = VALUES(module),
			   description = VALUES(description), is_sensitive = VALUES(is_sensitive)`,
			p.Code, string(p.Scope()), module, p.Description, p.Sensitive); err != nil {
			return err
		}
		codes = append(codes, p.Code)
	}
	if len(codes) == 0 {
		return nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(codes)), ",")
	res, err := s.repo.q.ExecContext(ctx, `DELETE FROM permissions WHERE code NOT IN (`+marks+`)`, codes...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 && log != nil {
		log.Warn("roles: removed permissions no longer in code (grants cascaded)", "count", n)
	}
	return nil
}
