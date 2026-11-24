package terminals

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"
)

// ActivationTTL is how long an issued code stays valid. Long enough to walk
// to the till and type it; short enough that a code on a sticky note dies.
const ActivationTTL = 24 * time.Hour

// codeAlphabet leaves out characters people misread off a screen: 0/O, 1/I/L.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// codeLen gives 31^10 ≈ 8×10^14 combinations; with expiry and the API rate
// limit, guessing is not practical.
const codeLen = 10

// NewActivationCode returns a code formatted for reading aloud: ABCDE-FGHJK.
func NewActivationCode() (string, error) {
	b := make([]byte, codeLen)
	size := big.NewInt(int64(len(codeAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, size)
		if err != nil {
			return "", err
		}
		b[i] = codeAlphabet[n.Int64()]
	}
	return string(b[:5]) + "-" + string(b[5:]), nil
}

// NormalizeCode makes a typed code comparable: case and separators (dash,
// space, underscore) are forgiven; every other character must match. The
// hash of the normalized form is what is stored.
func NormalizeCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.NewReplacer("-", "", " ", "", "_", "").Replace(s)
}
