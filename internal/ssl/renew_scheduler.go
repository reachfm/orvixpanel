package ssl

import (
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"context"
	"gorm.io/gorm"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RenewScheduler handles automatic certificate renewal.
type RenewScheduler struct {
	db       *gorm.DB
	config   *Config
	provider Provider
	storage  *Storage
	manager  *Manager
}

// NewRenewScheduler creates a new renewal scheduler.
func NewRenewScheduler(db *gorm.DB, config *Config, provider Provider, storage *Storage) *RenewScheduler {
	return &RenewScheduler{
		db:       db,
		config:   config,
		provider: provider,
		storage:  storage,
	}
}

// SetManager sets the SSL manager for scheduler operations.
func (s *RenewScheduler) SetManager(mgr *Manager) {
	s.manager = mgr
}

// RunDailyJob runs the daily renewal check.
// Should be called by a scheduler (cron or background goroutine).
func (s *RenewScheduler) RunDailyJob(ctx context.Context) error {
	// Acquire lock to prevent duplicate runs
	if err := s.acquireLock(ctx); err != nil {
		return err
	}
	defer s.releaseLock()

	// Find certificates expiring within renewal window
	expiringCerts, err := s.findExpiringCertificates(ctx)
	if err != nil {
		return err
	}

	// Process each certificate
	for _, cert := range expiringCerts {
		if err := s.processRenewal(ctx, cert); err != nil {
			// Log error but continue with other certificates
			s.logRenewalFailure(ctx, cert.ID, err)
		}
	}

	// Update expiring_soon status for certificates approaching expiry
	if err := s.updateExpiringStatus(ctx); err != nil {
		return err
	}

	return nil
}

// acquireLock attempts to acquire the renewal lock.
func (s *RenewScheduler) acquireLock(ctx context.Context) error {
	// Create lock directory if needed
	lockDir := filepath.Dir(s.config.RenewalLockFile)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return &Error{Op: "create lock dir", Err: err}
	}

	// Check for existing lock
	if info, err := os.Stat(s.config.RenewalLockFile); err == nil {
		// Lock exists - check if stale (older than 24 hours)
		if time.Since(info.ModTime()) > 24*time.Hour {
			// Stale lock - remove it
			os.Remove(s.config.RenewalLockFile)
		} else {
			return &Error{Op: "acquire lock", Err: fmt.Errorf("renewal already in progress")}
		}
	}

	// Create lock file with PID
	pid := strconv.Itoa(os.Getpid())
	timestamp := time.Now().UTC().Format(time.RFC3339)
	lockContent := fmt.Sprintf("%s\n%s\n", pid, timestamp)

	if err := os.WriteFile(s.config.RenewalLockFile, []byte(lockContent), 0644); err != nil {
		return &Error{Op: "write lock file", Err: err}
	}

	return nil
}

// releaseLock removes the renewal lock.
func (s *RenewScheduler) releaseLock() {
	os.Remove(s.config.RenewalLockFile)
}

// findExpiringCertificates finds certificates that need renewal.
func (s *RenewScheduler) findExpiringCertificates(ctx context.Context) ([]*models.SSLCertificate, error) {
	renewalWindow := time.Now().AddDate(0, 0, s.config.RenewalWindowDays)

	var certs []*models.SSLCertificate
	err := s.db.WithContext(ctx).
		Where("auto_renew = ?", true).
		Where("status IN ?", []string{models.CertStatusIssued, models.CertStatusExpiringSoon}).
		Where("expires_at <= ?", renewalWindow).
		Where("expires_at > ?", time.Now()).
		Find(&certs).Error

	if err != nil {
		return nil, &Error{Op: "find expiring certs", Err: err}
	}

	// Filter out recently attempted renewals (within 24 hours)
	var result []*models.SSLCertificate
	for _, cert := range certs {
		if cert.LastRenewalAttempt != nil {
			if time.Since(*cert.LastRenewalAttempt) < 24*time.Hour {
				continue // Skip recently attempted
			}
		}
		result = append(result, cert)
	}

	return result, nil
}

