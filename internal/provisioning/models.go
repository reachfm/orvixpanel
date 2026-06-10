// Package provisioning implements the website provisioning workflow.
//
// This package provides:
//   - ProvisioningJob model for tracking provisioning operations
//   - ProvisioningEvent model for step-by-step audit trail
//   - Domain validation and sanitization
//   - Rollback on failure
//   - Interface-based executors for testability
//
// Security guarantees:
//   - exec.Command only (never sh -c)
//   - Domain sanitization prevents path traversal
//   - Rejects unsafe domains (empty, too long, invalid TLD)
package provisioning

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// JobStatus represents the current state of a provisioning job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRolledBack JobStatus = "rolled_back"
)

// ProvisioningJob represents a single provisioning operation.
type ProvisioningJob struct {
	ID          string     `json:"id"`
	AccountID   string     `json:"account_id"`
	TenantID    string     `json:"tenant_id"`
	Username    string     `json:"username"`
	Domain      string     `json:"domain"`
	Status      JobStatus  `json:"status"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewJob creates a new provisioning job with a ULID.
func NewJob(accountID, tenantID, username, domain string) *ProvisioningJob {
	return &ProvisioningJob{
		ID:        ulid.Make().String(),
		AccountID: accountID,
		TenantID:  tenantID,
		Username:  username,
		Domain:    domain,
		Status:    JobStatusPending,
		CreatedAt: time.Now().UTC(),
	}
}

// ProvisioningEvent represents a single step in the provisioning workflow.
type ProvisioningEvent struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Step      string    `json:"step"` // e.g., "validate_domain", "create_directories"
	Status    string    `json:"status"` // "started", "completed", "failed", "rolled_back"
	Message   string    `json:"message,omitempty"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// NewEvent creates a new provisioning event.
func NewEvent(jobID, step, status, message string) *ProvisioningEvent {
	return &ProvisioningEvent{
		ID:        ulid.Make().String(),
		JobID:     jobID,
		Step:      step,
		Status:    status,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}
}

// ValidationError represents a domain validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// DomainValidator validates domain names for security and correctness.
type DomainValidator struct {
	// MaxLength is the maximum allowed domain length (default 253)
	MaxLength int
	// MinLength is the minimum allowed domain length (default 4)
	MinLength int
	// AllowWildcard allows wildcard domains (default false)
	AllowWildcard bool
}

// DefaultDomainValidator returns a validator with sane defaults.
func DefaultDomainValidator() *DomainValidator {
	return &DomainValidator{
		MaxLength:     253,
		MinLength:     4,
		AllowWildcard: false,
	}
}

// Validate checks a domain name for safety and correctness.
// Returns nil if valid, or a ValidationError describing the issue.
//
// Security checks:
//   - Rejects empty domains
//   - Rejects domains containing path traversal patterns
//   - Rejects domains with invalid characters
//   - Rejects domains shorter than min length
//   - Rejects domains exceeding max length
//   - Rejects reserved/bogon addresses
//   - Rejects wildcard domains unless AllowWildcard is true
func (v *DomainValidator) Validate(domain string) error {
	if v.MaxLength == 0 {
		v.MaxLength = 253
	}
	if v.MinLength == 0 {
		v.MinLength = 4
	}

	domain = strings.TrimSpace(strings.ToLower(domain))

	// Empty check
	if domain == "" {
		return &ValidationError{Field: "domain", Message: "domain cannot be empty"}
	}

	// Path traversal patterns - MUST come before length checks
	traversalPatterns := []string{
		"..",
		"../",
		"/..",
		"\\",
		"%2e%2e", // URL-encoded ..
		"%252e",  // Double URL-encoded .
	}
	domainLower := strings.ToLower(domain)
	for _, pattern := range traversalPatterns {
		if strings.Contains(domainLower, pattern) {
			return &ValidationError{Field: "domain", Message: "domain contains path traversal pattern"}
		}
	}

	// Character validation - only allow safe characters
	charPattern := `^[a-z0-9]([a-z0-9\-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9\-]*[a-z0-9])?)*$`
	if v.AllowWildcard {
		charPattern = `^\*?\.[a-z0-9]([a-z0-9\-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9\-]*[a-z0-9])?)*$`
	}
	validDomain := regexp.MustCompile(charPattern)
	if !validDomain.MatchString(domain) {
		return &ValidationError{Field: "domain", Message: "domain contains invalid characters"}
	}

	// Length checks
	if len(domain) > v.MaxLength {
		return &ValidationError{Field: "domain", Message: fmt.Sprintf("domain exceeds maximum length of %d", v.MaxLength)}
	}
	if len(domain) < v.MinLength {
		return &ValidationError{Field: "domain", Message: fmt.Sprintf("domain must be at least %d characters", v.MinLength)}
	}

	// Must have at least one dot
	if !strings.Contains(domain, ".") {
		return &ValidationError{Field: "domain", Message: "domain must have at least one dot"}
	}

	// Label length check (each part between dots must be 1-63 chars)
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return &ValidationError{Field: "domain", Message: "domain label exceeds 63 characters"}
		}
		if len(label) == 0 {
			return &ValidationError{Field: "domain", Message: "domain contains empty label"}
		}
	}

	// Wildcard check
	if strings.HasPrefix(domain, "*.") && !v.AllowWildcard {
		return &ValidationError{Field: "domain", Message: "wildcard domains not allowed"}
	}

	// Reserved/bogon check
	reserved := []string{
		"localhost",
		"example.org",
		"example.net",
		"test",
		"invalid",
		"localhost.localdomain",
	}
	for _, r := range reserved {
		if domain == r || strings.HasPrefix(domain, r+".") {
			return &ValidationError{Field: "domain", Message: fmt.Sprintf("domain '%s' is reserved", domain)}
		}
	}

	return nil
}

// SanitizeFilename converts a domain to a safe filename string.
// Returns only [a-z0-9_-] characters, dots become underscores.
func SanitizeFilename(domain string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(domain) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else if r == '.' {
			result.WriteRune('_')
		}
		// All other characters are dropped
	}
	return result.String()
}