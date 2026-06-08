// Package models holds the GORM data models.
//
// v0.1.0 scope: Phase 1 (auth) + minimal account/tenant/license.
// v0.3.0 scope: + API keys, custom RBAC roles, secrets vault, tenant
// quotas, encrypted license store. See ENTERPRISE_PLAN.md.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Base supplies ID + timestamps to every model. ID is a 26-char ULID
// generated at BeforeCreate.
type Base struct {
	ID        string         `gorm:"primarykey;type:varchar(26)" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate generates a ULID if the caller didn't supply one.
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = newID()
	}
	return nil
}

// Tenant — top-level isolation boundary.
type Tenant struct {
	Base
	Name   string `gorm:"not null" json:"name"`
	Slug   string `gorm:"uniqueIndex;not null" json:"slug"`
	Type   string `gorm:"not null;default:'admin'" json:"type"`
	Status string `gorm:"not null;default:'active'" json:"status"`
}

// User — authenticated identity.
type User struct {
	Base
	Email         string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string     `gorm:"not null" json:"-"`
	Role          string     `gorm:"not null;default:'account_owner';index" json:"role"`
	TenantID      string     `gorm:"index;not null" json:"tenant_id"`
	AccountID     string     `gorm:"index" json:"account_id,omitempty"`
	TOTPSecret    string     `json:"-"`
	TOTPEnabled   bool       `gorm:"default:false" json:"totp_enabled"`
	FailedLogins  int        `gorm:"default:0" json:"-"`
	LockedUntil   *time.Time `json:"locked_until,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP   string     `json:"last_login_ip,omitempty"`
	Status        string     `gorm:"default:'active'" json:"status"`
}

// UserSession — refresh-token tracking. The refresh token itself is
// never stored; only the SHA-256 hash.
type UserSession struct {
	Base
	UserID      string     `gorm:"index;not null" json:"user_id"`
	SessionID   string     `gorm:"uniqueIndex;not null" json:"session_id"`
	RefreshHash string     `gorm:"not null" json:"-"`
	UserAgent   string     `json:"user_agent"`
	IP          string     `json:"ip"`
	ExpiresAt   time.Time  `gorm:"index" json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

// IsActive returns true if the session hasn't expired or been revoked.
func (s *UserSession) IsActive() bool {
	if s.RevokedAt != nil {
		return false
	}
	return time.Now().UTC().Before(s.ExpiresAt)
}

// Account — a hosting account. Phase 1 stores the row only; Phase 2
// wires provisioning.
type Account struct {
	Base
	Username       string `gorm:"uniqueIndex;not null" json:"username"`
	Domain         string `gorm:"uniqueIndex;not null" json:"domain"`
	TenantID       string `gorm:"index;not null" json:"tenant_id"`
	Plan           string `gorm:"not null;default:'basic'" json:"plan"`
	DiskQuotaMB    int64  `gorm:"default:10240" json:"disk_quota_mb"`
	BandwidthGB    int    `gorm:"default:100" json:"bandwidth_gb"`
	BandwidthUsedGB int64 `gorm:"default:0" json:"bandwidth_used_gb"`
	DiskUsedMB     int64  `gorm:"default:0" json:"disk_used_mb"`
	Status         string `gorm:"default:'active'" json:"status"`
}

// AuditEntry — append-only log with SHA-256 hash chain. Each row's
// `hash` is sha256(prev_hash || row_content); tampering with any
// historical row invalidates every subsequent row.
type AuditEntry struct {
	Base
	Timestamp    time.Time `gorm:"index;not null" json:"timestamp"`
	UserID       string    `gorm:"index" json:"user_id,omitempty"`
	UserEmail    string    `json:"user_email,omitempty"`
	UserRole     string    `json:"user_role,omitempty"`
	ActorIP      string    `json:"actor_ip,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Action       string    `gorm:"not null;index" json:"action"`
	ResourceType string    `gorm:"index" json:"resource_type,omitempty"`
	ResourceID   string    `gorm:"index" json:"resource_id,omitempty"`
	ResourceName string    `json:"resource_name,omitempty"`
	Result       string    `gorm:"not null" json:"result"` // success | failure | denied
	DurationMS   int       `json:"duration_ms,omitempty"`
	Detail       string    `gorm:"type:text" json:"detail,omitempty"`
	PrevHash     string    `gorm:"type:varchar(64)" json:"prev_hash"`
	Hash         string    `gorm:"type:varchar(64);uniqueIndex" json:"hash"`
}

// -----------------------------------------------------------------------------
// v0.3.0 Enterprise Edition — additional models
// -----------------------------------------------------------------------------

// APIKey — long-lived credentials for automation. The full key
// (orx_live_<prefix>_<secret>) is never stored; only its SHA-256
// hash. The 8-char prefix is stored plain for operator lookup.
type APIKey struct {
	Base
	TenantID    string     `gorm:"index;not null" json:"tenant_id"`
	CreatedByID string     `gorm:"index;not null" json:"created_by_id"`
	Name        string     `gorm:"not null" json:"name"`
	KeyHash     string     `gorm:"uniqueIndex;not null" json:"-"`
	Prefix      string     `gorm:"index;not null;type:varchar(16)" json:"prefix"`
	Role        string     `gorm:"not null" json:"role"`
	Scopes      string     `gorm:"type:text" json:"scopes"` // JSON array of "resource.action"
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP  string     `json:"last_used_ip,omitempty"`
	RevokedAt   *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	RevokeReason string    `json:"revoke_reason,omitempty"`
}

