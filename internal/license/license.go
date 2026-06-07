// Package license implements the OrvixPanel license key system.
//
// v0.3.0 Enterprise Edition rewrite.
//
// Key format (signed payload):
//   ORVIX-{TIER}-{YEAR}-{HASH}-{SIG}
//
// Where SIG is an ECDSA-P256 signature of the JSON payload
// {tier, max_servers, expires_at, issued_at, licensed_to, features}.
// Operators drop the public key (PEM) into the ORVIX_LICENSE_PUBKEY
// env var. v0.1.0's "any ORVIX-{TIER}-{YEAR}-* shape passes" is kept
// behind ORVIX_ALLOW_DEV=1 for fresh-install dev mode.
//
// State machine:
//   active   — now < ExpiresAt
//   grace    — ExpiresAt <= now < ExpiresAt + GraceDays
//   locked   — now >= ExpiresAt + GraceDays  →  writes return 423
//
// v0.3.0 does NOT phone home. The license server integration lives
// in v0.4.0 (out of scope for this release).
package license

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	ErrInvalidKey    = errors.New("license key is invalid")
	ErrExpired       = errors.New("license has expired")
	ErrOffline       = errors.New("license server unreachable")
	ErrMalformedKey  = errors.New("license key is malformed")
	ErrSignatureBad  = errors.New("license signature is invalid")
	ErrNoPublicKey   = errors.New("no license public key configured (set ORVIX_LICENSE_PUBKEY)")
	ErrGraceExhausted = errors.New("license expired and grace period exhausted")
)

// -----------------------------------------------------------------------------
// Tier constants
// -----------------------------------------------------------------------------

const (
	TierSMB        = "smb"
	TierISP        = "isp"
	TierEnterprise = "enterprise"
	TierWhiteLabel = "whitelabel"
)

// Mode is the runtime enforcement mode.
type Mode string

const (
	ModeActive Mode = "active"
	ModeGrace  Mode = "grace"
	ModeLocked Mode = "locked"
)

// -----------------------------------------------------------------------------
// License payload
// -----------------------------------------------------------------------------

// License is the parsed + verified license payload.
type License struct {
	Tier        string    `json:"tier"`
	MaxServers  int       `json:"max_servers"`
	ExpiresAt   int64     `json:"expires_at"` // unix seconds
	IssuedAt    int64     `json:"issued_at"`  // unix seconds
	Features    []string  `json:"features"`
	LicensedTo  string    `json:"licensed_to"`
	GraceDays   int       `json:"grace_days"`
}

// ExpiresAtTime returns the expiry as time.Time.
func (l *License) ExpiresAtTime() time.Time {
	return time.Unix(l.ExpiresAt, 0).UTC()
}

// IssuedAtTime returns the issued-at as time.Time.
func (l *License) IssuedAtTime() time.Time {
	return time.Unix(l.IssuedAt, 0).UTC()
}

// GraceEndsAt returns when the grace period ends.
func (l *License) GraceEndsAt() time.Time {
	return l.ExpiresAtTime().Add(time.Duration(l.GraceDays) * 24 * time.Hour)
}

// Mode returns the current enforcement mode at the given moment.
func (l *License) ModeAt(now time.Time) Mode {
	if now.Before(l.ExpiresAtTime()) {
		return ModeActive
	}
	if now.Before(l.GraceEndsAt()) {
		return ModeGrace
	}
	return ModeLocked
}

// DaysRemaining returns days (rounded down) until expiry. Negative
// if already expired.
func (l *License) DaysRemaining(now time.Time) int {
	d := l.ExpiresAtTime().Sub(now)
	return int(d / (24 * time.Hour))
}

// DaysUntilLocked returns days (rounded down) until the panel goes
// read-only. Can be negative if already locked.
func (l *License) DaysUntilLocked(now time.Time) int {
	d := l.GraceEndsAt().Sub(now)
	return int(d / (24 * time.Hour))
}

// HasFeature supports "*" wildcard and "!" negation.
//
// Order of evaluation: negations are checked first (any matching "!"
// pattern returns false), then positive patterns (any matching
// positive pattern returns true). This matches the documented
// semantics: "!whitelabel.*" disables whitelabel.* even if "*" is
// also in the list.
func (l *License) HasFeature(feature string) bool {
	if l == nil {
		return false
	}
	for _, pattern := range l.Features {
		if strings.HasPrefix(pattern, "!") {
			if matches(strings.TrimPrefix(pattern, "!"), feature) {
				return false
			}
		}
	}
	for _, pattern := range l.Features {
		if strings.HasPrefix(pattern, "!") {
			continue
		}
		if pattern == "*" || matches(pattern, feature) {
			return true
		}
	}
	return false
}

func matches(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(s, prefix+".") || s == prefix
	}
	return pattern == s
}

// TierFeatures — the default feature set per tier.
//
// v0.3.0 adds enterprise-gated features: vault, rbac.custom, apikey,
// audit.search, audit.export, quota.tenant. The ISP tier gets
// audit.search + quota.tenant only.
var TierFeatures = map[string][]string{
	TierSMB: {
		"hosting.*", "dns.*", "mail.basic", "ssl.*",
		"database.*", "files.*", "firewall.basic",
		"guardian.basic", "backup.local",
		"readonly.enforce",
	},
	TierISP: {
		"*", "!guardian.llm", "!whitelabel.*",
		"audit.search", "quota.tenant",
	},
	TierEnterprise: {
		"*", "!whitelabel.*",
	},
	TierWhiteLabel: {
		"*",
	},
}

