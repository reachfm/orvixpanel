package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
)

// AuditSearchHandler — POST /api/v1/admin/audit-log/search
func AuditSearchHandler(a *audit.Auditor) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var req audit.SearchRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		// Sanitize: strip wildcards from action input to prevent
		// full-table scans via injected LIKE wildcards.
		req.Action = audit.SanitizeAction(req.Action)

		// Tenant scope: non-root users can only search their own
		// tenant. Root admin can pass any tenant_id or omit it.
		forceTenant := ""
		if claims.Role != auth.RoleRootAdmin {
			forceTenant = claims.TenantID
		}
		resp, err := a.Search(c.Context(), req, forceTenant)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "audit_search_failed")
		}
		return c.JSON(resp)
	}
}

// AuditExportHandler — POST /api/v1/admin/audit-log/export
func AuditExportHandler(a *audit.Auditor) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req audit.ExportRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		resp, err := a.Export(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "audit_export_failed:"+err.Error())
		}
		return c.JSON(resp)
	}
}