// IsActive reports whether the key is usable right now.
func (k *APIKey) IsActive() bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// CustomRole — admin-defined RBAC role. Built-in 12 roles are NOT
// stored here; only admin-created custom roles are. Built-in role
// lookups go through the auth.RolePermissions map.
type CustomRole struct {
	Base
	TenantID   string `gorm:"index;not null" json:"tenant_id"`
	Name       string `gorm:"uniqueIndex:idx_custom_role_tenant_name;not null" json:"name"`
	Permissions string `gorm:"type:text" json:"-"` // JSON: [{"resource":"domain","actions":["read","create"]}, ...]
	IsBuiltin  bool   `gorm:"default:false" json:"is_builtin"`
	Description string `gorm:"type:text" json:"description,omitempty"`
}

// Secret — encrypted secret in the per-tenant vault. Ciphertext
// and nonce are stored as base64 strings for easy inspection during
// audit. Plaintext is never stored, never logged.
type Secret struct {
	Base
	TenantID    string     `gorm:"index:idx_secret_tenant_name,unique;not null" json:"tenant_id"`
	Name        string     `gorm:"index:idx_secret_tenant_name,unique;not null" json:"name"`
	Ciphertext  string     `gorm:"type:text;not null" json:"-"` // base64(nonce || aesgcm(payload))
	Version     int        `gorm:"default:1" json:"version"`
	CreatedByID string     `gorm:"index;not null" json:"created_by_id"`
	RotatedAt   *time.Time `json:"rotated_at,omitempty"`
	RotatedByID *string    `json:"rotated_by_id,omitempty"`
}

// TenantQuota — resource bounds per tenant. Enforced on every
// resource create; checked at handler level.
//
// v0.3.0: no `gorm:"default:..."` tags on the numeric fields —
// GORM's default-tag behavior treats the zero value as "use
// default", which would silently turn a `MaxAccounts: 0` quota
// (legitimate "deny all" intent) into the schema default. The
// quota service fills values explicitly via tierDefaults.
type TenantQuota struct {
	Base
	TenantID       string `gorm:"uniqueIndex;not null" json:"tenant_id"`
	MaxAccounts    int    `json:"max_accounts"`
	MaxUsers       int    `json:"max_users"`
	MaxDomains     int    `json:"max_domains"`
	MaxStorageMB   int64  `json:"max_storage_mb"`
	MaxBandwidthGB int64  `json:"max_bandwidth_gb"`
	MaxAPIKeys     int    `json:"max_api_keys"`
	MaxCustomRoles int    `json:"max_custom_roles"`
	MaxSecrets     int    `json:"max_secrets"`
}

// LicenseStore — single-row table holding the encrypted license
// blob. ORVIX_LICENSE_PUBKEY ECDSA verification is performed on
// decrypt; in dev mode (ORVIX_ALLOW_DEV=1) parsing only.
type LicenseStore struct {
	Base
	// ID is always "singleton" — we use a unique row marker.
	KeyID         string    `gorm:"uniqueIndex;not null;default:'singleton'" json:"key_id"`
	Ciphertext    string    `gorm:"type:text;not null" json:"-"` // base64(nonce || aesgcm(json_payload))
	ParsedTier    string    `gorm:"index" json:"parsed_tier"`
	ParsedExpiresAt int64   `json:"parsed_expires_at"`
	ParsedIssuedAt  int64   `json:"parsed_issued_at"`
	UploadedByID  string    `gorm:"index" json:"uploaded_by_id"`
	UploadedAt    time.Time `gorm:"not null" json:"uploaded_at"`
}

// -----------------------------------------------------------------------------
// v0.4.0 DNS Engine — DNS models
// -----------------------------------------------------------------------------

// DNSZone represents a DNS zone (domain namespace).
type DNSZone struct {
	Base
	AccountID   string `gorm:"index;not null" json:"account_id"`
	TenantID    string `gorm:"index;not null" json:"tenant_id"`
	Domain      string `gorm:"uniqueIndex;not null" json:"domain"`
	Type        string `gorm:"not null;default:'native'" json:"type"` // native, master, slave
	Masters     string `gorm:"type:text" json:"masters,omitempty"`    // JSON array for slave zones
	SoaRefresh  int    `gorm:"default:10800" json:"soa_refresh"`
	SoaRetry    int    `gorm:"default:7200" json:"soa_retry"`
	SoaExpire   int    `gorm:"default:604800" json:"soa_expire"`
	SoaMinimum  int    `gorm:"default:3600" json:"soa_minimum"`
	Status      string `gorm:"default:'active'" json:"status"` // active, suspended, pending
}

// DNSRecord represents a single DNS record within a zone.
type DNSRecord struct {
	Base
	ZoneID    string `gorm:"index;not null" json:"zone_id"`
	Name      string `gorm:"not null" json:"name"`         // e.g., "@" or "www"
	Type      string `gorm:"not null;index" json:"type"`   // A, AAAA, MX, TXT, CNAME, NS, SRV, CAA
	Content   string `gorm:"type:text;not null" json:"content"`
	TTL       int    `gorm:"default:3600" json:"ttl"`
	Priority  int    `gorm:"default:0" json:"priority"`    // for MX, SRV records
	Disabled  bool   `gorm:"default:false" json:"disabled"`
}

// DNSZoneTemplate represents a reusable zone template.
type DNSZoneTemplate struct {
	Base
	TenantID    string `gorm:"index;not null" json:"tenant_id"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Records     string `gorm:"type:text;not null" json:"records"` // JSON array of record definitions
}
