// Package auth — API key support (v0.3.0 Enterprise Edition).
//
// API keys are long-lived credentials for automation. Format:
//   orx_live_<prefix8>_<secret32>
//
// Where:
//   - prefix8 = 8 random base32 chars (Crockford alphabet, public)
//   - secret32 = 32 random base32 chars (private, shown ONCE on create)
//
// The full key is never stored. We store SHA-256(prefix8 || secret32).
// On auth, we hash the presented key and look it up by hash + prefix.
//
// Auth precedence (set in middleware):
//   1. JWT in Authorization: Bearer (existing path)
//   2. API key in X-Orvix-Api-Key or Authorization: Bearer orx_live_...
//
// All API-key requests get an *auth.Claims synthesized from the key's
// stored role + tenant + scopes, then flow through the same
// TenantMiddleware + RBAC middleware as JWT users.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// Errors.
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")
	ErrAPIKeyExpired  = errors.New("api key expired")
	ErrAPIKeyMalformed = errors.New("api key malformed")
)

// KeyPrefix is the public part of every OrvixPanel API key.
const KeyPrefix = "orx_live_"

// CreateAPIKeyRequest is the body for POST /admin/api-keys.
type CreateAPIKeyRequest struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`     // built-in or custom
	Scopes    []string   `json:"scopes"`   // ["domain.read", "hosting.create", ...]
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse is returned ONCE on create. The Secret field
// is the only time the caller can see the full key.
type CreateAPIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	Secret    string     `json:"secret"` // full orx_live_<prefix>_<secret> — shown ONCE
	Role      string     `json:"role"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// KeyService is the API-key service.
type KeyService struct {
	db *gorm.DB
}

// NewKeyService constructs a KeyService.
func NewKeyService(db *gorm.DB) *KeyService {
	return &KeyService{db: db}
}

// Create issues a new API key. Returns the response with the
// full key — the secret portion is shown once and never recoverable.
func (s *KeyService) Create(ctx context.Context, tenantID, createdByID string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Role == "" {
		return nil, fmt.Errorf("role is required")
	}
	prefix, _, full, hash, err := generateKey()
	if err != nil {
		return nil, err
	}
	scopesBlob := strings.Join(req.Scopes, ",")
	row := models.APIKey{
		TenantID:    tenantID,
		CreatedByID: createdByID,
		Name:        req.Name,
		KeyHash:     hash,
		Prefix:      prefix,
		Role:        req.Role,
		Scopes:      scopesBlob,
		ExpiresAt:   req.ExpiresAt,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &CreateAPIKeyResponse{
		ID:        row.ID,
		Name:      row.Name,
		Prefix:    prefix,
		Secret:    full,
		Role:      row.Role,
		Scopes:    req.Scopes,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

// List returns the metadata for every API key in the tenant.
// Never returns the hash or the secret.
func (s *KeyService) List(ctx context.Context, tenantID string) ([]models.APIKey, error) {
	var rows []models.APIKey
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Revoke marks a key as revoked. Returns ErrAPIKeyNotFound if the
// ID doesn't exist in the tenant.
func (s *KeyService) Revoke(ctx context.Context, tenantID, keyID, reason string) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("tenant_id = ? AND id = ?", tenantID, keyID).
		Updates(map[string]any{
			"revoked_at":    &now,
			"revoke_reason": reason,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// Verify looks up a presented key. On success returns the row
// (caller reads .Role / .TenantID / .Scopes). Also bumps
// LastUsedAt + LastUsedIP — done async-safe (best effort).
func (s *KeyService) Verify(ctx context.Context, presented string, ip string) (*models.APIKey, error) {
	if !strings.HasPrefix(presented, KeyPrefix) {
		return nil, ErrAPIKeyMalformed
	}
	rest := strings.TrimPrefix(presented, KeyPrefix)
	// rest = "<prefix8>_<secret32>"
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 {
		return nil, ErrAPIKeyMalformed
	}
	prefix := parts[0]
	hash := hashKey(presented)

	var row models.APIKey
	err := s.db.WithContext(ctx).
		Where("key_hash = ? AND prefix = ?", hash, prefix).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	if !row.IsActive() {
		if row.RevokedAt != nil {
			return nil, ErrAPIKeyRevoked
		}
		return nil, ErrAPIKeyExpired
	}

	// Best-effort: bump LastUsedAt / LastUsedIP. If this fails we
	// don't fail the auth — usage tracking is observability, not
	// security.
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", row.ID).
		Updates(map[string]any{
			"last_used_at": &now,
			"last_used_ip": ip,
		}).Error
	row.LastUsedAt = &now
	row.LastUsedIP = ip

	return &row, nil
}

// Count returns the number of active API keys in a tenant. Used by
// the quota service.
func (s *KeyService) Count(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	if err := s.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("tenant_id = ? AND revoked_at IS NULL", tenantID).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ClaimsForKey synthesizes a *Claims from a verified API key. The
// SessionID is the API key's own ID so audit logs can correlate.
func ClaimsForKey(k *models.APIKey) *Claims {
	return &Claims{
		UserID:    "apikey:" + k.ID,
		Email:     "apikey:" + k.Prefix + "@" + k.TenantID,
		Role:      k.Role,
		TenantID:  k.TenantID,
		AccountID: k.CreatedByID,
		SessionID: "apikey:" + k.ID,
	}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// generateKey returns (prefix, secret, fullKey, sha256Hex).
// prefix = 8 chars Crockford base32. secret = 32 chars Crockford.
// full = "orx_live_" + prefix + "_" + secret.
func generateKey() (prefix, secret, full, hashHex string, err error) {
	p, err := randomBase32(8)
	if err != nil {
		return "", "", "", "", err
	}
	s, err := randomBase32(32)
	if err != nil {
		return "", "", "", "", err
	}
	full = KeyPrefix + p + "_" + s
	secret = s
	h := sha256.Sum256([]byte(full))
	hashHex = hex.EncodeToString(h[:])
	return p, s, full, hashHex, nil
}

func randomBase32(nChars int) (string, error) {
	// Crockford base32 has no padding and avoids look-alike chars.
	enc := base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)
	// nChars base32 chars = ceil(nChars*5/8) random bytes.
	nBytes := (nChars*5 + 7) / 8
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Mask off the trailing bits so the decoded length is exact.
	extraBits := nBytes*8 - nChars*5
	if extraBits > 0 {
		b[len(b)-1] &= (1 << (8 - extraBits)) - 1
	}
	s := enc.EncodeToString(b)
	if len(s) < nChars {
		return "", fmt.Errorf("base32 too short: %d for %d", len(s), nChars)
	}
	return s[:nChars], nil
}

// hashKey returns the lowercase hex SHA-256 of the full key.
func hashKey(full string) string {
	h := sha256.Sum256([]byte(full))
	return hex.EncodeToString(h[:])
}

// isKeyString returns true if the string looks like an API key.
func isKeyString(s string) bool {
	return strings.HasPrefix(s, KeyPrefix)
}

// ulid32 generates a fresh ULID for nonces / IDs.
func ulid32() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
}

// avoid unused imports in helpers — binary.LittleEndian is used by
// base32 length math elsewhere.
var _ = binary.LittleEndian
