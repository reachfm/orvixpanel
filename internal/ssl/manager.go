package ssl

import (
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"context"
	"gorm.io/gorm"
	"strings"
	"time"
)

// Manager orchestrates SSL certificate operations.
type Manager struct {
	db            *gorm.DB
	config       *Config
	provider     Provider
	storage      *Storage
	validator    *Validator
	healthScanner *HealthScanner
	scheduler    *RenewScheduler
	nginx        *NginxIntegration
	challenge    *ChallengeHandler
}

// NewManager creates a new SSL manager.
func NewManager(db *gorm.DB, config *Config) *Manager {
	storage := NewStorage(config.StorageDir)
	validator := NewValidator(storage)
	healthScanner := NewHealthScanner(db)
	nginx := NewNginxIntegration(config.NginxConfigDir, config.NginxBackupDir, config.StorageDir)
	challenge := NewChallengeHandler(config.ChallengeDir, storage)

	// Create Let's Encrypt provider
	provider := NewLetsEncryptProvider(config)

	manager := &Manager{
		db:            db,
		config:       config,
		provider:     provider,
		storage:      storage,
		validator:    validator,
		healthScanner: healthScanner,
		nginx:        nginx,
		challenge:    challenge,
	}

	// Set up scheduler with manager reference for nginx updates
	scheduler := NewRenewScheduler(db, config, provider, storage)
	scheduler.SetManager(manager)
	manager.scheduler = scheduler

	return manager
}

// IssueCertificate issues a new SSL certificate.
func (m *Manager) IssueCertificate(ctx context.Context, req *IssueRequest) (*models.SSLCertificate, error) {
	// Create certificate record in pending state
	cert := &models.SSLCertificate{
		CommonName:  req.Domain,
		SANNames:    joinStrings(req.SANs, ","),
		Provider:    req.Provider,
		Status:      models.CertStatusPending,
		AutoRenew:   true,
		TenantID:    "system", // Would be set from context in production
	}

	if err := m.db.WithContext(ctx).Create(cert).Error; err != nil {
		return nil, &Error{Op: "create cert record", Err: err}
	}

	// Log event
	m.logEvent(ctx, cert.ID, "issue_started", "Certificate issuance started", "")

	// Issue certificate via provider
	result, err := m.provider.IssueCertificate(ctx, *req)
	if err != nil {
		cert.Status = models.CertStatusFailed
		cert.LastError = err.Error()
		m.db.Save(cert)
		m.logEvent(ctx, cert.ID, "issue_failed", "Certificate issuance failed", err.Error())
		return nil, &Error{Op: "issue certificate", Err: err}
	}

	// Store certificate files
	paths, err := m.storage.WriteCertFiles(req.Domain, result)
	if err != nil {
		cert.Status = models.CertStatusFailed
		cert.LastError = err.Error()
		m.db.Save(cert)
		m.logEvent(ctx, cert.ID, "storage_failed", "Failed to store certificate files", err.Error())
		return nil, &Error{Op: "store certificate", Err: err}
	}

	// Update certificate record
	now := time.Now()
	cert.CertPath = paths.CertPath
	cert.KeyPath = paths.KeyPath
	cert.ChainPath = paths.ChainPath
	cert.FullChainPath = paths.FullChainPath
	cert.IssuedAt = &now
	cert.ExpiresAt = &result.NotAfter
	cert.SerialNumber = result.SerialNum
	cert.Fingerprint = result.Fingerprint
	cert.Status = models.CertStatusIssued
	cert.RenewalAttempts = 0
	m.db.Save(cert)

	// Update nginx vhost
	if err := m.UpdateNginxVhost(req.Domain); err != nil {
		m.logEvent(ctx, cert.ID, "nginx_update_failed", "Failed to update nginx vhost", err.Error())
		// Don't fail the issuance - nginx can be updated manually
	} else {
		m.logEvent(ctx, cert.ID, "nginx_updated", "Nginx vhost configured with SSL", "")
	}

	m.logEvent(ctx, cert.ID, "issued", "Certificate issued successfully", "")

	return cert, nil
}

