package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/hosting/plans"
	"gorm.io/gorm"
)

// HostingPlansDeps holds dependencies for hosting plans handlers.
type HostingPlansDeps struct {
	DB    *gorm.DB
	Plans *plans.GormStore
	Audit *audit.Auditor
}

// HostingPlanRequest is the request body for create/update plan.
type HostingPlanRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	DiskQuotaMB  int64    `json:"disk_quota_mb"`
	BandwidthGB  int64    `json:"bandwidth_gb"`
	MaxDomains   int      `json:"max_domains"`
	MaxUsers     int      `json:"max_users"`
	MaxSSL       int      `json:"max_ssl"`
	Features     []string `json:"features"`
	MonthlyPrice float64  `json:"monthly_price"`
	IsActive     bool     `json:"is_active"`
	IsDefault    bool     `json:"is_default"`
}

// ListPlansHandler returns all hosting plans.
func ListPlansHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Query params
		search := c.Query("search", "")
		status := c.Query("status", "")

		var planList []*plans.Plan
		var err error

		if search != "" || status != "" {
			// Manual filtering
			q := d.DB.Model(&plans.Plan{})
			if search != "" {
				q = q.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?",
					"%"+search+"%", "%"+search+"%", "%"+search+"%")
			}
			if status == "active" {
				q = q.Where("is_active = ?", true)
			} else if status == "inactive" {
				q = q.Where("is_active = ?", false)
			}
			var rows []*plans.Plan
			if err := q.Order("created_at DESC").Find(&rows).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "db_error")
			}
			planList = rows
		} else {
			planList, err = d.Plans.List(c.Context())
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "db_error")
			}
		}

		// Convert to response format
		response := make([]map[string]interface{}, 0, len(planList))
		for _, p := range planList {
			response = append(response, planToMap(p))
		}

		return c.JSON(fiber.Map{
			"plans": response,
			"total": len(response),
		})
	}
}

// GetPlanHandler returns a single plan by ID.
func GetPlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id_required")
		}

		plan, err := d.Plans.GetByID(c.Context(), id)
		if err != nil {
			if err == plans.ErrNotFound {
				return fiber.NewError(fiber.StatusNotFound, "plan_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "db_error")
		}

		return c.JSON(fiber.Map{
			"plan": planToMap(plan),
		})
	}
}

// CreatePlanHandler creates a new hosting plan.
func CreatePlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req HostingPlanRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		// Validation
		if req.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name_required")
		}
		if err := plans.ValidateName(req.Name); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.MonthlyPrice < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "monthly_price_cannot_be_negative")
		}

		// Check name uniqueness
		existing, _ := d.Plans.GetByName(c.Context(), req.Name)
		if existing != nil {
			return fiber.NewError(fiber.StatusConflict, "name_already_exists")
		}

		// Create plan
		plan := &plans.Plan{
			Name:         req.Name,
			DisplayName:  req.DisplayName,
			Description:  req.Description,
			DiskQuotaMB:  req.DiskQuotaMB,
			BandwidthGB:  req.BandwidthGB,
			MaxDomains:   req.MaxDomains,
			MaxUsers:     req.MaxUsers,
			MaxSSL:       req.MaxSSL,
			Features:     req.Features,
			MonthlyPrice: req.MonthlyPrice,
			IsActive:     req.IsActive,
			IsDefault:    req.IsDefault,
		}

		if err := d.Plans.Create(c.Context(), plan); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "create_failed")
		}

		// Audit log
		if d.Audit != nil {
			d.Audit.Record(c.Context(), audit.Event{
				Action:       "hosting.plan.create",
				Result:       "success",
				ResourceID:   plan.ID,
				ResourceType: "hosting_plan",
				Detail:       "Created plan: " + plan.Name,
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"plan": planToMap(plan),
		})
	}
}

