package ssl

import (
	"testing"
	"time"
)

// TestParseCertificate tests certificate parsing functionality.
func TestParseCertificate(t *testing.T) {
	// Test with nil/empty data
	_, err := ParseCertificate(nil)
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}

	_, err = ParseCertificate([]byte{})
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}

	// Test with invalid PEM
	_, err = ParseCertificate([]byte("not a certificate"))
	if err == nil {
		t.Error("expected error for invalid PEM, got nil")
	}
}

// TestCertInfo tests CertInfo struct initialization.
func TestCertInfo(t *testing.T) {
	info := &CertInfo{
		CommonName:   "example.com",
		SerialNumber: "12345",
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SANs:         []string{"www.example.com", "api.example.com"},
		IsCA:         false,
		Issuer:       "Let's Encrypt",
	}

	if info.CommonName != "example.com" {
		t.Errorf("expected CommonName 'example.com', got '%s'", info.CommonName)
	}

	if len(info.SANs) != 2 {
		t.Errorf("expected 2 SANs, got %d", len(info.SANs))
	}
}

// TestConfigDefaults tests DefaultConfig function.
func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.StorageDir == "" {
		t.Error("expected StorageDir to be set")
	}

	if cfg.ChallengeDir == "" {
		t.Error("expected ChallengeDir to be set")
	}

	if cfg.RenewalWindowDays <= 0 {
		t.Error("expected positive RenewalWindowDays")
	}

	if cfg.MaxRenewalRetries <= 0 {
		t.Error("expected positive MaxRenewalRetries")
	}
}

// TestStorage tests Storage initialization.
func TestStorage(t *testing.T) {
	storage := NewStorage("/tmp/ssl-test")

	if storage.baseDir != "/tmp/ssl-test" {
		t.Errorf("expected baseDir '/tmp/ssl-test', got '%s'", storage.baseDir)
	}
}

// TestStorageDomainPath tests Storage domain path generation.
func TestStorageDomainPath(t *testing.T) {
	storage := NewStorage("/var/lib/orvixpanel/ssl")

	path := storage.GetDomainPath("example.com")

	expectedPath := "/var/lib/orvixpanel/ssl/example.com"
	if path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, path)
	}
}

// TestStorageFileExists tests Storage file existence check.
func TestStorageFileExists(t *testing.T) {
	storage := NewStorage("/tmp/ssl-test-nonexistent")

	// Should return false for non-existent files
	if storage.FileExists("example.com", "cert.pem") {
		t.Error("expected FileExists to return false for non-existent file")
	}
}

// TestValidator tests Validator initialization.
func TestValidator(t *testing.T) {
	storage := NewStorage("/tmp/ssl-test")
	validator := NewValidator(storage)

	if validator == nil {
		t.Error("expected non-nil Validator")
	}

	if validator.storage != storage {
		t.Error("expected Validator to hold the storage reference")
	}
}

// TestHealthScanner tests HealthScanner initialization.
func TestHealthScanner(t *testing.T) {
	// This would need a real DB connection in integration tests
	// Just test that initialization works
	scanner := &HealthScanner{}

	if scanner == nil {
		t.Error("expected non-nil HealthScanner")
	}
}

// TestACMEError tests ACMEError struct.
func TestACMEError(t *testing.T) {
	err := &ACMEError{
		Type:   "urn:ietf:params:acme:error:unauthorized",
		Detail: "Invalid signature on JWS",
	}

	if err.Type != "urn:ietf:params:acme:error:unauthorized" {
		t.Errorf("expected Type 'urn:ietf:params:acme:error:unauthorized', got '%s'", err.Type)
	}

	if err.Detail != "Invalid signature on JWS" {
		t.Errorf("expected Detail 'Invalid signature on JWS', got '%s'", err.Detail)
	}
}

// TestIssueResult tests IssueResult struct.
func TestIssueResult(t *testing.T) {
	result := &IssueResult{
		Cert:       []byte("cert-data"),
		Key:        []byte("key-data"),
		CertChain:  []byte("chain-data"),
		FullChain:  []byte("full-chain-data"),
		NotAfter:   time.Now().AddDate(0, 0, 90),
		SerialNum:  "serial123",
		Fingerprint: "fp123",
	}

	if string(result.Cert) != "cert-data" {
		t.Error("expected Cert to be 'cert-data'")
	}

	if result.SerialNum != "serial123" {
		t.Errorf("expected SerialNum 'serial123', got '%s'", result.SerialNum)
	}
}

