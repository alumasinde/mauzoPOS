package bindings

import (
	"errors"
	"time"
)

const (
	pinMaxAttempts = 5
	pinLockFor     = 5 * time.Minute
)

// Cashier is a sign-in candidate from the synced working set.
type Cashier struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RoleName string `json:"role_name"`
}

// Cashiers lists sign-in candidates for the picker screen.
func (a *App) Cashiers() ([]Cashier, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	rows, err := a.db.Cashiers(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Cashier, 0, len(rows))
	for _, c := range rows {
		out = append(out, Cashier{ID: c.ID, Name: c.Name, RoleName: c.RoleName})
	}
	return out, nil
}

// Login verifies a cashier PIN — online against the server when reachable
// (fresh permissions + revocation), otherwise against the synced argon2 hash.
func (a *App) Login(cashierID, pin string) (*Session, error) {
	if a.db == nil {
		return nil, errors.New("local database not ready")
	}
	if a.pinLocked(cashierID) {
		return nil, errors.New("too many wrong PINs — try again in a few minutes")
	}
	sess, err := a.loginOnline(cashierID, pin)
	if err != nil && !errors.Is(err, ErrOffline) {
		a.recordPINFail(cashierID)
		return nil, err
	}
	if sess == nil { // offline (or offline error): local verify
		c, err := a.db.VerifyPIN(a.ctx, cashierID, pin)
		if err != nil {
			a.recordPINFail(cashierID)
			return nil, errors.New("wrong PIN")
		}
		sess = &Session{CashierID: c.ID, Name: c.Name, RoleName: c.RoleName, Permissions: c.Permissions}
	}
	a.clearPINFail(cashierID)
	sess.Since = time.Now().Format("15:04")
	a.mu.Lock()
	a.session = sess
	a.mu.Unlock()
	return sess, nil
}

// loginOnline asks the server to verify; returns (nil, ErrOffline) on network
// failure so the caller can verify locally.
func (a *App) loginOnline(cashierID, pin string) (*Session, error) {
	var out struct {
		OK          bool     `json:"ok"`
		Permissions []string `json:"permissions"`
	}
	err := a.request("POST", "/api/v1/pos/signin",
		map[string]string{"user_id": cashierID, "pin": pin}, &out)
	if errors.Is(err, ErrOffline) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	c, err := a.db.VerifyPIN(a.ctx, cashierID, pin) // hydrate name/role locally
	if err != nil {
		return nil, err
	}
	return &Session{CashierID: c.ID, Name: c.Name, RoleName: c.RoleName,
		Permissions: out.Permissions}, nil
}

// Logout ends the cashier session (till stays bound to the device).
func (a *App) Logout() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.session = nil
}

// CurrentSession returns the signed-in cashier, or nil.
func (a *App) CurrentSession() *Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.session
}

// Authorize checks a supervisor PIN for a sensitive action. Online it asks
// the server; offline it scans synced cashiers holding the permission.
func (a *App) Authorize(permission, pin string) (string, error) {
	if a.db == nil {
		return "", errors.New("local database not ready")
	}
	var out struct {
		OK           bool   `json:"ok"`
		AuthorizedBy string `json:"authorized_by"`
	}
	err := a.request("POST", "/api/v1/pos/authorize",
		map[string]string{"permission": permission, "pin": pin}, &out)
	if err == nil {
		return out.AuthorizedBy, nil
	}
	if !errors.Is(err, ErrOffline) {
		return "", err
	}
	name, lerr := a.db.AuthorizePIN(a.ctx, permission, pin)
	if lerr != nil {
		return "", errors.New("wrong PIN")
	}
	return name, nil
}

// Can reports whether the signed-in cashier holds a permission (UI hint —
// sensitive actions still go through Authorize).
func (a *App) Can(permission string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return false
	}
	for _, p := range a.session.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (a *App) pinLocked(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f, ok := a.fails[id]
	return ok && f.count >= pinMaxAttempts && time.Now().Before(f.until)
}

func (a *App) recordPINFail(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[id]
	if f == nil {
		f = &pinFail{}
		a.fails[id] = f
	}
	f.count++
	if f.count >= pinMaxAttempts {
		f.until = time.Now().Add(pinLockFor)
		f.count = 0
	}
}

func (a *App) clearPINFail(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.fails, id)
}
