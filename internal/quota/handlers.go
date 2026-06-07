package quota

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// GetHandler — GET /api/v1/admin/tenants/:id/quotas
func GetHandler(s *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		q, err := s.Get(c.Context(), tenantID, "smb")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "quota_get_failed")
		}
		return c.JSON(q)
	}
}

// PutHandler — PUT /api/v1/admin/tenants/:id/quotas
func PutHandler(s *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		var body models.TenantQuota
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		body.TenantID = tenantID
		if err := s.Put(c.Context(), &body); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "quota_put_failed")
		}
		return c.JSON(body)
	}
}

// MeHandler — GET /api/v1/me/quotas — returns the caller's own
// quota. Convenient for the UI without needing root_admin.
func MeHandler(s *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		q, err := s.Get(c.Context(), claims.TenantID, "smb")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "quota_get_failed")
		}
		return c.JSON(q)
	}
}
