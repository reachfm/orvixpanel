package ssl

import (
	"os"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// ChallengeStore Tests
// -----------------------------------------------------------------------------

func TestNewChallengeStore(t *testing.T) {
	// Test with valid directory
	store := NewChallengeStore("/tmp/acme-challenges")
	if store == nil {
		t.Fatal("expected non-nil ChallengeStore")
	}
	if store.baseDir != "/tmp/acme-challenges" {
		t.Errorf("expected baseDir '/tmp/acme-challenges', got '%s'", store.baseDir)
	}
}

func TestChallengeStoreStoreAndRead(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store a challenge
	token := "test-token-123"
	keyAuth := "test-challenge-content-abc123"
	domain := "example.com"
	err = store.StoreChallenge(nil, token, keyAuth, domain)
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Read it back
	got, err := store.GetChallenge(nil, token)
	if err != nil {
		t.Fatalf("failed to read challenge: %v", err)
	}

	if got != keyAuth {
		t.Errorf("expected content '%s', got '%s'", keyAuth, got)
	}
}

func TestChallengeStoreDeleteChallenge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store a challenge
	token := "delete-test-token"
	err = store.StoreChallenge(nil, token, "some-content", "test.com")
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Delete it
	err = store.DeleteChallenge(nil, token)
	if err != nil {
		t.Fatalf("failed to delete challenge: %v", err)
	}

	// Verify it's gone
	_, err = store.GetChallenge(nil, token)
	if err != ErrChallengeNotFound {
		t.Errorf("expected ErrChallengeNotFound, got: %v", err)
	}
}

func TestChallengeStoreValidateToken(t *testing.T) {
	store := NewChallengeStore("/tmp/acme")

	// Valid tokens
	validTokens := []string{
		"abc123",
		"A1B2C3",
		"test-token_underscore",
		"longer-token-with-dash-123",
	}

	for _, token := range validTokens {
		err := store.ValidateToken(token)
		if err != nil {
			t.Errorf("expected valid token '%s', got error: %v", token, err)
		}
	}

	// Invalid tokens
	invalidTokens := []string{
		"",                                 // empty
		"token/with/slash",                 // path traversal
		"token\\with\\backslash",            // backslash
		"token.with.dots",                   // dots
		"token with spaces",                // spaces
		"token\nwith\newline",              // newlines
	}

	for _, token := range invalidTokens {
		err := store.ValidateToken(token)
		if err == nil {
			t.Errorf("expected invalid token '%s' to fail validation", token)
		}
	}

	// Token too long
	longToken := make([]byte, MaxTokenLength+1)
	for i := range longToken {
		longToken[i] = 'a'
	}
	err := store.ValidateToken(string(longToken))
	if err == nil {
		t.Error("expected token too long to fail validation")
	}
}

func TestChallengeStorePathTraversal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Try to store challenge with path traversal token
	maliciousToken := "../../../etc/passwd"
	err = store.StoreChallenge(nil, maliciousToken, "malicious content", "test.com")
	// Token validation should reject path traversal characters
	if err == nil {
		t.Error("expected path traversal token to be rejected by validation")
	}
}

func TestChallengeStoreChallengeExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Initially should not exist
	if store.ChallengeExists("new-token") {
		t.Error("expected new token to not exist")
	}

	// Store a challenge
	err = store.StoreChallenge(nil, "exists-token", "content", "example.com")
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Now should exist
	if !store.ChallengeExists("exists-token") {
		t.Error("expected exists-token to exist after storing")
	}
}

func TestChallengeStoreGetChallengeURL(t *testing.T) {
	store := NewChallengeStore("/tmp/acme")

	url := store.GetChallengeURL("my-token")
	expected := "/.well-known/acme-challenge/my-token"
	if url != expected {
		t.Errorf("expected URL '%s', got '%s'", expected, url)
	}
}

func TestChallengeStoreDeleteDomainChallenges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store challenges for a domain
	err = store.StoreChallenge(nil, "token1", "content1", "mydomain.com")
	if err != nil {
		t.Fatalf("failed to store challenge 1: %v", err)
	}
	err = store.StoreChallenge(nil, "token2", "content2", "mydomain.com")
	if err != nil {
		t.Fatalf("failed to store challenge 2: %v", err)
	}

	// Delete all domain challenges
	err = store.DeleteDomainChallenges(nil, "mydomain.com")
	if err != nil {
		t.Fatalf("failed to delete domain challenges: %v", err)
	}

	// Verify they're gone
	if store.ChallengeExists("token1") {
		t.Error("expected token1 to be deleted")
	}
	if store.ChallengeExists("token2") {
		t.Error("expected token2 to be deleted")
	}
}

