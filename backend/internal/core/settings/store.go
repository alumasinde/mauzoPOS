package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/db"
)

// Entry is one tenant_settings row.
type Entry struct {
	Key       string          `db:"key"`
	Value     json.RawMessage `db:"value"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// Store reads and writes tenant_settings rows.
type Store struct {
	q db.Querier
}

// NewStore builds a Store.
func NewStore(conn *sqlx.DB) *Store { return &Store{q: conn} }

// WithQuerier returns a Store bound to q (typically a *sqlx.Tx) so settings
// writes can join a caller's transaction.
func (s *Store) WithQuerier(q db.Querier) *Store { return &Store{q: q} }

// Get returns one setting, or sql.ErrNoRows.
func (s *Store) Get(ctx context.Context, tenantID uint64, key string) (Entry, error) {
	var e Entry
	err := s.q.GetContext(ctx, &e,
		`SELECT `+"`key`"+`, value, updated_at FROM tenant_settings WHERE tenant_id = ? AND `+"`key` = ?", tenantID, key)
	return e, err
}

// List returns every setting for a tenant.
func (s *Store) List(ctx context.Context, tenantID uint64) ([]Entry, error) {
	var out []Entry
	err := s.q.SelectContext(ctx, &out,
		`SELECT `+"`key`"+`, value, updated_at FROM tenant_settings WHERE tenant_id = ? ORDER BY `+"`key`", tenantID)
	return out, err
}

// Set validates then upserts one setting.
func (s *Store) Set(ctx context.Context, tenantID uint64, key string, value json.RawMessage) error {
	if err := Validate(key, value); err != nil {
		if errors.Is(err, ErrUnknownKey) {
			return ErrUnknownKey
		}
		return fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO tenant_settings (tenant_id, `+"`key`"+`, value) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		tenantID, key, value)
	return err
}

// SetDefaults inserts keys that are absent and never overwrites existing ones —
// used by the profile engine so applying a profile stays idempotent.
func (s *Store) SetDefaults(ctx context.Context, tenantID uint64, defaults map[string]json.RawMessage) error {
	for key, value := range defaults {
		if err := Validate(key, value); err != nil {
			return fmt.Errorf("default %w", err)
		}
		// ON DUPLICATE KEY no-op rather than INSERT IGNORE: IGNORE would also
		// swallow real errors such as a foreign-key violation.
		if _, err := s.q.ExecContext(ctx,
			`INSERT INTO tenant_settings (tenant_id, `+"`key`"+`, value) VALUES (?, ?, ?)
			 ON DUPLICATE KEY UPDATE value = value`,
			tenantID, key, value); err != nil {
			return err
		}
	}
	return nil
}

// ErrInvalidValue wraps schema violations.
var ErrInvalidValue = errors.New("settings: invalid value")
