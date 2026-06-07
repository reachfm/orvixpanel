// Package license — encrypted license persistence (v0.3.0).
//
// The license JSON is encrypted with AES-256-GCM before it hits the
// database. The master key is sourced from ORVIX_MASTER_KEY env var
// (32 bytes hex) or, in dev mode, derived from server.secret_key.
//
// Cipher format: v1:<base64(nonce(12) || ciphertext || tag(16))>
// Nonce is freshly generated for every write — no reuse.
package license

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

const (
	licenseStoreKeyID = "singleton"
	cipherVersion     = "v1"
)

// MasterKeySource returns the 32-byte master key. The order is:
//   1. ORVIX_MASTER_KEY env (hex, 64 chars)
//   2. In dev mode (ORVIX_ALLOW_DEV=1): derived from
//      ORVIX_SECRET_KEY / server.secret_key (HKDF-like: just SHA-256
//      of the secret — fine for v0.3.0; KMS lands in v0.4.0)
func MasterKeySource() ([]byte, error) {
	if h := os.Getenv("ORVIX_MASTER_KEY"); h != "" {
		raw, err := base64.RawURLEncoding.DecodeString(h)
		if err != nil {
			return nil, fmt.Errorf("ORVIX_MASTER_KEY must be base64url: %w", err)
		}
		if len(raw) != 32 {
			return nil, fmt.Errorf("ORVIX_MASTER_KEY must decode to 32 bytes, got %d", len(raw))
		}
		return raw, nil
	}
	if os.Getenv("ORVIX_ALLOW_DEV") == "1" {
		sk := os.Getenv("ORVIX_SECRET_KEY")
		if sk == "" {
			return nil, errors.New("dev mode needs ORVIX_SECRET_KEY or ORVIX_MASTER_KEY")
		}
		// Derive a 32-byte key from the secret. SHA-256 is fine here
		// because the secret itself is high-entropy in dev. v0.4.0
		// will swap in HKDF-SHA-256.
		h := sha256Sum([]byte("orvix-master-key-v1:" + sk))
		return h, nil
	}
	return nil, errors.New("ORVIX_MASTER_KEY not set (or set ORVIX_ALLOW_DEV=1)")
}

// Store persists the encrypted license blob.
type Store struct {
	db  *gorm.DB
	gcm cipher.AEAD
}

// NewStore opens the license store. The master key is required.
func NewStore(db *gorm.DB, masterKey []byte) (*Store, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes, got %d", len(masterKey))
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}
	return &Store{db: db, gcm: gcm}, nil
}

// Save encrypts + persists the license. The plaintext is JSON
// (License struct). On success the parsed fields are stored
// alongside the ciphertext for easy admin querying.
func (s *Store) Save(ctx context.Context, lic *License, uploadedBy string) error {
	if lic == nil {
		return errors.New("license is nil")
	}
	payload, err := json.Marshal(lic)
	if err != nil {
		return fmt.Errorf("marshal license: %w", err)
	}
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("nonce: %w", err)
	}
	sealed := s.gcm.Seal(nil, nonce, payload, nil)
	blob := append(nonce, sealed...)
	cipherB64 := cipherVersion + ":" + base64.RawURLEncoding.EncodeToString(blob)

	row := models.LicenseStore{
		KeyID:           licenseStoreKeyID,
		Ciphertext:      cipherB64,
		ParsedTier:      lic.Tier,
		ParsedExpiresAt: lic.ExpiresAt,
		ParsedIssuedAt:  lic.IssuedAt,
		UploadedByID:    uploadedBy,
	}
	// Upsert: delete any existing row, then insert.
	if err := s.db.WithContext(ctx).Where("key_id = ?", licenseStoreKeyID).Delete(&models.LicenseStore{}).Error; err != nil {
		return fmt.Errorf("delete old license row: %w", err)
	}
	row.ID = newLicenseID()
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("insert license row: %w", err)
	}
	return nil
}

// Load decrypts + returns the current license. Returns
// ErrInvalidKey when no row exists.
func (s *Store) Load(ctx context.Context) (*License, error) {
	var row models.LicenseStore
	if err := s.db.WithContext(ctx).Where("key_id = ?", licenseStoreKeyID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidKey
		}
		return nil, fmt.Errorf("load license: %w", err)
	}
	return s.decrypt(row.Ciphertext)
}

// RenewalInfo returns the human-readable status used by the
// /admin/license/renewal-info endpoint.
func (s *Store) RenewalInfo(ctx context.Context) (map[string]any, error) {
	lic, err := s.Load(ctx)
	if err != nil {
		if errors.Is(err, ErrInvalidKey) {
			return map[string]any{
				"loaded": false,
				"mode":   string(ModeLocked),
			}, nil
		}
		return nil, err
	}
	now := timeNow()
	mode := lic.ModeAt(now)
	return map[string]any{
		"loaded":              true,
		"tier":                lic.Tier,
		"licensed_to":         lic.LicensedTo,
		"issued_at":           lic.IssuedAtTime().Format(timeRFC3339),
		"expires_at":          lic.ExpiresAtTime().Format(timeRFC3339),
		"grace_ends_at":       lic.GraceEndsAt().Format(timeRFC3339),
		"days_remaining":      lic.DaysRemaining(now),
		"days_until_locked":   lic.DaysUntilLocked(now),
		"mode":                string(mode),
		"max_servers":         lic.MaxServers,
		"feature_count":       len(lic.Features),
	}, nil
}

func (s *Store) decrypt(blob string) (*License, error) {
	if len(blob) < 3 || blob[:3] != cipherVersion+":" {
		return nil, fmt.Errorf("unsupported cipher version: %q", blob)
	}
	raw, err := base64.RawURLEncoding.DecodeString(blob[3:])
	if err != nil {
		return nil, fmt.Errorf("decode cipher blob: %w", err)
	}
	ns := s.gcm.NonceSize()
	if len(raw) < ns {
		return nil, errors.New("cipher blob too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := s.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}
	var lic License
	if err := json.Unmarshal(plain, &lic); err != nil {
		return nil, fmt.Errorf("unmarshal license: %w", err)
	}
	return &lic, nil
}
