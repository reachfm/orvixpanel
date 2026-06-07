package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/rs/zerolog/log"
)

// APIKeyMiddleware attempts to authenticate using an API key. If
// the request doesn't carry one, it falls through to the next
// handler (which should be AuthMiddleware for JWT). If a key is
// presented and valid, it sets c.Locals(LocalClaims) and short-
// circuits the JWT path.
//
// Header precedence:
//   1. X-Orvix-Api-Key: orx_live_...
//   2. Authorization: Bearer orx_live_...
//
// Anything else falls through.
func APIKeyMiddleware(svc *auth.KeyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-Orvix-Api-Key")
		if key == "" {
			h := c.Get("Authorization")
			if strings.HasPrefix(h, "Bearer ") {
				tok := strings.TrimSpace(h[len("Bearer "):])
				if strings.HasPrefix(tok, auth.KeyPrefix) {
					key = tok
				}
			}
		}
		if key == "" {
			return c.Next()
		}
		row, err := svc.Verify(c.Context(), key, c.IP())
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrAPIKeyRevoked):
				return fiber.NewError(fiber.StatusUnauthorized, "apikey_revoked")
			case errors.Is(err, auth.ErrAPIKeyExpired):
				return fiber.NewError(fiber.StatusUnauthorized, "apikey_expired")
			case errors.Is(err, auth.ErrAPIKeyNotFound):
				return fiber.NewError(fiber.StatusUnauthorized, "apikey_invalid")
			default:
				log.Debug().Err(err).Msg("api key verify failed")
				return fiber.NewError(fiber.StatusUnauthorized, "apikey_invalid")
			}
		}
		c.Locals(LocalClaims, auth.ClaimsForKey(row))
		c.Locals("apikey", row)
		return c.Next()
	}
}
