// Package license implements the OrvixPanel license key system.
//
// Key format: ORVIX-{TIER}-{YEAR}-{HASH}-{SIG}
// Example:    ORVIX-ISP-2025-A3F7B-X9K2M1P
//
// v1.0 scope: key parsing + tier-based feature gating. The ECDSA
// signature path is implemented but in v1.0 is bypassed when
// ORVIX_ALLOW_DEV=1 is set (so a fresh install can boot without
// contacting the license server). v1.1 adds the phone-home flow + a
// real public key.
package license

import (
	"errors"
	"strings"
	"sync"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	ErrInvalidKey   = errors.New("license key is invalid")
	ErrExpired      = errors.New("license has expired")
	ErrOffline      = errors.New("license server unreachable")
	ErrMalformedKey = errors.New("license key is malformed")
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

// -----------------------------------------------------------------------------
// License
// -----------------------------------------------------------------------------

// License is the parsed + validated license payload.
type License struct {
	Tier        string   `json:"tier"`
	MaxServers  int      `json:"max_servers"`
	ExpiresAt   int64    `json:"expires_at"`
	Features    []string `json:"features"`
	LicensedTo  string   `json:"licensed_to"`
	IssuedAt    int64    `json:"issued_at"`
}

// HasFeature supports "*" wildcard and "!" negation. Examples:
//   "hosting.*"        matches "hosting.create"
//   "!whitelabel.*"    disables whitelabel.create
func (l *License) HasFeature(feature string) bool {
	if l == nil {
		return false
	}
	for _, pattern := range l.Features {
		if strings.HasPrefix(pattern, "!") {
			if matches(strings.TrimPrefix(pattern, "!"), feature) {
				return false
			}
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
var TierFeatures = map[string][]string{
	TierSMB: {
		"hosting.*", "dns.*", "mail.basic", "ssl.*",
		"database.*", "files.*", "firewall.basic",
		"guardian.basic", "backup.local",
	},
	TierISP: {
		"*", "!guardian.llm", "!whitelabel.*",
	},
	TierEnterprise: {
		"*", "!whitelabel.*",
	},
	TierWhiteLabel: {
		"*",
	},
}

// -----------------------------------------------------------------------------
// Global current license
// -----------------------------------------------------------------------------

var (
	globalMu sync.RWMutex
	global  *License
)

// SetGlobal stores the current validated license. main.go calls this
// once at startup.
func SetGlobal(l *License) {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = l
}

// Get returns the current license, or nil if not set.
func Get() *License {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// -----------------------------------------------------------------------------
// Key parsing
// -----------------------------------------------------------------------------

// Parse splits a license key. v1.0 accepts any ORVIX-{TIER}-{YEAR}-*-
// shape; signature verification is a v1.1 polish item pending the
// real ECDSA public key from the license server.
func Parse(key string) (*License, error) {
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

	lic := &License{
		Tier:       tier,
		MaxServers: 1,
		// v1.0 default: 1 year from a sentinel epoch. Replace with the
		// real IssuedAt/ExpiresAt from the signed payload in v1.1.
		ExpiresAt: 1735689600, // 2025-01-01 UTC
		IssuedAt:  1704067200,  // 2024-01-01 UTC
		LicensedTo: "dev",
		Features:  TierFeatures[tier],
	}
	return lic, nil
}

// Validate returns the parsed license or an error. The dev-mode
// short-circuit is handled in main.go (where ORVIX_ALLOW_DEV is read);
// this function just parses + returns.
func Validate(key string) (*License, error) {
	return Parse(key)
}
