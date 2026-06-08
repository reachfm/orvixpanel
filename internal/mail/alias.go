package mail

import (
	"context"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// AliasManager handles alias and forwarder operations
type AliasManager struct {
	db *gorm.DB
}

// NewAliasManager creates a new alias manager
func NewAliasManager(db *gorm.DB) *AliasManager {
	return &AliasManager{db: db}
}

// CreateAlias creates a new alias
func (m *AliasManager) CreateAlias(ctx context.Context, alias *models.MailAlias) error {
	// Validate source email
	if err := validateEmail(alias.SourceEmail); err != nil {
		return NewMailError("CreateAlias", err, alias.SourceEmail)
	}

	// Check if alias already exists
	var existing models.MailAlias
	if err := m.db.Where("source_email = ?", alias.SourceEmail).First(&existing).Error; err == nil {
		return ErrAliasExists
	}

	// Set defaults
	if alias.ID == "" {
		alias.ID = generateID("al")
	}
	if alias.Status == "" {
		alias.Status = "active"
	}

	// Save alias
	if err := m.db.Create(alias).Error; err != nil {
		return NewMailError("CreateAlias", err, "database error")
	}

	return nil
}

// GetAlias retrieves an alias by ID
func (m *AliasManager) GetAlias(ctx context.Context, tenantID, aliasID string) (*models.MailAlias, error) {
	var alias models.MailAlias
	if err := m.db.Where("id = ? AND tenant_id = ?", aliasID, tenantID).First(&alias).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAliasNotFound
		}
		return nil, NewMailError("GetAlias", err, aliasID)
	}
	return &alias, nil
}

// ListAliases retrieves all aliases for a tenant
func (m *AliasManager) ListAliases(ctx context.Context, tenantID string, domainID string, page, pageSize int) ([]models.MailAlias, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var aliases []models.MailAlias
	var total int64

	query := m.db.Model(&models.MailAlias{}).Where("tenant_id = ?", tenantID)

	if domainID != "" {
		query = query.Where("mail_domain_id = ?", domainID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, NewMailError("ListAliases", err, "count error")
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&aliases).Error; err != nil {
		return nil, 0, NewMailError("ListAliases", err, "query error")
	}

	return aliases, total, nil
}

// DeleteAlias deletes an alias
func (m *AliasManager) DeleteAlias(ctx context.Context, tenantID, aliasID string) error {
	alias, err := m.GetAlias(ctx, tenantID, aliasID)
	if err != nil {
		return err
	}

	if err := m.db.Delete(alias).Error; err != nil {
		return NewMailError("DeleteAlias", err, aliasID)
	}

	return nil
}

// CreateForwarder creates a new forwarder
func (m *AliasManager) CreateForwarder(ctx context.Context, forwarder *models.MailForwarder) error {
	// Validate source email
	if err := validateEmail(forwarder.SourceEmail); err != nil {
		return NewMailError("CreateForwarder", err, forwarder.SourceEmail)
	}

	// Check if forwarder already exists
	var existing models.MailForwarder
	if err := m.db.Where("source_email = ?", forwarder.SourceEmail).First(&existing).Error; err == nil {
		return ErrForwarderExists
	}

	// Set defaults
	if forwarder.ID == "" {
		forwarder.ID = generateID("fw")
	}
	if forwarder.Status == "" {
		forwarder.Status = "active"
	}

	// Save forwarder
	if err := m.db.Create(forwarder).Error; err != nil {
		return NewMailError("CreateForwarder", err, "database error")
	}

	return nil
}

// GetForwarder retrieves a forwarder by ID
func (m *AliasManager) GetForwarder(ctx context.Context, tenantID, forwarderID string) (*models.MailForwarder, error) {
	var forwarder models.MailForwarder
	if err := m.db.Where("id = ? AND tenant_id = ?", forwarderID, tenantID).First(&forwarder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrForwarderNotFound
		}
		return nil, NewMailError("GetForwarder", err, forwarderID)
	}
	return &forwarder, nil
}

// ListForwarders retrieves all forwarders for a tenant
func (m *AliasManager) ListForwarders(ctx context.Context, tenantID string, domainID string, page, pageSize int) ([]models.MailForwarder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var forwarders []models.MailForwarder
	var total int64

	query := m.db.Model(&models.MailForwarder{}).Where("tenant_id = ?", tenantID)

	if domainID != "" {
		query = query.Where("mail_domain_id = ?", domainID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, NewMailError("ListForwarders", err, "count error")
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&forwarders).Error; err != nil {
		return nil, 0, NewMailError("ListForwarders", err, "query error")
	}

	return forwarders, total, nil
}

// DeleteForwarder deletes a forwarder
func (m *AliasManager) DeleteForwarder(ctx context.Context, tenantID, forwarderID string) error {
	forwarder, err := m.GetForwarder(ctx, tenantID, forwarderID)
	if err != nil {
		return err
	}

	if err := m.db.Delete(forwarder).Error; err != nil {
		return NewMailError("DeleteForwarder", err, forwarderID)
	}

	return nil
}

// GetAliasesForMailbox returns all aliases pointing to a specific mailbox
func (m *AliasManager) GetAliasesForMailbox(ctx context.Context, tenantID, mailboxEmail string) ([]models.MailAlias, error) {
	var aliases []models.MailAlias
	if err := m.db.Where("tenant_id = ? AND destinations LIKE ?", tenantID, "%"+mailboxEmail+"%").Find(&aliases).Error; err != nil {
		return nil, NewMailError("GetAliasesForMailbox", err, mailboxEmail)
	}
	return aliases, nil
}

// GetForwardersForMailbox returns all forwarders pointing to a specific mailbox
func (m *AliasManager) GetForwardersForMailbox(ctx context.Context, tenantID, mailboxEmail string) ([]models.MailForwarder, error) {
	var forwarders []models.MailForwarder
	if err := m.db.Where("tenant_id = ? AND destinations LIKE ?", tenantID, "%"+mailboxEmail+"%").Find(&forwarders).Error; err != nil {
		return nil, NewMailError("GetForwardersForMailbox", err, mailboxEmail)
	}
	return forwarders, nil
}