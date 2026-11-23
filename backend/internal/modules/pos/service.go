package pos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Sentinel errors.
var (
	ErrBadActivationCode = httpx.NewError(422, "bad_activation_code",
		"That activation code is invalid or already used.")
	ErrTerminalDisabled = httpx.NewError(403, "terminal_disabled", "This terminal is disabled.")
	ErrBadPIN           = httpx.NewError(401, "bad_pin", "Wrong PIN.")
	ErrPINLocked        = httpx.NewError(429, "pin_locked", "Too many wrong PINs. Try again shortly.")
	ErrHeldNotFound     = httpx.ErrNotFound
	ErrHeldEmpty        = httpx.NewError(422, "empty_cart", "A held sale needs at least one line.")
)

const (
	pinMaxAttempts = 5
	pinLockFor     = 5 * time.Minute
)

// Service owns terminal activation, device auth and PIN verification.
type Service struct {
	conn *sqlx.DB
	repo *Repo

	mu    sync.Mutex
	fails map[uint64]*pinFail
}

type pinFail struct {
	count int
	until time.Time
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo) *Service {
	return &Service{conn: conn, repo: repo, fails: map[uint64]*pinFail{}}
}

// ---- activation --------------------------------------------------------------

// ActivateRequest is POST /pos/activate.
type ActivateRequest struct {
	Code       string `json:"code"`
	DeviceName string `json:"device_name"`
	DeviceID   string `json:"device_id"`
}

// Activation is the activation result — the device token is shown once.
type Activation struct {
	Token    string   `json:"token"`
	Terminal Terminal `json:"terminal"`
}

// Terminal describes the bound device for the UI.
type Terminal struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Branch       string `json:"branch"`
	Tenant       string `json:"tenant"`
	Slug         string `json:"slug"`
	BusinessType string `json:"business_type"`
}

// Activate validates the one-time code and mints the device token.
func (s *Service) Activate(ctx context.Context, req ActivateRequest) (Activation, error) {
	if len(req.Code) < 6 || req.DeviceID == "" {
		return Activation{}, ErrBadActivationCode
	}
	row, err := s.repo.terminalByCodeHash(ctx, security.HashToken(req.Code))
	if err != nil {
		return Activation{}, ErrBadActivationCode
	}
	if row.ActivatedAt != nil {
		return Activation{}, ErrBadActivationCode
	}
	token, err := security.NewToken()
	if err != nil {
		return Activation{}, err
	}
	name := req.DeviceName
	if name == "" {
		name = row.Name
	}
	err = s.repo.Activate(ctx, s.conn, row.ID, req.DeviceID, name, security.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return Activation{}, ErrTerminalDisabled
	}
	if err != nil {
		return Activation{}, err
	}
	return Activation{Token: token, Terminal: Terminal{
		ID: row.PublicID, Name: name, Branch: row.BranchName,
		Tenant: row.TenantName, Slug: row.TenantSlug, BusinessType: row.BusinessType,
	}}, nil
}

// AuthenticateDevice resolves a bearer token to a Device (middleware hook).
func (s *Service) AuthenticateDevice(ctx context.Context, token string) (Device, error) {
	row, err := s.repo.terminalByTokenHash(ctx, security.HashToken(token))
	if err != nil {
		return Device{}, httpx.ErrUnauthorized
	}
	if row.Status != "active" {
		return Device{}, ErrTerminalDisabled
	}
	s.repo.Touch(ctx, row.ID)
	return Device{TerminalID: row.ID, TenantID: row.TenantID, BranchID: row.BranchID}, nil
}

// Whoami returns the terminal descriptor for an authenticated device.
func (s *Service) Whoami(ctx context.Context, d Device) (Terminal, error) {
	row, err := s.repo.terminalBy(ctx, `t.id = ?`, d.TerminalID)
	if err != nil {
		return Terminal{}, err
	}
	return Terminal{ID: row.PublicID, Name: row.Name, Branch: row.BranchName,
		Tenant: row.TenantName, Slug: row.TenantSlug, BusinessType: row.BusinessType}, nil
}

// ---- PIN sign-in --------------------------------------------------------------

