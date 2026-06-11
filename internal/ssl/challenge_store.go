package ssl

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// ChallengeDir is the default directory for HTTP-01 challenge files
	ChallengeDir = "/var/lib/orvixpanel/acme-challenges"

	// ChallengeFileMode is the file permission for challenge files (readable by nginx)
	ChallengeFileMode = 0644

	// ChallengeDirMode is the directory permission for challenge directory
	ChallengeDirMode = 0755

	// MaxTokenLength is the maximum length for an ACME token
	MaxTokenLength = 128

	// ChallengeValidityHours is how long a challenge file is valid
	ChallengeValidityHours = 7 * 24 // 7 days
)

var (
	// validTokenPattern matches safe ACME tokens (no path traversal chars)
	validTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// ChallengeStore handles ACME HTTP-01 challenge storage and retrieval.
type ChallengeStore struct {
	baseDir    string
	httpPrefix string // URL prefix for challenge serving
}

// NewChallengeStore creates a new challenge store.
func NewChallengeStore(baseDir string) *ChallengeStore {
	return &ChallengeStore{
		baseDir:    baseDir,
		httpPrefix: "/.well-known/acme-challenge",
	}
}

// ValidateToken checks if a token is safe (no path traversal).
func (s *ChallengeStore) ValidateToken(token string) error {
	if token == "" {
		return errors.New("token is empty")
	}

	if len(token) > MaxTokenLength {
		return fmt.Errorf("token exceeds maximum length of %d", MaxTokenLength)
	}

	if !validTokenPattern.MatchString(token) {
		return errors.New("token contains invalid characters")
	}

	// Ensure token doesn't contain path separators after normalization
	cleanPath := filepath.ToSlash(filepath.Clean(token))
	if strings.Contains(cleanPath, "/") {
		return errors.New("token contains path separators")
	}

	return nil
}

// StoreChallenge writes a challenge file to disk.
func (s *ChallengeStore) StoreChallenge(ctx context.Context, token, keyAuth, domain string) error {
	// Validate token first
	if err := s.ValidateToken(token); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Validate keyAuth format (should be token + "." + thumbprint)
	if keyAuth == "" {
		return errors.New("keyAuth is empty")
	}

	// Create directory structure: baseDir/domain/token
	domainDir := filepath.Join(s.baseDir, sanitizeDomain(domain))
	if err := os.MkdirAll(domainDir, ChallengeDirMode); err != nil {
		return fmt.Errorf("create domain dir: %w", err)
	}

	// Write challenge file
	filePath := filepath.Join(domainDir, token)
	if err := os.WriteFile(filePath, []byte(keyAuth), ChallengeFileMode); err != nil {
		return fmt.Errorf("write challenge file: %w", err)
	}

	return nil
}

// GetChallenge retrieves a challenge file content.
func (s *ChallengeStore) GetChallenge(ctx context.Context, token string) (string, error) {
	// Validate token
	if err := s.ValidateToken(token); err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	// Find challenge file (search all domain directories)
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrChallengeNotFound
		}
		return "", fmt.Errorf("read base dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Verify domain is safe
		if err := s.ValidateToken(entry.Name()); err != nil {
			continue // Skip invalid domain directories
		}

		filePath := filepath.Join(s.baseDir, entry.Name(), token)
		content, err := os.ReadFile(filePath)
		if err == nil {
			return string(content), nil
		}
		if !os.IsNotExist(err) {
			continue
		}
	}

	return "", ErrChallengeNotFound
}

// DeleteChallenge removes a challenge file.
func (s *ChallengeStore) DeleteChallenge(ctx context.Context, token string) error {
	// Validate token
	if err := s.ValidateToken(token); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Search for and remove the challenge file
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return fmt.Errorf("read base dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := s.ValidateToken(entry.Name()); err != nil {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name(), token)
		if err := os.Remove(filePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("remove challenge: %w", err)
		}

		return nil
	}

	return nil
}

// DeleteDomainChallenges removes all challenge files for a domain.
func (s *ChallengeStore) DeleteDomainChallenges(ctx context.Context, domain string) error {
	safeDomain := sanitizeDomain(domain)
	domainDir := filepath.Join(s.baseDir, safeDomain)

	if err := os.RemoveAll(domainDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove domain challenges: %w", err)
	}

	return nil
}

// CleanupExpiredChallenges removes challenge files older than maxAge.
func (s *ChallengeStore) CleanupExpiredChallenges(maxAge time.Duration) error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read base dir: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, domainEntry := range entries {
		if !domainEntry.IsDir() {
			continue
		}

		if err := s.ValidateToken(domainEntry.Name()); err != nil {
			// Remove invalid domain directories
			os.RemoveAll(filepath.Join(s.baseDir, domainEntry.Name()))
			continue
		}

		domainDir := filepath.Join(s.baseDir, domainEntry.Name())
		challenges, err := os.ReadDir(domainDir)
		if err != nil {
			continue
		}

		for _, challenge := range challenges {
			info, err := challenge.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(domainDir, challenge.Name()))
			}
		}
	}

	return nil
}

// ChallengeExists checks if a challenge file exists.
func (s *ChallengeStore) ChallengeExists(token string) bool {
	if err := s.ValidateToken(token); err != nil {
		return false
	}

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := s.ValidateToken(entry.Name()); err != nil {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name(), token)
		if _, err := os.Stat(filePath); err == nil {
			return true
		}
	}

	return false
}

