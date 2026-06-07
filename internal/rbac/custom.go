// Package rbac implements role-based access control.
//
// v0.1.0: 12 built-in roles hardcoded in api/middleware/rbac.go.
// v0.3.0: + custom admin-defined roles (CustomRole model). The
// HasPermission check first tries the built-in map; on miss it
// looks up the custom role by name in the DB and matches the
// permission against the role's stored JSON permission list.
package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// PermissionGrant is one row in a CustomRole.Permissions JSON blob.
type PermissionGrant struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

// Errors.
var (
	ErrInvalidRoleName   = errors.New("role name must be alphanumeric+dash, 1-64 chars")
	ErrBuiltinRoleClash  = errors.New("role name clashes with a built-in role")
	ErrRoleNotFound      = errors.New("role not found")
	ErrBuiltinImmutable  = errors.New("built-in roles are immutable")
)

// Service is the custom-role entry point.
type Service struct {
	db *gorm.DB
}

// New constructs a Service.
func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ValidateName enforces the role-name rules: 1-64 chars,
// alphanumeric + dash + underscore, no leading dash or
// underscore, no whitespace. Built-in role names are reserved
// and cannot be used for custom roles.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return ErrInvalidRoleName
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			// ok
		case r == '-' || r == '_':
			if i == 0 {
				return ErrInvalidRoleName
			}
		default:
			return ErrInvalidRoleName
		}
	}
	// Reserved built-in names.
	switch name {
	case "root_admin", "reseller_admin", "reseller_agent",
		"account_owner", "account_dev", "account_viewer",
		"mail_admin", "db_admin", "security_admin",
		"monitor", "support", "billing":
		return ErrBuiltinRoleClash
	}
	return nil
}

// Create adds a new custom role. Returns the persisted model.
func (s *Service) Create(ctx context.Context, tenantID, name, description string, perms []PermissionGrant) (*models.CustomRole, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if len(perms) == 0 {
		return nil, fmt.Errorf("%w: permissions cannot be empty", ErrInvalidRoleName)
	}
	for _, p := range perms {
		if strings.TrimSpace(p.Resource) == "" {
			return nil, fmt.Errorf("%w: resource cannot be empty", ErrInvalidRoleName)
		}
		if len(p.Actions) == 0 {
			return nil, fmt.Errorf("%w: actions cannot be empty for resource %q", ErrInvalidRoleName, p.Resource)
		}
	}
	blob, err := json.Marshal(perms)
	if err != nil {
		return nil, fmt.Errorf("marshal permissions: %w", err)
	}
	row := models.CustomRole{
		TenantID:     tenantID,
		Name:         name,
		Permissions:  string(blob),
		Description:  description,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create custom role: %w", err)
	}
	return &row, nil
}

// Update replaces the permissions + description of a custom role.
// Built-in roles are immutable and will return ErrBuiltinImmutable.
func (s *Service) Update(ctx context.Context, tenantID, name, description string, perms []PermissionGrant) error {
	row, err := s.Get(ctx, tenantID, name)
	if err != nil {
		return err
	}
	if row.IsBuiltin {
		return ErrBuiltinImmutable
	}
	blob, err := json.Marshal(perms)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}
	row.Permissions = string(blob)
	row.Description = description
	return s.db.WithContext(ctx).Save(row).Error
}

// Delete removes a custom role. Built-in roles cannot be deleted.
func (s *Service) Delete(ctx context.Context, tenantID, name string) error {
	row, err := s.Get(ctx, tenantID, name)
	if err != nil {
		return err
	}
	if row.IsBuiltin {
		return ErrBuiltinImmutable
	}
	return s.db.WithContext(ctx).Where("id = ?", row.ID).Delete(&models.CustomRole{}).Error
}

// Get returns a custom role by (tenant, name).
func (s *Service) Get(ctx context.Context, tenantID, name string) (*models.CustomRole, error) {
	var row models.CustomRole
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("get custom role: %w", err)
	}
	return &row, nil
}

// List returns every custom role in the tenant.
func (s *Service) List(ctx context.Context, tenantID string) ([]models.CustomRole, error) {
	var rows []models.CustomRole
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list custom roles: %w", err)
	}
	return rows, nil
}

// ParsePermissions returns the parsed PermissionGrant slice from a
// custom role's JSON blob.
func ParsePermissions(row *models.CustomRole) ([]PermissionGrant, error) {
	var out []PermissionGrant
	if row.Permissions == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(row.Permissions), &out); err != nil {
		return nil, fmt.Errorf("parse permissions: %w", err)
	}
	return out, nil
}

