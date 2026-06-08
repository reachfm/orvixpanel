package ssl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NginxIntegration handles nginx vhost SSL configuration updates.
type NginxIntegration struct {
	configDir   string
	backupDir   string
	storageDir  string
}

// NewNginxIntegration creates a new nginx integration handler.
func NewNginxIntegration(configDir, backupDir, storageDir string) *NginxIntegration {
	return &NginxIntegration{
		configDir:  configDir,
		backupDir:  backupDir,
		storageDir: storageDir,
	}
}

// UpdateVhostResult represents the result of a vhost update operation.
type UpdateVhostResult struct {
	Success     bool
	BackupPath  string
	Error       error
	RollbackOK  bool
}

// UpdateVhostSSL updates a nginx vhost with SSL configuration.
func (n *NginxIntegration) UpdateVhostSSL(ctx context.Context, domain string) (*UpdateVhostResult, error) {
	result := &UpdateVhostResult{}

	// Find vhost config file
	vhostPath := filepath.Join(n.configDir, domain+".conf")
	if _, err := os.Stat(vhostPath); os.IsNotExist(err) {
		return nil, &Error{Op: "find vhost", Err: fmt.Errorf("vhost config not found: %s", vhostPath)}
	}

	// Read original vhost config
	originalContent, err := os.ReadFile(vhostPath)
	if err != nil {
		return nil, &Error{Op: "read vhost", Err: err}
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(n.backupDir, 0755); err != nil {
		return nil, &Error{Op: "create backup dir", Err: err}
	}

	// Create backup with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(n.backupDir, domain+"."+timestamp+".conf.bak")
	if err := os.WriteFile(backupPath, originalContent, 0644); err != nil {
		return nil, &Error{Op: "write backup", Err: err}
	}
	result.BackupPath = backupPath

	// Get certificate paths
	certPaths := n.getCertPaths(domain)

	// Check if certificate files exist
	if err := n.validateCertPaths(certPaths); err != nil {
		// Rollback - remove backup
		os.Remove(backupPath)
		return nil, &Error{Op: "validate cert paths", Err: err}
	}

	// Generate SSL configuration
	sslConfig := n.generateSSLConfig(domain, certPaths)

	// Insert SSL config into vhost
	updatedContent := n.insertSSLConfig(string(originalContent), sslConfig)

	// Write updated vhost
	if err := os.WriteFile(vhostPath, []byte(updatedContent), 0644); err != nil {
		// Rollback
		os.WriteFile(vhostPath, originalContent, 0644)
		os.Remove(backupPath)
		return nil, &Error{Op: "write vhost", Err: err}
	}

	// Validate nginx config
	if err := n.validateNginxConfig(); err != nil {
		// Rollback
		os.WriteFile(vhostPath, originalContent, 0644)
		return nil, &Error{Op: "validate nginx", Err: err}
	}

	// Reload nginx
	if err := n.reloadNginx(); err != nil {
		// Rollback
		os.WriteFile(vhostPath, originalContent, 0644)
		return nil, &Error{Op: "reload nginx", Err: err}
	}

	result.Success = true
	return result, nil
}

// RollbackVhostSSL rolls back a vhost to its backup configuration.
func (n *NginxIntegration) RollbackVhostSSL(domain string, backupPath string) error {
	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return &Error{Op: "find backup", Err: fmt.Errorf("backup not found")}
	}

	// Read backup content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		return &Error{Op: "read backup", Err: err}
	}

	// Find current vhost config
	vhostPath := filepath.Join(n.configDir, domain+".conf")

	// Write backup content to vhost
	if err := os.WriteFile(vhostPath, backupContent, 0644); err != nil {
		return &Error{Op: "write vhost", Err: err}
	}

	// Validate and reload
	if err := n.validateNginxConfig(); err != nil {
		return &Error{Op: "validate after rollback", Err: err}
	}

	return n.reloadNginx()
}

// getCertPaths returns the certificate file paths for a domain.
func (n *NginxIntegration) getCertPaths(domain string) *CertPaths {
	return &CertPaths{
		CertPath:      filepath.Join(n.storageDir, domain, "cert.pem"),
		KeyPath:       filepath.Join(n.storageDir, domain, "privkey.pem"),
		FullChainPath: filepath.Join(n.storageDir, domain, "fullchain.pem"),
	}
}

// validateCertPaths checks if all required certificate files exist.
func (n *NginxIntegration) validateCertPaths(paths *CertPaths) error {
	if _, err := os.Stat(paths.CertPath); os.IsNotExist(err) {
		return &Error{Op: "check cert", Err: fmt.Errorf("certificate file not found")}
	}
	if _, err := os.Stat(paths.KeyPath); os.IsNotExist(err) {
		return &Error{Op: "check key", Err: fmt.Errorf("private key file not found")}
	}
	if _, err := os.Stat(paths.FullChainPath); os.IsNotExist(err) {
		return &Error{Op: "check chain", Err: fmt.Errorf("fullchain file not found")}
	}
	return nil
}

