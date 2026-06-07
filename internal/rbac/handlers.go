package rbac

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"gorm.io/gorm"
)

// claimsKey is the Fiber Locals key used by the api/middleware
// package. Duplicated here (instead of importing the middleware) to
// avoid an import cycle: middleware → rbac → middleware.
const claimsKey = "claims"

// Deps bundles the dependencies.
type Deps struct {
	Service *Service
	DB      *gorm.DB
	Audit   *audit.Auditor
}

// CreateRequest is the POST /admin/roles body.
type CreateRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []PermissionGrant `json:"permissions"`
}

// CreateHandler — POST /api/v1/admin/roles
func CreateHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var req CreateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		row, err := d.Service.Create(c.Context(), claims.TenantID, strings.TrimSpace(req.Name), req.Description, req.Permissions)
		if err != nil {
			return mapRBACError(err)
		}
		auditCustomRole(c, d.Audit, claims, "create", row.Name, true, "")
		return c.Status(fiber.StatusCreated).JSON(row)
	}
}

// UpdateHandler — PUT /api/v1/admin/roles/:name
func UpdateHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		name := c.Params("name")
		var req CreateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		if err := d.Service.Update(c.Context(), claims.TenantID, name, req.Description, req.Permissions); err != nil {
			return mapRBACError(err)
		}
		auditCustomRole(c, d.Audit, claims, "update", name, true, "")
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteHandler — DELETE /api/v1/admin/roles/:name
func DeleteHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		name := c.Params("name")
		if err := d.Service.Delete(c.Context(), claims.TenantID, name); err != nil {
			return mapRBACError(err)
		}
		auditCustomRole(c, d.Audit, claims, "delete", name, true, "")
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ListHandler — GET /api/v1/admin/roles
func ListHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		rows, err := d.Service.List(c.Context(), claims.TenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "rbac_list_failed")
		}
		// Include parsed permissions in the response.
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			perms, _ := ParsePermissions(&rows[i])
			out = append(out, map[string]any{
				"id":          rows[i].ID,
				"name":        rows[i].Name,
				"description": rows[i].Description,
				"permissions": perms,
				"is_builtin":  rows[i].IsBuiltin,
				"created_at":  rows[i].CreatedAt,
				"updated_at":  rows[i].UpdatedAt,
			})
		}
		return c.JSON(fiber.Map{"roles": out})
	}
}

// AssignRoleRequest is the POST /admin/users/:id/role body.
type AssignRoleRequest struct {
	Role string `json:"role"`
}

// AssignRoleHandler — POST /api/v1/admin/users/:id/role
func AssignRoleHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actor, ok := c.Locals(claimsKey).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		userID := c.Params("id")
		var req AssignRoleRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		// Tenant check: target user must be in the same tenant.
		var target struct {
			TenantID string `gorm:"column:tenant_id"`
		}
		if err := d.DB.WithContext(c.Context()).
			Table("users").Select("tenant_id").Where("id = ?", userID).
			Scan(&target).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "user_not_found")
		}
		if target.TenantID != actor.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "user_not_in_your_tenant")
		}
		// Custom-role scope: must be in the same tenant.
		if !isBuiltin(req.Role) {
			if _, err := d.Service.Get(c.Context(), actor.TenantID, req.Role); err != nil {
				return mapRBACError(err)
			}
		}
		if err := AssignRole(c.Context(), d.DB, userID, req.Role); err != nil {
			return mapRBACError(err)
		}
		auditCustomRole(c, d.Audit, actor, "assign", req.Role+" to "+userID, true, "")
		return c.JSON(fiber.Map{"user_id": userID, "role": req.Role})
	}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func isBuiltin(role string) bool {
	switch role {
	case "root_admin", "reseller_admin", "reseller_agent",
		"account_owner", "account_dev", "account_viewer",
		"mail_admin", "db_admin", "security_admin",
		"monitor", "support", "billing":
		return true
	}
	return false
}

func mapRBACError(err error) error {
	switch {
	case errors.Is(err, ErrRoleNotFound):
		return fiber.NewError(fiber.StatusNotFound, "role_not_found")
	case errors.Is(err, ErrBuiltinRoleClash):
		return fiber.NewError(fiber.StatusBadRequest, "role_name_reserved")
	case errors.Is(err, ErrBuiltinImmutable):
		return fiber.NewError(fiber.StatusForbidden, "builtin_role_immutable")
	case errors.Is(err, ErrInvalidRoleName):
		return fiber.NewError(fiber.StatusBadRequest, "invalid_role_name")
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "rbac_failed:"+err.Error())
	}
}

func auditCustomRole(c *fiber.Ctx, a *audit.Auditor, claims *auth.Claims, action, name string, ok bool, detail string) {
	if a == nil || claims == nil {
		return
	}
	result := "success"
	if !ok {
		result = "denied"
	}
	_ = a.Record(c.Context(), audit.Event{
		Action:       "rbac.custom." + action,
		ResourceType: "role",
		ResourceName: name,
		Result:       result,
		Detail:       detail,
		UserID:       claims.UserID,
		UserEmail:    claims.Email,
		UserRole:     claims.Role,
		ActorIP:      c.IP(),
	})
}
