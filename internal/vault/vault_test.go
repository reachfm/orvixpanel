package vault

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestVault(t *testing.T) (*Vault, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Secret{}, &models.TenantQuota{}))
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	v, err := New(db, key)
	require.NoError(t, err)
	return v, db
}

func TestPutGetReadDelete(t *testing.T) {
	v, _ := newTestVault(t)
	ctx := context.Background()

	// Put
	meta, err := v.Put(ctx, "tenant-1", "db_password", "s3cr3t!", "user-1")
	require.NoError(t, err)
	require.Equal(t, "db_password", meta.Name)
	require.Equal(t, 1, meta.Version)

	// Get (no plaintext)
	got, err := v.Get(ctx, "tenant-1", "db_password")
	require.NoError(t, err)
	require.Equal(t, "db_password", got.Name)
	require.Equal(t, 1, got.Version)

	// Read (plaintext)
	plain, _, err := v.Read(ctx, "tenant-1", "db_password")
	require.NoError(t, err)
	require.Equal(t, "s3cr3t!", plain)

	// Rotate
	meta2, err := v.Put(ctx, "tenant-1", "db_password", "new-s3cr3t", "user-1")
	require.NoError(t, err)
	require.Equal(t, 2, meta2.Version)
	require.NotNil(t, meta2.RotatedAt)

	plain2, _, err := v.Read(ctx, "tenant-1", "db_password")
	require.NoError(t, err)
	require.Equal(t, "new-s3cr3t", plain2)

	// List
	list, err := v.List(ctx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Tenant isolation: tenant-2 cannot read tenant-1 secret
	_, _, err = v.Read(ctx, "tenant-2", "db_password")
	require.ErrorIs(t, err, ErrSecretNotFound)

	// Delete
	require.NoError(t, v.Delete(ctx, "tenant-1", "db_password"))
	_, _, err = v.Read(ctx, "tenant-1", "db_password")
	require.ErrorIs(t, err, ErrSecretNotFound)
}

func TestEmptyName(t *testing.T) {
	v, _ := newTestVault(t)
	_, err := v.Put(context.Background(), "t1", "", "value", "u1")
	require.ErrorIs(t, err, ErrEmptyName)
}

func TestTooLarge(t *testing.T) {
	v, _ := newTestVault(t)
	huge := make([]byte, maxSecretBytes+1)
	for i := range huge {
		huge[i] = 'a'
	}
	_, err := v.Put(context.Background(), "t1", "huge", string(huge), "u1")
	require.ErrorIs(t, err, ErrSecretTooLarge)
}

func TestTamperedCiphertextFails(t *testing.T) {
	v, db := newTestVault(t)
	ctx := context.Background()
	_, err := v.Put(ctx, "t1", "name", "secret-value", "u1")
	require.NoError(t, err)

	// Tamper: flip a byte in the ciphertext column.
	var row models.Secret
	require.NoError(t, db.Where("tenant_id = ?", "t1").First(&row).Error)
	tampered := []byte(row.Ciphertext)
	pos := len(tampered) / 2
	if tampered[pos] == 'A' {
		tampered[pos] = 'B'
	} else {
		tampered[pos] = 'A'
	}
	require.NoError(t, db.Model(&row).Update("ciphertext", string(tampered)).Error)

	_, _, err = v.Read(ctx, "t1", "name")
	require.Error(t, err, "tampered ciphertext should fail to decrypt")
}

func TestCount(t *testing.T) {
	v, _ := newTestVault(t)
	ctx := context.Background()
	_, err := v.Put(ctx, "t1", "a", "1", "u1")
	require.NoError(t, err)
	_, err = v.Put(ctx, "t1", "b", "2", "u1")
	require.NoError(t, err)
	n, err := v.Count(ctx, "t1")
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	n, err = v.Count(ctx, "t2")
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

func TestWrongMasterKeyRejects(t *testing.T) {
	v1, db := newTestVault(t)
	ctx := context.Background()
	_, err := v1.Put(ctx, "t1", "name", "value", "u1")
	require.NoError(t, err)

	// Build a second vault with a different key against the same DB.
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 100)
	}
	v2, err := New(db, key2)
	require.NoError(t, err)

	_, _, err = v2.Read(ctx, "t1", "name")
	require.Error(t, err, "wrong key should fail")
}