// processRenewal processes a single certificate renewal.
func (s *RenewScheduler) processRenewal(ctx context.Context, cert *models.SSLCertificate) error {
	// Mark attempt
	cert.LastRenewalAttempt = timePtr(time.Now())
	cert.RenewalAttempts++
	s.db.Save(cert)

	// Log start
	s.logEvent(ctx, cert.ID, "renewal_started", "Starting automatic renewal", "")

	// Parse SANs
	var sans []string
	if cert.SANNames != "" {
		sans = strings.Split(cert.SANNames, ",")
	}

	// Attempt renewal
	result, err := s.provider.RenewCertificate(ctx, cert.ID, IssueRequest{
		Domain:        cert.CommonName,
		SANs:          sans,
		ACMEAccountID: cert.ACMEAccountID,
		Provider:      cert.Provider,
	})

	if err != nil {
		// Renewal failed
		cert.LastError = err.Error()
		s.db.Save(cert)
		s.logEvent(ctx, cert.ID, "renewal_failed", "Automatic renewal failed", err.Error())
		return err
	}

	// Renewal succeeded - update certificate files
	paths, err := s.storage.WriteCertFiles(cert.CommonName, result)
	if err != nil {
		cert.LastError = err.Error()
		s.db.Save(cert)
		s.logEvent(ctx, cert.ID, "renewal_failed", "Failed to store certificate files", err.Error())
		return err
	}

	// Update certificate record
	now := time.Now()
	cert.IssuedAt = &now
	cert.ExpiresAt = &result.NotAfter
	cert.SerialNumber = result.SerialNum
	cert.Fingerprint = result.Fingerprint
	cert.CertPath = paths.CertPath
	cert.KeyPath = paths.KeyPath
	cert.ChainPath = paths.ChainPath
	cert.FullChainPath = paths.FullChainPath
	cert.LastRenewalAt = &now
	cert.LastError = ""
	cert.Status = models.CertStatusIssued
	s.db.Save(cert)

	// Update nginx if manager is available
	if s.manager != nil {
		if err := s.manager.UpdateNginxVhost(cert.CommonName); err != nil {
			s.logEvent(ctx, cert.ID, "nginx_update_failed", "Failed to update nginx", err.Error())
			// Don't fail renewal - nginx can be updated manually
		} else {
			s.logEvent(ctx, cert.ID, "nginx_updated", "Nginx vhost updated with new certificate", "")
		}
	}

	s.logEvent(ctx, cert.ID, "renewed", "Certificate renewed successfully", fmt.Sprintf("Expires: %s", cert.ExpiresAt.Format("2006-01-02")))

	return nil
}

// updateExpiringStatus updates status for certificates approaching expiry.
func (s *RenewScheduler) updateExpiringStatus(ctx context.Context) error {
	deadline := time.Now().AddDate(0, 0, 30)

	return s.db.WithContext(ctx).
		Model(&models.SSLCertificate{}).
		Where("expires_at <= ? AND expires_at > ?", deadline, time.Now()).
		Where("status = ?", models.CertStatusIssued).
		Update("status", models.CertStatusExpiringSoon).Error
}

// logEvent logs an SSL event.
func (s *RenewScheduler) logEvent(ctx context.Context, certID, eventType, message, errorDetail string) {
	event := &models.SSLEvent{
		CertificateID: certID,
		EventType:    eventType,
		Message:     message,
		ErrorDetail: errorDetail,
	}
	s.db.WithContext(ctx).Create(event)
}

// logRenewalFailure logs a renewal failure.
func (s *RenewScheduler) logRenewalFailure(ctx context.Context, certID string, err error) {
	event := &models.SSLEvent{
		CertificateID: certID,
		EventType:     "renewal_failed",
		Message:      "Automatic renewal failed",
		ErrorDetail:  err.Error(),
	}
	s.db.WithContext(ctx).Create(event)
}

// ShouldRenew returns true if the certificate should be renewed.
func (s *RenewScheduler) ShouldRenew(cert *models.SSLCertificate) bool {
	if !cert.AutoRenew {
		return false
	}

	if cert.Status == models.CertStatusRevoked || cert.Status == models.CertStatusFailed {
		return false
	}

	if cert.ExpiresAt == nil {
		return false
	}

	// Check if within renewal window
	renewalDate := cert.ExpiresAt.AddDate(0, 0, -s.config.RenewalWindowDays)
	return time.Now().After(renewalDate)
}

// GetNextRenewalDate returns when the next renewal should occur.
func (s *RenewScheduler) GetNextRenewalDate(cert *models.SSLCertificate) *time.Time {
	if cert.ExpiresAt == nil {
		return nil
	}

	nextRenewal := cert.ExpiresAt.AddDate(0, 0, -s.config.RenewalWindowDays)
	if nextRenewal.Before(time.Now()) {
		nextRenewal = time.Now()
	}

	return &nextRenewal
}

// timePtr returns a pointer to a time.
func timePtr(t time.Time) *time.Time {
	return &t
}