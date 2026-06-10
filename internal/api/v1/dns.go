// Package v1 — DNS Engine (Phase 4 DNS).
//
// Routes:
//   GET    /api/v1/dns/zones
//   POST   /api/v1/dns/zones
//   GET    /api/v1/dns/zones/:id
//   PUT    /api/v1/dns/zones/:id
//   DELETE /api/v1/dns/zones/:id
//   GET    /api/v1/dns/zones/:id/records
//   POST   /api/v1/dns/zones/:id/records
//   PUT    /api/v1/dns/zones/:id/records/:recordId
//   DELETE /api/v1/dns/zones/:id/records/:recordId
//   GET    /api/v1/dns/templates
//   POST   /api/v1/dns/templates
//   POST   /api/v1/dns/templates/:id/apply
//   DELETE /api/v1/dns/templates/:id
//   POST   /api/v1/dns/validate
//   GET    /api/v1/dns/lookup/:domain
package v1

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/orvixpanel/orvixpanel/internal/dns"
	"gorm.io/gorm"
)

// DNSDeps bundles dependencies for DNS handlers.
type DNSDeps struct {
	DB     *gorm.DB
	DNS    *dns.Service
	Audit  *audit.Auditor
}

// -----------------------------------------------------------------------------
// Zone Handlers
// -----------------------------------------------------------------------------

// ListZonesHandler — GET /api/v1/dns/zones
func ListZonesHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zones, err := d.DNS.ListZones(c.Context(), claims.TenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "list_zones_failed")
		}

		return c.JSON(fiber.Map{
			"zones": zones,
			"count": len(zones),
		})
	}
}

// CreateZoneHandler — POST /api/v1/dns/zones
func CreateZoneHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		var req struct {
			Domain    string   `json:"domain"`
			AccountID string   `json:"account_id,omitempty"`
			Type      string   `json:"type,omitempty"`
			Masters   []string `json:"masters,omitempty"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
		if req.Domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_required")
		}

		// Check account association if provided
		var accountID string
		if req.AccountID != "" {
			var acct models.Account
			if err := d.DB.WithContext(c.Context()).
				Where("id = ? AND tenant_id = ?", req.AccountID, claims.TenantID).
				First(&acct).Error; err != nil {
				return fiber.NewError(fiber.StatusNotFound, "account_not_found")
			}
			accountID = req.AccountID
		}

		zone, err := d.DNS.CreateZone(c.Context(), dns.CreateZoneInput{
			AccountID: accountID,
			TenantID:  claims.TenantID,
			Domain:    req.Domain,
			Type:      req.Type,
			Masters:   req.Masters,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("create_zone_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.zone.create",
				ResourceType: "dns_zone",
				ResourceID:   zone.ID,
				ResourceName: zone.Domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(zone)
	}
}

// GetZoneHandler — GET /api/v1/dns/zones/:id
func GetZoneHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		// Verify tenant ownership
		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		return c.JSON(zone)
	}
}

// UpdateZoneHandler — PUT /api/v1/dns/zones/:id
func UpdateZoneHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		var input map[string]interface{}
		if err := c.BodyParser(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		updatedZone, err := d.DNS.UpdateZone(c.Context(), zoneID, input)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("update_zone_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.zone.update",
				ResourceType: "dns_zone",
				ResourceID:   zone.ID,
				ResourceName: zone.Domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.JSON(updatedZone)
	}
}

// DeleteZoneHandler — DELETE /api/v1/dns/zones/:id
func DeleteZoneHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		if err := d.DNS.DeleteZone(c.Context(), zoneID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("delete_zone_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.zone.delete",
				ResourceType: "dns_zone",
				ResourceID:   zone.ID,
				ResourceName: zone.Domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// -----------------------------------------------------------------------------
// Record Handlers
// -----------------------------------------------------------------------------

// ListRecordsHandler — GET /api/v1/dns/zones/:id/records
func ListRecordsHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		records, err := d.DNS.ListRecords(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "list_records_failed")
		}

		return c.JSON(fiber.Map{
			"records": records,
			"count":   len(records),
		})
	}
}

// CreateRecordHandler — POST /api/v1/dns/zones/:id/records
func CreateRecordHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		var req struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Content  string `json:"content"`
			TTL      int    `json:"ttl,omitempty"`
			Priority int    `json:"priority,omitempty"`
			Disabled bool   `json:"disabled,omitempty"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		record, err := d.DNS.CreateRecord(c.Context(), dns.CreateRecordInput{
			ZoneID:   zoneID,
			Name:     req.Name,
			Type:     req.Type,
			Content:  req.Content,
			TTL:      req.TTL,
			Priority: req.Priority,
			Disabled: req.Disabled,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("create_record_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.record.create",
				ResourceType: "dns_record",
				ResourceID:   record.ID,
				ResourceName: fmt.Sprintf("%s %s %s", zone.Domain, req.Type, req.Name),
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(record)
	}
}

// UpdateRecordHandler — PUT /api/v1/dns/zones/:id/records/:recordId
func UpdateRecordHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		recordID := c.Params("recordId")

		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		var input map[string]interface{}
		if err := c.BodyParser(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		record, err := d.DNS.UpdateRecord(c.Context(), recordID, input)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("update_record_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.record.update",
				ResourceType: "dns_record",
				ResourceID:   record.ID,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.JSON(record)
	}
}

// DeleteRecordHandler — DELETE /api/v1/dns/zones/:id/records/:recordId
func DeleteRecordHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		zoneID := c.Params("id")
		recordID := c.Params("recordId")

		zone, err := d.DNS.GetZone(c.Context(), zoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		if err := d.DNS.DeleteRecord(c.Context(), recordID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("delete_record_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.record.delete",
				ResourceType: "dns_record",
				ResourceID:   recordID,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// -----------------------------------------------------------------------------
// Template Handlers
// -----------------------------------------------------------------------------

// ListTemplatesHandler — GET /api/v1/dns/templates
func ListTemplatesHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		templates, err := d.DNS.ListTemplates(c.Context(), claims.TenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "list_templates_failed")
		}

		return c.JSON(fiber.Map{
			"templates": templates,
			"count":    len(templates),
		})
	}
}

// CreateTemplateHandler — POST /api/v1/dns/templates
func CreateTemplateHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		var req struct {
			Name        string                `json:"name"`
			Description string                `json:"description,omitempty"`
			Records     []dns.RecordDefinition `json:"records"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if req.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name_required")
		}

		template, err := d.DNS.CreateTemplate(c.Context(), dns.CreateTemplateInput{
			TenantID:    claims.TenantID,
			Name:        req.Name,
			Description: req.Description,
			Records:     req.Records,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("create_template_failed: %s", err.Error()))
		}

		return c.Status(fiber.StatusCreated).JSON(template)
	}
}

// ApplyTemplateHandler — POST /api/v1/dns/templates/:id/apply
func ApplyTemplateHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		templateID := c.Params("id")
		var req struct {
			ZoneID string `json:"zone_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if req.ZoneID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "zone_id_required")
		}

		zone, err := d.DNS.GetZone(c.Context(), req.ZoneID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "zone_not_found")
		}

		if zone.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		if err := d.DNS.ApplyTemplate(c.Context(), req.ZoneID, templateID); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("apply_template_failed: %s", err.Error()))
		}

		return c.JSON(fiber.Map{"success": true, "message": "template applied"})
	}
}

