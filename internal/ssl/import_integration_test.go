// Package ssl provides integration tests for the SSL API endpoints.
// Phase 2B: Real API Integration Test for POST /api/v1/ssl/import
package ssl

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// -----------------------------------------------------------------------------
// Test Fixtures
// -----------------------------------------------------------------------------

// requireDefaultStorageWritable checks if the SSL DefaultConfig storage
// path (/var/lib/orvixpanel/ssl/certs) is writable in the current
// environment. Tests that exercise the full handler write path
// (file storage) call this first; if the path is not writable, the
// test is skipped with a clear message rather than failing on a
// permission error that is unrelated to the code under test.
//
// In a normal install the directory exists with 0700 root. In a
// sandboxed CI / dev environment the path may be missing or owned
// by another user; skipping is the honest answer.
func requireDefaultStorageWritable(t *testing.T) {
	t.Helper()
	defaultPath := DefaultConfig().StorageDir
	if err := os.MkdirAll(defaultPath, 0700); err != nil {
		t.Skipf("SSL default storage path %q is not writable in this environment (%v); skipping filesystem-dependent test", defaultPath, err)
	}
}

// testEnv holds all test dependencies.
type testEnv struct {
	app     *fiber.App
	db      *gorm.DB
	storage *Storage
	deps    SSLDeps
}

// setupTestEnv creates a fresh test environment with in-memory SQLite.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "ssl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Open SQLite in-memory database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.SSLCertificate{},
		&models.SSLEvent{},
		&models.AuditEntry{},
	); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	// Create storage
	storage := NewStorage(tmpDir)

	// Create Fiber app
	app := fiber.New()

	// Setup test deps
	deps := SSLDeps{
		DB: db,
	}

	// Create a minimal router with auth middleware
	authMiddleware := func(c *fiber.Ctx) error {
		// Inject test claims directly
		claims := &auth.Claims{
			UserID:   "test-user-id",
			Email:    "test@example.com",
			Role:     "admin",
			TenantID: "test-tenant-id",
		}
		c.Locals(middleware.LocalClaims, claims)
		return c.Next()
	}

	// Register import handler
	app.Post("/api/v1/ssl/import", authMiddleware, ImportCertificateHandler(deps, nil))

	return &testEnv{
		app:     app,
		db:      db,
		storage: storage,
		deps:    deps,
	}
}

