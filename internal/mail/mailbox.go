package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MailboxManager handles mailbox operations
type MailboxManager struct {
	db *gorm.DB
}

// NewMailboxManager creates a new mailbox manager
func NewMailboxManager(db *gorm.DB) *MailboxManager {
	return &MailboxManager{db: db}
}

// CreateMailbox creates a new mailbox
func (m *MailboxManager) CreateMailbox(ctx context.Context, mailbox *models.Mailbox) error {
	// Validate email
	if err := validateEmail(mailbox.Email); err != nil {
		return NewMailError("CreateMailbox", err, mailbox.Email)
	}

	// Check if mailbox already exists
	var existing models.Mailbox
	if err := m.db.Where("email = ?", mailbox.Email).First(&existing).Error; err == nil {
		return ErrMailboxExists
	}

	// Parse email to get local part and domain
	parts := strings.Split(mailbox.Email, "@")
	if len(parts) != 2 {
		return NewMailError("CreateMailbox", ErrInvalidEmail, mailbox.Email)
	}

	// Set defaults
	if mailbox.ID == "" {
		mailbox.ID = generateID("mb")
	}
	if mailbox.Status == "" {
		mailbox.Status = "active"
	}
	if mailbox.QuotaMB == 0 {
		mailbox.QuotaMB = 1024 // Default 1GB
	}
	mailbox.LocalPart = parts[0]

	// Hash password if provided
	if mailbox.Password != "" {
		hash, err := HashPassword(mailbox.Password)
		if err != nil {
			return NewMailError("CreateMailbox", err, "password hashing")
		}
		mailbox.Password = hash
	}

	// Save mailbox
	if err := m.db.Create(mailbox).Error; err != nil {
		return NewMailError("CreateMailbox", err, "database error")
	}

	return nil
}

// GetMailbox retrieves a mailbox by ID
func (m *MailboxManager) GetMailbox(ctx context.Context, tenantID, mailboxID string) (*models.Mailbox, error) {
	var mailbox models.Mailbox
	if err := m.db.Where("id = ? AND tenant_id = ?", mailboxID, tenantID).First(&mailbox).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMailboxNotFound
		}
		return nil, NewMailError("GetMailbox", err, mailboxID)
	}
	return &mailbox, nil
}

// GetMailboxByEmail retrieves a mailbox by email
func (m *MailboxManager) GetMailboxByEmail(ctx context.Context, email string) (*models.Mailbox, error) {
	var mailbox models.Mailbox
	if err := m.db.Where("email = ?", email).First(&mailbox).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMailboxNotFound
		}
		return nil, NewMailError("GetMailboxByEmail", err, email)
	}
	return &mailbox, nil
}