// RenewCertificate renews an existing certificate.
func (m *Manager) RenewCertificate(ctx context.Context, certID string) (*models.SSLCertificate, error) {
	var cert models.SSLCertificate
	if err := m.db.WithContext(ctx).First(&cert, "id = ?", certID).Error; err != nil {
		return nil, &Error{Op: "find certificate", Err: err}
	}

	// Log start
	m.logEvent(ctx, cert.ID, "renewal_started", "Manual renewal started", "")

	// Parse SANs
	var sans []string
	if cert.SANNames != "" {
		sans = splitStrings(cert.SANNames)
	}

	// Renew via provider
	result, err := m.provider.RenewCertificate(ctx, certID, IssueRequest{
		Domain:        cert.CommonName,
		SANs:          sans,
		ACMEAccountID: cert.ACMEAccountID,
		Provider:      cert.Provider,
	})

	if err != nil {
		cert.LastError = err.Error()
		cert.RenewalAttempts++
		cert.LastRenewalAttempt = timePtr(time.Now())
		m.db.Save(cert)
		m.logEvent(ctx, cert.ID, "renewal_failed", "Certificate renewal failed", err.Error())
		return nil, &Error{Op: "renew certificate", Err: err}
	}

	// Store new certificate files
	paths, err := m.storage.WriteCertFiles(cert.CommonName, result)
	if err != nil {
		cert.LastError = err.Error()
		cert.RenewalAttempts++
		cert.LastRenewalAttempt = timePtr(time.Now())
		m.db.Save(cert)
		m.logEvent(ctx, cert.ID, "storage_failed", "Failed to store renewed certificate", err.Error())
		return nil, &Error{Op: "store renewed cert", Err: err}
	}

	// Update certificate record
	now := time.Now()
	cert.CertPath = paths.CertPath
	cert.KeyPath = paths.KeyPath
	cert.ChainPath = paths.ChainPath
	cert.FullChainPath = paths.FullChainPath
	cert.IssuedAt = &now
	cert.ExpiresAt = &result.NotAfter
	cert.SerialNumber = result.SerialNum
	cert.Fingerprint = result.Fingerprint
	cert.LastRenewalAt = &now
	cert.LastError = ""
	cert.RenewalAttempts = 0
	cert.Status = models.CertStatusIssued
	m.db.Save(cert)

	// Update nginx
	if err := m.UpdateNginxVhost(cert.CommonName); err != nil {
		m.logEvent(ctx, cert.ID, "nginx_update_failed", "Failed to update nginx", err.Error())
	} else {
		m.logEvent(ctx, cert.ID, "nginx_updated", "Nginx vhost updated with renewed certificate", "")
	}

	m.logEvent(ctx, cert.ID, "renewed", "Certificate renewed successfully", "")

	return &cert, nil
}

// RevokeCertificate revokes a certificate.
func (m *Manager) RevokeCertificate(ctx context.Context, certID string) error {
	var cert models.SSLCertificate
	if err := m.db.WithContext(ctx).First(&cert, "id = ?", certID).Error; err != nil {
		return &Error{Op: "find certificate", Err: err}
	}

	// Revoke via provider
	if err := m.provider.RevokeCertificate(ctx, cert.CertPath, cert.KeyPath); err != nil {
		m.logEvent(ctx, cert.ID, "revoke_failed", "Provider revoke failed", err.Error())
		// Continue with local revocation even if provider fails
	}

	// Update status
	cert.Status = models.CertStatusRevoked
	m.db.Save(cert)

	// Remove nginx SSL config
	if err := m.nginx.RemoveSSLVhost(cert.CommonName); err != nil {
		m.logEvent(ctx, cert.ID, "nginx_removal_failed", "Failed to remove nginx SSL config", err.Error())
	}

	m.logEvent(ctx, cert.ID, "revoked", "Certificate revoked", "")

	return nil
}

