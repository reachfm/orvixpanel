package plans

import (
	"context"
	"testing"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.HostingPlan{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestGormStore_Create(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plan := &Plan{
		Name:         "starter",
		DisplayName:  "Starter Plan",
		Description:  "Basic hosting plan",
		DiskQuotaMB:  5120,
		BandwidthGB:  50,
		MaxDomains:   2,
		MaxUsers:     5,
		MaxSSL:       1,
		Features:     []string{"backup", "email"},
		IsActive:     true,
		IsDefault:    true,
		MonthlyPrice: 499,
	}

	err := store.Create(context.Background(), plan)
	if err != nil {
		t.Fatalf("Create() = %v, want nil", err)
	}
	if plan.ID == "" {
		t.Error("plan.ID should be set after Create")
	}

	// Verify can retrieve
	retrieved, err := store.GetByID(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("GetByID() = %v, want nil", err)
	}
	if retrieved.Name != "starter" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "starter")
	}
	if len(retrieved.Features) != 2 {
		t.Errorf("Features = %v, want 2 items", retrieved.Features)
	}
}

func TestGormStore_Create_InvalidName(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	// Test invalid name (spaces not allowed)
	plan := &Plan{Name: "bad name", DisplayName: "Test"}
	err := store.Create(context.Background(), plan)
	if err == nil {
		t.Error("Create() with space in name = nil, want error")
	}

	// Test empty name
	plan = &Plan{Name: "", DisplayName: "Test"}
	err = store.Create(context.Background(), plan)
	if err == nil {
		t.Error("Create() with empty name = nil, want error")
	}
}

func TestGormStore_GetByName(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plan := &Plan{Name: "pro", DisplayName: "Pro Plan", IsActive: true}
	if err := store.Create(context.Background(), plan); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	retrieved, err := store.GetByName(context.Background(), "pro")
	if err != nil {
		t.Fatalf("GetByName() = %v, want nil", err)
	}
	if retrieved.DisplayName != "Pro Plan" {
		t.Errorf("DisplayName = %q, want %q", retrieved.DisplayName, "Pro Plan")
	}
}

func TestGormStore_GetByName_NotFound(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	_, err := store.GetByName(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Errorf("GetByName() = %v, want ErrNotFound", err)
	}
}

func TestGormStore_List(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plans := []*Plan{
		{Name: "starter", DisplayName: "Starter"},
		{Name: "pro", DisplayName: "Pro"},
		{Name: "enterprise", DisplayName: "Enterprise"},
	}
	for _, p := range plans {
		if err := store.Create(context.Background(), p); err != nil {
			t.Fatalf("Create() = %v", err)
		}
	}

	all, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v, want nil", err)
	}
	if len(all) != 3 {
		t.Errorf("List() returned %d plans, want 3", len(all))
	}
}

func TestGormStore_Update(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plan := &Plan{Name: "starter", DisplayName: "Starter Plan", MonthlyPrice: 499}
	if err := store.Create(context.Background(), plan); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	// Update the plan
	plan.DisplayName = "Starter Plan Updated"
	plan.MonthlyPrice = 599
	plan.Features = []string{"backup", "staging", "email"}

	if err := store.Update(context.Background(), plan); err != nil {
		t.Fatalf("Update() = %v, want nil", err)
	}

	retrieved, err := store.GetByID(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("GetByID() = %v", err)
	}
	if retrieved.DisplayName != "Starter Plan Updated" {
		t.Errorf("DisplayName = %q, want %q", retrieved.DisplayName, "Starter Plan Updated")
	}
	if retrieved.MonthlyPrice != 599 {
		t.Errorf("MonthlyPrice = %v, want 599", retrieved.MonthlyPrice)
	}
	if len(retrieved.Features) != 3 {
		t.Errorf("Features = %v, want 3 items", retrieved.Features)
	}
}

func TestGormStore_Delete(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plan := &Plan{Name: "temp", DisplayName: "Temp Plan"}
	if err := store.Create(context.Background(), plan); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := store.Delete(context.Background(), plan.ID); err != nil {
		t.Fatalf("Delete() = %v, want nil", err)
	}

	_, err := store.GetByID(context.Background(), plan.ID)
	if err != ErrNotFound {
		t.Errorf("GetByID() after delete = %v, want ErrNotFound", err)
	}
}

func TestGormStore_GetActivePlans(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plans := []*Plan{
		{Name: "starter", DisplayName: "Starter", IsActive: true},
		{Name: "pro", DisplayName: "Pro", IsActive: true},
		{Name: "archived", DisplayName: "Archived", IsActive: false},
	}
	for _, p := range plans {
		if err := store.Create(context.Background(), p); err != nil {
			t.Fatalf("Create() = %v", err)
		}
	}

	active, err := store.GetActivePlans(context.Background())
	if err != nil {
		t.Fatalf("GetActivePlans() = %v, want nil", err)
	}
	if len(active) != 2 {
		t.Errorf("GetActivePlans() returned %d plans, want 2", len(active))
	}
}

func TestGormStore_GetDefaultPlan(t *testing.T) {
	db := setupDB(t)
	store := NewStore(db)

	plans := []*Plan{
		{Name: "starter", DisplayName: "Starter", IsDefault: true},
		{Name: "pro", DisplayName: "Pro"},
	}
	for _, p := range plans {
		if err := store.Create(context.Background(), p); err != nil {
			t.Fatalf("Create() = %v", err)
		}
	}

	defaultPlan, err := store.GetDefaultPlan(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultPlan() = %v, want nil", err)
	}
	if defaultPlan.Name != "starter" {
		t.Errorf("defaultPlan.Name = %q, want %q", defaultPlan.Name, "starter")
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"starter", "pro", "enterprise", "my-plan-123"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "bad name", "bad:name", "bad/name"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}