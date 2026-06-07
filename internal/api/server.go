// Package api wires the Fiber v2 HTTP server. v0.3.0 ships:
//   - JWT auth (login/refresh/logout) + API key auth
//   - RBAC (built-in + custom) + license gating middleware
//   - Token-bucket rate limiter
//   - Audit log with hash chain + search + CEF export
//   - Encrypted secrets vault (AES-256-GCM)
//   - Tenant quotas
//   - Encrypted license persistence + read-only mode
//
// Routes that the spec calls for in later phases (DNS, mail, SSL,
// firewall, files, backups, WAF, Guardian, reseller, provisioning)
// return 501 Not Implemented — see stubs.go.
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/api/v1"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/hosting"
	"github.com/orvixpanel/orvixpanel/internal/license"
	"github.com/orvixpanel/orvixpanel/internal/quota"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
	"github.com/orvixpanel/orvixpanel/internal/vault"
	"github.com/orvixpanel/orvixpanel/internal/web"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Deps is the constructor input.
type Deps struct {
	Config       *config.Config
	DB           *gorm.DB
	Auth         *auth.Service
	Audit        *audit.Auditor
	LicenseStore *license.Store
	RBAC         *rbac.Service
	Vault        *vault.Vault
	Quota        *quota.Service
	APIKeys      *auth.KeyService
	Hosting      *hosting.Service
}

// Server is the *fiber.App wrapper.
type Server struct {
	app  *fiber.App
	deps Deps
}

// NewServer builds the Fiber app with the full middleware stack.
func NewServer(d Deps) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "orvixpanel",
		DisableStartupMessage: true,
		ReadTimeout:           d.Config.Server.ReadTimeout,
		WriteTimeout:          d.Config.Server.WriteTimeout,
	})

	rl := middleware.NewRateLimiter(100.0/60.0, 30.0) // 100/min, burst 30
	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.AccessLogMiddleware())
	app.Use(rl.Middleware())
	app.Use(recoverMiddleware())
	app.Use(securityHeadersMiddleware())
	app.Use(depsMiddleware(d)) // injects db/auditor into Locals
	app.Use(middleware.AuditMiddleware(d.Audit))

	// Health probes.
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", func(c *fiber.Ctx) error {
		sqlDB, err := d.DB.DB()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db_unavailable"})
		}
		pingCtx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db_unavailable"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// Public auth.
	app.Post("/auth/login", v1.LoginHandler(d.Auth))
	app.Post("/auth/refresh", v1.RefreshHandler(d.Auth))

	// Authenticated auth.
	authGrp := app.Group("/auth", middleware.AuthMiddleware(d.Auth))
	authGrp.Post("/logout", v1.LogoutHandler(d.Auth))

	// Authenticated v1 API. Middleware order:
	//   1. ReadOnlyEnforcer — license-expired panels reject writes
	//   2. APIKeyMiddleware — try API-key auth first
	//   3. AuthMiddleware   — fall back to JWT
	//   4. TenantMiddleware — license + tenant scope check
	v1grp := app.Group("/api/v1",
		middleware.ReadOnlyEnforcer(),
		middleware.APIKeyMiddleware(d.APIKeys),
		middleware.AuthMiddleware(d.Auth),
		middleware.TenantMiddleware(),
	)
	registerV1(v1grp, d)

	log.Info().Int("routes", len(app.GetRoutes())).Msg("http server initialized")

	// Mount the Enterprise UI last so the API routes take priority.
	// The web package's catch-all only fires for paths the API
	// didn't already match.
	web.RegisterRoutes(app)

	return &Server{app: app, deps: d}
}

// Listen starts the HTTP server. Blocks until Shutdown. v1.0 does
// NOT support TLS termination in the Go binary — the install guide
// puts a reverse proxy (nginx, Caddy) in front. The ListenTLS path
// lands in v1.1 alongside the cert manager integration.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// ShutdownWithContext gracefully drains in-flight requests.
func (s *Server) ShutdownWithContext(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

// App exposes the underlying *fiber.App for tests.
func (s *Server) App() *fiber.App { return s.app }

// errorHandler — converts *fiber.Error into a stable JSON shape.
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal_error"
	if fe, ok := err.(*fiber.Error); ok {
		code = fe.Code
		msg = fe.Message
	} else {
		log.Error().Err(err).Str("path", c.Path()).Msg("unhandled error")
	}
	return c.Status(code).JSON(fiber.Map{
		"error":      msg,
		"request_id": c.Locals("request_id"),
	})
}

// recoverMiddleware catches panics.
func recoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("path", c.Path()).
					Msg("panic recovered")
				err = fiber.NewError(fiber.StatusInternalServerError, "internal_panic")
			}
		}()
		return c.Next()
	}
}

// securityHeadersMiddleware sets the baseline security headers on
// every response.
func securityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

// buildInfo is exposed via v1.BuildInfo — the v1 package owns the
// /admin/system response shape.
//
// keep fmt imported for future debug prints.
var _ = fmt.Sprintf
