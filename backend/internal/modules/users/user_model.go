package users

import "time"

// User is a tenant employee. Role and RoleName are joined from roles.
type User struct {
	ID            uint64     `db:"id"`
	TenantID      uint64     `db:"tenant_id"`
	PublicID      string     `db:"public_id"`
	Name          string     `db:"name"`
	Email         *string    `db:"email"`
	Username      *string    `db:"username"`
	PasswordHash  *string    `db:"password_hash"`
	PINHash       *string    `db:"pin_hash"`
	RoleID        uint64     `db:"role_id"`
	Role          string     `db:"role"`
	RoleName      string     `db:"role_name"`
	RolePublicID  string     `db:"role_public_id"`
	AllBranches   bool       `db:"all_branches"`
	Status        string     `db:"status"`
	TOTPSecret    *string    `db:"totp_secret"`
	TOTPEnabled   bool       `db:"totp_enabled"`
	RecoveryCodes []byte     `db:"recovery_codes"`
	LastLoginAt   *time.Time `db:"last_login_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`

	BranchIDs []uint64 `db:"-"`
}