// UpdatePlanHandler updates an existing plan.
func UpdatePlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id_required")
		}

		var req HostingPlanRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		// Get existing plan
		existing, err := d.Plans.GetByID(c.Context(), id)
		if err != nil {
			if err == plans.ErrNotFound {
				return fiber.NewError(fiber.StatusNotFound, "plan_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "db_error")
		}

		// Validation
		if req.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name_required")
		}
		if err := plans.ValidateName(req.Name); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.MonthlyPrice < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "monthly_price_cannot_be_negative")
		}

		// Check name uniqueness (excluding current plan)
		byName, _ := d.Plans.GetByName(c.Context(), req.Name)
		if byName != nil && byName.ID != id {
			return fiber.NewError(fiber.StatusConflict, "name_already_exists")
		}

		// Update fields
		existing.Name = req.Name
		existing.DisplayName = req.DisplayName
		existing.Description = req.Description
		existing.DiskQuotaMB = req.DiskQuotaMB
		existing.BandwidthGB = req.BandwidthGB
		existing.MaxDomains = req.MaxDomains
		existing.MaxUsers = req.MaxUsers
		existing.MaxSSL = req.MaxSSL
		existing.Features = req.Features
		existing.MonthlyPrice = req.MonthlyPrice
		existing.IsActive = req.IsActive
		existing.IsDefault = req.IsDefault

		if err := d.Plans.Update(c.Context(), existing); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "update_failed")
		}

		// Audit log
		if d.Audit != nil {
			d.Audit.Record(c.Context(), audit.Event{
				Action:       "hosting.plan.update",
				Result:       "success",
				ResourceID:   existing.ID,
				ResourceType: "hosting_plan",
				Detail:       "Updated plan: " + existing.Name,
			})
		}

		return c.JSON(fiber.Map{
			"plan": planToMap(existing),
		})
	}
}

// DeletePlanHandler deletes a plan.
func DeletePlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id_required")
		}

		// Get existing plan
		existing, err := d.Plans.GetByID(c.Context(), id)
		if err != nil {
			if err == plans.ErrNotFound {
				return fiber.NewError(fiber.StatusNotFound, "plan_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "db_error")
		}

		// Cannot delete default plan
		if existing.IsDefault {
			return fiber.NewError(fiber.StatusConflict, "cannot_delete_default_plan")
		}

		if err := d.Plans.Delete(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "delete_failed")
		}

		// Audit log
		if d.Audit != nil {
			d.Audit.Record(c.Context(), audit.Event{
				Action:       "hosting.plan.delete",
				Result:       "success",
				ResourceID:   id,
				ResourceType: "hosting_plan",
				Detail:       "Deleted plan: " + existing.Name,
			})
		}

		return c.JSON(fiber.Map{
			"message": "plan_deleted",
			"plan_id": id,
		})
	}
}

// ActivatePlanHandler activates a plan.
func ActivatePlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id_required")
		}

		plan, err := d.Plans.GetByID(c.Context(), id)
		if err != nil {
			if err == plans.ErrNotFound {
				return fiber.NewError(fiber.StatusNotFound, "plan_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "db_error")
		}

		plan.IsActive = true

		if err := d.Plans.Update(c.Context(), plan); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "activate_failed")
		}

		// Audit log
		if d.Audit != nil {
			d.Audit.Record(c.Context(), audit.Event{
				Action:       "hosting.plan.activate",
				Result:       "success",
				ResourceID:   plan.ID,
				ResourceType: "hosting_plan",
				Detail:       "Activated plan: " + plan.Name,
			})
		}

		return c.JSON(fiber.Map{
			"plan":    planToMap(plan),
			"message": "plan_activated",
		})
	}
}

// DeactivatePlanHandler deactivates a plan.
func DeactivatePlanHandler(d HostingPlansDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id_required")
		}

		plan, err := d.Plans.GetByID(c.Context(), id)
		if err != nil {
			if err == plans.ErrNotFound {
				return fiber.NewError(fiber.StatusNotFound, "plan_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "db_error")
		}

		// Cannot deactivate default plan
		if plan.IsDefault {
			return fiber.NewError(fiber.StatusConflict, "cannot_deactivate_default_plan")
		}

		plan.IsActive = false

		if err := d.Plans.Update(c.Context(), plan); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "deactivate_failed")
		}

		// Audit log
		if d.Audit != nil {
			d.Audit.Record(c.Context(), audit.Event{
				Action:       "hosting.plan.deactivate",
				Result:       "success",
				ResourceID:   plan.ID,
				ResourceType: "hosting_plan",
				Detail:       "Deactivated plan: " + plan.Name,
			})
		}

		return c.JSON(fiber.Map{
			"plan":    planToMap(plan),
			"message": "plan_deactivated",
		})
	}
}

// planToMap converts a Plan to a map for JSON response.
func planToMap(p *plans.Plan) map[string]interface{} {
	return map[string]interface{}{
		"id":            p.ID,
		"name":          p.Name,
		"display_name":  p.DisplayName,
		"description":   p.Description,
		"disk_quota_mb": p.DiskQuotaMB,
		"bandwidth_gb":  p.BandwidthGB,
		"max_domains":   p.MaxDomains,
		"max_users":     p.MaxUsers,
		"max_ssl":       p.MaxSSL,
		"features":      p.Features,
		"monthly_price": p.MonthlyPrice,
		"is_active":     p.IsActive,
		"is_default":    p.IsDefault,
		"created_at":    p.CreatedAt,
		"updated_at":    p.UpdatedAt,
	}
}