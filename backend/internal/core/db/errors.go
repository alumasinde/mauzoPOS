package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicate reports a unique-key violation (MySQL/MariaDB error 1062). Use
// it as the race-safe backstop behind service-level uniqueness checks.
func IsDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