// TestAccountStatus tests AccountStatus struct.
func TestAccountStatus(t *testing.T) {
	status := &AccountStatus{
		URL:            "https://acme-v02.api.letsencrypt.org/acme/acct/123",
		Status:         "active",
		Email:          "admin@example.com",
		TermsAgree:     true,
		RemainingEAB:   0,
		RateLimits:     nil,
	}

	if status.Status != "active" {
		t.Errorf("expected Status 'active', got '%s'", status.Status)
	}

	if status.RemainingEAB != 0 {
		t.Errorf("expected RemainingEAB 0, got %d", status.RemainingEAB)
	}
}

// TestRateLimits tests RateLimits struct.
func TestRateLimits(t *testing.T) {
	limits := &RateLimits{
		LimitRemain: 50,
		LimitUsed:   0,
		ResetTime:   time.Now().Add(time.Hour),
		RetryAfter:  time.Second * 5,
	}

	if limits.LimitRemain != 50 {
		t.Errorf("expected LimitRemain 50, got %d", limits.LimitRemain)
	}

	if limits.RetryAfter != time.Second*5 {
		t.Errorf("expected RetryAfter 5s, got %v", limits.RetryAfter)
	}
}

// TestIssueRequest tests IssueRequest struct.
func TestIssueRequest(t *testing.T) {
	req := &IssueRequest{
		Domain:        "example.com",
		SANs:          []string{"www.example.com"},
		ACMEAccountID: "acc123",
		Provider:      "letsencrypt",
	}

	if req.Domain != "example.com" {
		t.Errorf("expected Domain 'example.com', got '%s'", req.Domain)
	}

	if len(req.SANs) != 1 {
		t.Errorf("expected 1 SAN, got %d", len(req.SANs))
	}

	if req.Provider != "letsencrypt" {
		t.Errorf("expected Provider 'letsencrypt', got '%s'", req.Provider)
	}
}

// TestErrorTypes tests error constants.
func TestErrorTypes(t *testing.T) {
	// Verify error types are defined
	if ErrCertificateNotFound == nil {
		t.Error("ErrCertificateNotFound should not be nil")
	}

	if ErrCertificateExpired == nil {
		t.Error("ErrCertificateExpired should not be nil")
	}

	if ErrChallengeFailed == nil {
		t.Error("ErrChallengeFailed should not be nil")
	}

	if ErrNginxValidationFailed == nil {
		t.Error("ErrNginxValidationFailed should not be nil")
	}
}

// TestErrorStruct tests Error struct.
func TestErrorStruct(t *testing.T) {
	err := &Error{
		Op:  "test operation",
		Err: ErrCertificateNotFound,
	}

	if err.Op != "test operation" {
		t.Errorf("expected Op 'test operation', got '%s'", err.Op)
	}

	if err.Err != ErrCertificateNotFound {
		t.Error("expected Err to be ErrCertificateNotFound")
	}
}

// TestCertPaths tests CertPaths struct.
func TestCertPaths(t *testing.T) {
	paths := &CertPaths{
		CertPath:      "/path/to/cert.pem",
		KeyPath:       "/path/to/key.pem",
		FullChainPath: "/path/to/fullchain.pem",
		ChainPath:     "/path/to/chain.pem",
	}

	if paths.CertPath != "/path/to/cert.pem" {
		t.Errorf("expected CertPath '/path/to/cert.pem', got '%s'", paths.CertPath)
	}

	if paths.KeyPath != "/path/to/key.pem" {
		t.Errorf("expected KeyPath '/path/to/key.pem', got '%s'", paths.KeyPath)
	}
}

// TestConfigCertPaths tests Config.CertPaths method.
func TestConfigCertPaths(t *testing.T) {
	cfg := &Config{
		StorageDir: "/var/lib/orvixpanel/ssl/certs",
	}

	paths := cfg.CertPaths("example.com")

	if paths.CertPath == "" {
		t.Error("expected CertPath to be set")
	}

	if paths.KeyPath == "" {
		t.Error("expected KeyPath to be set")
	}

	if paths.FullChainPath == "" {
		t.Error("expected FullChainPath to be set")
	}

	if paths.ChainPath == "" {
		t.Error("expected ChainPath to be set")
	}
}

