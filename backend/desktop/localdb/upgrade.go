package localdb

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// A till keeps its SQLite file across app updates, and CREATE TABLE IF NOT
// EXISTS never adds columns to a table that already exists. upgrade compares
// every table in schema with the file on disk and adds the columns that are
// missing, so a new version never fails with "table … has no column named …".
// Only additive changes are supported (new columns); a column is added with
// its declared default, or as nullable when it has none, because SQLite
// cannot add a NOT NULL column without a default.

var (
	createRe = regexp.MustCompile(`(?is)CREATE TABLE IF NOT EXISTS\s+(\w+)\s*\((.*?)\n\);`)
	colRe    = regexp.MustCompile(`^\s*(\w+)\s+(.+?)\s*,?\s*(--.*)?$`)
)

// tableColumns parses schema into table → ordered column definitions.
func tableColumns(schemaSQL string) map[string][][2]string {
	out := map[string][][2]string{}
	for _, m := range createRe.FindAllStringSubmatch(schemaSQL, -1) {
		table := m[1]
		for _, line := range strings.Split(m[2], "\n") {
			line = strings.TrimSpace(line)
			upper := strings.ToUpper(line)
			if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(upper, "PRIMARY KEY") ||
				strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "FOREIGN KEY") ||
				strings.HasPrefix(upper, "CHECK") || strings.HasPrefix(upper, "CONSTRAINT") {
				continue
			}
			c := colRe.FindStringSubmatch(line)
			if c == nil {
				continue
			}
			out[table] = append(out[table], [2]string{c[1], strings.TrimSuffix(strings.TrimSpace(c[2]), ",")})
		}
	}
	return out
}

// addable turns a CREATE TABLE column definition into one ALTER TABLE ADD
// COLUMN accepts: no PRIMARY KEY / UNIQUE, and NOT NULL only with a default.
func addable(def string) string {
	upper := strings.ToUpper(def)
	for _, kw := range []string{"PRIMARY KEY", "UNIQUE", "AUTOINCREMENT"} {
		if i := strings.Index(upper, kw); i >= 0 {
			def = strings.TrimSpace(def[:i] + def[i+len(kw):])
			upper = strings.ToUpper(def)
		}
	}
	if strings.Contains(upper, "NOT NULL") && !strings.Contains(upper, "DEFAULT") {
		i := strings.Index(upper, "NOT NULL")
		def = strings.TrimSpace(def[:i] + def[i+len("NOT NULL"):])
	}
	return def
}

func upgrade(ctx context.Context, d *DB) error {
	for table, cols := range tableColumns(schema) {
		var have []struct {
			Name string `db:"name"`
		}
		if err := d.SelectContext(ctx, &have, `SELECT name FROM pragma_table_info(?)`, table); err != nil {
			return fmt.Errorf("inspect %s: %w", table, err)
		}
		present := map[string]bool{}
		for _, h := range have {
			present[strings.ToLower(h.Name)] = true
		}
		for _, c := range cols {
			if present[strings.ToLower(c[0])] {
				continue
			}
			stmt := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, c[0], addable(c[1]))
			if _, err := d.ExecContext(ctx, stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("add %s.%s: %w", table, c[0], err)
			}
		}
	}
	return nil
}
