package mail

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// DomainManager handles mail domain operations
type DomainManager struct {
	db *gorm.DB
}

// NewDomainManager creates a new domain manager
func NewDomainManager(db *gorm.DB) *DomainManager {
	return &DomainManager{db: db}
}

// CreateDomain creates a new mail domain
func (m *DomainManager) CreateDomain(ctx context.Context, domain *models.MailDomain) error {
	// Validate domain name
	if err := validateDomain(domain.Domain); err != nil {
		return NewMailError("CreateDomain", err, domain.Domain)
	}

	// Check if domain already exists
	var existing models.MailDomain
	if err := m.db.Where("domain = ?", domain.Domain).First(&existing).Error; err == nil {
		return ErrDomainExists
	}

	// Set defaults
	if domain.ID == "" {
		domain.ID = generateID("md")
	}
	if domain.Status == "" {
		domain.Status = "active"
	}
	if domain.DKIMSelector == "" {
		domain.DKIMSelector = "default"
	}

	// Generate SPF record
	domain.SPFRecord = generateSPFRecord()

	// Generate DKIM keys
	if err := m.GenerateDKIM(ctx, domain.ID, domain.DKIMSelector); err != nil {
		return NewMailError("CreateDomain", err, "DKIM generation failed")
	}

	// Save domain
	if err := m.db.Create(domain).Error; err != nil {
		return NewMailError("CreateDomain", err, "database error")
	}

	return nil
}

// GetDomain retrieves a domain by ID
func (m *DomainManager) GetDomain(ctx context.Context, tenantID, domainID string) (*models.MailDomain, error) {
	var domain models.MailDomain
	if err := m.db.Where("id = ? AND tenant_id = ?", domainID, tenantID).First(&domain).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDomainNotFound
		}
		return nil, NewMailError("GetDomain", err, domainID)
	}
	return &domain, nil
}

// GetDomainByName retrieves a domain by name
func (m *DomainManager) GetDomainByName(ctx context.Context, tenantID, domainName string) (*models.MailDomain, error) {
	var domain models.MailDomain
	if err := m.db.Where("domain = ? AND tenant_id = ?", domainName, tenantID).First(&domain).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDomainNotFound
		}
		return nil, NewMailError("GetDomainByName", err, domainName)
	}
	return &domain, nil
}

// ListDomains retrieves all domains for a tenant
func (m *DomainManager) ListDomains(ctx context.Context, tenantID string, page, pageSize int) ([]models.MailDomain, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var domains []models.MailDomain
	var total int64

	// Count total
	if err := m.db.Model(&models.MailDomain{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, 0, NewMailError("ListDomains", err, "count error")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := m.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&domains).Error; err != nil {
		return nil, 0, NewMailError("ListDomains", err, "query error")
	}

	return domains, total, nil
}

// UpdateDomain updates a domain
func (m *DomainManager) UpdateDomain(ctx context.Context, domain *models.MailDomain) error {
	// Check if domain exists
	existing, err := m.GetDomain(ctx, domain.TenantID, domain.ID)
	if err != nil {
		return err
	}

	// Update fields
	existing.IsCatchAll = domain.IsCatchAll
	existing.MaxMailboxes = domain.MaxMailboxes
	existing.DMARCPolicy = domain.DMARCPolicy
	existing.Status = domain.Status

	if err := m.db.Save(existing).Error; err != nil {
		return NewMailError("UpdateDomain", err, domain.ID)
	}

	return nil
}

// DeleteDomain deletes a domain
func (m *DomainManager) DeleteDomain(ctx context.Context, tenantID, domainID string) error {
	domain, err := m.GetDomain(ctx, tenantID, domainID)
	if err != nil {
		return err
	}

	// Delete associated mailboxes, aliases, forwarders
	m.db.Where("mail_domain_id = ?", domainID).Delete(&models.Mailbox{})
	m.db.Where("mail_domain_id = ?", domainID).Delete(&models.MailAlias{})
	m.db.Where("mail_domain_id = ?", domainID).Delete(&models.MailForwarder{})

	// Delete domain
	if err := m.db.Delete(domain).Error; err != nil {
		return NewMailError("DeleteDomain", err, domainID)
	}

	return nil
}

// GenerateDKIM generates DKIM keys for a domain
func (m *DomainManager) GenerateDKIM(ctx context.Context, domainID, selector string) error {
	domain, err := m.GetDomain(ctx, "", domainID)
	if err != nil {
		return err
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ErrDKIMGeneration
	}

	// Encode private key
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Encode public key in DNS format
	publicKeyDER, _ := x509.MarshalPKIXPublicKey(privateKey.PublicKey)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	})

	// Store in domain
	domain.DKIMPrivate = string(privateKeyPEM)
	domain.DKIMPublic = string(publicKeyPEM)
	domain.DKIMSelector = selector

	if err := m.db.Save(domain).Error; err != nil {
		return NewMailError("GenerateDKIM", err, "database error")
	}

	return nil
}