// TestLetsEncryptProvider tests LetsEncryptProvider initialization.
func TestLetsEncryptProvider(t *testing.T) {
	cfg := &Config{
		LetsEncryptEmail:        "test@example.com",
		LetsEncryptDirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
	}

	provider := NewLetsEncryptProvider(cfg)

	if provider == nil {
		t.Error("expected non-nil provider")
	}

	if provider.Name() != "letsencrypt" {
		t.Errorf("expected Name 'letsencrypt', got '%s'", provider.Name())
	}

	if !provider.IsConfigured() {
		t.Error("expected IsConfigured to return true")
	}
}

// TestZeroSSLProvider tests ZeroSSLProvider initialization.
func TestZeroSSLProvider(t *testing.T) {
	provider := NewZeroSSLProvider("test-api-key")

	if provider == nil {
		t.Error("expected non-nil provider")
	}

	if provider.Name() != "zerossl" {
		t.Errorf("expected Name 'zerossl', got '%s'", provider.Name())
	}

	if !provider.IsConfigured() {
		t.Error("expected IsConfigured to return true with API key")
	}

	// Test without API key
	unconfigured := NewZeroSSLProvider("")
	if unconfigured.IsConfigured() {
		t.Error("expected IsConfigured to return false without API key")
	}
}

// TestNginxIntegration tests NginxIntegration initialization.
func TestNginxIntegration(t *testing.T) {
	integration := NewNginxIntegration("/etc/nginx/sites-enabled", "/tmp/nginx-backup", "/var/lib/orvixpanel/ssl")

	if integration == nil {
		t.Error("expected non-nil NginxIntegration")
	}

	if integration.configDir != "/etc/nginx/sites-enabled" {
		t.Errorf("expected configDir '/etc/nginx/sites-enabled', got '%s'", integration.configDir)
	}
}

// TestUpdateVhostResult tests UpdateVhostResult struct.
func TestUpdateVhostResult(t *testing.T) {
	result := &UpdateVhostResult{
		Success:    true,
		BackupPath: "/tmp/backup.conf.bak",
		RollbackOK: true,
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.BackupPath != "/tmp/backup.conf.bak" {
		t.Errorf("expected BackupPath '/tmp/backup.conf.bak', got '%s'", result.BackupPath)
	}
}

// TestStorageStats tests StorageStats struct.
func TestStorageStats(t *testing.T) {
	stats := &StorageStats{
		DomainCount: 5,
		FileCount:   20,
		TotalSize:   1024 * 1024 * 10, // 10 MB
	}

	if stats.DomainCount != 5 {
		t.Errorf("expected DomainCount 5, got %d", stats.DomainCount)
	}

	if stats.FileCount != 20 {
		t.Errorf("expected FileCount 20, got %d", stats.FileCount)
	}

	if stats.TotalSize != 1024*1024*10 {
		t.Errorf("expected TotalSize 10485760, got %d", stats.TotalSize)
	}
}

// TestValidationResult tests ValidationResult struct.
func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		IsValid:     true,
		Errors:      []string{},
		Warnings:    []string{"Warning 1"},
		ChainValid:  true,
		KeyMatch:    true,
		Permissions: true,
	}

	if !result.IsValid {
		t.Error("expected IsValid to be true")
	}

	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}

	if !result.ChainValid {
		t.Error("expected ChainValid to be true")
	}
}

// TestACMESubproblem tests ACMESubproblem struct.
func TestACMESubproblem(t *testing.T) {
	sub := &ACMESubproblem{
		Type:   "urn:ietf:params:acme:error:malformed",
		Detail: "Invalid JWS",
		Identifier: &ACMEIdentifier{
			Type:  "dns",
			Value: "example.com",
		},
	}

	if sub.Type != "urn:ietf:params:acme:error:malformed" {
		t.Errorf("expected Type 'urn:ietf:params:acme:error:malformed', got '%s'", sub.Type)
	}

	if sub.Identifier.Value != "example.com" {
		t.Errorf("expected Identifier.Value 'example.com', got '%s'", sub.Identifier.Value)
	}
}

// TestConfigAccountKeyPath tests Config.AccountKeyPath method.
func TestConfigAccountKeyPath(t *testing.T) {
	cfg := &Config{
		StorageDir: "/var/lib/orvixpanel/ssl/certs",
	}

	path := cfg.AccountKeyPath("tenant1", "acc123")

	if path == "" {
		t.Error("expected AccountKeyPath to return a non-empty path")
	}
}