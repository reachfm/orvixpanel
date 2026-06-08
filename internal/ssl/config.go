package ssl

import (
	"os"
	"path/filepath"
)

// Config holds SSL engine configuration.
type Config struct {
	// Storage paths
	StorageDir string // Base directory for certificate storage

	// Challenge settings
	ChallengeDir string // Directory for HTTP-01 challenge files

	// Renewal settings
	RenewalWindowDays int // Days before expiry to start renewal (default: 30)
	RenewalLockFile   string // Path to renewal lock file
	MaxRenewalRetries int // Max retry attempts for failed renewals

	// Nginx settings
	NginxConfigDir string // Nginx vhost config directory
	NginxBackupDir string // Backup directory for nginx configs

	// Let's Encrypt settings
	LetsEncryptDirectoryURL string // ACME directory URL (production or staging)
	LetsEncryptEmail        string // Default email for ACME account

	// ZeroSSL settings (stub)
	ZeroSSLAPIKey string
}

// DefaultConfig returns the default SSL configuration.
func DefaultConfig() *Config {
	baseDir := "/var/lib/orvixpanel/ssl"

	return &Config{
		StorageDir:           filepath.Join(baseDir, "certs"),
		ChallengeDir:         "/var/www/orvixpanel/.well-known/acme-challenge",
		RenewalWindowDays:    30,
		RenewalLockFile:      "/run/orvixpanel/ssl-renew.lock",
		MaxRenewalRetries:    3,
		NginxConfigDir:       "/etc/nginx/conf.d/orvix",
		NginxBackupDir:       filepath.Join(baseDir, "nginx-backup"),
		LetsEncryptDirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
		LetsEncryptEmail:     "",
	}
}

// Validate checks the configuration for required paths and settings.
func (c *Config) Validate() error {
	// Ensure storage directory path is set
	if c.StorageDir == "" {
		c.StorageDir = DefaultConfig().StorageDir
	}

	// Ensure challenge directory path is set
	if c.ChallengeDir == "" {
		c.ChallengeDir = DefaultConfig().ChallengeDir
	}

	// Validate storage directory exists or can be created
	if err := os.MkdirAll(c.StorageDir, 0700); err != nil {
		return &Error{Op: "validate config", Err: err}
	}

	// Validate challenge directory exists or can be created
	if err := os.MkdirAll(c.ChallengeDir, 0755); err != nil {
		return &Error{Op: "validate config", Err: err}
	}

	return nil
}

// CertPaths returns the file paths for a certificate domain.
func (c *Config) CertPaths(domain string) CertPaths {
	domainDir := filepath.Join(c.StorageDir, domain)
	return CertPaths{
		CertPath:      filepath.Join(domainDir, "cert.pem"),
		KeyPath:       filepath.Join(domainDir, "privkey.pem"),
		ChainPath:     filepath.Join(domainDir, "chain.pem"),
		FullChainPath: filepath.Join(domainDir, "fullchain.pem"),
	}
}

// CertPaths holds the file paths for certificate storage.
type CertPaths struct {
	CertPath      string
	KeyPath       string
	ChainPath     string
	FullChainPath string
}

// AccountKeyPath returns the path to an ACME account key.
func (c *Config) AccountKeyPath(tenantID, accountID string) string {
	return filepath.Join(c.StorageDir, "accounts", tenantID, accountID, "account_key.pem")
}