// DeleteCertificate deletes a certificate.
func (m *Manager) DeleteCertificate(ctx context.Context, certID string) error {
	var cert models.SSLCertificate
	if err := m.db.WithContext(ctx).First(&cert, "id = ?", certID).Error; err != nil {
		return &Error{Op: "find certificate", Err: err}
	}

	// Remove nginx SSL config
	m.nginx.RemoveSSLVhost(cert.CommonName)

	// Delete certificate files
	if err := m.storage.DeleteCertFiles(cert.CommonName); err != nil {
		m.logEvent(ctx, cert.ID, "file_delete_warning", "Failed to delete some certificate files", err.Error())
	}

	// Delete certificate record
	m.logEvent(ctx, cert.ID, "deleted", "Certificate deleted", "")
	m.db.Delete(&cert)

	return nil
}

// ImportCertificate imports an existing certificate.
func (m *Manager) ImportCertificate(ctx context.Context, domain, certPEM, keyPEM, chainPEM string) (*models.SSLCertificate, error) {
	// Parse and validate certificate
	certInfo, err := m.provider.ValidateCertificate("")
	if err != nil {
		return nil, &Error{Op: "validate imported cert", Err: err}
	}

	// Store certificate files
	paths, err := m.storage.ImportCertFiles(domain, certPEM, keyPEM, chainPEM)
	if err != nil {
		return nil, &Error{Op: "import certificate files", Err: err}
	}

	// Create certificate record
	cert := &models.SSLCertificate{
		CommonName:   domain,
		Provider:     "imported",
		Status:       models.CertStatusIssued,
		AutoRenew:    false,
		CertPath:     paths.CertPath,
		KeyPath:      paths.KeyPath,
		ChainPath:    paths.ChainPath,
		FullChainPath: paths.FullChainPath,
		IssuedAt:     &certInfo.NotBefore,
		ExpiresAt:    &certInfo.NotAfter,
		SerialNumber: certInfo.SerialNumber,
		Fingerprint: certInfo.Fingerprint,
		TenantID:     "system",
	}

	if err := m.db.WithContext(ctx).Create(cert).Error; err != nil {
		// Cleanup files
		m.storage.DeleteCertFiles(domain)
		return nil, &Error{Op: "create cert record", Err: err}
	}

	// Update nginx
	if err := m.UpdateNginxVhost(domain); err != nil {
		m.logEvent(ctx, cert.ID, "nginx_update_failed", "Failed to update nginx", err.Error())
	}

	m.logEvent(ctx, cert.ID, "imported", "Certificate imported", "")

	return cert, nil
}

// UpdateNginxVhost updates nginx vhost with SSL configuration.
func (m *Manager) UpdateNginxVhost(domain string) error {
	result, err := m.nginx.UpdateVhostSSL(context.Background(), domain)
	if err != nil {
		return err
	}

	if result.BackupPath != "" {
		// Store backup path for potential rollback
		// Would typically store in certificate record
	}

	return nil
}

// GetCertificate retrieves a certificate by ID.
func (m *Manager) GetCertificate(ctx context.Context, certID string) (*models.SSLCertificate, error) {
	var cert models.SSLCertificate
	if err := m.db.WithContext(ctx).First(&cert, "id = ?", certID).Error; err != nil {
		return nil, &Error{Op: "find certificate", Err: err}
	}
	return &cert, nil
}

// ListCertificates returns all certificates.
func (m *Manager) ListCertificates(ctx context.Context) ([]*models.SSLCertificate, error) {
	var certs []*models.SSLCertificate
	if err := m.db.WithContext(ctx).Find(&certs).Error; err != nil {
		return nil, &Error{Op: "list certificates", Err: err}
	}
	return certs, nil
}

// GetHealthReport returns certificate health report.
func (m *Manager) GetHealthReport(ctx context.Context) (*HealthScanReport, error) {
	return m.healthScanner.ScanAll(ctx)
}

// logEvent logs an SSL event.
func (m *Manager) logEvent(ctx context.Context, certID, eventType, message, errorDetail string) {
	event := &models.SSLEvent{
		CertificateID: certID,
		EventType:     eventType,
		Message:      message,
		ErrorDetail:  errorDetail,
	}
	m.db.WithContext(ctx).Create(event)
}

// joinStrings joins a slice of strings with commas.
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}

// splitStrings splits a comma-separated string into a slice.
func splitStrings(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}