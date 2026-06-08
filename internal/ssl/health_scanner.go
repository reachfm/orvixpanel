package ssl

import (
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"context"
	"gorm.io/gorm"
	"fmt"
	"time"
)

// HealthScanner scans for certificate health issues.
type HealthScanner struct {
	db *gorm.DB
}

// NewHealthScanner creates a new health scanner.
func NewHealthScanner(db *gorm.DB) *HealthScanner {
	return &HealthScanner{db: db}
}

// HealthStatus represents certificate health status.
type HealthStatus struct {
	Status         string    `json:"status"` // healthy, warning, critical, unknown
	Message        string    `json:"message,omitempty"`
	DaysUntilExpiry int      `json:"days_until_expiry,omitempty"`
	FileMissing    bool      `json:"file_missing,omitempty"`
	InvalidChain   bool      `json:"invalid_chain,omitempty"`
	PermissionsBad bool      `json:"permissions_bad,omitempty"`
}

// ScanCertificate performs health scan on a single certificate.
func (s *HealthScanner) ScanCertificate(ctx context.Context, certID string) (*HealthStatus, error) {
	var cert models.SSLCertificate
	if err := s.db.WithContext(ctx).First(&cert, "id = ?", certID).Error; err != nil {
		return nil, &Error{Op: "find certificate", Err: err}
	}

	status := &HealthStatus{}

	// Check if certificate has expired
	if cert.ExpiresAt != nil {
		if time.Now().After(*cert.ExpiresAt) {
			status.Status = "critical"
			status.Message = "Certificate has expired"
			status.DaysUntilExpiry = -1
			return status, nil
		}
		status.DaysUntilExpiry = int(time.Until(*cert.ExpiresAt).Hours() / 24)
	}

	// Check status field
	switch cert.Status {
	case models.CertStatusExpired:
		status.Status = "critical"
		status.Message = "Certificate marked as expired"
	case models.CertStatusRevoked:
		status.Status = "critical"
		status.Message = "Certificate has been revoked"
	case models.CertStatusFailed:
		status.Status = "critical"
		status.Message = "Certificate issuance/renewal failed"
		if cert.LastError != "" {
			status.Message += ": " + cert.LastError
		}
	case models.CertStatusExpiringSoon:
		if status.DaysUntilExpiry <= 7 {
			status.Status = "critical"
			status.Message = fmt.Sprintf("Certificate expires in %d days", status.DaysUntilExpiry)
		} else {
			status.Status = "warning"
			status.Message = fmt.Sprintf("Certificate expires in %d days", status.DaysUntilExpiry)
		}
	case models.CertStatusPending:
		status.Status = "unknown"
		status.Message = "Certificate issuance in progress"
	default:
		// Check by actual expiry
		if status.DaysUntilExpiry <= 0 {
			status.Status = "critical"
			status.Message = "Certificate has expired"
		} else if status.DaysUntilExpiry <= 7 {
			status.Status = "critical"
			status.Message = fmt.Sprintf("Certificate expires in %d days", status.DaysUntilExpiry)
		} else if status.DaysUntilExpiry <= 14 {
			status.Status = "warning"
			status.Message = fmt.Sprintf("Certificate expires in %d days", status.DaysUntilExpiry)
		} else if status.DaysUntilExpiry <= 30 {
			status.Status = "warning"
			status.Message = fmt.Sprintf("Certificate expires in %d days", status.DaysUntilExpiry)
		} else {
			status.Status = "healthy"
			status.Message = fmt.Sprintf("Certificate valid for %d days", status.DaysUntilExpiry)
		}
	}

	return status, nil
}

// ScanAll scans all certificates and returns a summary.
func (s *HealthScanner) ScanAll(ctx context.Context) (*HealthScanReport, error) {
	var certs []models.SSLCertificate
	if err := s.db.WithContext(ctx).Find(&certs).Error; err != nil {
		return nil, &Error{Op: "find all certificates", Err: err}
	}

	report := &HealthScanReport{
		Timestamp:    time.Now(),
		Total:        len(certs),
		Healthy:      0,
		Warning:      0,
		Critical:     0,
		Unknown:      0,
		Certificates: make([]*CertHealthDetail, 0, len(certs)),
	}

	for _, cert := range certs {
		status, err := s.ScanCertificate(ctx, cert.ID)
		if err != nil {
			continue
		}

		detail := &CertHealthDetail{
			ID:               cert.ID,
			CommonName:       cert.CommonName,
			Status:           status.Status,
			Message:          status.Message,
			DaysUntilExpiry:  status.DaysUntilExpiry,
			AutoRenew:        cert.AutoRenew,
			Provider:         cert.Provider,
		}
		report.Certificates = append(report.Certificates, detail)

		switch status.Status {
		case "healthy":
			report.Healthy++
		case "warning":
			report.Warning++
		case "critical":
			report.Critical++
		default:
			report.Unknown++
		}
	}

	return report, nil
}

// HealthScanReport represents a complete health scan report.
type HealthScanReport struct {
	Timestamp    time.Time
	Total        int
	Healthy      int
	Warning      int
	Critical     int
	Unknown      int
	Certificates []*CertHealthDetail
}

// CertHealthDetail represents health details for a single certificate.
type CertHealthDetail struct {
	ID               string `json:"id"`
	CommonName       string `json:"common_name"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	DaysUntilExpiry  int    `json:"days_until_expiry,omitempty"`
	AutoRenew        bool   `json:"auto_renew"`
	Provider         string `json:"provider"`
}

// GetExpiringCertificates returns certificates expiring within specified days.
func (s *HealthScanner) GetExpiringCertificates(ctx context.Context, withinDays int) ([]*models.SSLCertificate, error) {
	deadline := time.Now().AddDate(0, 0, withinDays)

	var certs []*models.SSLCertificate
	err := s.db.WithContext(ctx).
		Where("expires_at <= ? AND expires_at > ?", deadline, time.Now()).
		Where("status IN ?", []string{models.CertStatusIssued, models.CertStatusExpiringSoon}).
		Find(&certs).Error

	if err != nil {
		return nil, &Error{Op: "find expiring certificates", Err: err}
	}

	return certs, nil
}

// GetExpiredCertificates returns all expired certificates.
func (s *HealthScanner) GetExpiredCertificates(ctx context.Context) ([]*models.SSLCertificate, error) {
	var certs []*models.SSLCertificate
	err := s.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Where("status NOT IN ?", []string{models.CertStatusRevoked, models.CertStatusExpired}).
		Find(&certs).Error

	if err != nil {
		return nil, &Error{Op: "find expired certificates", Err: err}
	}

	return certs, nil
}

// GetFailedCertificates returns certificates with failed issuance/renewal.
func (s *HealthScanner) GetFailedCertificates(ctx context.Context) ([]*models.SSLCertificate, error) {
	var certs []*models.SSLCertificate
	err := s.db.WithContext(ctx).
		Where("status = ?", models.CertStatusFailed).
		Find(&certs).Error

	if err != nil {
		return nil, &Error{Op: "find failed certificates", Err: err}
	}

	return certs, nil
}

// MarkAsExpiringSoon marks certificates expiring within 30 days.
func (s *HealthScanner) MarkAsExpiringSoon(ctx context.Context) error {
	deadline := time.Now().AddDate(0, 0, 30)

	return s.db.WithContext(ctx).
		Model(&models.SSLCertificate{}).
		Where("expires_at <= ? AND expires_at > ?", deadline, time.Now()).
		Where("status = ?", models.CertStatusIssued).
		Update("status", models.CertStatusExpiringSoon).Error
}