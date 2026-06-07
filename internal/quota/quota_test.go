package quota

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Account{},
		&models.APIKey{},
		&models.CustomRole{},
		&models.Secret{},
		&models.TenantQuota{},
	))
	return db
}

func TestGetAutoCreates(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	q, err := s.Get(ctx, "t1", "enterprise")
	require.NoError(t, err)
	require.Equal(t, "t1", q.TenantID)
	require.Equal(t, 5000, q.MaxAccounts)
	require.Equal(t, 1000, q.MaxSecrets)

	// Second call should return the same row.
	q2, err := s.Get(ctx, "t1", "enterprise")
	require.NoError(t, err)
	require.Equal(t, q.ID, q2.ID)
}

func TestPutOverridesDefaults(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	_, err := s.Get(ctx, "t1", "smb")
	require.NoError(t, err)

	err = s.Put(ctx, &models.TenantQuota{
		TenantID:       "t1",
		MaxAccounts:    10,
		MaxUsers:       20,
		MaxAPIKeys:     5,
		MaxCustomRoles: 3,
		MaxSecrets:     7,
		MaxDomains:     8,
		MaxStorageMB:   1000,
		MaxBandwidthGB: 100,
	})
	require.NoError(t, err)

	q, err := s.Get(ctx, "t1", "smb")
	require.NoError(t, err)
	require.Equal(t, 10, q.MaxAccounts)
	require.Equal(t, 5, q.MaxAPIKeys)
	require.Equal(t, 7, q.MaxSecrets)
}

func TestCheckEnforcesLimits(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	err := s.Put(ctx, &models.TenantQuota{
		TenantID:    "t1",
		MaxAccounts: 0, // never any accounts
		MaxUsers:    2,
		MaxSecrets:  1,
		MaxAPIKeys:  1,
	})
	require.NoError(t, err)

	q, err := s.Get(ctx, "t1", "smb")
	require.NoError(t, err)
	require.Equal(t, 0, q.MaxAccounts)
	require.Equal(t, 2, q.MaxUsers)
	require.Equal(t, 1, q.MaxSecrets)

	// No accounts yet — 0 >= 0 so the check fails.
	ok, code, err := s.Check(ctx, "t1", ResourceAccount)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, "quota_accounts_exceeded", code)

	// Users: create 2, then check.
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&models.User{
			Email:        fmt.Sprintf("u%d@x", i),
			PasswordHash: "x", Role: "account_owner", TenantID: "t1",
		}).Error)
	}
	ok, code, err = s.Check(ctx, "t1", ResourceUser)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, "quota_users_exceeded", code)

	// Secrets: create 1, then check.
	require.NoError(t, db.Create(&models.Secret{
		TenantID: "t1", Name: "x", Ciphertext: "abc", CreatedByID: "u1",
	}).Error)
	ok, code, err = s.Check(ctx, "t1", ResourceSecret)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, "quota_secrets_exceeded", code)
}
