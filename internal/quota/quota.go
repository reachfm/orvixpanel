// Package quota implements per-tenant resource limits and the
// enforcement helpers used by the vault, accounts, and api-keys
// handlers.
//
// v0.3.0 Enterprise Edition.
package quota

import (
	"context"
	"errors"
	"fmt"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// Resource enumerates the things we cap.
type Resource string

const (
	ResourceAccount    Resource = "account"
	ResourceUser       Resource = "user"
	ResourceDomain     Resource = "domain"
	ResourceAPIKey     Resource = "apikey"
	ResourceCustomRole Resource = "custom_role"
	ResourceSecret     Resource = "secret"
)

// Defaults by license tier. Applied when a tenant has no row.
var tierDefaults = map[string]models.TenantQuota{
	"smb":        {MaxAccounts: 50, MaxUsers: 100, MaxDomains: 100, MaxStorageMB: 102400, MaxBandwidthGB: 1024, MaxAPIKeys: 10, MaxCustomRoles: 5, MaxSecrets: 50},
	"isp":        {MaxAccounts: 500, MaxUsers: 2000, MaxDomains: 2000, MaxStorageMB: 1048576, MaxBandwidthGB: 10240, MaxAPIKeys: 50, MaxCustomRoles: 20, MaxSecrets: 200},
	"enterprise": {MaxAccounts: 5000, MaxUsers: 20000, MaxDomains: 20000, MaxStorageMB: 10485760, MaxBandwidthGB: 102400, MaxAPIKeys: 200, MaxCustomRoles: 100, MaxSecrets: 1000},
	"whitelabel": {MaxAccounts: 99999, MaxUsers: 99999, MaxDomains: 99999, MaxStorageMB: 104857600, MaxBandwidthGB: 1024000, MaxAPIKeys: 1000, MaxCustomRoles: 500, MaxSecrets: 5000},
}

// Service is the quota entry point.
type Service struct {
	db *gorm.DB
}

// New constructs a Service.
func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Errors.
var (
	ErrQuotaExceeded   = errors.New("quota exceeded")
	ErrTenantNotFound  = errors.New("tenant not found")
)

// Get returns the effective quota for a tenant. Auto-creates a row
// using the tier default if none exists.
func (s *Service) Get(ctx context.Context, tenantID, tier string) (*models.TenantQuota, error) {
	if tenantID == "" {
		return nil, ErrTenantNotFound
	}
	var q models.TenantQuota
	err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&q).Error
	if err == nil {
		return &q, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get quota: %w", err)
	}
	// Auto-create with tier default.
	def, ok := tierDefaults[tier]
	if !ok {
		def = tierDefaults["smb"]
	}
	q = models.TenantQuota{TenantID: tenantID}
	q.MaxAccounts = def.MaxAccounts
	q.MaxUsers = def.MaxUsers
	q.MaxDomains = def.MaxDomains
	q.MaxStorageMB = def.MaxStorageMB
	q.MaxBandwidthGB = def.MaxBandwidthGB
	q.MaxAPIKeys = def.MaxAPIKeys
	q.MaxCustomRoles = def.MaxCustomRoles
	q.MaxSecrets = def.MaxSecrets
	if err := s.db.WithContext(ctx).Create(&q).Error; err != nil {
		return nil, fmt.Errorf("auto-create quota: %w", err)
	}
	return &q, nil
}

// Put replaces a tenant's quota row. Used by the admin endpoint.
func (s *Service) Put(ctx context.Context, q *models.TenantQuota) error {
	if q.TenantID == "" {
		return ErrTenantNotFound
	}
	var existing models.TenantQuota
	err := s.db.WithContext(ctx).Where("tenant_id = ?", q.TenantID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// GORM BeforeCreate hook will fill the ID.
		return s.db.WithContext(ctx).Create(q).Error
	}
	if err != nil {
		return err
	}
	existing.MaxAccounts = q.MaxAccounts
	existing.MaxUsers = q.MaxUsers
	existing.MaxDomains = q.MaxDomains
	existing.MaxStorageMB = q.MaxStorageMB
	existing.MaxBandwidthGB = q.MaxBandwidthGB
	existing.MaxAPIKeys = q.MaxAPIKeys
	existing.MaxCustomRoles = q.MaxCustomRoles
	existing.MaxSecrets = q.MaxSecrets
	return s.db.WithContext(ctx).Save(&existing).Error
}

// Check reports whether a tenant can create one more of `res`.
// Returns (ok, errorCode, error). errorCode is the stable string the
// API uses when 403'ing ("quota_accounts_exceeded" etc).
func (s *Service) Check(ctx context.Context, tenantID string, res Resource) (bool, string, error) {
	q, err := s.Get(ctx, tenantID, "smb") // tier is auto-defaulted; admin can override later
	if err != nil {
		return false, "quota_lookup_failed", err
	}
	var current int64
	switch res {
	case ResourceAccount:
		if err := s.db.WithContext(ctx).Model(&models.Account{}).
			Where("tenant_id = ?", tenantID).Count(&current).Error; err != nil {
			return false, "quota_count_failed", err
		}
		if int(current) >= q.MaxAccounts {
			return false, "quota_accounts_exceeded", nil
		}
	case ResourceUser:
		if err := s.db.WithContext(ctx).Model(&models.User{}).
			Where("tenant_id = ?", tenantID).Count(&current).Error; err != nil {
			return false, "quota_count_failed", err
		}
		if int(current) >= q.MaxUsers {
			return false, "quota_users_exceeded", nil
		}
	case ResourceAPIKey:
		if err := s.db.WithContext(ctx).Model(&models.APIKey{}).
			Where("tenant_id = ?", tenantID).Count(&current).Error; err != nil {
			return false, "quota_count_failed", err
		}
		if int(current) >= q.MaxAPIKeys {
			return false, "quota_apikeys_exceeded", nil
		}
	case ResourceCustomRole:
		if err := s.db.WithContext(ctx).Model(&models.CustomRole{}).
			Where("tenant_id = ?", tenantID).Count(&current).Error; err != nil {
			return false, "quota_count_failed", err
		}
		if int(current) >= q.MaxCustomRoles {
			return false, "quota_custom_roles_exceeded", nil
		}
	case ResourceSecret:
		if err := s.db.WithContext(ctx).Model(&models.Secret{}).
			Where("tenant_id = ?", tenantID).Count(&current).Error; err != nil {
			return false, "quota_count_failed", err
		}
		if int(current) >= q.MaxSecrets {
			return false, "quota_secrets_exceeded", nil
		}
	case ResourceDomain:
		// Domain model isn't in v0.3.0 yet (lands in v0.2.0). v0.3.0
		// declares the quota + the check shape; the actual count
		// query will be added when the domain model lands.
		// For now this path always passes — the quota value is
		// informational.
	}
	return true, "", nil
}
