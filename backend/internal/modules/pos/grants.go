package pos

import (
	"context"
	"time"

	"mauzo/internal/core/httpx"
	"mauzo/internal/core/security"
)

// Supervisor grants are the proof that a sensitive action was authorized on
// this terminal: /pos/authorize mints a single-use token bound to the device,
// the authorizing user and the permission; the action endpoint consumes it.
// Like PIN lockout they live in memory — a restart simply requires
// re-authorization.
const grantTTL = 2 * time.Minute

type grant struct {
	tenantID   uint64 // grants are bound to the device that minted them
	userID     uint64
	permission string
	expires    time.Time
}

// ErrGrantInvalid rejects a missing/expired/wrong-permission grant.
var ErrGrantInvalid = httpx.NewError(403, "grant_invalid",
	"That supervisor authorization is missing or expired.")

// AuthorizeResult is /pos/authorize output: the supervisor's name plus a
// one-time grant token for the action that needed approval.
type AuthorizeResult struct {
	Name  string `json:"name"`
	Grant string `json:"grant"`
}

// Authorize verifies a supervisor PIN for perm and mints a grant token.
func (s *Service) Authorize(ctx context.Context, d Device, perm, pin string) (AuthorizeResult, error) {
	users, err := s.repo.UsersWithPermission(ctx, d.TenantID, perm)
	if err != nil {
		return AuthorizeResult{}, err
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
			tok, err := security.NewToken()
			if err != nil {
				return AuthorizeResult{}, err
			}
			s.mu.Lock()
			s.grants[tok] = grant{tenantID: d.TenantID, userID: u.ID, permission: perm, expires: time.Now().Add(grantTTL)}
			s.mu.Unlock()
			return AuthorizeResult{Name: u.Name, Grant: tok}, nil
		}
		s.recordFail(u.ID)
	}
	return AuthorizeResult{}, ErrBadPIN
}

// ConsumeGrant validates a grant token for perm and returns the authorizing
// user id. Single-use: a grant is deleted on read.
func (s *Service) ConsumeGrant(d Device, token, perm string) (*uint64, error) {
	if token == "" {
		return nil, nil
	}
	s.mu.Lock()
	g, ok := s.grants[token]
	delete(s.grants, token)
	s.mu.Unlock()
	if !ok || time.Now().After(g.expires) || g.permission != perm || g.tenantID != d.TenantID {
		return nil, ErrGrantInvalid
	}
	id := g.userID
	return &id, nil
}
