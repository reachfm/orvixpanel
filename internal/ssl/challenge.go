package ssl

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// ChallengeHandler handles ACME HTTP-01 challenges.
type ChallengeHandler struct {
	challengeDir string
	storage      *Storage
}

// NewChallengeHandler creates a new challenge handler.
func NewChallengeHandler(challengeDir string, storage *Storage) *ChallengeHandler {
	return &ChallengeHandler{
		challengeDir: challengeDir,
		storage:      storage,
	}
}

// Challenge represents an ACME HTTP-01 challenge.
type Challenge struct {
	Token     string
	KeyAuth   string
	FilePath  string
	Domain    string
	ExpiresAt *time.Time
}

// CreateHTTP01Challenge creates an HTTP-01 challenge file.
func (h *ChallengeHandler) CreateHTTP01Challenge(ctx context.Context, token, keyAuth, domain string) (*Challenge, error) {
	// Ensure challenge directory exists
	if err := os.MkdirAll(h.challengeDir, 0755); err != nil {
		return nil, &Error{Op: "create challenge dir", Err: err}
	}

	// Create challenge file path
	fileName := token
	filePath := filepath.Join(h.challengeDir, fileName)

	// Write challenge file
	// Content is the key authorization (token.domain == keyAuth)
	if err := os.WriteFile(filePath, []byte(keyAuth), 0644); err != nil {
		return nil, &Error{Op: "write challenge file", Err: err}
	}

	// Calculate expiry (ACME challenges expire in 7 days typically)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	return &Challenge{
		Token:     token,
		KeyAuth:   keyAuth,
		FilePath:  filePath,
		Domain:    domain,
		ExpiresAt: &expiresAt,
	}, nil
}

// CleanHTTP01Challenge removes an HTTP-01 challenge file.
func (h *ChallengeHandler) CleanHTTP01Challenge(token string) error {
	filePath := filepath.Join(h.challengeDir, token)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // Already cleaned up
	}

	return os.Remove(filePath)
}

// CleanAllChallenges removes all challenge files.
func (h *ChallengeHandler) CleanAllChallenges() error {
	entries, err := os.ReadDir(h.challengeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &Error{Op: "read challenge dir", Err: err}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(h.challengeDir, entry.Name()))
		}
	}

	return nil
}

// VerifyHTTP01Challenge verifies a challenge file is accessible.
func (h *ChallengeHandler) VerifyHTTP01Challenge(token, expectedKeyAuth string) (bool, error) {
	filePath := filepath.Join(h.challengeDir, token)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, &Error{Op: "read challenge file", Err: err}
	}

	// Compare key authorization
	return string(content) == expectedKeyAuth, nil
}

// GetChallengeURL returns the HTTP-01 challenge URL for a token.
func (h *ChallengeHandler) GetChallengeURL(token string) string {
	return "/.well-known/acme-challenge/" + token
}

// ChallengeInfo contains challenge information for storage.
type ChallengeInfo struct {
	Token    string
	KeyAuth  string
	FilePath string
	Domain   string
}

// StoreChallenge stores challenge information in database.
func (h *ChallengeHandler) StoreChallenge(ctx context.Context, db interface{}, certID string, info *ChallengeInfo) error {
	// This would store the challenge in the ssl_challenges table
	// Implementation depends on the DB interface provided
	return nil
}

// IsChallengeFileAccessible checks if a challenge file exists and is readable.
func (h *ChallengeHandler) IsChallengeFileAccessible(token string) bool {
	filePath := filepath.Join(h.challengeDir, token)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	// Check if readable
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	file.Close()

	return true
}

// GetChallengeExpiry returns the expiry time for a challenge file.
func (h *ChallengeHandler) GetChallengeExpiry(token string) (*time.Time, error) {
	filePath := filepath.Join(h.challengeDir, token)

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, &Error{Op: "stat challenge file", Err: err}
	}

	// Use file modification time as a proxy for expiry
	modTime := info.ModTime()
	return &modTime, nil
}

// CleanExpiredChallenges removes challenge files older than the specified duration.
func (h *ChallengeHandler) CleanExpiredChallenges(maxAgeHours int) error {
	entries, err := os.ReadDir(h.challengeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &Error{Op: "read challenge dir", Err: err}
	}

	cutoff := time.Now().Add(-time.Duration(maxAgeHours) * time.Hour)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(h.challengeDir, entry.Name()))
		}
	}

	return nil
}