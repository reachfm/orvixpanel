// Package plans implements the Hosting Plans CRUD.
//
// v0.7.2: Hosting Plans define provisioning templates (starter, pro,
// enterprise) that can be assigned to accounts for resource limits
// and feature flags.
package plans

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// Plan represents a hosting plan template.
type Plan struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`           // e.g., "starter", "pro", "enterprise"
	DisplayName  string    `json:"display_name"`    // e.g., "Starter Plan"
	Description  string    `json:"description"`     // Human-readable description
	DiskQuotaMB  int64     `json:"disk_quota_mb"`   // Disk space limit in MB
	BandwidthGB  int64     `json:"bandwidth_gb"`    // Monthly bandwidth limit in GB
	MaxDomains   int       `json:"max_domains"`     // Max domains per account
	MaxUsers     int       `json:"max_users"`        // Max users per account
	MaxSSL       int       `json:"max_ssl"`          // Max SSL certificates
	Features     []string   `json:"features"`        // Enabled features (backup, staging, etc.)
	IsActive     bool      `json:"is_active"`        // Whether plan is available
	IsDefault    bool      `json:"is_default"`       // Default plan for new accounts
	MonthlyPrice float64   `json:"monthly_price"`    // Price in cents (e.g., 999 = $9.99)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Store defines the persistence layer for plans.
type Store interface {
	Create(ctx context.Context, plan *Plan) error
	GetByID(ctx context.Context, id string) (*Plan, error)
	GetByName(ctx context.Context, name string) (*Plan, error)
	List(ctx context.Context) ([]*Plan, error)
	Update(ctx context.Context, plan *Plan) error
	Delete(ctx context.Context, id string) error
	GetActivePlans(ctx context.Context) ([]*Plan, error)
	GetDefaultPlan(ctx context.Context) (*Plan, error)
}

// GormStore implements Store using GORM.
type GormStore struct {
	db *gorm.DB
}

// NewStore creates a GormStore.
func NewStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// Create inserts a new plan.
func (s *GormStore) Create(ctx context.Context, plan *Plan) error {
	if plan.Name == "" {
		return errors.New("plan name is required")
	}
	if err := ValidateName(plan.Name); err != nil {
		return err
	}
	row := toModel(plan)
	plan.ID = models.NewID()
	row.ID = plan.ID
	return s.db.WithContext(ctx).Create(row).Error
}

// GetByID retrieves a plan by ID.
func (s *GormStore) GetByID(ctx context.Context, id string) (*Plan, error) {
	var row models.HostingPlan
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fromModel(&row), nil
}

// GetByName retrieves a plan by name.
func (s *GormStore) GetByName(ctx context.Context, name string) (*Plan, error) {
	var row models.HostingPlan
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fromModel(&row), nil
}

// List returns all plans.
func (s *GormStore) List(ctx context.Context) ([]*Plan, error) {
	var rows []models.HostingPlan
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	plans := make([]*Plan, len(rows))
	for i := range rows {
		plans[i] = fromModel(&rows[i])
	}
	return plans, nil
}

// Update modifies an existing plan.
func (s *GormStore) Update(ctx context.Context, plan *Plan) error {
	var row models.HostingPlan
	if err := s.db.WithContext(ctx).Where("id = ?", plan.ID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	row.Name = plan.Name
	row.DisplayName = plan.DisplayName
	row.Description = plan.Description
	row.DiskQuotaMB = plan.DiskQuotaMB
	row.BandwidthGB = plan.BandwidthGB
	row.MaxDomains = plan.MaxDomains
	row.MaxUsers = plan.MaxUsers
	row.MaxSSL = plan.MaxSSL
	// Serialize features to JSON
	if len(plan.Features) > 0 {
		b, _ := json.Marshal(plan.Features)
		row.FeaturesJSON = string(b)
	} else {
		row.FeaturesJSON = ""
	}
	row.IsActive = plan.IsActive
	row.IsDefault = plan.IsDefault
	row.MonthlyPrice = plan.MonthlyPrice
	return s.db.WithContext(ctx).Save(&row).Error
}

// Delete removes a plan (soft delete via GORM).
func (s *GormStore) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.HostingPlan{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetActivePlans returns only active plans.
func (s *GormStore) GetActivePlans(ctx context.Context) ([]*Plan, error) {
	var rows []models.HostingPlan
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("monthly_price ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	plans := make([]*Plan, len(rows))
	for i := range rows {
		plans[i] = fromModel(&rows[i])
	}
	return plans, nil
}

// GetDefaultPlan returns the default plan.
func (s *GormStore) GetDefaultPlan(ctx context.Context) (*Plan, error) {
	var row models.HostingPlan
	if err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return starter plan as fallback
			return s.GetByName(ctx, "starter")
		}
		return nil, err
	}
	return fromModel(&row), nil
}

// toModel converts a Plan to a GORM model.
func toModel(p *Plan) *models.HostingPlan {
	var featuresJSON string
	if len(p.Features) > 0 {
		b, _ := json.Marshal(p.Features)
		featuresJSON = string(b)
	}
	// Apply numeric defaults if zero
	diskQuotaMB := p.DiskQuotaMB
	if diskQuotaMB == 0 {
		diskQuotaMB = 10240
	}
	bandwidthGB := p.BandwidthGB
	if bandwidthGB == 0 {
		bandwidthGB = 100
	}
	maxDomains := p.MaxDomains
	if maxDomains == 0 {
		maxDomains = 5
	}
	maxUsers := p.MaxUsers
	if maxUsers == 0 {
		maxUsers = 10
	}
	maxSSL := p.MaxSSL
	if maxSSL == 0 {
		maxSSL = 3
	}
	return &models.HostingPlan{
		Base:         models.Base{ID: p.ID},
		Name:         p.Name,
		DisplayName:  p.DisplayName,
		Description: p.Description,
		DiskQuotaMB: diskQuotaMB,
		BandwidthGB: bandwidthGB,
		MaxDomains:  maxDomains,
		MaxUsers:    maxUsers,
		MaxSSL:      maxSSL,
		FeaturesJSON: featuresJSON,
		IsActive:    p.IsActive,
		IsDefault:   p.IsDefault,
		MonthlyPrice: p.MonthlyPrice,
	}
}

// fromModel converts a GORM model to a Plan.
func fromModel(m *models.HostingPlan) *Plan {
	if m == nil {
		return nil
	}
	var features []string
	if m.FeaturesJSON != "" {
		_ = json.Unmarshal([]byte(m.FeaturesJSON), &features)
	}
	return &Plan{
		ID:           m.ID,
		Name:         m.Name,
		DisplayName:  m.DisplayName,
		Description:  m.Description,
		DiskQuotaMB:  m.DiskQuotaMB,
		BandwidthGB:  m.BandwidthGB,
		MaxDomains:   m.MaxDomains,
		MaxUsers:     m.MaxUsers,
		MaxSSL:       m.MaxSSL,
		Features:     features,
		IsActive:     m.IsActive,
		IsDefault:    m.IsDefault,
		MonthlyPrice: m.MonthlyPrice,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// ErrNotFound indicates the requested plan doesn't exist.
var ErrNotFound = errors.New("plan not found")

// Validation errors.
var (
	ErrInvalidName     = errors.New("plan name must be alphanumeric with dashes")
	ErrDuplicateName   = errors.New("plan name already exists")
	ErrCannotDeleteDefault = errors.New("cannot delete the default plan")
)

// ValidateName checks plan name format.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("plan name is required")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return ErrInvalidName
		}
	}
	return nil
}