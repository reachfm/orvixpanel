package v1

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/orvixpanel/orvixpanel/internal/license"
	"gorm.io/gorm"
)

// mustDB pulls the *gorm.DB out of Fiber Locals. Set by the api
// package's deps middleware in a real wire-up; for tests the test
// harness injects it directly.
func mustDB(c *fiber.Ctx) *gorm.DB {
	v := c.Locals("db")
	if v == nil {
		return nil
	}
	db, _ := v.(*gorm.DB)
	return db
}

// mustAuditor pulls the *audit.Auditor out of Locals.
func mustAuditor(c *fiber.Ctx) *audit.Auditor {
	v := c.Locals("auditor")
	if v == nil {
		return nil
	}
	a, _ := v.(*audit.Auditor)
	return a
}

// AdminSystem — GET /api/v1/admin/system.
func AdminSystem(c *fiber.Ctx) error {
	// AuthMiddleware already ran (we're under the /admin group which
	// requires the "admin" permission). We don't need to read the
	// claims here — the route returns server build info.
	return c.JSON(BuildInfo())
}

// BuildInfo is the v1.0 build metadata returned by /admin/system.
func BuildInfo() map[string]any {
	return map[string]any{
		"name":      "OrvixPanel",
		"version":   "1.0.0",
		"uptime_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// AdminLicense — GET /api/v1/admin/license.
func AdminLicense(c *fiber.Ctx) error {
	lic := license.Get()
	if lic == nil {
		return c.JSON(fiber.Map{"tier": "none", "features": []string{}})
	}
	return c.JSON(lic)
}

// AdminAuditLog — GET /api/v1/admin/audit-log?limit=50.
func AdminAuditLog(c *fiber.Ctx) error {
	db := mustDB(c)
	if db == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db_not_injected")
	}
	limit := c.QueryInt("limit", 50)
	if limit > 500 {
		limit = 500
	}
	var entries []models.AuditEntry
	if err := db.Order("timestamp DESC").Limit(limit).Find(&entries).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db_error")
	}
	return c.JSON(fiber.Map{"entries": entries})
}

// AdminAuditVerify — POST /api/v1/admin/audit-log/verify.
func AdminAuditVerify(c *fiber.Ctx) error {
	a := mustAuditor(c)
	if a == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "audit_not_injected")
	}
	idx, err := a.VerifyChain(c.Context())
	if err != nil {
		return c.JSON(fiber.Map{
			"tampered":       true,
			"first_bad_row":  idx,
			"error":          err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"tampered":      false,
		"first_bad_row": -1,
	})
}

// AdminLicenseRenewal — GET /api/v1/admin/license/renewal-info
// Returns the human-readable license status (mode, days remaining,
// grace days, etc). Wired to *license.Store by the api package.
func AdminLicenseRenewal(store *license.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "license_store_unavailable")
		}
		info, err := store.RenewalInfo(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "license_renewal_failed")
		}
		return c.JSON(info)
	}
}

// LicenseUploadRequest is the body for PUT /api/v1/admin/license.
type LicenseUploadRequest struct {
	Key string `json:"key"`
}

// AdminLicenseUpload — PUT /api/v1/admin/license
// Parses + verifies the key and persists it (encrypted). v0.3.0
// keeps the dev-mode signature bypass; in production the public
// key must be configured.
func AdminLicenseUpload(store *license.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "license_store_unavailable")
		}
		var req LicenseUploadRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		if req.Key == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_key")
		}
		// In dev mode we accept the key without signature.
		// In production, license.ParseWithPublicKey will fail.
		lic, err := license.Parse(req.Key)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "license_parse_failed:"+err.Error())
		}
		// Persist.
		claims, _ := c.Locals(middleware.LocalClaims).(*auth.Claims)
		uploader := ""
		if claims != nil {
			uploader = claims.UserID
		}
		if err := store.Save(c.Context(), lic, uploader); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "license_save_failed")
		}
		// Re-apply globally.
		license.SetGlobal(lic)
		return c.JSON(lic)
	}
}
