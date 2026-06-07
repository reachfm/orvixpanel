package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
)

// Permission is the (resource, action) tuple the RBAC layer checks.
type Permission struct {
	Resource string
	Action   string
}

// RolePermissions is the default role → permission map.
//
// In v1.0 this lives in code. v1.1 moves it to the DB so operators
// can add custom roles without recompiling.
var RolePermissions = map[string][]Permission{
	"root_admin": {{"*", "*"}},
	"reseller_admin": {
		{"reseller", "*"},
		{"account", "*"},
		{"domain", "*"},
		{"hosting", "*"},
		{"dns", "*"},
		{"mail", "*"},
		{"database", "*"},
		{"files", "*"},
		{"ssl", "*"},
		{"firewall", "read"},
		{"guardian", "read"},
		{"metrics", "read"},
		{"audit", "read"},
	},
	"reseller_agent": {
		{"account", "read"},
		{"domain", "read"},
		{"hosting", "read"},
		{"metrics", "read"},
	},
	"account_owner": {
		{"domain", "*"},
		{"hosting", "*"},
		{"database", "*"},
		{"files", "*"},
		{"mail", "*"},
		{"ssl", "*"},
		{"dns", "*"},
		{"backup", "*"},
		{"metrics", "read"},
		{"firewall", "read"},
	},
	"account_dev": {
		{"domain", "read"},
		{"hosting", "*"},
		{"database", "*"},
		{"files", "*"},
		{"metrics", "read"},
	},
	"account_viewer": {
		{"domain", "read"},
		{"hosting", "read"},
		{"database", "read"},
		{"files", "read"},
		{"metrics", "read"},
	},
	"mail_admin":   {{"mail", "*"}, {"ssl", "read"}, {"domain", "read"}},
	"db_admin":     {{"database", "*"}, {"domain", "read"}},
	"security_admin": {
		{"firewall", "*"},
		{"waf", "*"},
		{"ids", "*"},
		{"ssl", "*"},
		{"audit", "read"},
	},
	"monitor": {
		{"metrics", "read"},
		{"audit", "read"},
		{"guardian", "read"},
	},
	"support": {
		{"account", "read"},
		{"domain", "read"},
		{"hosting", "read"},
	},
	"billing": {
		{"billing", "*"},
		{"license", "*"},
		{"account", "read"},
	},
}

// HasPermission returns true if the role's permission list allows
// (resource, action). Wildcard "*" matches anything.
func HasPermission(role, resource, action string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if match(p.Resource, resource) && match(p.Action, action) {
			return true
		}
	}
	return false
}

func match(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == s
}

// RequirePermission returns a middleware that 403s if the caller's
// role doesn't allow (resource, action).
//
// v0.3.0: when the user's role is not in the built-in map, fall
// through to the custom-role DB lookup. The custom role must live
// in the user's tenant. The check is cached in-process via
// rbac.RoleAssignable — pass nil to disable caching.
func RequirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		// Built-in: use the in-memory map.
		if HasPermission(claims.Role, resource, action) {
			return c.Next()
		}
		// Custom role: look it up in the DB.
		svc := rbacServiceFromCtx(c)
		if svc == nil {
			return fiber.NewError(fiber.StatusForbidden, "permission_denied")
		}
		if rbac.HasPermissionFor(c.Context(), svc, claims.TenantID, claims.Role, resource, action) {
			return c.Next()
		}
		return fiber.NewError(fiber.StatusForbidden, "permission_denied")
	}
}

// rbacServiceFromCtx pulls the *rbac.Service injected by the api
// package's depsMiddleware. We import the rbac package lazily to
// avoid a hard import cycle (rbac → db only; middleware → rbac
// already exists from the existing tenant middleware).
func rbacServiceFromCtx(c *fiber.Ctx) *rbac.Service {
	v := c.Locals("rbac")
	if v == nil {
		return nil
	}
	s, _ := v.(*rbac.Service)
	return s
}
