package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/api/v1"
)

// registerV1 wires every /api/v1/* route.
//
// v1.0 ships:
//   - /me         — who am I (returns the JWT claims)
//   - /admin/*    — license + audit-log view (root_admin only)
//
// All other spec routes (account/domain/hosting/dns/mail/ssl/firewall/
// guardian/reseller/etc.) return 501 in v1.0 — they live in
// internal/api/v1/stubs.go with a single "phase_X_pending" message.
func registerV1(g fiber.Router, d Deps) {
	g.Get("/me", v1.MeHandler).Name("auth.me")

	// Admin group — root_admin only. /admin/system returns buildInfo
	// (placeholder); /admin/license returns the parsed license; the
	// audit-log endpoints return rows + a chain-verification result.
	admin := g.Group("/admin", middleware.RequirePermission("admin", "*"))
	admin.Get("/system", v1.AdminSystem).Name("admin.system")
	admin.Get("/license", v1.AdminLicense).Name("admin.license.read")
	admin.Get("/audit-log", v1.AdminAuditLog).Name("admin.audit.read")
	admin.Post("/audit-log/verify", v1.AdminAuditVerify).Name("admin.audit.verify")
}