// ServeChallenge serves a challenge file via HTTP.
// This is used by the HTTP handler to respond to ACME validation requests.
func (s *ChallengeStore) ServeChallenge(w http.ResponseWriter, r *http.Request, token string) error {
	// Only allow GET
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return errors.New("method not allowed")
	}

	// Get challenge content
	content, err := s.GetChallenge(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrChallengeNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return err
		}
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	// Serve content
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(http.StatusOK)

	_, err = w.Write([]byte(content))
	return err
}

// GetChallengeURL returns the HTTP-01 challenge URL for a token.
func (s *ChallengeStore) GetChallengeURL(token string) string {
	return s.httpPrefix + "/" + token
}

// GenerateKeyAuth generates a key authorization string for HTTP-01.
// token: the ACME challenge token
// thumbprint: the SHA256 thumbprint of the account public key
func GenerateKeyAuth(token, thumbprint string) string {
	return token + "." + thumbprint
}

// GenerateThumbprint generates a JWK thumbprint for an account key.
func GenerateThumbprint(accountKey *AccountKey) (string, error) {
	if accountKey == nil || accountKey.Key == nil {
		return "", errors.New("nil account key")
	}

	// Generate JWK thumbprint (simplified - in production would use proper JWK serialization)
	// For now, just return a hash of the private key bytes
	hash := sha256.Sum256(accountKey.Key.PrivateKeyBytes)
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

// EnsureChallengeDir creates the challenge directory if it doesn't exist.
func EnsureChallengeDir(baseDir string) error {
	return os.MkdirAll(baseDir, ChallengeDirMode)
}

// sanitizeDomain creates a safe directory name from a domain.
func sanitizeDomain(domain string) string {
	// Replace dots and special chars with underscores
	safe := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(domain, "_")
	return strings.ToLower(safe)
}

// GetChallengeExpiry returns the expiry time for a challenge file.
func (s *ChallengeStore) GetChallengeExpiry(token string) (*time.Time, error) {
	if err := s.ValidateToken(token); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read base dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := s.ValidateToken(entry.Name()); err != nil {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name(), token)
		info, err := os.Stat(filePath)
		if err == nil {
			modTime := info.ModTime()
			expiry := modTime.Add(ChallengeValidityHours * time.Hour)
			return &expiry, nil
		}
	}

	return nil, ErrChallengeNotFound
}

// ErrChallengeNotFound indicates the challenge file was not found.
var ErrChallengeNotFound = errors.New("challenge not found")

// AccountKey holds the ACME account key pair.
type AccountKey struct {
	ID      string
	TenantID string
	Email   string
	Key     *AccountPrivateKey
}

// AccountPrivateKey wraps the RSA private key for ACME operations.
type AccountPrivateKey struct {
	*rsaPrivateKeyWrapper
}

// rsaPrivateKeyWrapper wraps an RSA private key for serialization.
type rsaPrivateKeyWrapper struct {
	PrivateKeyBytes []byte // PKCS8 or PKCS1 encoded
	CreatedAt       time.Time
}

// LoadAccountKey loads an account key from disk.
func LoadAccountKey(path string) (*AccountKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	_, err = ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	// Create wrapper (simplified - real implementation would include metadata)
	return &AccountKey{
		Key: &AccountPrivateKey{
			rsaPrivateKeyWrapper: &rsaPrivateKeyWrapper{
				PrivateKeyBytes: data,
				CreatedAt:       time.Now(),
			},
		},
	}, nil
}

// SaveAccountKey saves an account key to disk.
func SaveAccountKey(path string, key *AccountKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}

	return os.WriteFile(path, key.Key.PrivateKeyBytes, 0600)
}

// GenerateAccountKey generates a new ACME account key pair.
func GenerateAccountKey(tenantID, email string) (*AccountKey, error) {
	// Generate RSA key
	privateKey, err := generateRSAKey(2048)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	keyBytes, err := encodePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("encode key: %w", err)
	}

	return &AccountKey{
		ID:       generateID(),
		TenantID: tenantID,
		Email:    email,
		Key: &AccountPrivateKey{
			rsaPrivateKeyWrapper: &rsaPrivateKeyWrapper{
				PrivateKeyBytes: keyBytes,
				CreatedAt:       time.Now(),
			},
		},
	}, nil
}

// generateRSAKey generates an RSA private key.
func generateRSAKey(bits int) (*rsaPrivateKeyWrapper, error) {
	// Use crypto/rsa to generate key
	// This is a placeholder - real implementation would use crypto/rsa.GenerateKey
	return &rsaPrivateKeyWrapper{}, nil
}

// encodePrivateKey encodes a private key to PEM format.
func encodePrivateKey(key *rsaPrivateKeyWrapper) ([]byte, error) {
	// Simplified - real implementation would encode RSA key
	return key.PrivateKeyBytes, nil
}

// generateID generates a unique ID for account keys.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ReadTokenFromRequest extracts the challenge token from a URL path.
func ReadTokenFromRequest(path string) (string, error) {
	// Expected format: /.well-known/acme-challenge/<token>
	const prefix = "/.well-known/acme-challenge/"

	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid challenge path")
	}

	token := strings.TrimPrefix(path, prefix)
	if token == "" {
		return "", errors.New("empty token")
	}

	// Check for trailing content (e.g., query string)
	if idx := strings.IndexAny(token, "?#"); idx != -1 {
		token = token[:idx]
	}

	return token, nil
}

// ServeChallengeHTTP is an HTTP handler for serving ACME challenges.
// This can be registered directly with the Fiber app.
func (s *ChallengeStore) ServeChallengeHTTP(w http.ResponseWriter, r *http.Request) error {
	token, err := ReadTokenFromRequest(r.URL.Path)
	if err != nil {
		return err
	}

	return s.ServeChallenge(w, r, token)
}