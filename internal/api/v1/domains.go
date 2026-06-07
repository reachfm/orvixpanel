// Package v1 — Domain CRUD (Phase 2 Core Hosting Engine).
//
// Each domain belongs to an account. Creating a domain:
//   1. Validates the domain shape
//   2. Calls hosting.CreateDomain to make the document root
//   3. Writes the nginx vhost
//   4. Writes the php-fpm pool
//   5. Runs nginx -t and php-fpm -t to validate
//   6. Reloads nginx and php-fpm
//
// If any of the validation steps fail, the partial state is
// rolled back as best-effort and the API returns 500 with the
// command output.
package v1

import (
	"errors"
	"fmt"
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

// DomainDeps bundles dependencies for domain handlers.
type DomainDeps struct {
	DB      *gorm.DB
	Hosting *hosting.Service
	Audit   *audit.Auditor
}

// DomainRow is the persisted domain. We use the Account row as
// the parent and stash the domain as a column on the same row's
// JSON. v0.2.0 keeps the storage simple: the primary "domain"
// lives on Account.Domain; addons/subdomains can come in v0.2.1
// with a separate domain table. To not break the schema, the
// handler below returns a synthetic view of (account_id, name).
type DomainView struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	TenantID    string `json:"tenant_id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	DocumentRoot string `json:"document_root"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// CreateDomainRequest is the POST /accounts/:id/domains body.
type CreateDomainRequest struct {
	Domain string `json:"domain"`
	Port   int    `json:"port,omitempty"` // default 80
}

// CreateDomainHandler — POST /api/v1/accounts/:id/domains
func CreateDomainHandler(d DomainDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		accountID := c.Params("id")
		var acct models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", accountID, claims.TenantID).
			First(&acct).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		var req CreateDomainRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
		if req.Domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_domain")
		}
		if req.Port == 0 {
			req.Port = 80
		}
		if err := d.Hosting.ValidateDomain(req.Domain); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_domain:"+err.Error())
		}

		// 1. Create document root + placeholder index/info.php.
		if err := d.Hosting.CreateDomain(acct.Username, req.Domain); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "docroot_create_failed:"+err.Error())
		}

		// 2. Write nginx vhost.
		vhost, err := hosting.GenerateNginxConfig(hosting.VHostConfig{
			Username:     acct.Username,
			Domain:       req.Domain,
			Port:         req.Port,
			PHP:          true,
			OpenBasedir:  d.Hosting.Paths.DomainDocumentRoot(acct.Username, req.Domain),
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "vhost_generate_failed:"+err.Error())
		}
		if err := d.Hosting.WriteVHostConfig(acct.Username, req.Domain, vhost); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "vhost_write_failed:"+err.Error())
		}

		// 3. Write php-fpm pool.
		pool, err := hosting.GenerateFPMPool(hosting.FPMConfig{
			Username:  acct.Username,
			Domain:    req.Domain,
			PHPVersion: detectPHPVersion(),
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "fpm_generate_failed:"+err.Error())
		}
		if err := d.Hosting.WriteFPMPool(acct.Username, req.Domain, pool); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "fpm_write_failed:"+err.Error())
		}

		// 4. Validate configs.
		if err := d.Hosting.TestNginx(); err != nil {
			// Roll back.
			_ = d.Hosting.RemoveVHostConfig(acct.Username, req.Domain)
			return fiber.NewError(fiber.StatusInternalServerError, "nginx_test_failed:"+err.Error())
		}
		if err := d.Hosting.TestPHP(); err != nil {
			_ = d.Hosting.RemoveVHostConfig(acct.Username, req.Domain)
			_ = d.Hosting.RemoveFPMPool(acct.Username, req.Domain)
			return fiber.NewError(fiber.StatusInternalServerError, "fpm_test_failed:"+err.Error())
		}

		// 5. Reload services.
		if err := d.Hosting.ReloadNginx(); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "nginx_reload_failed:"+err.Error())
		}
		if err := d.Hosting.ReloadPHP(); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "fpm_reload_failed:"+err.Error())
		}

		// 6. Persist: update the account's primary domain column.
		// v0.2.0 keeps it simple — the account has one primary
		// domain. Addons come in v0.2.1 with a separate table.
		acct.Domain = req.Domain
		_ = d.DB.WithContext(c.Context()).Save(&acct).Error

		// 7. Audit.
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "domain.create",
				ResourceType: "domain",
				ResourceID:   req.Domain,
				ResourceName: req.Domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
				Detail:       fmt.Sprintf("account=%s", acct.Username),
			})
		}

		view := DomainView{
			ID:           uuid.NewString(),
			AccountID:    acct.ID,
			TenantID:     acct.TenantID,
			Username:     acct.Username,
			Name:         req.Domain,
			DocumentRoot: d.Hosting.Paths.DomainDocumentRoot(acct.Username, req.Domain),
			Status:       "active",
			CreatedAt:    acct.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		return c.Status(fiber.StatusCreated).JSON(view)
	}
}

// DeleteDomainHandler — DELETE /api/v1/accounts/:id/domains/:domain
func DeleteDomainHandler(d DomainDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		accountID := c.Params("id")
		domain := strings.ToLower(c.Params("domain"))
		var acct models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", accountID, claims.TenantID).
			First(&acct).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		if err := d.Hosting.DeleteDomain(acct.Username, domain); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "domain_delete_failed:"+err.Error())
		}
	_ = d.Hosting.ReloadNginx()
	// Reload php-fpm too so the master process drops the pool
	// config. Without this, userdel refuses the next DeleteAccount
	// call with "user is currently used by process <N>".
	_ = d.Hosting.ReloadPHP()
	if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "domain.delete",
				ResourceType: "domain",
				ResourceID:   domain,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ListDomainsHandler — GET /api/v1/accounts/:id/domains
func ListDomainsHandler(d DomainDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		accountID := c.Params("id")
		var acct models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", accountID, claims.TenantID).
			First(&acct).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "account_not_found")
		}
		// v0.2.0: single primary domain per account.
		if acct.Domain == "" {
			return c.JSON(fiber.Map{"domains": []any{}})
		}
		view := DomainView{
			ID:           acct.ID,
			AccountID:    acct.ID,
			TenantID:     acct.TenantID,
			Username:     acct.Username,
			Name:         acct.Domain,
			DocumentRoot: d.Hosting.Paths.DomainDocumentRoot(acct.Username, acct.Domain),
			Status:       acct.Status,
			CreatedAt:    acct.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		return c.JSON(fiber.Map{"domains": []DomainView{view}})
	}
}

// detectPHPVersion is a tiny shim. v0.2.0 reads ORVIX_FPM_VERSION
// from the env (set by the operator) and defaults to "8.5". The
// install script writes this env to /etc/default/orvixpanel.
func detectPHPVersion() string {
	// Inline import-free: this is a placeholder for the real
	// detection done by the install script. v0.2.1 will read
	// /etc/php/<ver>/fpm/pool.d/ and pick the highest version.
	if v := strings.TrimSpace(getenv("ORVIX_FPM_VERSION")); v != "" {
		return v
	}
	return "8.5"
}

func getenv(k string) string {
	// small wrapper to keep the v1 package's import list thin.
	return osGetenv(k)
}

// osGetenv is a thin re-export so we don't import "os" twice in
// different files. The real call is in osenv.go.
var osGetenv = func(k string) string { return osGetenvImpl(k) }

// avoid unused
var _ = errors.Is