// ListMailboxes retrieves all mailboxes for a tenant
func (m *MailboxManager) ListMailboxes(ctx context.Context, tenantID string, domainID string, page, pageSize int) ([]models.Mailbox, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var mailboxes []models.Mailbox
	var total int64

	query := m.db.Model(&models.Mailbox{}).Where("tenant_id = ?", tenantID)

	// Filter by domain if specified
	if domainID != "" {
		query = query.Where("mail_domain_id = ?", domainID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, NewMailError("ListMailboxes", err, "count error")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&mailboxes).Error; err != nil {
		return nil, 0, NewMailError("ListMailboxes", err, "query error")
	}

	return mailboxes, total, nil
}

// UpdateMailbox updates a mailbox
func (m *MailboxManager) UpdateMailbox(ctx context.Context, mailbox *models.Mailbox) error {
	// Check if mailbox exists
	existing, err := m.GetMailbox(ctx, mailbox.TenantID, mailbox.ID)
	if err != nil {
		return err
	}

	// Update fields
	existing.EnableIMAP = mailbox.EnableIMAP
	existing.EnablePOP3 = mailbox.EnablePOP3
	existing.EnableSMTP = mailbox.EnableSMTP
	existing.QuotaMB = mailbox.QuotaMB
	existing.Status = mailbox.Status

	if err := m.db.Save(existing).Error; err != nil {
		return NewMailError("UpdateMailbox", err, mailbox.ID)
	}

	return nil
}

// DeleteMailbox deletes a mailbox
func (m *MailboxManager) DeleteMailbox(ctx context.Context, tenantID, mailboxID string) error {
	mailbox, err := m.GetMailbox(ctx, tenantID, mailboxID)
	if err != nil {
		return err
	}

	// Delete associated aliases and forwarders
	m.db.Where("source_email LIKE ?", mailbox.Email+"%").Delete(&models.MailAlias{})
	m.db.Where("source_email = ?", mailbox.Email).Delete(&models.MailForwarder{})

	// Delete mailbox
	if err := m.db.Delete(mailbox).Error; err != nil {
		return NewMailError("DeleteMailbox", err, mailboxID)
	}

	return nil
}

// ChangePassword changes a mailbox password
func (m *MailboxManager) ChangePassword(ctx context.Context, tenantID, mailboxID, newPassword string) error {
	mailbox, err := m.GetMailbox(ctx, tenantID, mailboxID)
	if err != nil {
		return err
	}

	if mailbox.Status == "suspended" {
		return ErrMailboxSuspended
	}

	// Hash new password
	hash, err := HashPassword(newPassword)
	if err != nil {
		return NewMailError("ChangePassword", err, "password hashing")
	}

	mailbox.Password = hash

	if err := m.db.Save(mailbox).Error; err != nil {
		return NewMailError("ChangePassword", err, "database error")
	}

	return nil
}

// SuspendMailbox suspends a mailbox
func (m *MailboxManager) SuspendMailbox(ctx context.Context, tenantID, mailboxID string) error {
	mailbox, err := m.GetMailbox(ctx, tenantID, mailboxID)
	if err != nil {
		return err
	}

	mailbox.Status = "suspended"

	if err := m.db.Save(mailbox).Error; err != nil {
		return NewMailError("SuspendMailbox", err, mailboxID)
	}

	return nil
}

// ReactivateMailbox reactivates a suspended mailbox
func (m *MailboxManager) ReactivateMailbox(ctx context.Context, tenantID, mailboxID string) error {
	mailbox, err := m.GetMailbox(ctx, tenantID, mailboxID)
	if err != nil {
		return err
	}

	mailbox.Status = "active"

	if err := m.db.Save(mailbox).Error; err != nil {
		return NewMailError("ReactivateMailbox", err, mailboxID)
	}

	return nil
}

// GetQuotaUsage returns quota usage for a mailbox
func (m *MailboxManager) GetQuotaUsage(ctx context.Context, mailboxID string) (map[string]interface{}, error) {
	mailbox, err := m.GetMailboxByEmail(ctx, "")
	if err != nil {
		// Try by ID
		mailbox, err = m.GetMailbox(ctx, "", mailboxID)
		if err != nil {
			return nil, err
		}
	}

	usedPercent := 0.0
	if mailbox.QuotaMB > 0 {
		usedPercent = float64(mailbox.QuotaUsedMB) / float64(mailbox.QuotaMB) * 100
	}

	return map[string]interface{}{
		"used_mb":      mailbox.QuotaUsedMB,
		"limit_mb":     mailbox.QuotaMB,
		"used_percent": usedPercent,
		"status":       "ok",
	}, nil
}

// UpdateQuotaUsage updates the quota usage for a mailbox
func (m *MailboxManager) UpdateQuotaUsage(ctx context.Context, mailboxID string, usedMB int) error {
	mailbox, err := m.GetMailbox(ctx, "", mailboxID)
	if err != nil {
		return err
	}

	mailbox.QuotaUsedMB = usedMB

	// Check if quota exceeded
	if usedMB > mailbox.QuotaMB {
		mailbox.Status = "suspended"
	}

	if err := m.db.Save(mailbox).Error; err != nil {
		return NewMailError("UpdateQuotaUsage", err, mailboxID)
	}

	return nil
}

// VerifyPassword verifies a mailbox password
func (m *MailboxManager) VerifyPassword(ctx context.Context, email, password string) (bool, error) {
	mailbox, err := m.GetMailboxByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	if mailbox.Status == "suspended" {
		return false, ErrMailboxSuspended
	}

	err = bcrypt.CompareHashAndPassword([]byte(mailbox.Password), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}

// HashPassword creates a bcrypt hash for a password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// validateEmail validates an email address
func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ErrInvalidEmail
	}

	localPart := parts[0]
	domain := parts[1]

	// Check for empty local part or domain
	if localPart == "" || domain == "" {
		return ErrInvalidEmail
	}

	if len(localPart) > 64 {
		return ErrInvalidEmail
	}

	// Validate domain part
	if err := validateDomain(domain); err != nil {
		return ErrInvalidEmail
	}

	return nil
}

// GetMailboxCount returns the number of mailboxes for a domain
func (m *MailboxManager) GetMailboxCount(ctx context.Context, domainID string) (int64, error) {
	var count int64
	if err := m.db.Model(&models.Mailbox{}).Where("mail_domain_id = ?", domainID).Count(&count).Error; err != nil {
		return 0, NewMailError("GetMailboxCount", err, domainID)
	}
	return count, nil
}

// CheckDomainMailboxLimit checks if domain has reached mailbox limit
func (m *MailboxManager) CheckDomainMailboxLimit(ctx context.Context, domainID string, maxMailboxes int) error {
	count, err := m.GetMailboxCount(ctx, domainID)
	if err != nil {
		return err
	}

	if count >= int64(maxMailboxes) {
		return fmt.Errorf("domain mailbox limit reached (%d/%d)", count, maxMailboxes)
	}

	return nil
}

// ParseDestinations parses a JSON string of destinations
func ParseDestinations(destinationsJSON string) ([]string, error) {
	if destinationsJSON == "" {
		return []string{}, nil
	}

	var destinations []string
	if err := json.Unmarshal([]byte(destinationsJSON), &destinations); err != nil {
		return nil, err
	}

	return destinations, nil
}

// SerializeDestinations serializes destinations to JSON
func SerializeDestinations(destinations []string) (string, error) {
	data, err := json.Marshal(destinations)
	if err != nil {
		return "", err
	}
	return string(data), nil
}