// setupTestEnvNoAuth creates a test env without auth middleware (for 401 tests).
func setupTestEnvNoAuth(t *testing.T) *testEnv {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ssl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	if err := db.AutoMigrate(&models.SSLCertificate{}, &models.SSLEvent{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	storage := NewStorage(tmpDir)
	app := fiber.New()
	deps := SSLDeps{DB: db}

	// Register WITHOUT auth middleware
	app.Post("/api/v1/ssl/import", ImportCertificateHandler(deps, nil))

	return &testEnv{
		app:     app,
		db:      db,
		storage: storage,
		deps:    deps,
	}
}

// generateTestKey generates an RSA 2048-bit private key.
func generateTestKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// generateTestCert generates a self-signed certificate.
func generateTestCert(key *rsa.PrivateKey, domain string, sans []string) ([]byte, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              sans,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}), nil
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Valid Request
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_Valid(t *testing.T) {
	requireDefaultStorageWritable(t)
	env := setupTestEnv(t)

	// Generate test key and certificate
	key, err := generateTestKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	domain := "test-import-valid.example.com"
	sans := []string{domain, "www." + domain}

	certPEM, err := generateTestCert(key, domain, sans)
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}

	keyPEM := generateTestKeyPEM(key)

	// Build request
	reqBody := map[string]interface{}{
		"domain":    domain,
		"cert_pem":  string(certPEM),
		"key_pem":   string(keyPEM),
		"chain_pem": "", // optional
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status
	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify response JSON
	var cert models.SSLCertificate
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &cert); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify response fields
	if cert.CommonName != domain {
		t.Errorf("expected CommonName '%s', got '%s'", domain, cert.CommonName)
	}
	if cert.Status != models.CertStatusIssued {
		t.Errorf("expected status 'issued', got '%s'", cert.Status)
	}
	if cert.Fingerprint == "" {
		t.Error("expected non-empty fingerprint")
	}
	if cert.ExpiresAt == nil {
		t.Error("expected non-nil ExpiresAt")
	}
	if cert.TenantID != "test-tenant-id" {
		t.Errorf("expected TenantID 'test-tenant-id', got '%s'", cert.TenantID)
	}

	// Verify NO private key in response
	if strings.Contains(string(bodyBytes), "PRIVATE KEY") {
		t.Error("response should NOT contain private key material")
	}
	if strings.Contains(string(bodyBytes), "key_pem") {
		t.Error("response should NOT contain key_pem field")
	}
	if strings.Contains(string(bodyBytes), "privkey") {
		t.Error("response should NOT contain private key data")
	}

	// Verify database record
	var dbCert models.SSLCertificate
	if err := env.db.First(&dbCert, "id = ?", cert.ID).Error; err != nil {
		t.Errorf("failed to find certificate in DB: %v", err)
	}
	if dbCert.CommonName != domain {
		t.Errorf("DB: expected CommonName '%s', got '%s'", domain, dbCert.CommonName)
	}
	if dbCert.Fingerprint != cert.Fingerprint {
		t.Errorf("DB: fingerprint mismatch: expected '%s', got '%s'", cert.Fingerprint, dbCert.Fingerprint)
	}
	if dbCert.TenantID != "test-tenant-id" {
		t.Errorf("DB: expected TenantID 'test-tenant-id', got '%s'", dbCert.TenantID)
	}

	// Verify filesystem
	certPath := filepath.Join(env.storage.baseDir, domain, "cert.pem")
	keyPath := filepath.Join(env.storage.baseDir, domain, "privkey.pem")
	fullChainPath := filepath.Join(env.storage.baseDir, domain, "fullchain.pem")

	// Check cert file exists and has 0644 permissions
	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Errorf("cert file not found at %s: %v", certPath, err)
	} else {
		perm := certInfo.Mode().Perm()
		expectedPerm := os.FileMode(0644)
		if perm != expectedPerm {
			t.Errorf("cert permissions: expected %o, got %o", expectedPerm, perm)
		}
	}

	// Check key file exists and has 0600 permissions
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Errorf("key file not found at %s: %v", keyPath, err)
	} else {
		perm := keyInfo.Mode().Perm()
		expectedPerm := os.FileMode(0600)
		if perm != expectedPerm {
			t.Errorf("key permissions: expected %o, got %o", expectedPerm, perm)
		}
	}

	// Check fullchain file exists
	if _, err := os.Stat(fullChainPath); err != nil {
		t.Errorf("fullchain file not found at %s: %v", fullChainPath, err)
	}

	// Verify cert content in file matches what we sent
	certContent, _ := os.ReadFile(certPath)
	if string(certContent) != string(certPEM) {
		t.Error("cert file content mismatch")
	}

	// Verify key content in file matches what we sent
	keyContent, _ := os.ReadFile(keyPath)
	if string(keyContent) != string(keyPEM) {
		t.Error("key file content mismatch")
	}

	t.Logf("✓ Valid import test passed")
	t.Logf("  Domain: %s", domain)
	t.Logf("  Fingerprint: %s", cert.Fingerprint)
	t.Logf("  Expires: %s", cert.ExpiresAt.Format(time.RFC3339))
	t.Logf("  Cert path: %s", certPath)
	t.Logf("  Key path: %s", keyPath)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Invalid Certificate PEM
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_InvalidCert(t *testing.T) {
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	keyPEM := generateTestKeyPEM(key)

	reqBody := map[string]interface{}{
		"domain":   "test-invalid-cert.example.com",
		"cert_pem": "-----BEGIN CERTIFICATE-----\nINVALID\n-----END CERTIFICATE-----",
		"key_pem":  string(keyPEM),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status is 400 Bad Request
	if resp.StatusCode != fiber.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify no certificate was created in DB
	var count int64
	env.db.Model(&models.SSLCertificate{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 certificates in DB, got %d", count)
	}

	t.Logf("✓ Invalid cert test passed: got status %d", resp.StatusCode)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Invalid Private Key
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_InvalidKey(t *testing.T) {
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	certPEM, _ := generateTestCert(key, "test-invalid-key.example.com", nil)

	reqBody := map[string]interface{}{
		"domain":   "test-invalid-key.example.com",
		"cert_pem": string(certPEM),
		"key_pem":  "-----BEGIN RSA PRIVATE KEY-----\nINVALID\n-----END RSA PRIVATE KEY-----",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status is 400 Bad Request
	if resp.StatusCode != fiber.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify no certificate was created in DB
	var count int64
	env.db.Model(&models.SSLCertificate{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 certificates in DB, got %d", count)
	}

	t.Logf("✓ Invalid key test passed: got status %d", resp.StatusCode)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Key-Cert Mismatch
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_KeyCertMismatch(t *testing.T) {
	env := setupTestEnv(t)

	// Generate two different keys
	key1, _ := generateTestKey()
	key2, _ := generateTestKey()

	// Certificate signed with key1
	certPEM, _ := generateTestCert(key1, "test-mismatch.example.com", nil)

	// But we send key2 (doesn't match)
	keyPEM := generateTestKeyPEM(key2)

	reqBody := map[string]interface{}{
		"domain":   "test-mismatch.example.com",
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status is 400 Bad Request
	if resp.StatusCode != fiber.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify error message contains mismatch info
	bodyBytes, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(bodyBytes), "mismatch") && !strings.Contains(string(bodyBytes), "match") {
		t.Logf("Warning: error response doesn't explicitly mention mismatch: %s", string(bodyBytes))
	}

	// Verify no certificate was created in DB
	var count int64
	env.db.Model(&models.SSLCertificate{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 certificates in DB, got %d", count)
	}

	t.Logf("✓ Key-cert mismatch test passed: got status %d", resp.StatusCode)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Missing Authorization
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_MissingAuth(t *testing.T) {
	env := setupTestEnvNoAuth(t)

	key, _ := generateTestKey()
	certPEM, _ := generateTestCert(key, "test-no-auth.example.com", nil)
	keyPEM := generateTestKeyPEM(key)

	reqBody := map[string]interface{}{
		"domain":   "test-no-auth.example.com",
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status is 401 Unauthorized
	if resp.StatusCode != fiber.StatusUnauthorized {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 401, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	t.Logf("✓ Missing auth test passed: got status %d", resp.StatusCode)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Missing Required Fields
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_MissingFields(t *testing.T) {
	env := setupTestEnv(t)

	// Missing domain
	reqBody := map[string]interface{}{
		"cert_pem": "-----BEGIN CERTIFICATE-----\nMIID\n-----END CERTIFICATE-----",
		"key_pem":  "-----BEGIN RSA PRIVATE KEY-----\nMIID\n-----END RSA PRIVATE KEY-----",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status is 400 Bad Request
	if resp.StatusCode != fiber.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 400, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	t.Logf("✓ Missing fields test passed: got status %d", resp.StatusCode)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - With Chain PEM
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_WithChain(t *testing.T) {
	requireDefaultStorageWritable(t)
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	domain := "test-with-chain.example.com"
	certPEM, _ := generateTestCert(key, domain, []string{domain})
	keyPEM := generateTestKeyPEM(key)

	// Create a chain (self-signed cert acts as CA)
	caKey, _ := generateTestKey()
	caCertPEM, _ := generateTestCert(caKey, "Test CA", nil)
	chainPEM := string(caCertPEM)

	reqBody := map[string]interface{}{
		"domain":    domain,
		"cert_pem":  string(certPEM),
		"key_pem":   string(keyPEM),
		"chain_pem": chainPEM,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify HTTP status
	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify fullchain file contains both cert and chain
	fullChainPath := filepath.Join(env.storage.baseDir, domain, "fullchain.pem")
	fullChainContent, _ := os.ReadFile(fullChainPath)

	// Fullchain should contain 2 certificates (cert + chain)
	certCount := 0
	remaining := fullChainContent
	for {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			certCount++
		}
		remaining = rest
	}

	if certCount != 2 {
		t.Errorf("expected fullchain to contain 2 certs, got %d", certCount)
	}

	// Verify chain file exists
	chainPath := filepath.Join(env.storage.baseDir, domain, "chain.pem")
	if _, err := os.Stat(chainPath); err != nil {
		t.Errorf("chain file not found at %s: %v", chainPath, err)
	}

	t.Logf("✓ With chain test passed: fullchain contains %d certs", certCount)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Tenant Isolation
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_TenantIsolation(t *testing.T) {
	requireDefaultStorageWritable(t)
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	certPEM, _ := generateTestCert(key, "tenant-test.example.com", nil)
	keyPEM := generateTestKeyPEM(key)

	reqBody := map[string]interface{}{
		"domain":   "tenant-test.example.com",
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("request failed with status %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var cert models.SSLCertificate
	json.Unmarshal(bodyBytes, &cert)

	// Verify tenant ID is set correctly from auth claims
	if cert.TenantID != "test-tenant-id" {
		t.Errorf("expected TenantID 'test-tenant-id', got '%s'", cert.TenantID)
	}

	// Verify in database
	var dbCert models.SSLCertificate
	env.db.First(&dbCert, "id = ?", cert.ID)

	if dbCert.TenantID != "test-tenant-id" {
		t.Errorf("DB TenantID mismatch: expected 'test-tenant-id', got '%s'", dbCert.TenantID)
	}

	t.Logf("✓ Tenant isolation test passed: cert belongs to tenant '%s'", cert.TenantID)
}

// -----------------------------------------------------------------------------
// Test: Import Certificate - Duplicate Domain
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_DuplicateDomain(t *testing.T) {
	requireDefaultStorageWritable(t)
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	domain := "duplicate.example.com"
	certPEM, _ := generateTestCert(key, domain, nil)
	keyPEM := generateTestKeyPEM(key)

	reqBody := map[string]interface{}{
		"domain":   domain,
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}

	// First import - should succeed
	body, _ := json.Marshal(reqBody)
	req1 := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := env.app.Test(req1)

	if resp1.StatusCode != fiber.StatusCreated {
		t.Fatalf("first import failed with status %d", resp1.StatusCode)
	}

	// Second import with same domain - should fail or replace
	// The current implementation may allow duplicate, so we just verify behavior
	body2, _ := json.Marshal(reqBody)
	req2 := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := env.app.Test(req2)

	// Current behavior: should create another record (not enforced unique on domain)
	// In production, you might want unique constraint on tenant_id + common_name
	t.Logf("Duplicate domain import: first=%d, second=%d", resp1.StatusCode, resp2.StatusCode)

	var count int64
	env.db.Model(&models.SSLCertificate{}).Where("common_name = ?", domain).Count(&count)
	t.Logf("✓ Duplicate test: %d certificates for domain '%s'", count, domain)
}

// -----------------------------------------------------------------------------
// Test: Response JSON Excludes Sensitive Fields
// -----------------------------------------------------------------------------

func TestImportCertificateHandler_ResponseExcludesPrivateKey(t *testing.T) {
	requireDefaultStorageWritable(t)
	env := setupTestEnv(t)

	key, _ := generateTestKey()
	domain := "security-test.example.com"
	certPEM, _ := generateTestCert(key, domain, nil)
	keyPEM := generateTestKeyPEM(key)

	reqBody := map[string]interface{}{
		"domain":   domain,
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/ssl/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	// List of sensitive patterns that should NOT appear in response
	sensitivePatterns := []string{
		"PRIVATE KEY",
		"key_pem",
		"privkey",
		"key_path",
		"-----BEGIN RSA PRIVATE KEY-----",
		"-----BEGIN EC PRIVATE KEY-----",
		"-----BEGIN PRIVATE KEY-----",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(bodyStr, pattern) {
			t.Errorf("response contains sensitive pattern '%s'", pattern)
		}
	}

	// Verify the response is valid JSON
	var cert models.SSLCertificate
	if err := json.Unmarshal(bodyBytes, &cert); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}

	t.Logf("✓ Response security test passed: no private key material exposed")
}

// Verify testEnv implements interface correctly
var _ interface {
	App() *fiber.App
} = (*testEnv)(nil)

func (e *testEnv) App() *fiber.App { return e.app }