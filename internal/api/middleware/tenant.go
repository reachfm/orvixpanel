package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/license"
)

// TenantMiddleware resolves the caller's tenant and applies license
// feature gating. v1.0 is a thin shim: every authenticated user is
// in the "default" tenant (the bootstrap one), and feature gating
// falls back to the license check only.
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		// v1.0: single-tenant. The claim is the source of truth.
		if claims.TenantID == "" {
			return fiber.NewError(fiber.StatusForbidden, "tenant_missing")
		}
		// License gating: if a route is named and the license doesn't
		// allow the feature, 403. v1.0 has the mapping for the
		// foundation routes only.
		lic := license.Get()
		if lic != nil {
			routeName := ""
			if r := c.Route(); r != nil {
				routeName = r.Name
			}
			if routeName != "" && !lic.HasFeature(routeName) {
				return fiber.NewError(fiber.StatusForbidden, "feature_not_licensed")
			}
		}
		return c.Next()
	}
}
