package models

import (
	"time"
)

// MailDomain represents a mail domain with DNS configuration
type MailDomain struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID     string    `json:"tenant_id" gorm:"size:36;index;not null"`
	AccountID    string    `json:"account_id,omitempty" gorm:"size:36;index"`
	Domain       string    `json:"domain" gorm:"size:255;uniqueIndex;not null"`
	DKIMSelector string    `json:"dkim_selector" gorm:"size:63;default:default"`
	DKIMPrivate  string    `json:"-" gorm:"column:dkim_private;type:text"`     // Encrypted at rest
	DKIMPublic   string    `json:"dkim_public" gorm:"column:dkim_public;type:text"`
	SPFRecord    string    `json:"spf_record" gorm:"size:255"`
	DMARCPolicy  string    `json:"dmarc_policy" gorm:"size:50;default:v=none"`
	IsCatchAll   bool      `json:"is_catch_all" gorm:"default:false"`
	MaxMailboxes int       `json:"max_mailboxes" gorm:"default:100"`
	Status       string    `json:"status" gorm:"size:20;default:active"` // active, suspended, deleted
	CreatedBy    string    `json:"created_by" gorm:"size:36"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Mailbox represents a user mailbox
type Mailbox struct {
	ID           string     `json:"id" gorm:"primaryKey;size:36"`
	TenantID     string     `json:"tenant_id" gorm:"size:36;index;not null"`
	AccountID    string     `json:"account_id,omitempty" gorm:"size:36;index"`
	MailDomainID string     `json:"mail_domain_id" gorm:"size:36;index;not null"`
	LocalPart    string     `json:"local_part" gorm:"size:64;not null"` // Before @
	Email        string     `json:"email" gorm:"size:255;uniqueIndex;not null"`
	Password     string     `json:"-" gorm:"column:password;type:varchar(255)"` // Dovecot bcrypt
	QuotaMB      int        `json:"quota_mb" gorm:"default:1024"`
	QuotaUsedMB  int        `json:"quota_used_mb" gorm:"default:0"`
	EnableIMAP   bool       `json:"enable_imap" gorm:"default:true"`
	EnablePOP3   bool       `json:"enable_pop3" gorm:"default:false"`
	EnableSMTP   bool       `json:"enable_smtp" gorm:"default:true"`
	Status       string     `json:"status" gorm:"size:20;default:active"` // active, suspended, deleted
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedBy    string     `json:"created_by" gorm:"size:36"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MailAlias represents email aliases
type MailAlias struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID     string    `json:"tenant_id" gorm:"size:36;index;not null"`
	AccountID    string    `json:"account_id,omitempty" gorm:"size:36;index"`
	MailDomainID string    `json:"mail_domain_id" gorm:"size:36;index;not null"`
	SourceEmail  string    `json:"source_email" gorm:"size:255;uniqueIndex;not null"`
	Destinations string    `json:"destinations" gorm:"type:text"` // JSON: ["dest1@example.com", "dest2@example.com"]
	IsCatchAll   bool      `json:"is_catch_all" gorm:"default:false"`
	Status       string    `json:"status" gorm:"size:20;default:active"`
	CreatedBy    string    `json:"created_by" gorm:"size:36"`
	CreatedAt    time.Time `json:"created_at"`
}

// MailForwarder represents email forwarders
type MailForwarder struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID     string    `json:"tenant_id" gorm:"size:36;index;not null"`
	AccountID    string    `json:"account_id,omitempty" gorm:"size:36;index"`
	MailDomainID string    `json:"mail_domain_id" gorm:"size:36;index;not null"`
	SourceEmail  string    `json:"source_email" gorm:"size:255;uniqueIndex;not null"`
	Destinations string    `json:"destinations" gorm:"type:text"` // JSON array
	KeepCopy     bool      `json:"keep_copy" gorm:"default:true"`
	Status       string    `json:"status" gorm:"size:20;default:active"`
	CreatedBy    string    `json:"created_by" gorm:"size:36"`
	CreatedAt    time.Time `json:"created_at"`
}

// MailRateLimit for anti-abuse controls
type MailRateLimit struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID      string    `json:"tenant_id" gorm:"size:36;index;not null"`
	AccountID     string    `json:"account_id,omitempty" gorm:"size:36;index"`
	MailboxID     string    `json:"mailbox_id,omitempty" gorm:"size:36;index"`
	RateType      string    `json:"rate_type" gorm:"size:20;not null"` // outbound, inbound, relay
	MaxMessages   int       `json:"max_messages" gorm:"default:100"`
	WindowMinutes int       `json:"window_minutes" gorm:"default:60"`
	MaxSizeMB     int       `json:"max_size_mb" gorm:"default:50"`
	Status        string    `json:"status" gorm:"size:20;default:active"`
	CreatedAt     time.Time `json:"created_at"`
}

// MailAuditLog for compliance and debugging
type MailAuditLog struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID  string    `json:"tenant_id" gorm:"size:36;index;not null"`
	MailboxID string    `json:"mailbox_id,omitempty" gorm:"size:36;index"`
	Action    string    `json:"action" gorm:"size:50;not null"` // sent, received, login, failed_login, quota_exceeded
	Direction string    `json:"direction" gorm:"size:20"`        // inbound, outbound
	FromEmail string    `json:"from_email" gorm:"size:255"`
	ToEmail   string    `json:"to_email" gorm:"size:255"`
	Subject   string    `json:"subject,omitempty" gorm:"size:500"`
	MessageID string    `json:"message_id,omitempty" gorm:"size:255"`
	SizeBytes int64     `json:"size_bytes"`
	Status    string    `json:"status" gorm:"size:20"` // delivered, bounced, rejected, deferred
	ErrorCode string    `json:"error_code,omitempty" gorm:"size:20"`
	RemoteIP  string    `json:"remote_ip,omitempty" gorm:"size:45"`
	UserAgent string    `json:"user_agent,omitempty" gorm:"size:255"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// MigrateMailModels runs AutoMigrate for mail models
func MigrateMailModels(db interface{}) error {
	return nil // Placeholder for GORM migration
}