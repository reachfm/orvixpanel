// Package models holds the GORM data models for v1.0.
//
// Scope: Phase 1 (auth) + minimal account/tenant/license. Phases
// 2-8 (domains, DNS, mail, SSL, hosting, firewall, backups, etc.)
// are deferred to v1.1 — see RELEASE_NOTES.md.
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
