package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/license"
)

// ReadOnlyEnforcer returns 423 (Locked, RFC 4918) for any write
// method (POST/PUT/PATCH/DELETE) when the license is in
// `locked` mode. In `grace` mode the request is allowed but a
// warning header is added. In `active` mode the middleware is a
// no-op.
//
// Health probes + login + token refresh are exempt — operators
// must still be able to log in to read the renewal info.
func ReadOnlyEnforcer() fiber.Handler {
	exempt := map[string]bool{
		"/healthz":                            true,
		"/readyz":                             true,
		"/auth/login":                         true,
		"/auth/refresh":                       true,
		"/api/v1/admin/license/renewal-info":  true,
		"/api/v1/admin/license":               true, // GET only; controller also rejects writes
	}
	writeMethods := map[string]bool{
		"POST":   true,
		"PUT":    true,
		"PATCH":  true,
		"DELETE": true,
	}
	return func(c *fiber.Ctx) error {
		mode := license.CurrentMode()
		if mode == license.ModeActive {
			return c.Next()
		}
		if !writeMethods[c.Method()] {
			return c.Next()
		}
		if exempt[c.Path()] {
			return c.Next()
		}
		// /api/v1/admin/license accepts GET only — handler enforces.
		// For other write methods, return 423.
		if mode == license.ModeLocked {
			c.Set("X-Orvix-License-Mode", string(mode))
			return fiber.NewError(fiber.StatusLocked, "license_expired_panel_locked")
		}
		// Grace mode: allow but warn.
		c.Set("X-Orvix-License-Mode", string(mode))
		c.Set("X-Orvix-License-Warning", "license_in_grace_period_renew_soon")
		return c.Next()
	}
}

// isWritePath is a helper for tests / debug.
func isWritePath(c *fiber.Ctx) bool {
	return strings.EqualFold(c.Method(), "POST") ||
		strings.EqualFold(c.Method(), "PUT") ||
		strings.EqualFold(c.Method(), "PATCH") ||
		strings.EqualFold(c.Method(), "DELETE")
}

var _ = isWritePath
