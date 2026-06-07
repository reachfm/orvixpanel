// Package vault implements the per-tenant encrypted secrets store.
//
// v0.3.0 Enterprise Edition.
//
// Each Secret row stores: name (per-tenant unique), ciphertext
// (base64(nonce || aesgcm(plaintext))), and metadata. The plaintext
// is never persisted, never logged, and never returned by list/get —
// the API only returns the metadata + version + created_at. To read
// the actual secret value, the caller invokes Read which returns
// the plaintext in memory only.
//
// Crypto: AES-256-GCM, master key from license.MasterKeySource().
// New nonce per write — never reused.
package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// Errors.
var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrSecretExists   = errors.New("secret already exists")
	ErrSecretTooLarge = errors.New("secret too large (max 64KB)")
	ErrEmptyName      = errors.New("secret name cannot be empty")
)

const maxSecretBytes = 64 * 1024

// Vault is the entry point.
type Vault struct {
	db  *gorm.DB
	gcm cipher.AEAD
}

// New constructs a Vault.
func New(db *gorm.DB, masterKey []byte) (*Vault, error) {
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
	return &Vault{db: db, gcm: gcm}, nil
}

// -----------------------------------------------------------------------------
// CRUD
// -----------------------------------------------------------------------------

// SecretMeta is the metadata returned by list/get (no ciphertext).
type SecretMeta struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Name      string     `json:"name"`
	Version   int        `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy string     `json:"created_by_id"`
	RotatedAt *time.Time `json:"rotated_at,omitempty"`
	RotatedBy *string    `json:"rotated_by_id,omitempty"`
}

// Put stores (or rotates) a secret's plaintext value. The plaintext
// is encrypted before hitting the DB. The audit is recorded by the
// caller; this function just touches the vault.
func (v *Vault) Put(ctx context.Context, tenantID, name, plaintext, actorID string) (*SecretMeta, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if len(plaintext) > maxSecretBytes {
		return nil, ErrSecretTooLarge
	}
	ct, err := v.encrypt([]byte(plaintext))
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	now := time.Now().UTC()
	var existing models.Secret
	err = v.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		First(&existing).Error
	if err == nil {
		// rotate
		existing.Ciphertext = ct
		existing.Version++
		existing.RotatedAt = &now
		existing.RotatedByID = &actorID
		if err := v.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("update secret: %w", err)
		}
		return metaOf(&existing), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup secret: %w", err)
	}

	row := models.Secret{
		TenantID:    tenantID,
		Name:        name,
		Ciphertext:  ct,
		Version:     1,
		CreatedByID: actorID,
	}
	// GORM BeforeCreate hook fills the ID.
	if err := v.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}
	return metaOf(&row), nil
}

// Get returns the metadata for a single secret. Plaintext is NOT
// returned. Use Read to retrieve the decrypted value.
func (v *Vault) Get(ctx context.Context, tenantID, name string) (*SecretMeta, error) {
	var row models.Secret
	if err := v.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("lookup secret: %w", err)
	}
	return metaOf(&row), nil
}

// Read returns the decrypted plaintext value. The audit hook lives
// in the handler so we can attribute the access.
func (v *Vault) Read(ctx context.Context, tenantID, name string) (string, *SecretMeta, error) {
	var row models.Secret
	if err := v.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrSecretNotFound
		}
		return "", nil, fmt.Errorf("lookup secret: %w", err)
	}
	plain, err := v.decrypt(row.Ciphertext)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), metaOf(&row), nil
}

// List returns metadata for every secret in the tenant.
func (v *Vault) List(ctx context.Context, tenantID string) ([]SecretMeta, error) {
	var rows []models.Secret
	if err := v.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	out := make([]SecretMeta, 0, len(rows))
	for i := range rows {
		out = append(out, *metaOf(&rows[i]))
	}
	return out, nil
}

// Delete removes a secret. Returns ErrSecretNotFound when the name
// doesn't exist (so the caller can audit the miss).
func (v *Vault) Delete(ctx context.Context, tenantID, name string) error {
	res := v.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		Delete(&models.Secret{})
	if res.Error != nil {
		return fmt.Errorf("delete secret: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrSecretNotFound
	}
	return nil
}

// Count returns the number of secrets in a tenant. Used by the
// quota service to enforce MaxSecrets.
func (v *Vault) Count(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	if err := v.db.WithContext(ctx).
		Model(&models.Secret{}).
		Where("tenant_id = ?", tenantID).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// -----------------------------------------------------------------------------
// crypto helpers
// -----------------------------------------------------------------------------

func (v *Vault) encrypt(plain []byte) (string, error) {
	nonce := make([]byte, v.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := v.gcm.Seal(nil, nonce, plain, nil)
	blob := append(nonce, sealed...)
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func (v *Vault) decrypt(blob string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	ns := v.gcm.NonceSize()
	if len(raw) < ns {
		return nil, errors.New("cipher blob too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	return v.gcm.Open(nil, nonce, ct, nil)
}

// -----------------------------------------------------------------------------
// meta helper
// -----------------------------------------------------------------------------

func metaOf(row *models.Secret) *SecretMeta {
	return &SecretMeta{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		Version:   row.Version,
		CreatedAt: row.CreatedAt,
		CreatedBy: row.CreatedByID,
		RotatedAt: row.RotatedAt,
		RotatedBy: row.RotatedByID,
	}
}

// -----------------------------------------------------------------------------
// audit helpers
// -----------------------------------------------------------------------------

// RecordAccess writes a vault access audit event. Called by handlers.
func RecordAccess(ctx context.Context, a *audit.Auditor, tenantID, userID, email, role, action, name string, success bool) {
	if a == nil {
		return
	}
	result := "success"
	if !success {
		result = "denied"
	}
	_ = a.Record(ctx, audit.Event{
		Action:       "vault." + action,
		ResourceType: "secret",
		ResourceName: name,
		Result:       result,
		Detail:       "tenant=" + tenantID,
		UserID:       userID,
		UserEmail:    email,
		UserRole:     role,
	})
}
