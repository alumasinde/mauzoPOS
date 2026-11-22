package audit

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/httpx"
)

// Repo reads audit_logs. Writes go through core/audit.Writer.
type Repo struct {
	db *sqlx.DB
}

// NewRepo builds a Repo.
func NewRepo(conn *sqlx.DB) *Repo {
	return &Repo{db: conn}
}

const cols = `id, tenant_id, branch_id, actor_type, actor_id, supervisor_id,
	action, entity_type, entity_id, data, ip_address, user_agent, request_id, created_at`

// List returns a page of audit rows, newest first. tenantID nil = platform
// scope (all rows); non-nil restricts to that tenant. actorType may filter by
// "platform_user", "user", etc.
func (r *Repo) List(ctx context.Context, tenantID *uint64, actorType string, p httpx.Page) ([]Log, int, error) {
	var parts []string
	var args []any
	if tenantID != nil {
		parts = append(parts, "tenant_id = ?")
		args = append(args, *tenantID)
	}
	if actorType != "" {
		parts = append(parts, "actor_type = ?")
		args = append(args, actorType)
	}
	if p.Query != "" {
		parts = append(parts, "(action LIKE ? OR entity_id LIKE ?)")
		like := "%" + p.Query + "%"
		args = append(args, like, like)
	}
	where := ""
	if len(parts) > 0 {
		where = " WHERE " + strings.Join(parts, " AND ")
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM audit_logs`+where, args...); err != nil {
		return nil, 0, err
	}
	var out []Log
	err := r.db.SelectContext(ctx, &out,
		`SELECT `+cols+` FROM audit_logs`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, p.Limit(), p.Offset())...)
	return out, total, err
}
