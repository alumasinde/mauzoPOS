package security

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

var fastParams = Argon2Params{MemoryKB: 8192, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestArgon2RoundTrip(t *testing.T) {
	hash, err := fastParams.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Fatalf("hash not in PHC format: %s", hash)
	}
	if err := VerifyPassword("correct horse battery staple", hash); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if err := VerifyPassword("wrong", hash); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, h := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$AA$AA", "$argon2id$v=19$bad"} {
		if err := VerifyPassword("x", h); errors.Is(err, ErrPasswordMismatch) || err == nil {
			t.Fatalf("expected format error for %q, got %v", h, err)
		}
	}
}

func TestTokenHashDeterministic(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 43 {
		t.Fatalf("token length = %d", len(tok))
	}
	if h := HashToken(tok); h != HashToken(tok) {
		t.Fatal("hash not deterministic")
	}
	other, _ := NewToken()
	if HashToken(tok) == HashToken(other) {
		t.Fatal("hash collision between distinct tokens")
	}
}

func TestPublicIDIsULID(t *testing.T) {
	id := NewPublicID()
	if len(id) != 26 {
		t.Fatalf("public id length = %d", len(id))
	}
}

func TestTOTPRoundTrip(t *testing.T) {
	secret, url, err := GenerateTOTPSecret("MauzoPOS", "admin@mauzo.local")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("unexpected url %q", url)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ValidateTOTP(secret, code) {
		t.Fatal("fresh TOTP code did not validate")
	}
	if ValidateTOTP(secret, "000000") && code != "000000" {
		t.Fatal("invalid code accepted")
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes, err := NewRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 8 {
		t.Fatalf("got %d codes", len(codes))
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if len(c) != 9 || c[4] != '-' {
			t.Fatalf("bad code format %q", c)
		}
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
	}
}