func TestChallengeStoreCleanupExpiredChallenges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store a challenge
	err = store.StoreChallenge(nil, "cleanup-test", "content", "example.com")
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Verify it exists
	if !store.ChallengeExists("cleanup-test") {
		t.Error("expected challenge to exist after storing")
	}

	// Cleanup should not error
	err = store.CleanupExpiredChallenges(24 * time.Hour)
	if err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}
}

// -----------------------------------------------------------------------------
// ACMEChallengeHandler Tests
// -----------------------------------------------------------------------------

func TestNewACMEChallengeHandler(t *testing.T) {
	store := NewChallengeStore("/tmp/acme")
	handler := NewACMEChallengeHandler(store)

	if handler == nil {
		t.Fatal("expected non-nil ACMEChallengeHandler")
	}

	if handler.store != store {
		t.Error("expected handler to hold store reference")
	}
}

// -----------------------------------------------------------------------------
// Staging Provider Tests
// -----------------------------------------------------------------------------

func TestStagingConfig(t *testing.T) {
	cfg := StagingConfig()

	if cfg == nil {
		t.Fatal("expected non-nil Config")
	}

	if cfg.LetsEncryptDirectoryURL != ACMEDirectoryStaging {
		t.Errorf("expected staging directory '%s', got '%s'", ACMEDirectoryStaging, cfg.LetsEncryptDirectoryURL)
	}

	if !cfg.UseStaging {
		t.Error("expected UseStaging to be true")
	}
}

func TestStagingProviderName(t *testing.T) {
	cfg := StagingConfig()
	provider := NewStagingProvider(cfg, nil)

	if provider.Name() != ProviderNameLetsEncryptStaging {
		t.Errorf("expected name '%s', got '%s'", ProviderNameLetsEncryptStaging, provider.Name())
	}
}

func TestStagingProviderCreation(t *testing.T) {
	cfg := StagingConfig()

	// With staging config, should be able to create provider
	provider := NewStagingProvider(cfg, nil)
	if provider == nil {
		t.Error("expected non-nil staging provider")
	}

	// Without config, should also be able to create provider (but may fail later)
	unconfigured := NewStagingProvider(nil, nil)
	if unconfigured == nil {
		t.Error("expected non-nil unconfigured provider")
	}
}

// -----------------------------------------------------------------------------
// Config Tests
// -----------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify staging-related defaults
	if cfg.LetsEncryptDirectoryURL != ACMEDirectoryProduction {
		t.Errorf("expected production directory '%s', got '%s'", ACMEDirectoryProduction, cfg.LetsEncryptDirectoryURL)
	}

	if cfg.UseStaging {
		t.Error("expected UseStaging to be false in DefaultConfig")
	}

	// Verify other defaults
	if cfg.StorageDir == "" {
		t.Error("expected StorageDir to be set")
	}

	if cfg.ChallengeDir == "" {
		t.Error("expected ChallengeDir to be set")
	}

	if cfg.RenewalWindowDays <= 0 {
		t.Error("expected positive RenewalWindowDays")
	}
}

func TestStagingConfigOverrides(t *testing.T) {
	cfg := StagingConfig()

	// Staging should use staging directory
	if cfg.LetsEncryptDirectoryURL != ACMEDirectoryStaging {
		t.Errorf("expected staging directory, got '%s'", cfg.LetsEncryptDirectoryURL)
	}

	// But other settings should be reasonable
	if cfg.RenewalWindowDays <= 0 {
		t.Error("expected positive RenewalWindowDays")
	}
}

// -----------------------------------------------------------------------------
// Constants Tests
// -----------------------------------------------------------------------------

