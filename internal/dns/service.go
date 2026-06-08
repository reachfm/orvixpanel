// Package dns provides DNS zone and record management with optional
// PowerDNS integration.
package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// Service provides DNS zone and record management.
type Service struct {
	db     *gorm.DB
	pdns   *PowerDNSClient
}

// NewService creates a new DNS service.
func NewService(db *gorm.DB) (*Service, error) {
	pdns, err := NewPowerDNSClient()
	if err != nil {
		return nil, fmt.Errorf("powerdns client: %w", err)
	}
	return &Service{
		db:   db,
		pdns: pdns,
	}, nil
}

// IsPowerDNSEnabled returns true if PowerDNS sync is configured.
func (s *Service) IsPowerDNSEnabled() bool {
	return s.pdns != nil && s.pdns.IsConfigured()
}

// -----------------------------------------------------------------------------
// Zone Management
// -----------------------------------------------------------------------------

// CreateZoneInput represents input for creating a DNS zone.
type CreateZoneInput struct {
	AccountID string
	TenantID  string
	Domain    string
	Type      string // native, master, slave
	Masters   []string
}

// CreateZone creates a new DNS zone.
func (s *Service) CreateZone(ctx context.Context, input CreateZoneInput) (*models.DNSZone, error) {
	if err := ValidateZoneDomain(input.Domain); err != nil {
		return nil, fmt.Errorf("validate domain: %w", err)
	}

	// Check if zone already exists
	var existing models.DNSZone
	if err := s.db.WithContext(ctx).
		Where("domain = ?", input.Domain).
		First(&existing).Error; err == nil {
		return nil, fmt.Errorf("zone already exists: %s", input.Domain)
	}

	zoneType := input.Type
	if zoneType == "" {
		zoneType = "native"
	}

	zone := &models.DNSZone{
		AccountID: input.AccountID,
		TenantID:  input.TenantID,
		Domain:    strings.ToLower(input.Domain),
		Type:      zoneType,
		Status:    "active",
	}

	if len(input.Masters) > 0 {
		mastersJSON, _ := json.Marshal(input.Masters)
		zone.Masters = string(mastersJSON)
	}

	// Use transaction for PowerDNS mode
	if s.IsPowerDNSEnabled() {
		tx := s.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return nil, fmt.Errorf("begin transaction: %w", tx.Error)
		}

		if err := tx.Create(zone).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("create zone: %w", err)
		}

		// Sync to PowerDNS
		if err := s.pdns.CreateZone(PowerDNSZone{
			Name: zone.Domain,
			Kind: zone.Type,
		}); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("powerdns sync failed, rolled back: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			// Try to clean up PowerDNS even if DB commit fails
			_ = s.pdns.DeleteZone(zone.Domain)
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		return zone, nil
	}

	// Local-only mode: no transaction needed
	if err := s.db.WithContext(ctx).Create(zone).Error; err != nil {
		return nil, fmt.Errorf("create zone: %w", err)
	}

	return zone, nil
}

// GetZone retrieves a zone by ID.
func (s *Service) GetZone(ctx context.Context, zoneID string) (*models.DNSZone, error) {
	var zone models.DNSZone
	if err := s.db.WithContext(ctx).Where("id = ?", zoneID).First(&zone).Error; err != nil {
		return nil, fmt.Errorf("get zone: %w", err)
	}
	return &zone, nil
}

// GetZoneByDomain retrieves a zone by domain name.
func (s *Service) GetZoneByDomain(ctx context.Context, domain string) (*models.DNSZone, error) {
	var zone models.DNSZone
	if err := s.db.WithContext(ctx).Where("domain = ?", domain).First(&zone).Error; err != nil {
		return nil, fmt.Errorf("get zone: %w", err)
	}
	return &zone, nil
}

// ListZones lists all zones for a tenant.
func (s *Service) ListZones(ctx context.Context, tenantID string) ([]models.DNSZone, error) {
	var zones []models.DNSZone
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&zones).Error; err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	return zones, nil
}

// ListZonesByAccount lists all zones for an account.
func (s *Service) ListZonesByAccount(ctx context.Context, accountID string) ([]models.DNSZone, error) {
	var zones []models.DNSZone
	if err := s.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&zones).Error; err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	return zones, nil
}

// UpdateZone updates a zone.
func (s *Service) UpdateZone(ctx context.Context, zoneID string, input map[string]interface{}) (*models.DNSZone, error) {
	var zone models.DNSZone
	if err := s.db.WithContext(ctx).Where("id = ?", zoneID).First(&zone).Error; err != nil {
		return nil, fmt.Errorf("get zone: %w", err)
	}

	// Apply updates
	if status, ok := input["status"].(string); ok {
		zone.Status = status
	}
	if soaRefresh, ok := input["soa_refresh"].(float64); ok {
		zone.SoaRefresh = int(soaRefresh)
	}
	if soaRetry, ok := input["soa_retry"].(float64); ok {
		zone.SoaRetry = int(soaRetry)
	}
	if soaExpire, ok := input["soa_expire"].(float64); ok {
		zone.SoaExpire = int(soaExpire)
	}
	if soaMinimum, ok := input["soa_minimum"].(float64); ok {
		zone.SoaMinimum = int(soaMinimum)
	}

	if err := s.db.WithContext(ctx).Save(&zone).Error; err != nil {
		return nil, fmt.Errorf("update zone: %w", err)
	}

	return &zone, nil
}