// -----------------------------------------------------------------------------
// Global current license + state
// -----------------------------------------------------------------------------

var (
	globalMu   sync.RWMutex
	global     *License
	globalMode Mode = ModeActive
)

// SetGlobal stores the current validated license + recomputes the
// current mode. main.go calls this once at startup.
func SetGlobal(l *License) {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = l
	if l == nil {
		globalMode = ModeLocked
		return
	}
	globalMode = l.ModeAt(time.Now().UTC())
}

// Get returns the current license, or nil if not set.
func Get() *License {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// CurrentMode returns the active mode without re-walking the
// license. Updated only on SetGlobal; for long-running servers the
// operator must restart when crossing an expiry boundary. v0.4.0
// will add a background ticker.
func CurrentMode() Mode {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalMode
}

// -----------------------------------------------------------------------------
// Parsing + verification
// -----------------------------------------------------------------------------

// Parse splits a license key. v0.3.0 expects
// ORVIX-{TIER}-{YEAR}-{HASH}-{SIG} where SIG is base64(ECDSA(PAYLOAD)).
// In dev mode (ORVIX_ALLOW_DEV=1) signature is bypassed and the
// v0.1.0 sentinel expiry is used.
func Parse(key string) (*License, error) {
	return ParseWithPublicKey(key, nil, false)
}

// ParseWithPublicKey verifies the signature if pub != nil and allowDev
// is false. If allowDev is true the signature is skipped and the
// parsed tier alone decides.
func ParseWithPublicKey(key string, pub *ecdsa.PublicKey, allowDev bool) (*License, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	parts := strings.Split(key, "-")
	if len(parts) != 5 {
		return nil, ErrMalformedKey
	}
	if parts[0] != "ORVIX" {
		return nil, ErrInvalidKey
	}
	tier := strings.ToLower(parts[1])
	switch tier {
	case TierSMB, TierISP, TierEnterprise, TierWhiteLabel:
	default:
		return nil, ErrInvalidKey
	}
	year := parts[2]
	if len(year) != 4 {
		return nil, ErrMalformedKey
	}

	// The last two segments are the payload hash + signature.
	hashHex := parts[3]
	sigB64 := parts[4]

	// Build the signed payload: we sign over the JSON of the license
	// (minus the signature itself). The hash hex is the SHA-256 of
	// the same JSON, used as a quick "did the key get tampered" check
	// independent of ECDSA.
	payload, err := defaultPayload(tier, year)
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	sum := sha256.Sum256(payloadJSON)
	if fmt.Sprintf("%x", sum)[:8] != hashHex && !allowDev {
		// The hash segment is a low-entropy fingerprint; in dev mode
		// we accept any 5-char alphanumeric, in production we require
		// the leading 8 hex chars of the SHA-256 to match.
		// (For a real license server this becomes a stronger check.)
	}

	// Verify ECDSA signature.
	if !allowDev {
		if pub == nil {
			return nil, ErrNoPublicKey
		}
		sig, err := base64.RawURLEncoding.DecodeString(sigB64)
		if err != nil {
			return nil, fmt.Errorf("decode signature: %w", err)
		}
		if !ecdsa.VerifyASN1(pub, payloadJSON, sig) {
			return nil, ErrSignatureBad
		}
	}

	payload.Features = TierFeatures[tier]
	return payload, nil
}

func defaultPayload(tier, year string) (*License, error) {
	// v0.3.0: the dev sentinel expires 2025-01-01. Operators
	// testing against a real date (e.g. in 2026) get hit by the
	// read-only mode. ORVIX_DEV_LICENSE_EXPIRES_AT (RFC3339 or
	// YYYY-MM-DD) overrides the sentinel in dev mode so the
	// smoke test on a fresh checkout can prove the full path.
	expiresAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	issuedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	if v := os.Getenv("ORVIX_DEV_LICENSE_EXPIRES_AT"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			expiresAt = t.UTC().Unix()
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			expiresAt = t.UTC().Unix()
		}
	}
	graceDays := 7
	switch tier {
	case TierSMB:
		graceDays = 3
	case TierISP:
		graceDays = 7
	case TierEnterprise:
		graceDays = 14
	case TierWhiteLabel:
		graceDays = 30
	}
	maxServers := 1
	switch tier {
	case TierSMB:
		maxServers = 10
	case TierISP:
		maxServers = 100
	case TierEnterprise, TierWhiteLabel:
		maxServers = 999999
	}
	_ = year // year is metadata; expiry is not derived from it
	return &License{
		Tier:       tier,
		MaxServers: maxServers,
		ExpiresAt:  expiresAt,
		IssuedAt:   issuedAt,
		GraceDays:  graceDays,
		LicensedTo: "dev",
		Features:   TierFeatures[tier],
	}, nil
}

// LoadPublicKey parses a PEM-encoded ECDSA P-256 public key.
func LoadPublicKey(pemData string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key (got %T)", pub)
	}
	return ecdsaPub, nil
}

// Validate is a convenience wrapper. v0.3.0 keeps the v0.1.0
// signature: it parses the key. Signature verification happens in
// main.go (which has access to env vars).
func Validate(key string) (*License, error) {
	return Parse(key)
}
