package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/audit"
)

// claimsKey is the Fiber Locals key used by the api/middleware
// package to stash *Claims. Duplicated here (instead of importing
// the middleware) to avoid an import cycle: auth → middleware → auth.
const claimsKey = "claims"

// APIKeyDeps bundles the dependencies.
type APIKeyDeps struct {
	Keys  *KeyService
	Audit *audit.Auditor
}

// CreateHandler — POST /api/v1/admin/api-keys
func CreateAPIKeyHandler(d APIKeyDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		var req CreateAPIKeyRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		resp, err := d.Keys.Create(c.Context(), claims.TenantID, claims.UserID, req)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "apikey_create_failed:"+err.Error())
		}
		// Audit (no key material in the audit; just metadata)
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "apikey.create",
				ResourceType: "apikey",
				ResourceID:   resp.ID,
				ResourceName: resp.Name,
				Result:       "success",
				UserID:       claims.UserID,
				UserEmail:    claims.Email,
				UserRole:     claims.Role,
				ActorIP:      c.IP(),
			})
		}
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

// ListHandler — GET /api/v1/admin/api-keys
func ListAPIKeyHandler(d APIKeyDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		rows, err := d.Keys.List(c.Context(), claims.TenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "apikey_list_failed")
		}
		// Strip the key hash before returning.
		out := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			out = append(out, map[string]any{
				"id":            r.ID,
				"name":          r.Name,
				"prefix":        r.Prefix,
				"role":          r.Role,
				"scopes":        r.Scopes,
				"expires_at":    r.ExpiresAt,
				"last_used_at":  r.LastUsedAt,
				"last_used_ip":  r.LastUsedIP,
				"revoked_at":    r.RevokedAt,
				"revoke_reason": r.RevokeReason,
				"created_at":    r.CreatedAt,
				"created_by_id": r.CreatedByID,
			})
		}
		return c.JSON(fiber.Map{"api_keys": out})
	}
}

// RevokeHandler — DELETE /api/v1/admin/api-keys/:id
func RevokeAPIKeyHandler(d APIKeyDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		err := d.Keys.Revoke(c.Context(), claims.TenantID, id, "revoked_by_admin")
		if err != nil {
			if errors.Is(err, ErrAPIKeyNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "apikey_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "apikey_revoke_failed")
		}
		if d.Audit != nil {
			_ = d.Audit.Record(c.Context(), audit.Event{
				Action:       "apikey.revoke",
				ResourceType: "apikey",
				ResourceID:   id,
				Result:       "success",
				UserID:       claims.UserID,
				UserEmail:    claims.Email,
				ActorIP:      c.IP(),
			})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}
