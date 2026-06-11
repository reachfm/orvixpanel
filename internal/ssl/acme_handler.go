package ssl

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ACMEChallengeHandler handles HTTP-01 ACME challenges.
// This is registered at GET /.well-known/acme-challenge/:token
type ACMEChallengeHandler struct {
	store *ChallengeStore
}

// NewACMEChallengeHandler creates a new ACME challenge handler.
func NewACMEChallengeHandler(store *ChallengeStore) *ACMEChallengeHandler {
	return &ACMEChallengeHandler{store: store}
}

// Handle serves an ACME HTTP-01 challenge response.
func (h *ACMEChallengeHandler) Handle(c *fiber.Ctx) error {
	token := c.Params("token")

	// Validate token format
	if err := h.store.ValidateToken(token); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid token")
	}

	// Get challenge content
	content, err := h.store.GetChallenge(c.Context(), token)
	if err != nil {
		if err == ErrChallengeNotFound {
			return c.Status(fiber.StatusNotFound).SendString("challenge not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("internal error")
	}

	// Set proper headers for ACME challenge response
	c.Set("Content-Type", "text/plain")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")

	return c.SendString(content)
}

// Middleware returns a Fiber middleware for ACME challenge handling.
// This can be used with app.Use() to handle challenges before other routes.
func (h *ACMEChallengeHandler) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Check if this is an ACME challenge request
		if !strings.HasPrefix(path, "/.well-known/acme-challenge/") {
			return c.Next()
		}

		// Extract token from path
		token := strings.TrimPrefix(path, "/.well-known/acme-challenge/")
		if token == "" {
			return c.Status(fiber.StatusBadRequest).SendString("missing token")
		}

		// Validate token
		if err := h.store.ValidateToken(token); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("invalid token")
		}

		// Get challenge content
		content, err := h.store.GetChallenge(c.Context(), token)
		if err != nil {
			if err == ErrChallengeNotFound {
				return c.Status(fiber.StatusNotFound).SendString("challenge not found")
			}
			return c.Status(fiber.StatusInternalServerError).SendString("internal error")
		}

		// Set headers and respond
		c.Set("Content-Type", "text/plain")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.SendString(content)
	}
}

// HTTPHandler returns an http.Handler interface for use with standard http mux.
type HTTPHandler struct {
	store *ChallengeStore
}

// NewHTTPHandler creates an HTTP handler for ACME challenges.
func NewHTTPHandler(store *ChallengeStore) *HTTPHandler {
	return &HTTPHandler{store: store}
}

// ServeHTTP implements http.Handler for standard library compatibility.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token, err := ReadTokenFromRequest(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Validate token
	if err := h.store.ValidateToken(token); err != nil {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	// Get challenge content
	content, err := h.store.GetChallenge(ctx, token)
	if err != nil {
		if err == ErrChallengeNotFound {
			http.Error(w, "challenge not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)

	io.Copy(w, bytes.NewReader([]byte(content)))
}

// RegisterWithFiber registers the challenge handler with a Fiber app.
func (h *ACMEChallengeHandler) RegisterWithFiber(app *fiber.App) {
	// Register as a specific route
	app.Get("/.well-known/acme-challenge/:token", h.Handle)

	// Also register the middleware for any unmatched .well-known paths
	app.Use(h.Middleware())
}

// RegisterWithHTTPServer registers with a standard http.Server.
func (h *HTTPHandler) RegisterWithHTTPServer(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}

// ValidateChallenge verifies a challenge is accessible at the expected URL.
// This is used for testing and validation purposes.
func (h *ChallengeStore) ValidateChallenge(ctx context.Context, token, expectedKeyAuth string) (bool, error) {
	content, err := h.GetChallenge(ctx, token)
	if err != nil {
		return false, err
	}

	return content == expectedKeyAuth, nil
}

// CheckChallengeAccessible verifies a challenge file exists and is readable.
func (h *ChallengeStore) CheckChallengeAccessible(token string) error {
	if err := h.ValidateToken(token); err != nil {
		return err
	}

	if !h.ChallengeExists(token) {
		return ErrChallengeNotFound
	}

	return nil
}

// GetChallengeFileInfo returns information about a challenge file.
func (h *ChallengeStore) GetChallengeFileInfo(token string) (*ChallengeFileInfo, error) {
	if err := h.ValidateToken(token); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(h.baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := h.ValidateToken(entry.Name()); err != nil {
			continue
		}

		filePath := filepath.Join(h.baseDir, entry.Name(), token)
		info, err := os.Stat(filePath)
		if err == nil {
			return &ChallengeFileInfo{
				Path:       filePath,
				Domain:     entry.Name(),
				Token:      token,
				Size:       info.Size(),
				ModTime:    info.ModTime(),
				Mode:       info.Mode(),
				ReadOK:     isFileReadable(filePath),
			}, nil
		}
	}

	return nil, ErrChallengeNotFound
}

// ChallengeFileInfo holds information about a challenge file.
type ChallengeFileInfo struct {
	Path    string
	Domain  string
	Token   string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	ReadOK  bool
}

// isFileReadable checks if a file is readable.
func isFileReadable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// EnsureChallengeSetup creates the challenge directory and sets permissions.
func EnsureChallengeSetup(baseDir string) error {
	if err := os.MkdirAll(baseDir, ChallengeDirMode); err != nil {
		return err
	}

	// Ensure directory is accessible by nginx (www-data or nginx user)
	// In production, you may need to setfacl or change ownership
	return nil
}

// CleanupAllChallenges removes all challenge files.
func (h *ChallengeStore) CleanupAllChallenges() error {
	return os.RemoveAll(h.baseDir)
}

// ListChallengeTokens returns all active challenge tokens.
func (h *ChallengeStore) ListChallengeTokens() []string {
	var tokens []string

	entries, err := os.ReadDir(h.baseDir)
	if err != nil {
		return tokens
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := h.ValidateToken(entry.Name()); err != nil {
			continue
		}

		domainDir := filepath.Join(h.baseDir, entry.Name())
		challenges, err := os.ReadDir(domainDir)
		if err != nil {
			continue
		}

		for _, challenge := range challenges {
			if err := h.ValidateToken(challenge.Name()); err == nil {
				tokens = append(tokens, challenge.Name())
			}
		}
	}

	return tokens
}

// ChallengeStats holds statistics about challenges.
type ChallengeStats struct {
	DomainCount    int
	ChallengeCount int
	TotalSize      int64
	OldestChallenge *time.Time
	NewestChallenge *time.Time
}

// GetStats returns challenge storage statistics.
func (h *ChallengeStore) GetStats() (*ChallengeStats, error) {
	stats := &ChallengeStats{}

	entries, err := os.ReadDir(h.baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := h.ValidateToken(entry.Name()); err != nil {
			continue
		}

		stats.DomainCount++
		domainDir := filepath.Join(h.baseDir, entry.Name())
		challenges, err := os.ReadDir(domainDir)
		if err != nil {
			continue
		}

		for _, challenge := range challenges {
			info, err := challenge.Info()
			if err != nil {
				continue
			}

			stats.ChallengeCount++
			stats.TotalSize += info.Size()

			if stats.OldestChallenge == nil || info.ModTime().Before(*stats.OldestChallenge) {
				t := info.ModTime()
				stats.OldestChallenge = &t
			}
			if stats.NewestChallenge == nil || info.ModTime().After(*stats.NewestChallenge) {
				t := info.ModTime()
				stats.NewestChallenge = &t
			}
		}
	}

	return stats, nil
}