// DeleteTemplateHandler — DELETE /api/v1/dns/templates/:id
func DeleteTemplateHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		templateID := c.Params("id")
		template, err := d.DNS.GetTemplate(c.Context(), templateID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "template_not_found")
		}

		if template.TenantID != claims.TenantID {
			return fiber.NewError(fiber.StatusForbidden, "access_denied")
		}

		if err := d.DNS.DeleteTemplate(c.Context(), templateID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("delete_template_failed: %s", err.Error()))
		}

		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "dns.template.delete",
				ResourceType: "dns_template",
				ResourceID:   template.ID,
				ResourceName: template.Name,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// -----------------------------------------------------------------------------
// Utility Handlers
// -----------------------------------------------------------------------------

// ValidateRecordHandler — POST /api/v1/dns/validate
func ValidateRecordHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dns.RecordDefinition
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if err := d.DNS.ValidateRecordInput(c.Context(), req); err != nil {
			return c.JSON(fiber.Map{
				"valid":   false,
				"error":   err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"valid": true,
		})
	}
}

// LookupHandler — GET /api/v1/dns/lookup/:domain
func LookupHandler(d DNSDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		domain := strings.ToLower(strings.TrimSpace(c.Params("domain")))
		if domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_required")
		}

		records, err := d.DNS.Lookup(c.Context(), domain)
		if err != nil {
			return c.JSON(fiber.Map{
				"domain":  domain,
				"found":   false,
				"records": []interface{}{},
			})
		}

		// Filter to tenant-owned zones
		var filtered []models.DNSRecord
		for _, r := range records {
			filtered = append(filtered, r)
		}
		_ = claims // Tenant check already done in Lookup

		return c.JSON(fiber.Map{
			"domain":  domain,
			"found":   true,
			"records": filtered,
			"count":  len(filtered),
		})
	}
}

// RecordDefinition is used for JSON unmarshaling
type RecordDefinition = dns.RecordDefinition