// generateSSLConfig generates nginx SSL configuration.
func (n *NginxIntegration) generateSSLConfig(domain string, paths *CertPaths) string {
	return fmt.Sprintf(`
    # SSL configuration for %s
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate %s;
    ssl_certificate_key %s;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
`, domain, paths.FullChainPath, paths.KeyPath)
}

// insertSSLConfig inserts SSL configuration into existing vhost.
func (n *NginxIntegration) insertSSLConfig(original string, sslConfig string) string {
	// Find the server block and insert SSL config after the server_name directive
	var buffer bytes.Buffer
	lines := strings.Split(original, "\n")

	inServerBlock := false
	inserted := false

	for i, line := range lines {
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Detect server block start
		if strings.TrimSpace(line) == "server {" {
			inServerBlock = true
		}

		// Insert SSL config after server_name in server block
		if inServerBlock && !inserted && strings.HasPrefix(strings.TrimSpace(line), "server_name") {
			// Insert SSL config after this line
			buffer.WriteString(sslConfig)
			inserted = true
		}

		_ = i // unused but needed for loop
	}

	// If SSL config wasn't inserted (no server_name found), append after server {
	if !inserted {
		return original + sslConfig
	}

	return buffer.String()
}

// validateNginxConfig validates nginx configuration with `nginx -t`.
func (n *NginxIntegration) validateNginxConfig() error {
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &Error{
			Op:  "nginx -t",
			Err: fmt.Errorf("nginx config test failed: %s", string(output)),
		}
	}

	return nil
}

// reloadNginx reloads nginx configuration with `systemctl reload nginx`.
func (n *NginxIntegration) reloadNginx() error {
	cmd := exec.Command("systemctl", "reload", "nginx")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &Error{
			Op:  "systemctl reload nginx",
			Err: fmt.Errorf("nginx reload failed: %s", string(output)),
		}
	}

	return nil
}

// RemoveSSLVhost removes SSL configuration from a vhost.
func (n *NginxIntegration) RemoveSSLVhost(domain string) error {
	vhostPath := filepath.Join(n.configDir, domain+".conf")

	// Read current config
	content, err := os.ReadFile(vhostPath)
	if err != nil {
		return &Error{Op: "read vhost", Err: err}
	}

	// Remove SSL-related lines
	lines := strings.Split(string(content), "\n")
	var newContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect SSL listen directives
		if strings.Contains(trimmed, "listen") && strings.Contains(trimmed, "ssl") {
			continue // Skip SSL listen lines
		}

		// Detect SSL configuration lines
		if strings.HasPrefix(trimmed, "ssl_") {
			continue // Skip SSL config lines
		}

		// Detect SSL headers
		if strings.Contains(trimmed, "X-Frame-Options") ||
			strings.Contains(trimmed, "X-Content-Type-Options") ||
			strings.Contains(trimmed, "X-XSS-Protection") ||
			strings.Contains(trimmed, "Strict-Transport-Security") {
			continue // Skip security headers that are SSL-related
		}

		// Detect SSL certificate directives
		if strings.HasPrefix(trimmed, "ssl_certificate") {
			continue
		}

		newContent = append(newContent, line)
	}

	// Write updated config
	if err := os.WriteFile(vhostPath, []byte(strings.Join(newContent, "\n")), 0644); err != nil {
		return &Error{Op: "write vhost", Err: err}
	}

	// Validate and reload
	if err := n.validateNginxConfig(); err != nil {
		return &Error{Op: "validate after SSL removal", Err: err}
	}

	return n.reloadNginx()
}

// GetVhostConfig reads and returns the current vhost configuration.
func (n *NginxIntegration) GetVhostConfig(domain string) (string, error) {
	vhostPath := filepath.Join(n.configDir, domain+".conf")

	content, err := os.ReadFile(vhostPath)
	if err != nil {
		return "", &Error{Op: "read vhost", Err: err}
	}

	return string(content), nil
}

// ListSSLVhosts returns a list of vhosts with SSL configuration.
func (n *NginxIntegration) ListSSLVhosts() ([]string, error) {
	entries, err := os.ReadDir(n.configDir)
	if err != nil {
		return nil, &Error{Op: "read config dir", Err: err}
	}

	var sslVhosts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := os.ReadFile(filepath.Join(n.configDir, entry.Name()))
		if err != nil {
			continue
		}

		if strings.Contains(string(content), "ssl_certificate") {
			domain := strings.TrimSuffix(entry.Name(), ".conf")
			sslVhosts = append(sslVhosts, domain)
		}
	}

	return sslVhosts, nil
}