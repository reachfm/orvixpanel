package middleware

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/rs/zerolog/log"
)

// RequestIDMiddleware tags every request with a UUID correlation id
// (X-Request-ID header) used by structured logs + the audit chain.
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Locals("request_id", id)
		c.Set("X-Request-ID", id)
		return c.Next()
	}
}

// AccessLogMiddleware — one structured line per request.
func AccessLogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		dur := time.Since(start)

		ev := log.Info()
		if err != nil {
			ev = log.Warn().Err(err)
		}
		ev.
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Str("ip", c.IP()).
			Int("duration_ms", int(dur.Milliseconds())).
			Str("request_id", c.Locals("request_id").(string)).
			Msg("http")
		return err
	}
}

// AuditMiddleware records the request to the audit log. Skips
// health probes + the public theme endpoint (no useful action).
func AuditMiddleware(a *audit.Auditor) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		path := c.Path()
		switch path {
		case "/healthz", "/readyz", "/api/v1/public/theme":
			return err
		}

		ev := audit.Event{
			Action:     c.Method() + " " + path,
			Result:     "success",
			DurationMS: int(time.Since(start).Milliseconds()),
			ActorIP:    c.IP(),
		}
		if claims, ok := c.Locals(LocalClaims).(*auth.Claims); ok && claims != nil {
			ev.UserID = claims.UserID
			ev.UserEmail = claims.Email
			ev.UserRole = claims.Role
			ev.SessionID = claims.SessionID
		}
		if err != nil {
			ev.Result = "failure"
			var fe *fiber.Error
			if errors.As(err, &fe) {
				if fe.Code == fiber.StatusUnauthorized || fe.Code == fiber.StatusForbidden {
					ev.Result = "denied"
				}
				ev.Detail = strconv.Itoa(fe.Code) + " " + fe.Message
			} else {
				ev.Detail = err.Error()
			}
		}
		if recErr := a.Record(c.Context(), ev); recErr != nil {
			log.Error().Err(recErr).Msg("audit record failed")
		}
		return err
	}
}
