package pos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"mauzo/internal/core/authz"
	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
	"mauzo/internal/modules/mpesa"
	"mauzo/internal/modules/sales"
	"mauzo/internal/modules/shifts"
	"mauzo/internal/modules/terminals"
)

// Sentinel errors.
var (
	ErrBadActivationCode = httpx.NewError(422, "bad_activation_code",
		"That code is not valid. Check it on the back-office screen (letters and digits, e.g. ABCDE-FGH23) — or issue a new one.")
	ErrActivationExpired = httpx.NewError(422, "activation_code_expired",
		"That code has expired. Issue a new one in the back office (Team & access → POS terminals).")
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
	conn     *sqlx.DB
	repo     *Repo
	sales    Sales
	receipts Receipts
	mpesa    Mpesa
	shifts   Shifts

	mu     sync.Mutex
	fails  map[uint64]*pinFail
	grants map[string]grant
}

type pinFail struct {
	count int
	until time.Time
}

// NewService builds a Service.
func NewService(conn *sqlx.DB, repo *Repo) *Service {
	return &Service{conn: conn, repo: repo, fails: map[uint64]*pinFail{}, grants: map[string]grant{}}
}

// BindSales attaches the sales service (wired post-construction to avoid an
// import cycle between the device API and the sales module).
func (s *Service) BindSales(v Sales) { s.sales = v }

// Sales is the sale-recording surface the device endpoints use.
type Sales interface {
	Complete(ctx context.Context, in sales.CompleteInput) (sales.SaleDetail, bool, error)
	GetByReceipt(ctx context.Context, tenantID, branchID uint64, ref string) (sales.SaleDetail, error)
	Authz(ctx context.Context, tenantID, userID uint64) (*sales.CashierAuthz, error)
}

// BindMpesa attaches the M-Pesa service (nil when the module is disabled).
func (s *Service) BindMpesa(v Mpesa) { s.mpesa = v }

// Mpesa is the M-Pesa surface the till uses — STK pushes, status polls, the
// live collection feed and allocation.
type Mpesa interface {
	InitiateSTK(ctx context.Context, tenantID, branchID, byUserID uint64,
		phone, accountRef, note string, amount decimal.Decimal) (*mpesa.Transaction, error)
	STKStatus(ctx context.Context, tenantID uint64, publicID string) (*mpesa.Transaction, error)
	FeedForTill(ctx context.Context, tenantID, branchID uint64) ([]mpesa.Transaction, error)
	ServeFeed(w http.ResponseWriter, r *http.Request, tenantID, branchID uint64)
	Allocate(ctx context.Context, tenantID, actorID uint64, txnPublicID, salePublicID string) error
}

// BindShifts attaches the shifts service (nil when the module is disabled).
func (s *Service) BindShifts(v Shifts) { s.shifts = v }

// Shifts is the drawer-session surface the till uses.
type Shifts interface {
	Current(ctx context.Context, tenantID, terminalID uint64) (*shifts.Shift, error)
	Open(ctx context.Context, in shifts.OpenInput) (*shifts.Shift, error)
	AddMovement(ctx context.Context, in shifts.MovementInput) (*shifts.Movement, error)
	DrawerOpen(ctx context.Context, tenantID, branchID, terminalID, cashierID uint64) error
	Report(ctx context.Context, sh shifts.Shift) (*shifts.Report, error)
	ReportByPublic(ctx context.Context, tenantID uint64, publicID string) (*shifts.Report, error)
	Close(ctx context.Context, in shifts.CloseInput) (*shifts.Report, error)
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
	code := terminals.NormalizeCode(req.Code)
	if len(code) < 6 || req.DeviceID == "" {
		return Activation{}, ErrBadActivationCode
	}
	row, err := s.repo.terminalByCodeHash(ctx, security.HashToken(code))
	if err != nil {
		return Activation{}, ErrBadActivationCode
	}
	if row.Status == "disabled" {
		return Activation{}, ErrTerminalDisabled
	}
	if row.CodeExpiresAt != nil && time.Now().After(*row.CodeExpiresAt) {
		return Activation{}, ErrActivationExpired
	}
	// A terminal that already had a device is being re-paired (new PC,
	// reinstall): the new device takes over and the old token stops working.
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

func (s *Service) rolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	var locked bool
	if err := s.conn.GetContext(ctx, &locked, `SELECT is_locked FROM roles WHERE id = ?`, roleID); err != nil {
		return nil, err
	}
	if locked { // the owner: every permission, no stored rows
		return authz.AllOf().Codes(authz.ScopeTenant), nil
	}
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
