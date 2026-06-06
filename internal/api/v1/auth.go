// Package v1 holds the v1.0 HTTP handlers. The file split:
//
//	auth.go       — login, refresh, logout, me
//	admin.go      — /admin/system, /admin/license, /admin/audit-log
//	stubs.go      — 501 Not Implemented placeholders for spec routes
//	              that arrive in v1.1+
//
// Every handler is a `func(c *fiber.Ctx) error` and returns stable
// error messages so the frontend can switch on them.
package v1

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/auth"
)

// LoginRequest is the POST body for /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginHandler — POST /auth/login.
func LoginHandler(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		if req.Email == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing_credentials")
		}

		res, err := svc.Login(c.Context(), req.Email, req.Password, c.IP())
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidCredentials):
				return fiber.NewError(fiber.StatusUnauthorized, "invalid_credentials")
			case errors.Is(err, auth.ErrUserLocked):
				return fiber.NewError(fiber.StatusTooManyRequests, "account_locked")
			case errors.Is(err, auth.ErrUserSuspended):
				return fiber.NewError(fiber.StatusForbidden, "account_suspended")
			default:
				return fiber.NewError(fiber.StatusInternalServerError, "login_failed")
			}
		}
		return c.JSON(fiber.Map{
			"access_token":  res.Tokens.AccessToken,
			"refresh_token": res.Tokens.RefreshToken,
			"expires_at":    res.Tokens.ExpiresAt,
			"user": fiber.Map{
				"id":    res.User.ID,
				"email": res.User.Email,
				"role":  res.User.Role,
			},
		})
	}
}

// RefreshRequest is the POST body for /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshHandler — POST /auth/refresh.
func RefreshHandler(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}
		pair, err := svc.Refresh(c.Context(), req.RefreshToken, c.IP())
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid_refresh")
		}
		return c.JSON(fiber.Map{
			"access_token":  pair.AccessToken,
			"refresh_token": pair.RefreshToken,
			"expires_at":    pair.ExpiresAt,
		})
	}
}

// LogoutHandler — POST /auth/logout.
func LogoutHandler(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		if err := svc.Logout(c.Context(), claims); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "logout_failed")
		}
		return c.JSON(fiber.Map{"logged_out": true})
	}
}

// MeHandler — GET /api/v1/me. Returns the caller's claims.
func MeHandler(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
	if !ok {
		return fiber.ErrUnauthorized
	}
	return c.JSON(fiber.Map{
		"user_id":    claims.UserID,
		"email":      claims.Email,
		"role":       claims.Role,
		"tenant_id":  claims.TenantID,
		"account_id": claims.AccountID,
		"session_id": claims.SessionID,
	})
}