// SignIn verifies a cashier PIN. Online verify; the terminal also caches
// hashes locally for offline sign-in.
func (s *Service) SignIn(ctx context.Context, d Device, userID uint64, pin string) (*SyncCashier, []string, error) {
	if s.locked(userID) {
		return nil, nil, ErrPINLocked
	}
	hash, roleID, status, err := s.repo.UserPIN(ctx, d.TenantID, userID)
	if err != nil || status != "active" || security.VerifyPassword(pin, hash) != nil {
		s.recordFail(userID)
		return nil, nil, ErrBadPIN
	}
	s.clearFail(userID)
	perms, err := s.rolePermissions(ctx, roleID)
	if err != nil {
		return nil, nil, err
	}
	return &SyncCashier{ID: userID}, perms, nil
}

// Authorize verifies a PIN against any user holding the given permission —
// the supervisor-override check for sensitive POS actions.
func (s *Service) Authorize(ctx context.Context, d Device, perm, pin string) (string, error) {
	users, err := s.repo.UsersWithPermission(ctx, d.TenantID, perm)
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if s.locked(u.ID) {
			continue
		}
		hash, _, _, err := s.repo.UserPIN(ctx, d.TenantID, u.ID)
		if err != nil {
			continue
		}
		if security.VerifyPassword(pin, hash) == nil {
			s.clearFail(u.ID)
			return u.Name, nil
		}
		s.recordFail(u.ID)
	}
	return "", ErrBadPIN
}

func (s *Service) rolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	var out []string
	err := s.conn.SelectContext(ctx, &out,
		`SELECT permission FROM role_permissions WHERE role_id = ?`, roleID)
	return out, err
}

func (s *Service) locked(userID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.fails[userID]
	return ok && f.count >= pinMaxAttempts && time.Now().Before(f.until)
}

func (s *Service) recordFail(userID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fails[userID]
	if f == nil {
		f = &pinFail{}
		s.fails[userID] = f
	}
	f.count++
	if f.count >= pinMaxAttempts {
		f.until = time.Now().Add(pinLockFor)
		f.count = 0
	}
}

func (s *Service) clearFail(userID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fails, userID)
}

// ---- held sales ---------------------------------------------------------------

// HoldRequest is POST /pos/held.
type HoldRequest struct {
	UserPublicID string          `json:"user_id"`
	Label        string          `json:"label"`
	Cart         json.RawMessage `json:"cart"`
}

// Hold parks a cart on the server (shared across terminals on the branch).
func (s *Service) Hold(ctx context.Context, d Device, req HoldRequest) (HeldSale, error) {
	if len(req.Cart) == 0 || string(req.Cart) == "[]" || string(req.Cart) == "null" {
		return HeldSale{}, ErrHeldEmpty
	}
	userID, err := s.userByPublicID(ctx, d.TenantID, req.UserPublicID)
	if err != nil {
		return HeldSale{}, httpx.NewError(422, "bad_user", "Unknown cashier.")
	}
	h := HeldSale{
		TenantID: d.TenantID, BranchID: d.BranchID, TerminalID: d.TerminalID,
		UserID: userID, PublicID: security.NewPublicID(), Label: req.Label, Cart: string(req.Cart),
	}
	id, err := s.repo.HoldCreate(ctx, h)
	if err != nil {
		return HeldSale{}, err
	}
	h.ID = id
	return h, nil
}

// Held lists parked carts for the branch.
func (s *Service) Held(ctx context.Context, d Device) ([]HeldSale, error) {
	return s.repo.HeldList(ctx, d.TenantID, d.BranchID)
}

// RecallHeld returns the cart and marks it recalled.
func (s *Service) RecallHeld(ctx context.Context, d Device, publicID string) (HeldSale, error) {
	h, err := s.repo.HeldByPublicID(ctx, d.TenantID, d.BranchID, publicID)
	if err != nil {
		return HeldSale{}, ErrHeldNotFound
	}
	if h.Status != "held" {
		return HeldSale{}, ErrHeldNotFound
	}
	if err := s.repo.HeldSetStatus(ctx, d.TenantID, h.ID, "recalled"); err != nil {
		return HeldSale{}, err
	}
	return h, nil
}

// CancelHeld voids a parked cart.
func (s *Service) CancelHeld(ctx context.Context, d Device, publicID string) error {
	h, err := s.repo.HeldByPublicID(ctx, d.TenantID, d.BranchID, publicID)
	if err != nil {
		return ErrHeldNotFound
	}
	return s.repo.HeldSetStatus(ctx, d.TenantID, h.ID, "cancelled")
}

func (s *Service) userByPublicID(ctx context.Context, tenantID uint64, publicID string) (uint64, error) {
	var id uint64
	err := s.conn.GetContext(ctx, &id,
		`SELECT id FROM users WHERE tenant_id = ? AND public_id = ?`, tenantID, publicID)
	return id, err
}
