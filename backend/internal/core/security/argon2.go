// Package security provides password hashing (argon2id), opaque token
// generation/hashing and TOTP multi-factor authentication helpers.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params are the work factors for argon2id hashing. They are encoded
// into each hash so parameters can change without invalidating old hashes.
type Argon2Params struct {
	MemoryKB    uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  int
	KeyLength   uint32
}

// DefaultArgon2Params matches the .env.example defaults (RFC 9106 "recommended").
var DefaultArgon2Params = Argon2Params{
	MemoryKB:    65536,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// ErrPasswordMismatch is returned when a plaintext password does not match a hash.
var ErrPasswordMismatch = errors.New("security: password does not match")

// HashPassword returns a PHC-format argon2id hash of password.
func (p Argon2Params) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("security: empty password")
	}
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKB, p.Parallelism, p.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKB, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword checks password against a PHC-format argon2id hash. It parses
// the parameters from the hash, so hashes made with older parameters still verify.
func VerifyPassword(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("security: unsupported hash format")
	}
	var version int
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return errors.New("security: unsupported argon2 version")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return errors.New("security: malformed argon2 parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return errors.New("security: malformed salt")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) > 1<<20 {
		return errors.New("security: malformed key")
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want))) //nolint:gosec // length bounded above
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}
