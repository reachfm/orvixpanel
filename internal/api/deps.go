package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"gorm.io/gorm"
)

// depsMiddleware injects *gorm.DB and *audit.Auditor into request
// Locals so handlers can pull them. v1.0 keeps the surface small
// because we don't have other singletons yet.
func depsMiddleware(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("db", d.DB)
		c.Locals("auditor", d.Audit)
		return c.Next()
	}
}

// Compile-time guards for the optional types.
var _ *gorm.DB
var _ *audit.Auditor