// DeleteZone deletes a zone and all its records.
func (s *Service) DeleteZone(ctx context.Context, zoneID string) error {
	var zone models.DNSZone
	if err := s.db.WithContext(ctx).Where("id = ?", zoneID).First(&zone).Error; err != nil {
		return fmt.Errorf("get zone: %w", err)
	}

	// Use transaction for PowerDNS mode
	if s.IsPowerDNSEnabled() {
		// First delete from PowerDNS
		if err := s.pdns.DeleteZone(zone.Domain); err != nil {
			return fmt.Errorf("powerdns delete failed: %w", err)
		}

		// Then delete from DB
		tx := s.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return fmt.Errorf("begin transaction: %w", tx.Error)
		}

		if err := tx.Where("zone_id = ?", zoneID).Delete(&models.DNSRecord{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("delete records: %w", err)
		}

		if err := tx.Delete(&zone).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("delete zone: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}

		return nil
	}

	// Local-only mode: no PowerDNS sync needed
	if err := s.db.WithContext(ctx).
		Where("zone_id = ?", zoneID).
		Delete(&models.DNSRecord{}).Error; err != nil {
		return fmt.Errorf("delete records: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&zone).Error; err != nil {
		return fmt.Errorf("delete zone: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Record Management
// -----------------------------------------------------------------------------

// CreateRecordInput represents input for creating a DNS record.
type CreateRecordInput struct {
	ZoneID   string
	Name     string
	Type     string
	Content  string
	TTL      int
	Priority int
	Disabled bool
}

// CreateRecord creates a new DNS record.
func (s *Service) CreateRecord(ctx context.Context, input CreateRecordInput) (*models.DNSRecord, error) {
	// Validate the record
	recordDef := RecordDefinition{
		Name:     input.Name,
		Type:     input.Type,
		Content:  input.Content,
		TTL:      input.TTL,
		Priority: input.Priority,
	}
	if err := ValidateRecord(recordDef); err != nil {
		return nil, fmt.Errorf("validate record: %w", err)
	}

	// Set default TTL if not specified
	ttl := input.TTL
	if ttl == 0 {
		ttl = 3600
	}

	record := &models.DNSRecord{
		ZoneID:   input.ZoneID,
		Name:     input.Name,
		Type:     strings.ToUpper(input.Type),
		Content:  input.Content,
		TTL:      ttl,
		Priority: input.Priority,
		Disabled: input.Disabled,
	}

	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, fmt.Errorf("create record: %w", err)
	}

	// Sync to PowerDNS if configured
	s.syncRecordToPowerDNS(ctx, input.ZoneID)

	return record, nil
}

// GetRecord retrieves a record by ID.
func (s *Service) GetRecord(ctx context.Context, recordID string) (*models.DNSRecord, error) {
	var record models.DNSRecord
	if err := s.db.WithContext(ctx).Where("id = ?", recordID).First(&record).Error; err != nil {
		return nil, fmt.Errorf("get record: %w", err)
	}
	return &record, nil
}

// ListRecords lists all records for a zone.
func (s *Service) ListRecords(ctx context.Context, zoneID string) ([]models.DNSRecord, error) {
	var records []models.DNSRecord
	if err := s.db.WithContext(ctx).
		Where("zone_id = ?", zoneID).
		Order("type, name").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	return records, nil
}

// UpdateRecord updates a DNS record.
func (s *Service) UpdateRecord(ctx context.Context, recordID string, input map[string]interface{}) (*models.DNSRecord, error) {
	var record models.DNSRecord
	if err := s.db.WithContext(ctx).Where("id = ?", recordID).First(&record).Error; err != nil {
		return nil, fmt.Errorf("get record: %w", err)
	}

	// Apply updates
	if name, ok := input["name"].(string); ok {
		record.Name = name
	}
	if recordType, ok := input["type"].(string); ok {
		record.Type = strings.ToUpper(recordType)
	}
	if content, ok := input["content"].(string); ok {
		record.Content = content
	}
	if ttl, ok := input["ttl"].(float64); ok {
		record.TTL = int(ttl)
	}
	if priority, ok := input["priority"].(float64); ok {
		record.Priority = int(priority)
	}
	if disabled, ok := input["disabled"].(bool); ok {
		record.Disabled = disabled
	}

	if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		return nil, fmt.Errorf("update record: %w", err)
	}

	// Sync to PowerDNS if configured
	s.syncRecordToPowerDNS(ctx, record.ZoneID)

	return &record, nil
}

// DeleteRecord deletes a DNS record.
func (s *Service) DeleteRecord(ctx context.Context, recordID string) error {
	var record models.DNSRecord
	if err := s.db.WithContext(ctx).Where("id = ?", recordID).First(&record).Error; err != nil {
		return fmt.Errorf("get record: %w", err)
	}

	zoneID := record.ZoneID

	if err := s.db.WithContext(ctx).Delete(&record).Error; err != nil {
		return fmt.Errorf("delete record: %w", err)
	}

	// Sync to PowerDNS if configured
	s.syncRecordToPowerDNS(ctx, zoneID)

	return nil
}

// syncRecordToPowerDNS syncs zone records to PowerDNS.
func (s *Service) syncRecordToPowerDNS(ctx context.Context, zoneID string) {
	if !s.IsPowerDNSEnabled() {
		return
	}

	var zone models.DNSZone
	if err := s.db.WithContext(ctx).Where("id = ?", zoneID).First(&zone).Error; err != nil {
		return
	}

	var records []models.DNSRecord
	if err := s.db.WithContext(ctx).Where("zone_id = ?", zoneID).Find(&records).Error; err != nil {
		return
	}

	pdnsRecords := make([]PowerDNSRecord, len(records))
	for i, r := range records {
		pdnsRecords[i] = PowerDNSRecord{
			Name:     r.Name,
			Type:     r.Type,
			Content:  r.Content,
			TTL:      r.TTL,
			Disabled: r.Disabled,
		}
	}

	if err := s.pdns.SyncZone(zone.Domain, pdnsRecords); err != nil {
		fmt.Printf("powerdns sync warning: %v\n", err)
	}
}

// -----------------------------------------------------------------------------
// Template Management
// -----------------------------------------------------------------------------

// CreateTemplateInput represents input for creating a zone template.
type CreateTemplateInput struct {
	TenantID    string
	Name        string
	Description string
	Records     []RecordDefinition
}

// CreateTemplate creates a new zone template.
func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput) (*models.DNSZoneTemplate, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("template name required")
	}

	recordsJSON, err := json.Marshal(input.Records)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}

	template := &models.DNSZoneTemplate{
		TenantID:    input.TenantID,
		Name:        input.Name,
		Description: input.Description,
		Records:     string(recordsJSON),
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}

	return template, nil
}

