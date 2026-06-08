// Package models holds the GORM data models.
// v0.5.0 SSL Engine — Certificate management with Let's Encrypt.
package models

import (
	"time"
)

// -----------------------------------------------------------------------------
// v0.5.0 SSL Engine — SSL Certificate models
// -----------------------------------------------------------------------------

// Certificate status constants.
const (
	CertStatusPending      = "pending"
	CertStatusIssued       = "issued"
	CertStatusExpiringSoon  = "expiring_soon"
	CertStatusExpired       = "expired"
	CertStatusRevoked       = "revoked"
	CertStatusFailed        = "failed"
)

// Provider constants.
const (
	ProviderLetsEncrypt = "letsencrypt"
	ProviderZeroSSL     = "zerossl"
)

// SSLChallenge types.
const (
	ChallengeHTTP01 = "http-01"
	ChallengeDNS01  = "dns-01"
)

// Challenge status constants.
const (
	ChallengeStatusPending = "pending"
	ChallengeStatusValid   = "valid"
	ChallengeStatusInvalid = "invalid"
	ChallengeStatusRevoked = "revoked"
)

// ACMEAccount represents an ACME (Let's Encrypt/ZeroSSL) account for certificate issuance.
type ACMEAccount struct {
	Base
	TenantID   string `gorm:"index;not null" json:"tenant_id"`
	Email      string `gorm:"uniqueIndex:idx_acme_tenant_email;not null" json:"email"`
	Provider   string `gorm:"size:20;not null" json:"provider"` // letsencrypt, zerossl
	Status     string `gorm:"default:'active'" json:"status"`   // active, deactivated, revoked
	TermsAgreed bool `gorm:"default:false" json:"terms_agreed"`

	// ACME server account URL (returned after registration)
	AccountURL string `gorm:"size:500" json:"account_url,omitempty"`

	// Stored encrypted - never log these
	AccountKey string `gorm:"type:text" json:"-"` // Encrypted ACME account private key

	// Rate limiting info from ACME server
	RateLimits string `gorm:"type:text" json:"rate_limits,omitempty"` // JSON with remaining quotas

	// Metadata
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// IsActive returns true if the ACME account is active and usable.
func (a *ACMEAccount) IsActive() bool {
	return a.Status == "active" && a.TermsAgreed && a.AccountURL != ""
}

// SSLCertificate represents a managed SSL/TLS certificate.
// Certificates are stored as files; DB stores metadata and paths only.
type SSLCertificate struct {
	Base
	DomainID   string `gorm:"index" json:"domain_id,omitempty"`
	AccountID  string `gorm:"index" json:"account_id,omitempty"`
	TenantID   string `gorm:"index;not null" json:"tenant_id"`

	// Certificate metadata
	Provider    string `gorm:"size:20;default:'letsencrypt'" json:"provider"` // letsencrypt, zerossl
	CommonName  string `gorm:"index;size:253;not null" json:"common_name"`
	SANNames    string `gorm:"type:text" json:"san_names,omitempty"` // JSON array of Subject Alternative Names

	// Status
	Status    string `gorm:"size:20;index;default:'pending'" json:"status"` // pending, issued, expiring_soon, expired, revoked, failed
	AutoRenew bool   `gorm:"default:true" json:"auto_renew"`

	// ACME account reference
	ACMEAccountID string `gorm:"size:26" json:"acme_account_id,omitempty"`

	// Timestamps
	IssuedAt      *time.Time `json:"issued_at,omitempty"`
	ExpiresAt     *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastRenewalAt *time.Time `json:"last_renewal_at,omitempty"`

	// File paths (relative to SSL storage directory)
	// Certificates are stored as FILES, never as PEM content in DB
	CertPath      string `gorm:"size:500" json:"cert_path,omitempty"`       // e.g., /var/lib/orvixpanel/ssl/certs/example.com/cert.pem
	KeyPath       string `gorm:"size:500" json:"key_path,omitempty"`      // e.g., /var/lib/orvixpanel/ssl/certs/example.com/privkey.pem
	ChainPath     string `gorm:"size:500" json:"chain_path,omitempty"`     // e.g., /var/lib/orvixpanel/ssl/certs/example.com/chain.pem
	FullChainPath string `gorm:"size:500" json:"fullchain_path,omitempty"` // e.g., /var/lib/orvixpanel/ssl/certs/example.com/fullchain.pem

	// Certificate metadata (from parsed PEM)
	SerialNumber string `gorm:"size:100" json:"serial_number,omitempty"`
	Fingerprint  string `gorm:"size:100;uniqueIndex" json:"fingerprint,omitempty"`

	// Error tracking for failed issuance/renewal
	LastError string `gorm:"type:text" json:"last_error,omitempty"`

	// Renewal tracking
	RenewalAttempts int `gorm:"default:0" json:"renewal_attempts"`
	LastRenewalAttempt *time.Time `json:"last_renewal_attempt,omitempty"`
}

// IsExpired returns true if the certificate has expired.
func (c *SSLCertificate) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// DaysUntilExpiry returns the number of days until the certificate expires.
// Returns negative value if already expired.
func (c *SSLCertificate) DaysUntilExpiry() int {
	if c.ExpiresAt == nil {
		return -1
	}
	return int(time.Until(*c.ExpiresAt).Hours() / 24)
}

// NeedsRenewal returns true if the certificate should be renewed (30 days or less).
func (c *SSLCertificate) NeedsRenewal() bool {
	return c.DaysUntilExpiry() <= 30
}

// SSLEvent represents an immutable audit log entry for certificate operations.
type SSLEvent struct {
	Base
	CertificateID string `gorm:"index;not null" json:"certificate_id"`
	EventType     string `gorm:"size:50;index;not null" json:"event_type"` // issued, renewed, failed, revoked, deleted, challenge_requested, challenge_verified, nginx_updated, nginx_rollback

	Message string `gorm:"type:text" json:"message,omitempty"`

	// Detailed error information for failed events
	ErrorDetail string `gorm:"type:text" json:"error_detail,omitempty"`

	// Challenge tracking (for debugging ACME issues)
	ChallengeToken string `gorm:"size:255" json:"challenge_token,omitempty"`
	ChallengeURL   string `gorm:"size:500" json:"challenge_url,omitempty"`

	// Request metadata for debugging
	RequestID string `gorm:"size:100" json:"request_id,omitempty"`
	UserID    string `gorm:"index" json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

// SSLChallenge represents an ACME HTTP-01 or DNS-01 challenge.
// Used for domain verification during certificate issuance.
type SSLChallenge struct {
	Base
	CertificateID string `gorm:"index;not null" json:"certificate_id"`
	Domain        string `gorm:"size:253;not null" json:"domain"`
	Token         string `gorm:"size:255;uniqueIndex;not null" json:"token"`
	KeyAuth       string `gorm:"type:text;not null" json:"-"` // ACME key authorization - never log

	ChallengeType string `gorm:"size:20;not null" json:"challenge_type"` // http-01, dns-01

	Status string `gorm:"size:20;default:'pending'" json:"status"` // pending, valid, invalid, revoked

	// For HTTP-01: path where the challenge file should be placed
	FilePath string `gorm:"size:500" json:"file_path,omitempty"`

	// Verification timestamps
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`

	// ACME server response
	ValidatedAt   *time.Time `json:"validated_at,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
}

// IsValid returns true if the challenge was successfully validated.
func (c *SSLChallenge) IsValid() bool {
	return c.Status == ChallengeStatusValid && c.ValidatedAt != nil
}

// HasExpired returns true if the challenge has expired.
func (c *SSLChallenge) HasExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}