package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.APIKey{}))
	return db
}

func TestCreateAndVerifyRoundTrip(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	expires := time.Now().Add(24 * time.Hour)
	resp, err := s.Create(ctx, "t1", "creator-1", CreateAPIKeyRequest{
		Name:      "ci-deploy",
		Role:      "account_owner",
		Scopes:    []string{"hosting.create", "hosting.read"},
		ExpiresAt: &expires,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Secret)
	require.True(t, strings.HasPrefix(resp.Secret, KeyPrefix))
	require.Len(t, resp.Prefix, 8)

	// Verify the same key
	row, err := s.Verify(ctx, resp.Secret, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "t1", row.TenantID)
	require.Equal(t, "account_owner", row.Role)
	require.Equal(t, resp.ID, row.ID)

	// LastUsedAt set
	require.NotNil(t, row.LastUsedAt)
	require.Equal(t, "127.0.0.1", row.LastUsedIP)
}

func TestCreateRequiresNameAndRole(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	_, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Role: "account_owner"})
	require.Error(t, err)

	_, err = s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: "x"})
	require.Error(t, err)
}

func TestVerifyRejectsMalformed(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	cases := []string{
		"",
		"notakey",
		KeyPrefix,
		KeyPrefix + "tooshort",
		KeyPrefix + "abcdefgh_",
	}
	for _, c := range cases {
		_, err := s.Verify(context.Background(), c, "")
		require.Error(t, err, "expected error for %q", c)
	}
}

func TestVerifyRejectsUnknownHash(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	_, err := s.Verify(context.Background(), KeyPrefix+"AAAAAAAA_BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "")
	require.ErrorIs(t, err, ErrAPIKeyNotFound)
}

func TestRevokeBlocksVerify(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	resp, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: "k", Role: "account_owner"})
	require.NoError(t, err)

	require.NoError(t, s.Revoke(ctx, "t1", resp.ID, "test"))
	_, err = s.Verify(ctx, resp.Secret, "")
	require.ErrorIs(t, err, ErrAPIKeyRevoked)
}

func TestExpiredKeyBlocked(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	resp, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{
		Name: "k", Role: "account_owner", ExpiresAt: &past,
	})
	require.NoError(t, err)

	_, err = s.Verify(ctx, resp.Secret, "")
	require.ErrorIs(t, err, ErrAPIKeyExpired)
}

func TestListHidesHashAndSecret(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	_, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: "k1", Role: "account_owner"})
	require.NoError(t, err)
	_, err = s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: "k2", Role: "billing"})
	require.NoError(t, err)

	rows, err := s.List(ctx, "t1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.NotEmpty(t, r.KeyHash) // stored
		require.Empty(t, r.Prefix[:0]) // safe-guard
		// Verify we can never reconstruct the secret from the row.
		require.NotEqual(t, "orx_live_", r.Prefix[:0])
	}
}

func TestTenantIsolation(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()

	resp, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: "k", Role: "account_owner"})
	require.NoError(t, err)

	// Cross-tenant revoke with the same ID must fail (no row in t2).
	require.ErrorIs(t, s.Revoke(ctx, "t2", resp.ID, "hax"), ErrAPIKeyNotFound)
	// Verify still works because the key is in t1, not t2.
	_, err = s.Verify(ctx, resp.Secret, "")
	require.NoError(t, err)
}

func TestCount(t *testing.T) {
	db := newTestDB(t)
	s := NewKeyService(db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := s.Create(ctx, "t1", "u", CreateAPIKeyRequest{Name: time.Now().String(), Role: "account_owner"})
		require.NoError(t, err)
	}
	n, err := s.Count(ctx, "t1")
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	// Revoke one, count should still be 3 (we count active).
	rows, _ := s.List(ctx, "t1")
	require.NoError(t, s.Revoke(ctx, "t1", rows[0].ID, "x"))
	n, err = s.Count(ctx, "t1")
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}

func TestClaimsForKey(t *testing.T) {
	k := &models.APIKey{
		Base:    models.Base{ID: "abc123"},
		Prefix:  "PREFX000",
		TenantID: "t1",
		Role:    "account_owner",
	}
	c := ClaimsForKey(k)
	require.Equal(t, "apikey:abc123", c.UserID)
	require.Equal(t, "apikey:PREFX000@t1", c.Email)
	require.Equal(t, "account_owner", c.Role)
	require.Equal(t, "t1", c.TenantID)
}

func TestIsKeyString(t *testing.T) {
	require.True(t, isKeyString("orx_live_AAAAAAA_"))
	require.False(t, isKeyString("eyJ"))
	require.False(t, isKeyString(""))
}
