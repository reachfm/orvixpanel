package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/rs/zerolog/log"
)

// AuthMiddleware verifies the JWT and looks up the session. On
// success it puts the *auth.Claims into c.Locals("claims").
func AuthMiddleware(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			header = c.Query("access_token", "")
		}
		token := ""
		const prefix = "Bearer "
		if strings.HasPrefix(header, prefix) {
			token = strings.TrimSpace(header[len(prefix):])
		} else {
			token = header
		}
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing_bearer_token")
		}
		claims, err := svc.VerifyAndCheckSession(c.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrExpiredToken):
				return fiber.NewError(fiber.StatusUnauthorized, "token_expired")
			case errors.Is(err, auth.ErrSessionRevoked):
				return fiber.NewError(fiber.StatusForbidden, "session_revoked")
			default:
				log.Debug().Err(err).Msg("token verify failed")
				return fiber.NewError(fiber.StatusUnauthorized, "invalid_token")
			}
		}
		c.Locals(LocalClaims, claims)
		return c.Next()
	}
}