// GetSPFRecord returns the SPF record for a domain
func (m *DomainManager) GetSPFRecord(domain string) string {
	return fmt.Sprintf("v=spf1 a mx ~all")
}

// GetDMARCRecord returns the DMARC record for a domain
func (m *DomainManager) GetDMARCRecord(domain string, policy string) string {
	if policy == "" {
		policy = "none"
	}
	return fmt.Sprintf("v=DMARC1; p=%s; rua=mailto:dmarc@%s", policy, domain)
}

// GetDNSRecords returns all DNS records for a domain
func (m *DomainManager) GetDNSRecords(ctx context.Context, tenantID, domainID string) (map[string]string, error) {
	domain, err := m.GetDomain(ctx, tenantID, domainID)
	if err != nil {
		return nil, err
	}

	records := map[string]string{
		"spf":  domain.SPFRecord,
		"dmarc": m.GetDMARCRecord(domain.Domain, domain.DMARCPolicy),
		"dkim": fmt.Sprintf("%s._domainkey.%s IN TXT ( \"%s\" )", domain.DKIMSelector, domain.Domain, domain.DKIMPublic),
	}

	return records, nil
}

// validateDomain validates a domain name
func validateDomain(domain string) error {
	if domain == "" {
		return ErrDomainInvalid
	}

	domain = strings.ToLower(domain)

	// Basic domain validation
	if len(domain) < 3 || len(domain) > 253 {
		return ErrDomainInvalid
	}

	// Check for valid characters and patterns
	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if part == "" {
			return ErrDomainInvalid // Empty part (from ..)
		}
		// Check for leading or trailing hyphens in each part
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return ErrDomainInvalid
		}
		// Check for valid characters
		for _, c := range part {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return ErrDomainInvalid
			}
		}
	}

	return nil
}

// generateSPFRecord generates a basic SPF record
func generateSPFRecord() string {
	return "v=spf1 a mx ~all"
}

// generateID generates a unique ID with prefix
func generateID(prefix string) string {
	timestamp := time.Now().UnixNano()
	random := make([]byte, 4)
	rand.Read(random)
	return fmt.Sprintf("%s_%d%x", prefix, timestamp, random)
}

// VerifyDomain verifies DNS records for a domain (stub - real implementation needs DNS lookup)
func (m *DomainManager) VerifyDomain(ctx context.Context, tenantID, domainID string) (map[string]bool, error) {
	domain, err := m.GetDomain(ctx, tenantID, domainID)
	if err != nil {
		return nil, err
	}

	// Stub verification - in production, would check actual DNS records
	result := map[string]bool{
		"spf":  domain.SPFRecord != "",
		"dkim": domain.DKIMPublic != "",
		"dmarc": domain.DMARCPolicy != "",
	}

	return result, nil
}