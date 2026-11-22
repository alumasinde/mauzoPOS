package users

import "time"

// Tenant user (employee) interim roles until the roles table lands in Phase 3.
const (
	UserRoleOwner      = "owner"
	UserRoleAdmin      = "admin"
	UserRoleManager    = "manager"
	UserRoleSupervisor = "supervisor"
	UserRoleCashier    = "cashier"
)

// User is a tenant employee.
type User struct {
	ID            uint64     `db:"id"`
	TenantID      uint64     `db:"tenant_id"`
	PublicID      string     `db:"public_id"`
	Name          string     `db:"name"`
	Email         *string    `db:"email"`
	Username      *string    `db:"username"`
	PasswordHash  *string    `db:"password_hash"`
	PINHash       *string    `db:"pin_hash"`
	Role          string     `db:"role"`
	Status        string     `db:"status"`
	TOTPSecret    *string    `db:"totp_secret"`
	TOTPEnabled   bool       `db:"totp_enabled"`
	RecoveryCodes []byte     `db:"recovery_codes"`
	LastLoginAt   *time.Time `db:"last_login_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}
