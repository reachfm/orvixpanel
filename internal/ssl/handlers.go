package ssl

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// HandlerDeps holds dependencies for SSL handlers.
type HandlerDeps struct {
	DB     *gorm.DB
	Audit  interface{} // Will be *audit.Service
	Quota  interface{} // Will be *quota.Service
}

// SSLDeps holds SSL-specific dependencies.
type SSLDeps struct {
	DB      *gorm.DB
	Audit   *audit.Auditor
	Manager *Manager // Optional: if provided, handlers will use this instead of creating a new one
}

// ListCertificatesHandler returns all SSL certificates.
// GET /api/v1/ssl
func ListCertificatesHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		var certs []models.SSLCertificate
		query := deps.DB.WithContext(c.Context())

		// Tenant isolation
		if claims.TenantID != "" {
			query = query.Where("tenant_id = ?", claims.TenantID)
		}

		if err := query.Find(&certs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_list_failed")
		}

		return c.JSON(certs)
	}
}

// GetCertificateHandler returns a single SSL certificate.
// GET /api/v1/ssl/:id
func GetCertificateHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		certID := c.Params("id")

		var cert models.SSLCertificate
		if err := deps.DB.WithContext(c.Context()).First(&cert, "id = ?", certID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusNotFound, "ssl_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_get_failed")
		}

		return c.JSON(cert)
	}
}

// IssueCertificateRequest represents a certificate issuance request.
type IssueCertificateRequest struct {
	Domain     string   `json:"domain"`
	SANs       []string `json:"san_names,omitempty"`
	Provider   string   `json:"provider"`
	AutoRenew  bool     `json:"auto_renew"`
	ACMEAccountID string `json:"acme_account_id,omitempty"`
}

// IssueCertificateHandler issues a new SSL certificate.
// POST /api/v1/ssl
func IssueCertificateHandler(deps SSLDeps, manager *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req IssueCertificateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if req.Domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_required")
		}

		if req.Provider == "" {
			req.Provider = models.ProviderLetsEncrypt
		}

		// Create manager for issuance
		cfg := DefaultConfig()
		mgr := NewManager(deps.DB, cfg)

		issueReq := &IssueRequest{
			Domain:         req.Domain,
			SANs:           req.SANs,
			ACMEAccountID:  req.ACMEAccountID,
			Provider:       req.Provider,
		}

		cert, err := mgr.IssueCertificate(c.Context(), issueReq)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_issue_failed")
		}

		return c.Status(fiber.StatusCreated).JSON(cert)
	}
}

// RenewCertificateHandler renews an existing certificate.
// POST /api/v1/ssl/:id/renew
func RenewCertificateHandler(deps SSLDeps, manager *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		certID := c.Params("id")

		cfg := DefaultConfig()
		mgr := NewManager(deps.DB, cfg)

		cert, err := mgr.RenewCertificate(c.Context(), certID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_renew_failed")
		}

		return c.JSON(cert)
	}
}

