package dns

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Migrate DNS models
	if err := db.AutoMigrate(&models.DNSZone{}, &models.DNSRecord{}, &models.DNSZoneTemplate{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestService_NewService(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	// Test that PowerDNS is not enabled (env vars not set)
	if svc.IsPowerDNSEnabled() {
		t.Error("IsPowerDNSEnabled() = true, want false")
	}
}

func TestService_CreateZone(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Test creating a valid zone
	zone, err := svc.CreateZone(ctx, CreateZoneInput{
		AccountID: "acc-123",
		TenantID:  "tenant-456",
		Domain:    "example.com",
		Type:      "native",
	})
	if err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if zone.Domain != "example.com" {
		t.Errorf("zone.Domain = %s, want example.com", zone.Domain)
	}
	if zone.Status != "active" {
		t.Errorf("zone.Status = %s, want active", zone.Status)
	}

	// Test duplicate zone
	_, err = svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-456",
		Domain:   "example.com",
	})
	if err == nil {
		t.Error("CreateZone() expected error for duplicate zone")
	}

	// Test invalid domain
	_, err = svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-456",
		Domain:   "invalid domain!",
	})
	if err == nil {
		t.Error("CreateZone() expected error for invalid domain")
	}
}

func TestService_GetZone(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Create a zone first
	created, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-456",
		Domain:   "gettest.com",
	})

	// Test getting the zone
	zone, err := svc.GetZone(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetZone() error = %v", err)
	}
	if zone.Domain != "gettest.com" {
		t.Errorf("zone.Domain = %s, want gettest.com", zone.Domain)
	}

	// Test non-existent zone
	_, err = svc.GetZone(ctx, "non-existent-id")
	if err == nil {
		t.Error("GetZone() expected error for non-existent zone")
	}
}

func TestService_GetZoneByDomain(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-456",
		Domain:   "bydomain.com",
	})

	zone, err := svc.GetZoneByDomain(ctx, "bydomain.com")
	if err != nil {
		t.Fatalf("GetZoneByDomain() error = %v", err)
	}
	if zone.Domain != "bydomain.com" {
		t.Errorf("zone.Domain = %s, want bydomain.com", zone.Domain)
	}
}

func TestService_ListZones(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Create multiple zones
	svc.CreateZone(ctx, CreateZoneInput{TenantID: "tenant-1", Domain: "zone1.com"})
	svc.CreateZone(ctx, CreateZoneInput{TenantID: "tenant-1", Domain: "zone2.com"})
	svc.CreateZone(ctx, CreateZoneInput{TenantID: "tenant-2", Domain: "zone3.com"})

	// List zones for tenant-1
	zones, err := svc.ListZones(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
	if len(zones) != 2 {
		t.Errorf("len(zones) = %d, want 2", len(zones))
	}

	// List zones for tenant-2
	zones, err = svc.ListZones(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
	if len(zones) != 1 {
		t.Errorf("len(zones) = %d, want 1", len(zones))
	}
}

func TestService_UpdateZone(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "updatetest.com",
	})

	updated, err := svc.UpdateZone(ctx, zone.ID, map[string]interface{}{
		"status":     "suspended",
		"soa_refresh": 7200,
	})
	if err != nil {
		t.Fatalf("UpdateZone() error = %v", err)
	}
	if updated.Status != "suspended" {
		t.Errorf("updated.Status = %s, want suspended", updated.Status)
	}
}

func TestService_DeleteZone(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "deletetest.com",
	})

	// Add a record to the zone
	svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID:  zone.ID,
		Name:    "www",
		Type:    "A",
		Content: "192.0.2.1",
		TTL:     3600,
	})

	// Delete the zone
	err := svc.DeleteZone(ctx, zone.ID)
	if err != nil {
		t.Fatalf("DeleteZone() error = %v", err)
	}

	// Verify zone is gone
	_, err = svc.GetZone(ctx, zone.ID)
	if err == nil {
		t.Error("GetZone() expected error after deletion")
	}
}

func TestService_CreateRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "recordtest.com",
	})

	record, err := svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID:   zone.ID,
		Name:     "www",
		Type:     "A",
		Content:  "192.0.2.1",
		TTL:      3600,
		Priority: 0,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	if record.Type != "A" {
		t.Errorf("record.Type = %s, want A", record.Type)
	}
	if record.TTL != 3600 {
		t.Errorf("record.TTL = %d, want 3600", record.TTL)
	}

	// Test invalid record
	_, err = svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID:  zone.ID,
		Name:    "bad",
		Type:    "A",
		Content: "not-an-ip",
	})
	if err == nil {
		t.Error("CreateRecord() expected error for invalid record")
	}
}

