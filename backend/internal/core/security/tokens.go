package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/oklog/ulid/v2"
)

// NewToken returns a URL-safe 256-bit random token. The raw token is shown to
// the client once; only its SHA-256 hash is stored.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("security: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 hex digest stored in place of a raw token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewPublicID returns a new ULID for the public_id column exposed in APIs.
func NewPublicID() string {
	return ulid.Make().String()
}

// NewRecoveryCodes returns n human-readable one-time codes (XXXX-XXXX) for MFA
// recovery. Store the SHA-256 hash of each, never the plaintext.
func NewRecoveryCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("security: recovery codes: %w", err)
		}
		codes[i] = fmt.Sprintf("%04x-%04x", b[:2], b[2:])
	}
	return codes, nil
}
