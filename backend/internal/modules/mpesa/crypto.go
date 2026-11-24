package mpesa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// secretbox encrypts Daraja credentials at rest with AES-256-GCM. The key
// comes from MPESA_CRED_KEY (falling back to ENCRYPTION_KEY); in development
// it falls back to a derived key so the config still works out of the box
// (production requires the env var — config validation enforces that).
type secretbox struct{ aead cipher.AEAD }

func newSecretbox(key string) (*secretbox, error) {
	if key == "" {
		return nil, errors.New("mpesa: MPESA_CRED_KEY is required")
	}
	k := sha256.Sum256([]byte(key))
	aead, err := newAEAD(k[:])
	if err != nil {
		return nil, err
	}
	return &secretbox{aead: aead}, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// seal encrypts a secret; empty input returns nil so optional fields stay NULL.
func (s *secretbox) seal(plain string) ([]byte, error) {
	if plain == "" {
		return nil, nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, []byte(plain), nil), nil
}

// open decrypts a sealed blob.
func (s *secretbox) open(blob []byte) (string, error) {
	if len(blob) == 0 {
		return "", nil
	}
	if len(blob) < s.aead.NonceSize() {
		return "", errors.New("mpesa: credential blob too short")
	}
	nonce, ct := blob[:s.aead.NonceSize()], blob[s.aead.NonceSize():]
	plain, err := s.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("mpesa: credential decrypt failed — wrong MPESA_CRED_KEY?")
	}
	return string(plain), nil
}

// mask shows only the last four characters of a secret for UI display.
func mask(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "••••"
	}
	return "••••" + secret[len(secret)-4:]
}

// newCallbackToken generates an unguessable URL path secret.
func newCallbackToken() string {
	b := make([]byte, 36)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err) // crypto/rand is a hard dependency everywhere else too
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
