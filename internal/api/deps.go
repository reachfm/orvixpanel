package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
	"gorm.io/gorm"
)

// depsMiddleware injects *gorm.DB, *audit.Auditor, and *rbac.Service
// into request Locals so handlers can pull them. v0.3.0 adds rbac
// for the RequirePermission middleware to look up custom roles.
func depsMiddleware(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("db", d.DB)
		c.Locals("auditor", d.Audit)
		c.Locals("rbac", d.RBAC)
		return c.Next()
	}
}

// Compile-time guards for the optional types.
var _ *gorm.DB
var _ *audit.Auditor
var _ *rbac.Service