// Count returns the number of custom roles in a tenant.
func (s *Service) Count(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	if err := s.db.WithContext(ctx).
		Model(&models.CustomRole{}).
		Where("tenant_id = ?", tenantID).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// -----------------------------------------------------------------------------
// Permission lookup (used by the RBAC middleware)
// -----------------------------------------------------------------------------

// Lookup returns the PermissionGrant slice for a role name. Checks
// the built-in auth.RolePermissions map first; on miss, hits the
// custom-role DB. Returns (nil, ErrRoleNotFound) for unknown roles.
func Lookup(ctx context.Context, s *Service, roleName string) ([]PermissionGrant, error) {
	if s == nil {
		return nil, nil
	}
	// The built-in map is the source of truth — no DB hit needed
	// for those. The middleware converts Permission → PermissionGrant
	// and consults this function.
	if _, ok := builtInSet[roleName]; ok {
		// Caller handles built-in directly; we return nil to signal
		// "use built-in map".
		return nil, nil
	}
	// Custom: look up. The caller is responsible for tenant scoping
	// (we don't know the tenant at this layer in the middleware; the
	// middleware pulls it from claims).
	// For a more efficient path, the middleware can pass the
	// tenant ID. We provide a convenience LookupWithTenant for that.
	return nil, ErrRoleNotFound
}

// builtInSet is the set of hardcoded role names. Filled at init().
var builtInSet = map[string]bool{
	"root_admin":      true,
	"reseller_admin":  true,
	"reseller_agent":  true,
	"account_owner":   true,
	"account_dev":     true,
	"account_viewer":  true,
	"mail_admin":      true,
	"db_admin":        true,
	"security_admin":  true,
	"monitor":         true,
	"support":         true,
	"billing":         true,
}

// LookupWithTenant is the full lookup. Returns the parsed
// PermissionGrant slice for a role in a tenant. If the role is
// built-in, the built-in permissions are converted into the same
// PermissionGrant shape.
func LookupWithTenant(ctx context.Context, s *Service, tenantID, roleName string) ([]PermissionGrant, error) {
	if builtInSet[roleName] {
		return fromBuiltIn(roleName), nil
	}
	row, err := s.Get(ctx, tenantID, roleName)
	if err != nil {
		return nil, err
	}
	return ParsePermissions(row)
}

// HasPermissionFor reports whether the given role has (resource, action)
// in a given tenant. Checks built-in + custom. Returns false on any
// lookup error.
func HasPermissionFor(ctx context.Context, s *Service, tenantID, roleName, resource, action string) bool {
	if roleName == "" {
		return false
	}
	grants, err := LookupWithTenant(ctx, s, tenantID, roleName)
	if err != nil {
		return false
	}
	for _, g := range grants {
		if matchR(g.Resource, resource) {
			for _, a := range g.Actions {
				if a == "*" || a == action {
					return true
				}
			}
		}
	}
	return false
}

func fromBuiltIn(role string) []PermissionGrant {
	switch role {
	case "root_admin":
		return []PermissionGrant{{Resource: "*", Actions: []string{"*"}}}
	}
	// For the rest, the api/middleware/rbac.go map is the source of
	// truth. v0.3.0 duplicates the built-in mapping here so
	// LookupWithTenant is self-contained.
	out := []PermissionGrant{}
	switch role {
	case "reseller_admin":
		out = append(out,
			PermissionGrant{"reseller", []string{"*"}},
			PermissionGrant{"account", []string{"*"}},
			PermissionGrant{"domain", []string{"*"}},
			PermissionGrant{"hosting", []string{"*"}},
			PermissionGrant{"dns", []string{"*"}},
			PermissionGrant{"mail", []string{"*"}},
			PermissionGrant{"database", []string{"*"}},
			PermissionGrant{"files", []string{"*"}},
			PermissionGrant{"ssl", []string{"*"}},
			PermissionGrant{"firewall", []string{"read"}},
			PermissionGrant{"guardian", []string{"read"}},
			PermissionGrant{"metrics", []string{"read"}},
			PermissionGrant{"audit", []string{"read"}},
		)
	case "reseller_agent":
		out = append(out,
			PermissionGrant{"account", []string{"read"}},
			PermissionGrant{"domain", []string{"read"}},
			PermissionGrant{"hosting", []string{"read"}},
			PermissionGrant{"metrics", []string{"read"}},
		)
	case "account_owner":
		out = append(out,
			PermissionGrant{"domain", []string{"*"}},
			PermissionGrant{"hosting", []string{"*"}},
			PermissionGrant{"database", []string{"*"}},
			PermissionGrant{"files", []string{"*"}},
			PermissionGrant{"mail", []string{"*"}},
			PermissionGrant{"ssl", []string{"*"}},
			PermissionGrant{"dns", []string{"*"}},
			PermissionGrant{"backup", []string{"*"}},
			PermissionGrant{"metrics", []string{"read"}},
			PermissionGrant{"firewall", []string{"read"}},
		)
	case "account_dev":
		out = append(out,
			PermissionGrant{"domain", []string{"read"}},
			PermissionGrant{"hosting", []string{"*"}},
			PermissionGrant{"database", []string{"*"}},
			PermissionGrant{"files", []string{"*"}},
			PermissionGrant{"metrics", []string{"read"}},
		)
	case "account_viewer":
		out = append(out,
			PermissionGrant{"domain", []string{"read"}},
			PermissionGrant{"hosting", []string{"read"}},
			PermissionGrant{"database", []string{"read"}},
			PermissionGrant{"files", []string{"read"}},
			PermissionGrant{"metrics", []string{"read"}},
		)
	case "mail_admin":
		out = append(out,
			PermissionGrant{"mail", []string{"*"}},
			PermissionGrant{"ssl", []string{"read"}},
			PermissionGrant{"domain", []string{"read"}},
		)
	case "db_admin":
		out = append(out,
			PermissionGrant{"database", []string{"*"}},
			PermissionGrant{"domain", []string{"read"}},
		)
	case "security_admin":
		out = append(out,
			PermissionGrant{"firewall", []string{"*"}},
			PermissionGrant{"waf", []string{"*"}},
			PermissionGrant{"ids", []string{"*"}},
			PermissionGrant{"ssl", []string{"*"}},
			PermissionGrant{"audit", []string{"read"}},
		)
	case "monitor":
		out = append(out,
			PermissionGrant{"metrics", []string{"read"}},
			PermissionGrant{"audit", []string{"read"}},
			PermissionGrant{"guardian", []string{"read"}},
		)
	case "support":
		out = append(out,
			PermissionGrant{"account", []string{"read"}},
			PermissionGrant{"domain", []string{"read"}},
			PermissionGrant{"hosting", []string{"read"}},
		)
	case "billing":
		out = append(out,
			PermissionGrant{"billing", []string{"*"}},
			PermissionGrant{"license", []string{"*"}},
			PermissionGrant{"account", []string{"read"}},
		)
	}
	return out
}

func matchR(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == s
}

// -----------------------------------------------------------------------------
// User role assignment
// -----------------------------------------------------------------------------

// AssignRole sets a user's role (built-in or custom). For custom
// roles, validates the role exists in the same tenant.
func AssignRole(ctx context.Context, db *gorm.DB, userID, roleName string) error {
	if roleName == "" {
		return fmt.Errorf("role name is empty")
	}
	if !builtInSet[roleName] {
		// Custom role: must exist somewhere. (Tenant isolation
		// is enforced at the handler level by joining on tenant_id.)
		var n int64
		if err := db.WithContext(ctx).Model(&models.CustomRole{}).
			Where("name = ?", roleName).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrRoleNotFound
		}
	}
	return db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("role", roleName).Error
}

// RoleAssignable is a thread-safe in-memory cache of custom-role
// permission sets to avoid hitting the DB on every request. v0.3.0
// exposes invalidate/clear for handlers that mutate roles.
type RoleAssignable struct {
	mu    sync.Mutex
	cache map[string][]PermissionGrant
}

// NewRoleCache returns a fresh cache.
func NewRoleCache() *RoleAssignable {
	return &RoleAssignable{cache: make(map[string][]PermissionGrant)}
}

// Get returns the cached grants for (tenant, role) or false.
func (c *RoleAssignable) Get(tenantID, role string) ([]PermissionGrant, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.cache[tenantID+":"+role]
	return g, ok
}

// Set stores the grants for (tenant, role).
func (c *RoleAssignable) Set(tenantID, role string, g []PermissionGrant) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[tenantID+":"+role] = g
}

// Invalidate removes the entry for (tenant, role). Used after a
// role update or delete.
func (c *RoleAssignable) Invalidate(tenantID, role string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, tenantID+":"+role)
}