// RevokeCertificateHandler revokes a certificate.
// POST /api/v1/ssl/:id/revoke
func RevokeCertificateHandler(deps SSLDeps, manager *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		certID := c.Params("id")

		cfg := DefaultConfig()
		mgr := NewManager(deps.DB, cfg)

		if err := mgr.RevokeCertificate(c.Context(), certID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_revoke_failed")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteCertificateHandler deletes a certificate.
// DELETE /api/v1/ssl/:id
func DeleteCertificateHandler(deps SSLDeps, manager *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		certID := c.Params("id")

		cfg := DefaultConfig()
		mgr := NewManager(deps.DB, cfg)

		if err := mgr.DeleteCertificate(c.Context(), certID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_delete_failed")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ImportCertificateRequest represents an import request.
type ImportCertificateRequest struct {
	Domain   string `json:"domain"`
	CertPEM  string `json:"cert_pem"`
	KeyPEM   string `json:"key_pem"`
	ChainPEM string `json:"chain_pem,omitempty"`
}

// ImportCertificateHandler imports an existing certificate.
// POST /api/v1/ssl/import
func ImportCertificateHandler(deps SSLDeps, manager *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		var req ImportCertificateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if req.Domain == "" || req.CertPEM == "" || req.KeyPEM == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_and_pem_required")
		}

		// Use injected Manager, or create one with default config
		var mgr *Manager
		if manager != nil {
			mgr = manager
		} else if deps.Manager != nil {
			mgr = deps.Manager
		} else {
			cfg := DefaultConfig()
			mgr = NewManager(deps.DB, cfg)
		}

		chainPEM := req.ChainPEM

		cert, err := mgr.ImportCertificate(c.Context(), claims.TenantID, req.Domain, req.CertPEM, req.KeyPEM, chainPEM)
		if err != nil {
			if deps.Audit != nil {
				_ = deps.Audit.Record(c.Context(), audit.Event{
					Action:       "ssl.certificate.import",
					ResourceType: "ssl_certificate",
					ResourceID:   "",
					ResourceName: req.Domain,
					Result:       "failure",
					Detail:       err.Error(),
					UserID:       claims.UserID,
					ActorIP:      c.IP(),
				})
			}
			return fiber.NewError(fiber.StatusBadRequest, "ssl_import_failed: "+err.Error())
		}

		if deps.Audit != nil {
			_ = deps.Audit.Record(c.Context(), audit.Event{
				Action:       "ssl.certificate.import",
				ResourceType: "ssl_certificate",
				ResourceID:   cert.ID,
				ResourceName: cert.CommonName,
				Result:       "success",
				UserID:       claims.UserID,
				ActorIP:      c.IP(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(cert)
	}
}

// GetCertificateEventsHandler returns events for a certificate.
// GET /api/v1/ssl/:id/events
func GetCertificateEventsHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		certID := c.Params("id")

		var events []models.SSLEvent
		if err := deps.DB.WithContext(c.Context()).
			Where("certificate_id = ?", certID).
			Order("created_at DESC").
			Find(&events).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_events_failed")
		}

		return c.JSON(events)
	}
}

// GetSSLEventsHandler returns all SSL events.
// GET /api/v1/ssl/events
func GetSSLEventsHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 100)

		var events []models.SSLEvent
		if err := deps.DB.WithContext(c.Context()).
			Order("created_at DESC").
			Limit(limit).
			Find(&events).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_events_failed")
		}

		return c.JSON(events)
	}
}

// GetHealthHandler returns SSL health status.
// GET /api/v1/ssl/health
func GetHealthHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scanner := NewHealthScanner(deps.DB)
		report, err := scanner.ScanAll(context.Background())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "ssl_health_failed")
		}

		return c.JSON(report)
	}
}

// GetDashboardStatsHandler returns dashboard statistics.
type DashboardStats struct {
	TotalActive    int64 `json:"total_active"`
	ExpiringSoon   int64 `json:"expiring_soon"`
	FailedRenewals int64 `json:"failed_renewals"`
	AutoRenewCount int64 `json:"auto_renew_enabled"`
}

// GetDashboardStatsHandler returns SSL dashboard statistics.
// GET /api/v1/ssl/dashboard
func GetDashboardStatsHandler(deps SSLDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats := &DashboardStats{}

		// Count active certificates
		deps.DB.WithContext(c.Context()).Model(&models.SSLCertificate{}).
			Where("status = ?", models.CertStatusIssued).
			Count(&stats.TotalActive)

		// Count expiring soon (within 30 days)
		deps.DB.WithContext(c.Context()).Model(&models.SSLCertificate{}).
			Where("status = ?", models.CertStatusExpiringSoon).
			Count(&stats.ExpiringSoon)

		// Count failed
		deps.DB.WithContext(c.Context()).Model(&models.SSLCertificate{}).
			Where("status = ?", models.CertStatusFailed).
			Count(&stats.FailedRenewals)

		// Count auto-renew enabled
		deps.DB.WithContext(c.Context()).Model(&models.SSLCertificate{}).
			Where("auto_renew = ?", true).
			Count(&stats.AutoRenewCount)

		return c.JSON(stats)
	}
}