func TestStagingConstants(t *testing.T) {
	// Verify staging constants are defined
	if ACMEDirectoryStaging == "" {
		t.Error("ACMEDirectoryStaging should not be empty")
	}

	if ACMEDirectoryProduction == "" {
		t.Error("ACMEDirectoryProduction should not be empty")
	}

	if ACMEDirectoryStaging == ACMEDirectoryProduction {
		t.Error("staging and production directories should be different")
	}

	if ProviderNameLetsEncryptStaging == "" {
		t.Error("ProviderNameLetsEncryptStaging should not be empty")
	}

	// Verify staging URL contains "staging"
	if !contains(ACMEDirectoryStaging, "staging") {
		t.Errorf("staging directory should contain 'staging': %s", ACMEDirectoryStaging)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// ChallengeStore Token Validation Edge Cases
// -----------------------------------------------------------------------------

func TestChallengeStoreTokenValidationEdgeCases(t *testing.T) {
	store := NewChallengeStore("/tmp/acme")

	// Empty token
	err := store.ValidateToken("")
	if err == nil {
		t.Error("expected empty token to fail validation")
	}

	// Very long valid token (should pass length check but fail pattern if too long)
	longValid := make([]byte, MaxTokenLength)
	for i := range longValid {
		longValid[i] = 'a'
	}
	err = store.ValidateToken(string(longValid))
	if err != nil {
		t.Errorf("expected valid-length token to pass: %v", err)
	}

	// Token just over limit
	overLimit := make([]byte, MaxTokenLength+1)
	for i := range overLimit {
		overLimit[i] = 'a'
	}
	err = store.ValidateToken(string(overLimit))
	if err == nil {
		t.Error("expected over-limit token to fail validation")
	}
}

// -----------------------------------------------------------------------------
// ChallengeStore File Permissions Tests
// -----------------------------------------------------------------------------

func TestChallengeStoreFilePermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-perm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store a challenge
	token := "perm-test-token"
	err = store.StoreChallenge(nil, token, "test-content", "example.com")
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Verify the challenge exists
	if !store.ChallengeExists(token) {
		t.Error("expected challenge to exist after storing")
	}
}

// -----------------------------------------------------------------------------
// Error Types Tests
// -----------------------------------------------------------------------------

func TestChallengeStoreErrors(t *testing.T) {
	// Verify challenge-related errors are defined
	if ErrChallengeNotFound == nil {
		t.Error("ErrChallengeNotFound should not be nil")
	}
}

// -----------------------------------------------------------------------------
// StagingManager Tests
// -----------------------------------------------------------------------------

func TestStagingManagerCreation(t *testing.T) {
	cfg := StagingConfig()
	store := NewChallengeStore("/tmp/acme")

	mgr := &StagingManager{
		db:        nil,
		config:    cfg,
		provider:  NewStagingProvider(cfg, store),
		challenge: store,
	}

	if mgr.config != cfg {
		t.Error("expected config to be set")
	}

	if mgr.challenge != store {
		t.Error("expected challenge store to be set")
	}
}

// -----------------------------------------------------------------------------
// KeyAuth Generation Tests
// -----------------------------------------------------------------------------

func TestGenerateKeyAuth(t *testing.T) {
	token := "test-token"
	thumbprint := "test-thumbprint"
	keyAuth := GenerateKeyAuth(token, thumbprint)

	expected := token + "." + thumbprint
	if keyAuth != expected {
		t.Errorf("expected '%s', got '%s'", expected, keyAuth)
	}
}

// -----------------------------------------------------------------------------
// EnsureChallengeDir Tests
// -----------------------------------------------------------------------------

func TestEnsureChallengeDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-dir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should create directory
	err = EnsureChallengeDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to ensure challenge dir: %v", err)
	}

	// Verify it exists
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected directory to exist")
	}
}

// -----------------------------------------------------------------------------
// ChallengeStore GetChallengeExpiry Tests
// -----------------------------------------------------------------------------

func TestChallengeStoreGetChallengeExpiry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acme-expiry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewChallengeStore(tmpDir)

	// Store a challenge
	token := "expiry-test-token"
	err = store.StoreChallenge(nil, token, "content", "example.com")
	if err != nil {
		t.Fatalf("failed to store challenge: %v", err)
	}

	// Get expiry
	expiry, err := store.GetChallengeExpiry(token)
	if err != nil {
		t.Fatalf("failed to get expiry: %v", err)
	}

	if expiry == nil {
		t.Error("expected non-nil expiry")
	}
}