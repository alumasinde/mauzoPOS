// Package migrations embeds the SQL migration files (goose format).
//
// Naming: NNNNN_<module>_<change>.sql, e.g. 00002_tenants_create.sql.
// Keep SQL inside the MySQL 8 / MariaDB 10.11+ common subset (see README §5).
// Create a new file with: go run ./cmd/migrate create <module>_<change>
package migrations

import "embed"

// FS embeds all SQL migration files.
//
//go:embed *.sql
var FS embed.FS