func TestService_ListRecords(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "listrecords.com",
	})

	// Create a single record with valid TTL
	record, err := svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID: zone.ID, Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}

	records, err := svc.ListRecords(ctx, zone.ID)
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("len(records) = %d, want 1 (created 1 record)", len(records))
	}
	// Verify the record we created is in the list
	if len(records) > 0 && records[0].ID != record.ID {
		t.Errorf("records[0].ID = %s, want %s", records[0].ID, record.ID)
	}
}

func TestService_UpdateRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "updaterecord.com",
	})

	record, _ := svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID: zone.ID, Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})

	updated, err := svc.UpdateRecord(ctx, record.ID, map[string]interface{}{
		"content": "192.0.2.99",
		"ttl":     7200.0,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
	if updated.Content != "192.0.2.99" {
		t.Errorf("updated.Content = %s, want 192.0.2.99", updated.Content)
	}
	if updated.TTL != 7200 {
		t.Errorf("updated.TTL = %d, want 7200", updated.TTL)
	}
}

func TestService_DeleteRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "deleterecord.com",
	})

	record, _ := svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID: zone.ID, Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})

	err := svc.DeleteRecord(ctx, record.ID)
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}

	_, err = svc.GetRecord(ctx, record.ID)
	if err == nil {
		t.Error("GetRecord() expected error after deletion")
	}
}

func TestService_Templates(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Create template
	template, err := svc.CreateTemplate(ctx, CreateTemplateInput{
		TenantID:    "tenant-1",
		Name:        "Basic Template",
		Description: "Basic DNS setup",
		Records: []RecordDefinition{
			{Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600},
			{Name: "@", Type: "MX", Content: "10 mail", TTL: 3600, Priority: 10},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate() error = %v", err)
	}

	// List templates
	templates, err := svc.ListTemplates(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 1 {
		t.Errorf("len(templates) = %d, want 1", len(templates))
	}

	// Get template
	got, err := svc.GetTemplate(ctx, template.ID)
	if err != nil {
		t.Fatalf("GetTemplate() error = %v", err)
	}
	if got.Name != "Basic Template" {
		t.Errorf("got.Name = %s, want Basic Template", got.Name)
	}

	// Delete template
	err = svc.DeleteTemplate(ctx, template.ID)
	if err != nil {
		t.Fatalf("DeleteTemplate() error = %v", err)
	}
}

func TestService_ApplyTemplate(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Create template
	template, _ := svc.CreateTemplate(ctx, CreateTemplateInput{
		TenantID: "tenant-1",
		Name:     "Apply Test",
		Records: []RecordDefinition{
			{Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600},
		},
	})

	// Create zone
	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "applytest.com",
	})

	// Apply template
	err := svc.ApplyTemplate(ctx, zone.ID, template.ID)
	if err != nil {
		t.Fatalf("ApplyTemplate() error = %v", err)
	}

	// Verify record was created
	records, _ := svc.ListRecords(ctx, zone.ID)
	if len(records) != 1 {
		t.Errorf("len(records) = %d, want 1", len(records))
	}
}

func TestService_Lookup(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	zone, _ := svc.CreateZone(ctx, CreateZoneInput{
		TenantID: "tenant-1",
		Domain:   "lookuptest.com",
	})

	svc.CreateRecord(ctx, CreateRecordInput{
		ZoneID: zone.ID, Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})

	records, err := svc.Lookup(ctx, "lookuptest.com")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("len(records) = %d, want 1", len(records))
	}
}

func TestService_ValidateRecordInput(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc, _ := NewService(db)
	ctx := context.Background()

	// Valid record
	err := svc.ValidateRecordInput(ctx, RecordDefinition{
		Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600,
	})
	if err != nil {
		t.Errorf("ValidateRecordInput() valid record error = %v", err)
	}

	// Invalid record
	err = svc.ValidateRecordInput(ctx, RecordDefinition{
		Name: "www", Type: "A", Content: "not-an-ip", TTL: 3600,
	})
	if err == nil {
		t.Error("ValidateRecordInput() expected error for invalid record")
	}
}

func TestService_PowerDNSEnvOverride(t *testing.T) {
	// This test verifies behavior when PowerDNS env vars are not set
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Ensure env vars are not set
	os.Unsetenv("ORVIX_POWERDNS_URL")
	os.Unsetenv("ORVIX_POWERDNS_API_KEY")

	svc, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if svc.IsPowerDNSEnabled() {
		t.Error("IsPowerDNSEnabled() = true, want false when env vars not set")
	}
}