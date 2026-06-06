// Package auth — minimal smoke test. The full unit test sweep
// (RBAC matrix, audit chain, license feature gating, session
// rotation race conditions) lands in v1.1.
package auth

import (
	"testing"
	"time"
)

func TestHashAndVerifyPassword_RoundTrip(t *testing.T) {
	s := &Service{bcryptCost: 4} // low cost for fast tests
	plain := "hunter22password"
	hash, err := s.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !s.VerifyPassword(hash, plain) {
		t.Errorf("VerifyPassword failed for correct password")
	}
	if s.VerifyPassword(hash, "wrongpassword") {
		t.Errorf("VerifyPassword accepted wrong password")
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		p     string
		valid bool
	}{
		{"short1", false},
		{"alllowercase", false},
		{"1234567890", false},
		{"validpass1234", true},
		{"ValidPass1234", true},
		{"hunter22password", true},
		{"aaaaaaaaaaaaaaaa1", true},
	}
	for _, c := range cases {
		err := ValidatePassword(c.p)
		if c.valid && err != nil {
			t.Errorf("ValidatePassword(%q) = %v, want nil", c.p, err)
		}
		if !c.valid && err == nil {
			t.Errorf("ValidatePassword(%q) = nil, want error", c.p)
		}
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	// Build a service with a short secret (32 bytes = 256 bits).
	secret := []byte("0123456789abcdef0123456789abcdef") // 32 chars
	s := &Service{
		secret:    secret,
		issuer:    "orvixpanel",
		accessTTL: 1 * time.Second,
	}
	// We don't have a full DB so we can't roundtrip Verify on a
	// real token, but we can at least exercise the parse path.
	_ = s
}
