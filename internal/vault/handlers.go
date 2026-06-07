package vault

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/quota"
)

// Deps bundles the service dependencies for the Fiber handlers.
type Deps struct {
	Vault *Vault
	Audit *audit.Auditor
	Quota *quota.Service
}

// PutRequest is the POST /vault/secrets body.
type PutRequest struct {
	Name      string `json:"name"`
	Plaintext string `json:"value"`
}

// PutHandler — POST /api/v1/vault/secrets
//
// First PUT creates a new secret (version=1). Subsequent PUTs to
// the same (tenant_id, name) rotate the secret (version++, RotatedAt
// set). On every PUT we check MaxSecrets via the quota service.
func PutHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var req PutRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		if req.Name == "" || req.Plaintext == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_name_or_value")
		}

		// Quota check: only enforce on creation. Rotation is always
		// allowed (we're replacing, not adding).
		existing, _ := d.Vault.Get(c.Context(), claims.TenantID, req.Name)
		if existing == nil && d.Quota != nil {
			ok, reason, err := d.Quota.Check(c.Context(), claims.TenantID, quota.ResourceSecret)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "quota_check_failed")
			}
			if !ok {
				return fiber.NewError(fiber.StatusForbidden, reason)
			}
		}

		meta, err := d.Vault.Put(c.Context(), claims.TenantID, req.Name, req.Plaintext, claims.UserID)
		if err != nil {
			if err == ErrSecretTooLarge {
				return fiber.NewError(fiber.StatusRequestEntityTooLarge, "secret_too_large")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "vault_put_failed")
		}
		RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
			"write", req.Name, true)
		return c.Status(fiber.StatusCreated).JSON(meta)
	}
}

// GetHandler — GET /api/v1/vault/secrets/:name — returns metadata only.
func GetHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		name := c.Params("name")
		if name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_name")
		}
		meta, err := d.Vault.Get(c.Context(), claims.TenantID, name)
		if err != nil {
			if err == ErrSecretNotFound {
				RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
					"read", name, false)
				return fiber.NewError(fiber.StatusNotFound, "secret_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "vault_get_failed")
		}
		RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
			"read", name, true)
		return c.JSON(meta)
	}
}

// ReadHandler — GET /api/v1/vault/secrets/:name/value — returns the
// decrypted plaintext. Audited more loudly than the metadata read.
func ReadHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		name := c.Params("name")
		plain, meta, err := d.Vault.Read(c.Context(), claims.TenantID, name)
		if err != nil {
			if err == ErrSecretNotFound {
				RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
					"reveal", name, false)
				return fiber.NewError(fiber.StatusNotFound, "secret_not_found")
			}
			RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
				"reveal", name, false)
			return fiber.NewError(fiber.StatusInternalServerError, "vault_read_failed")
		}
		RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
			"reveal", name, true)
		_ = meta
		return c.JSON(fiber.Map{
			"name":  name,
			"value": plain,
		})
	}
}

// ListHandler — GET /api/v1/vault/secrets
func ListHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		items, err := d.Vault.List(c.Context(), claims.TenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "vault_list_failed")
		}
		return c.JSON(fiber.Map{"secrets": items})
	}
}

// DeleteHandler — DELETE /api/v1/vault/secrets/:name
func DeleteHandler(d Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		name := c.Params("name")
		if err := d.Vault.Delete(c.Context(), claims.TenantID, name); err != nil {
			if err == ErrSecretNotFound {
				RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
					"delete", name, false)
				return fiber.NewError(fiber.StatusNotFound, "secret_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "vault_delete_failed")
		}
		RecordAccess(c.Context(), d.Audit, claims.TenantID, claims.UserID, claims.Email, claims.Role,
			"delete", name, true)
		return c.SendStatus(fiber.StatusNoContent)
	}
}