// IssueStagingCertificateRequest represents a staging certificate issuance request.
type IssueStagingCertificateRequest struct {
	Domain string   `json:"domain"`
	Email  string   `json:"email"`
	SANs   []string `json:"san_names,omitempty"`
}

// IssueStagingCertificateResponse represents the response for staging certificate issuance.
type IssueStagingCertificateResponse struct {
	CertificateID string   `json:"certificate_id"`
	Domain       string   `json:"domain"`
	Status       string   `json:"status"`
	Provider     string   `json:"provider"`
	IsStaging    bool     `json:"is_staging"`
	IssuedAt     string   `json:"issued_at,omitempty"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	Fingerprint  string   `json:"fingerprint,omitempty"`
	SerialNumber string   `json:"serial_number,omitempty"`
	Message      string   `json:"message,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// IssueStagingCertificateHandler issues a certificate using Let's Encrypt staging.
// POST /api/v1/ssl/certificates/issue
func IssueStagingCertificateHandler(deps SSLDeps, challengeStore *ChallengeStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req IssueStagingCertificateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
		}

		if req.Domain == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_required")
		}

		if req.Email == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email_required")
		}

		// Use staging configuration
		cfg := StagingConfig()
		cfg.LetsEncryptEmail = req.Email

		// Create staging provider
		stagingProvider := NewStagingProvider(cfg, challengeStore)

		// Create manager with staging provider
		cfg.UseStaging = true
		_ = &StagingManager{
			db:         deps.DB,
			config:     cfg,
			provider:   stagingProvider,
			challenge:  challengeStore,
		}

		// Issue certificate via staging
		result, err := stagingProvider.IssueCertificate(c.Context(), IssueRequest{
			Domain:   req.Domain,
			SANs:     req.SANs,
			Provider: ProviderNameLetsEncryptStaging,
		})

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "staging_issue_failed: "+err.Error())
		}

		// Create certificate record
		cert := &models.SSLCertificate{
			CommonName:  req.Domain,
			Provider:    ProviderNameLetsEncryptStaging,
			Status:      models.CertStatusIssued,
			AutoRenew:   false, // Staging certs should not auto-renew
			TenantID:    "system",
			Fingerprint: result.Fingerprint,
			SerialNumber: result.SerialNum,
		}

		if result.NotAfter.After(time.Now()) {
			cert.ExpiresAt = &result.NotAfter
		}

		if err := deps.DB.WithContext(c.Context()).Create(cert).Error; err != nil {
			// Log but don't fail
		}

		return c.Status(fiber.StatusCreated).JSON(IssueStagingCertificateResponse{
			CertificateID: cert.ID,
			Domain:        req.Domain,
			Status:        string(models.CertStatusIssued),
			Provider:      ProviderNameLetsEncryptStaging,
			IsStaging:     true,
			IssuedAt:      time.Now().Format(time.RFC3339),
			ExpiresAt:     result.NotAfter.Format(time.RFC3339),
			Fingerprint:   result.Fingerprint,
			SerialNumber:   result.SerialNum,
			Message:       "Certificate issued using Let's Encrypt STAGING. NOT FOR PRODUCTION USE.",
			Warnings: []string{
				"This is a STAGING certificate issued by Let's Encrypt",
				"Browsers will show security warnings",
				"Do not use in production environments",
			},
		})
	}
}

// StagingManager wraps the SSL manager for staging operations.
type StagingManager struct {
	db        *gorm.DB
	config    *Config
	provider  *StagingProvider
	challenge *ChallengeStore
}

// Note: ACMEChallengeHandler is defined in acme_handler.go
// This avoids duplicate declarations