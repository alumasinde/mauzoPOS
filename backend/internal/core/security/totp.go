package security

import (
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret creates a new TOTP secret for an account. The returned URL
// encodes the secret for QR provisioning in authenticator apps.
func GenerateTOTPSecret(issuer, account string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("security: totp generate: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTP checks a 6-digit code against secret, allowing one period of
// clock skew in each direction.
func ValidateTOTP(secret, code string) bool {
	ok, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && ok
}
