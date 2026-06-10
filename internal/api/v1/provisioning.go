// Package v1 — Provisioning API (Phase 2 Core Hosting Engine).
//
// Endpoints:
//   - POST   /api/v1/provisioning/websites       Create website provisioning job
//   - GET    /api/v1/provisioning/jobs           List provisioning jobs
//   - GET    /api/v1/provisioning/jobs/:id       Get provisioning job details
package v1

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/orvixpanel/orvixpanel/internal/provisioning"
	"gorm.io/gorm"
)

// ProvisioningDeps bundles dependencies for provisioning handlers.
type ProvisioningDeps struct {
	DB          *gorm.DB
	Audit       *audit.Auditor
	Provisioning *provisioning.Service
}

// CreateWebsiteRequest is the POST /api/v1/provisioning/websites body.
type CreateWebsiteRequest struct {
	AccountID   string `json:"account_id"`
	Domain      string `json:"domain"`
	PHPVersion  string `json:"php_version,omitempty"` // default "8.5"
}

// CreateWebsiteResponse is the POST response.
type CreateWebsiteResponse struct {
	JobID      string                     `json:"job_id"`
	Status     provisioning.JobStatus     `json:"status"`
	Domain     string                     `json:"domain"`
	CreatedAt  string                     `json:"created_at"`
	Steps      []provisioning.ProvisioningEvent `json:"steps,omitempty"`
}

// CreateWebsiteHandler — POST /api/v1/provisioning/websites
func CreateWebsiteHandler(d ProvisioningDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		var req CreateWebsiteRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		// Validate required fields
		if req.AccountID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_account_id")
		}
		if req.Domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_domain")
		}

		// Verify account belongs to tenant
		var acct models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", req.AccountID, claims.TenantID).
			First(&acct).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}

		// Check for duplicate domain (within tenant)
		var existingDomain models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("tenant_id = ? AND domain = ? AND id != ?", claims.TenantID, req.Domain, req.AccountID).
			First(&existingDomain).Error; err == nil {
			return fiber.NewError(fiber.StatusConflict, "domain_already_exists")
		}

		// Provision website
		job, err := d.Provisioning.ProvisionWebsite(c.Context(), provisioning.ProvisionWebsiteRequest{
			AccountID:  req.AccountID,
			TenantID:   claims.TenantID,
			Username:   acct.Username,
			Domain:     req.Domain,
			PHPVersion: req.PHPVersion,
		})
		if err != nil {
			// Audit failure
			if d.Audit != nil {
				_ = d.Audit.Record(c.Context(), audit.Event{
					Action:       "provisioning.website.create",
					ResourceType: "website",
					ResourceID:   req.Domain,
					ResourceName: req.Domain,
					Result:       "failure",
					UserID:       claims.UserID,
					ActorIP:      c.IP(),
					Detail:       fmt.Sprintf("account=%s error=%v", acct.Username, err),
				})
			}
			return fiber.NewError(fiber.StatusInternalServerError, "provisioning_failed:"+err.Error())
		}

		// Update account with new domain
		acct.Domain = req.Domain
		_ = d.DB.WithContext(c.Context()).Save(&acct).Error

		// Audit success
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "provisioning.website.create",
				ResourceType: "website",
				ResourceID:   job.ID,
				ResourceName: req.Domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
				Detail:       fmt.Sprintf("account=%s domain=%s job_id=%s", acct.Username, req.Domain, job.ID),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(CreateWebsiteResponse{
			JobID:     job.ID,
			Status:    job.Status,
			Domain:    job.Domain,
			CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// ListJobsHandler — GET /api/v1/provisioning/jobs
func ListJobsHandler(d ProvisioningDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		// Get account_id filter if provided
		accountID := c.Query("account_id")

		// Build query - scope to tenant
		query := d.DB.WithContext(c.Context()).Model(&provisioning.ProvisioningJob{}).
			Where("tenant_id = ?", claims.TenantID)

		if accountID != "" {
			query = query.Where("account_id = ?", accountID)
		}

		var jobs []provisioning.ProvisioningJob
		if err := query.Order("created_at DESC").Limit(100).Find(&jobs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "query_failed")
		}

		return c.JSON(fiber.Map{
			"jobs": jobs,
			"count": len(jobs),
		})
	}
}

// GetJobHandler — GET /api/v1/provisioning/jobs/:id
func GetJobHandler(d ProvisioningDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		jobID := c.Params("id")
		if jobID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_job_id")
		}

		var job provisioning.ProvisioningJob
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).
			First(&job).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusNotFound, "job_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "query_failed")
		}

		// Get events for this job
		var events []provisioning.ProvisioningEvent
		_ = d.DB.WithContext(c.Context()).
			Where("job_id = ?", jobID).
			Order("created_at ASC").
			Find(&events).Error

		return c.JSON(fiber.Map{
			"job":   job,
			"events": events,
		})
	}
}

// ListEventsHandler — GET /api/v1/provisioning/jobs/:id/events
func ListEventsHandler(d ProvisioningDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		jobID := c.Params("id")
		if jobID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_job_id")
		}

		// Verify job belongs to tenant
		var job provisioning.ProvisioningJob
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).
			First(&job).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusNotFound, "job_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "query_failed")
		}

		var events []provisioning.ProvisioningEvent
		if err := d.DB.WithContext(c.Context()).
			Where("job_id = ?", jobID).
			Order("created_at ASC").
			Find(&events).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "query_failed")
		}

		return c.JSON(fiber.Map{
			"events": events,
			"count":  len(events),
		})
	}
}