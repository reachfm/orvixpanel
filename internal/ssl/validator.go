package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"
)

// Validator handles certificate validation operations.
type Validator struct {
	storage *Storage
}

// NewValidator creates a new certificate validator.
func NewValidator(storage *Storage) *Validator {
	return &Validator{storage: storage}
}

// ValidationResult represents the result of certificate validation.
type ValidationResult struct {
	IsValid     bool
	Errors      []string
	Warnings    []string
	CertInfo    *CertInfo
	ChainValid  bool
	KeyMatch    bool
	Permissions bool
}

// ValidateCertificate performs comprehensive certificate validation.
func (v *Validator) ValidateCertificate(domain string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: true,
		Errors:  []string{},
		Warnings: []string{},
	}

	// Check if certificate files exist
	certPath := v.storage.GetDomainPath(domain) + "/cert.pem"
	keyPath := v.storage.GetDomainPath(domain) + "/privkey.pem"
	fullChainPath := v.storage.GetDomainPath(domain) + "/fullchain.pem"

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, "certificate file missing")
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, "private key file missing")
	}

	if _, err := os.Stat(fullChainPath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, "fullchain file missing")
	}

	if !result.IsValid {
		return result, nil
	}

	// Read and parse certificate
	certData, err := os.ReadFile(certPath)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "cannot read certificate: "+err.Error())
		return result, nil
	}

	cert, err := v.parseCertificate(certData)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "cannot parse certificate: "+err.Error())
		return result, nil
	}

	result.CertInfo = cert

	// Check expiration
	if cert.NotAfter.Before(time.Now()) {
		result.IsValid = false
		result.Errors = append(result.Errors, "certificate has expired")
	} else {
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysUntilExpiry <= 7 {
			result.Warnings = append(result.Warnings, "certificate expires in less than 7 days")
		} else if daysUntilExpiry <= 30 {
			result.Warnings = append(result.Warnings, "certificate expires in less than 30 days")
		}
	}

	// Validate chain
	chainData, err := os.ReadFile(fullChainPath)
	if err != nil {
		result.Warnings = append(result.Warnings, "cannot read chain file")
		result.ChainValid = false
	} else {
		result.ChainValid = v.validateChain(chainData)
		if !result.ChainValid {
			result.Warnings = append(result.Warnings, "certificate chain may be incomplete")
		}
	}

	// Check file permissions
	if err := v.checkPermissions(certPath, keyPath); err != nil {
		result.Warnings = append(result.Warnings, err.Error())
		result.Permissions = false
	} else {
		result.Permissions = true
	}

	return result, nil
}

// parseCertificate parses a PEM certificate.
func (v *Validator) parseCertificate(data []byte) (*CertInfo, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrInvalidCertificate
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	info := &CertInfo{
		CommonName: cert.Subject.CommonName,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		IsCA:       cert.IsCA,
	}

	// Extract SANs
	for _, dnsName := range cert.DNSNames {
		info.SANs = append(info.SANs, dnsName)
	}

	// Extract serial number
	info.SerialNumber = cert.SerialNumber.String()

	return info, nil
}

// validateChain validates the certificate chain.
func (v *Validator) validateChain(data []byte) bool {
	// Count PEM blocks in chain
	blocks := 0
	remaining := data
	for {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}
		blocks++
		remaining = rest
	}

	// Valid chain should have at least 2 blocks (cert + intermediate/root)
	return blocks >= 2
}

// checkPermissions verifies file permissions are secure.
func (v *Validator) checkPermissions(certPath, keyPath string) error {
	// Check key permissions (should be 0600 or stricter)
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		return err
	}

	keyMode := keyInfo.Mode().Perm()
	if keyMode > 0600 {
		return &Error{
			Op:  "check permissions",
			Err: &PermissionError{Path: keyPath, Mode: keyMode, Required: 0600},
		}
	}

	return nil
}

// PermissionError represents a file permission issue.
type PermissionError struct {
	Path     string
	Mode     os.FileMode
	Required os.FileMode
}

func (e *PermissionError) Error() string {
	return "insecure permissions " + e.Mode.String() + " on " + e.Path + " (should be " + e.Required.String() + ")"
}

// HealthCheck performs a health check on all certificates.
func (v *Validator) HealthCheck(domains []string) *HealthReport {
	report := &HealthReport{
		Timestamp:    time.Now(),
		TotalChecked: len(domains),
		Results:      make([]*ValidationResult, 0, len(domains)),
	}

	for _, domain := range domains {
		result, err := v.ValidateCertificate(domain)
		if err != nil {
			report.Errors++
			continue
		}

		report.Results = append(report.Results, result)

		if result.IsValid {
			if len(result.Warnings) > 0 {
				report.Warnings++
			} else {
				report.Healthy++
			}
		} else {
			report.Critical++
		}
	}

	return report
}

// HealthReport represents a certificate health check report.
type HealthReport struct {
	Timestamp time.Time
	TotalChecked int
	Healthy   int
	Warnings  int
	Critical  int
	Errors    int
	Results   []*ValidationResult
}

// GetExpiringCerts returns certificates expiring within the specified days.
func (v *Validator) GetExpiringCerts(domains []string, withinDays int) ([]string, error) {
	var expiring []string

	for _, domain := range domains {
		result, err := v.ValidateCertificate(domain)
		if err != nil {
			continue
		}

		if result.CertInfo != nil {
			daysUntil := int(time.Until(result.CertInfo.NotAfter).Hours() / 24)
			if daysUntil <= withinDays && daysUntil > 0 {
				expiring = append(expiring, domain)
			}
		}
	}

	return expiring, nil
}