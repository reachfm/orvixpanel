// Package v1 — Account CRUD (Phase 2 Core Hosting Engine).
//
// Account = a hosting account owned by a tenant. The HTTP
// handlers translate the request into calls to
// internal/hosting (which does the real Linux work) and persist
// the result in the accounts table.
package v1

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/orvixpanel/orvixpanel/internal/hosting"
	"gorm.io/gorm"
)

// AccountDeps bundles dependencies for account handlers.
type AccountDeps struct {
	DB      *gorm.DB
	Hosting *hosting.Service
	Audit   *audit.Auditor
}

// CreateAccountRequest is the POST /accounts body.
type CreateAccountRequest struct {
	Username     string `json:"username"`
	Domain       string `json:"domain"`
	Plan         string `json:"plan"`         // basic|pro|unlimited
	DiskQuotaMB  int    `json:"disk_quota_mb"` // default 10240
	BandwidthGB  int    `json:"bandwidth_gb"`  // default 100
}

// CreateAccountHandler — POST /api/v1/accounts
//
// On success returns 201 with the created account row + the
// freshly created domain (since "an account" is meaningless
// without a domain in this hosting model).
func CreateAccountHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var req CreateAccountRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))
		req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
		if req.Username == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_username")
		}
		if req.Plan == "" {
			req.Plan = "basic"
		}
		if req.DiskQuotaMB == 0 {
			req.DiskQuotaMB = 10240
		}
		if req.BandwidthGB == 0 {
			req.BandwidthGB = 100
		}

		// 1. Provision the system user (Linux-only).
		uid, err := d.Hosting.CreateAccount(req.Username)
		if err != nil {
			if errors.Is(err, hosting.ErrAccountExists) {
				return fiber.NewError(fiber.StatusConflict, "account_username_taken")
			}
			if errors.Is(err, hosting.ErrUnsupported) {
				return fiber.NewError(fiber.StatusNotImplemented, "hosting_linux_only")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "user_provision_failed:"+err.Error())
		}

		// 2. Persist the account row.
		row := models.Account{
			Base:         models.Base{ID: newAccountID()},
			Username:     req.Username,
			Domain:       req.Domain,
			TenantID:     claims.TenantID,
			Plan:         req.Plan,
			DiskQuotaMB:  int64(req.DiskQuotaMB),
			BandwidthGB:  req.BandwidthGB,
			Status:       "active",
		}
		if err := d.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
			// Roll back the system user on DB failure.
			_ = d.Hosting.DeleteAccount(req.Username)
			return fiber.NewError(fiber.StatusInternalServerError, "account_create_failed:"+err.Error())
		}
		_ = uid

		// 3. Audit.
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "account.create",
				ResourceType: "account",
				ResourceID:   row.ID,
				ResourceName: row.Username,
				Result:       "success",
				UserID:       claims.UserID,
				UserEmail:    claims.Email,
				ActorIP:      c.IP(),
			})
		}
		return c.Status(fiber.StatusCreated).JSON(row)
	}
}

// ListAccountsHandler — GET /api/v1/accounts
func ListAccountsHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var rows []models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("tenant_id = ?", claims.TenantID).
			Order("created_at DESC").
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "account_list_failed")
		}
		return c.JSON(fiber.Map{"accounts": rows})
	}
}

// GetAccountHandler — GET /api/v1/accounts/:id
func GetAccountHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var row models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "account_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "account_get_failed")
		}
		// Live usage (Linux-only).
		home := d.Hosting.Paths.AccountHome(row.Username)
		used, _ := d.Hosting.DiskUsed(home)
		row.DiskUsedMB = used / 1024 / 1024
		return c.JSON(row)
	}
}

// SuspendAccountHandler — POST /api/v1/accounts/:id/suspend
func SuspendAccountHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var row models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&row).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		if err := d.Hosting.SuspendAccount(row.Username); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "suspend_failed:"+err.Error())
		}
		row.Status = "suspended"
		_ = d.DB.WithContext(c.Context()).Save(&row).Error
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "account.suspend",
				ResourceType: "account",
				ResourceID:   row.ID,
				ResourceName: row.Username,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}
		return c.JSON(row)
	}
}

// UnsuspendAccountHandler — POST /api/v1/accounts/:id/unsuspend
func UnsuspendAccountHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var row models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&row).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		if err := d.Hosting.UnsuspendAccount(row.Username); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "unsuspend_failed:"+err.Error())
		}
		row.Status = "active"
		_ = d.DB.WithContext(c.Context()).Save(&row).Error
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "account.unsuspend",
				ResourceType: "account",
				ResourceID:   row.ID,
				ResourceName: row.Username,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}
		return c.JSON(row)
	}
}

// DeleteAccountHandler — DELETE /api/v1/accounts/:id
func DeleteAccountHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var row models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&row).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		if err := d.Hosting.DeleteAccount(row.Username); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "delete_failed:"+err.Error())
		}
		// Hard-delete: the soft-delete behavior keeps the row in
		// the unique index on accounts.domain, which blocks the
		// smoke test from re-creating an account with the same
		// domain. v0.2.1 adds a partial unique index that
		// excludes deleted_at; for now we hard-delete.
		_ = d.DB.WithContext(c.Context()).Unscoped().Delete(&row).Error
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "account.delete",
				ResourceType: "account",
				ResourceID:   row.ID,
				ResourceName: row.Username,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// AccountUsageHandler — GET /api/v1/accounts/:id/usage
func AccountUsageHandler(d AccountDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var row models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&row).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		home := d.Hosting.Paths.AccountHome(row.Username)
		bytes, _ := d.Hosting.DiskUsed(home)
		inodes, _ := d.Hosting.InodeCount(home)
		return c.JSON(fiber.Map{
			"account_id":     row.ID,
			"username":       row.Username,
			"disk_used_mb":   bytes / 1024 / 1024,
			"disk_used_bytes": bytes,
			"disk_quota_mb":  row.DiskQuotaMB,
			"inodes":         inodes,
			"bandwidth_gb":   row.BandwidthGB,
		})
	}
}

// newAccountID is a UUIDv4 — accounts use UUID for ease (no
// special ID-format requirement; the model.Base.BeforeCreate
// would also generate a ULID if ID was empty, but we set it
// explicitly here for clarity).
func newAccountID() string { return uuid.NewString() }
