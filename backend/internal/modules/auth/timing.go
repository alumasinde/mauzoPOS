package auth

import (
	"sync"

	"mauzo/internal/core/security"
)

// timingEqualiser makes sign-in attempts for unknown or password-less
// identities cost the same argon2id verification as a wrong password, so
// response time does not reveal which identities exist.
type timingEqualiser struct {
	once sync.Once
	hash string
}

// burn verifies password against a throwaway hash built lazily with the
// service's argon2 parameters. The result is discarded.
func (s *Service) burnPasswordCheck(password string) {
	s.timing.once.Do(func() {
		s.timing.hash, _ = s.params.HashPassword("mauzo-timing-equaliser")
	})
	if s.timing.hash != "" {
		_ = security.VerifyPassword(password, s.timing.hash)
	}
}