// ListTemplates lists all templates for a tenant.
func (s *Service) ListTemplates(ctx context.Context, tenantID string) ([]models.DNSZoneTemplate, error) {
	var templates []models.DNSZoneTemplate
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name").
		Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return templates, nil
}

// GetTemplate retrieves a template by ID.
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*models.DNSZoneTemplate, error) {
	var template models.DNSZoneTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", templateID).First(&template).Error; err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &template, nil
}

// DeleteTemplate deletes a zone template.
func (s *Service) DeleteTemplate(ctx context.Context, templateID string) error {
	var template models.DNSZoneTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", templateID).First(&template).Error; err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&template).Error; err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	return nil
}

// ApplyTemplate applies a template to a zone.
func (s *Service) ApplyTemplate(ctx context.Context, zoneID string, templateID string) error {
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}

	var records []RecordDefinition
	if err := json.Unmarshal([]byte(template.Records), &records); err != nil {
		return fmt.Errorf("parse template records: %w", err)
	}

	// Create records from template
	for _, rec := range records {
		_, err := s.CreateRecord(ctx, CreateRecordInput{
			ZoneID:   zoneID,
			Name:     rec.Name,
			Type:     rec.Type,
			Content:  rec.Content,
			TTL:      rec.TTL,
			Priority: rec.Priority,
		})
		if err != nil {
			return fmt.Errorf("create record from template: %w", err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// DNS Lookup (Local Storage)
// -----------------------------------------------------------------------------

// Lookup performs a local DNS lookup for a zone.
func (s *Service) Lookup(ctx context.Context, domain string) ([]models.DNSRecord, error) {
	// First find the zone
	var zone models.DNSZone
	if err := s.db.WithContext(ctx).
		Where("domain = ?", strings.ToLower(domain)).
		First(&zone).Error; err != nil {
		return nil, fmt.Errorf("zone not found: %s", domain)
	}

	// Get all records for this zone
	var records []models.DNSRecord
	if err := s.db.WithContext(ctx).
		Where("zone_id = ? AND disabled = ?", zone.ID, false).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("lookup records: %w", err)
	}

	return records, nil
}

// ValidateRecordInput validates a record without creating it.
func (s *Service) ValidateRecordInput(ctx context.Context, input RecordDefinition) error {
	return ValidateRecord(input)
}