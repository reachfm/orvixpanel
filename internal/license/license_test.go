package license

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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
	require.NoError(t, db.AutoMigrate(&models.LicenseStore{}))
	return db
}

func TestParseRejectsBadShape(t *testing.T) {
	cases := []string{
		"",
		"ORVIX",
		"ORVIX-SMB-2025",
		"ORVIX-SMB-2025-AAAAA",
		"WRONG-SMB-2025-AAAAA-BBBBBB",
		"ORVIX-BOGUS-2025-AAAAA-BBBBBB",
		"ORVIX-SMB-20-AAAAA-BBBBBB",
	}
	for _, c := range cases {
		_, err := ParseWithPublicKey(c, nil, true)
		require.Error(t, err, "expected error for %q", c)
	}
}

func TestParseAcceptsAllTiersInDev(t *testing.T) {
	tiers := []string{TierSMB, TierISP, TierEnterprise, TierWhiteLabel}
	for _, tier := range tiers {
		lic, err := ParseWithPublicKey("ORVIX-"+tier+"-2025-AAAAA-BBBBBB", nil, true)
		require.NoError(t, err)
		require.Equal(t, tier, lic.Tier)
	}
}

func TestParseRejectsBadSignatureInProduction(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, err = ParseWithPublicKey("ORVIX-SMB-2025-AAAAA-BBBBBB", &priv.PublicKey, false)
	require.ErrorIs(t, err, ErrSignatureBad)
}

func TestModeAt(t *testing.T) {
	lic := &License{
		ExpiresAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		GraceDays: 7,
	}
	active := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	grace := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	locked := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	require.Equal(t, ModeActive, lic.ModeAt(active))
	require.Equal(t, ModeGrace, lic.ModeAt(grace))
	require.Equal(t, ModeLocked, lic.ModeAt(locked))
}

func TestDaysRemainingAndUntilLocked(t *testing.T) {
	lic := &License{
		ExpiresAt: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC).Unix(),
		GraceDays: 7,
	}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 9, lic.DaysRemaining(now))
	require.Equal(t, 16, lic.DaysUntilLocked(now))
}

func TestHasFeatureWildcardAndNegation(t *testing.T) {
	lic := &License{
		Features: []string{"hosting.*", "!hosting.delete", "*"},
	}
	require.True(t, lic.HasFeature("hosting.create"))
	require.True(t, lic.HasFeature("vault.read"))
	require.False(t, lic.HasFeature("hosting.delete")) // explicitly negated
}

func TestSetGlobalUpdatesMode(t *testing.T) {
	SetGlobal(&License{
		Tier:       TierEnterprise,
		ExpiresAt:  time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		GraceDays:  14,
		Features:   TierFeatures[TierEnterprise],
	})
	require.Equal(t, ModeActive, CurrentMode())
	require.Equal(t, TierEnterprise, Get().Tier)
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	db := newTestDB(t)
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	store, err := NewStore(db, masterKey)
	require.NoError(t, err)

	lic := &License{
		Tier:       TierEnterprise,
		MaxServers: 999999,
		ExpiresAt:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC).Unix(),
		IssuedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		GraceDays:  14,
		LicensedTo: "Acme Corp",
		Features:   TierFeatures[TierEnterprise],
	}
	require.NoError(t, store.Save(context.Background(), lic, "admin-id"))

	got, err := store.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, lic.Tier, got.Tier)
	require.Equal(t, lic.MaxServers, got.MaxServers)
	require.Equal(t, lic.ExpiresAt, got.ExpiresAt)
	require.Equal(t, lic.GraceDays, got.GraceDays)
	require.Equal(t, lic.LicensedTo, got.LicensedTo)
}

func TestStoreRejectsWrongMasterKey(t *testing.T) {
	db := newTestDB(t)
	k1 := make([]byte, 32)
	k2 := make([]byte, 32)
	for i := range k1 {
		k1[i] = byte(i)
		k2[i] = byte(i + 100)
	}
	store1, err := NewStore(db, k1)
	require.NoError(t, err)
	store2, err := NewStore(db, k2)
	require.NoError(t, err)

	lic := &License{Tier: TierEnterprise, ExpiresAt: 9999999999, GraceDays: 7}
	require.NoError(t, store1.Save(context.Background(), lic, "x"))

	_, err = store2.Load(context.Background())
	require.Error(t, err, "load with wrong key should fail")
}

func TestStoreRejectsTamperedCiphertext(t *testing.T) {
	db := newTestDB(t)
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	store, err := NewStore(db, masterKey)
	require.NoError(t, err)

	lic := &License{Tier: TierEnterprise, ExpiresAt: 9999999999, GraceDays: 7}
	require.NoError(t, store.Save(context.Background(), lic, "x"))

	// Tamper a byte in the middle of the base64 portion of the
	// blob (skipping the "v1:" prefix). Pick a position that's well
	// past the nonce (16 base64 chars) so the flip lands on the
	// AES-GCM ciphertext + tag, not on the nonce.
	var row models.LicenseStore
	require.NoError(t, db.Where("key_id = ?", "singleton").First(&row).Error)
	tampered := []byte(row.Ciphertext)
	require.Greater(t, len(tampered), 30, "blob too short to test")
	pos := len(tampered) / 2 // middle of the base64
	if tampered[pos] == 'A' {
		tampered[pos] = 'B'
	} else {
		tampered[pos] = 'A'
	}
	require.NoError(t, db.Model(&row).Update("ciphertext", string(tampered)).Error)

	_, err = store.Load(context.Background())
	require.Error(t, err, "tampered ciphertext should fail to decrypt")
}

func TestLoadPublicKeyValidatesPEM(t *testing.T) {
	_, err := LoadPublicKey("not pem")
	require.Error(t, err)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	// Wrong format: this is a PEM but not ECDSA
	badPEM := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"
	_, err = LoadPublicKey(badPEM)
	require.Error(t, err)

	// Correct format
	der, err := x509MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pem := pemEncodePublicKey(der)
	_, err = LoadPublicKey(pem)
	require.NoError(t, err)
}

func TestRenewalInfoReportsMode(t *testing.T) {
	db := newTestDB(t)
	masterKey := make([]byte, 32)
	store, err := NewStore(db, masterKey)
	require.NoError(t, err)

	// No license loaded yet
	info, err := store.RenewalInfo(context.Background())
	require.NoError(t, err)
	require.False(t, info["loaded"].(bool))
	require.Equal(t, string(ModeLocked), info["mode"])

	// Active license
	lic := &License{
		Tier:      TierEnterprise,
		ExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		GraceDays: 14,
		Features:  TierFeatures[TierEnterprise],
	}
	require.NoError(t, store.Save(context.Background(), lic, "x"))
	info, err = store.RenewalInfo(context.Background())
	require.NoError(t, err)
	require.True(t, info["loaded"].(bool))
	require.Equal(t, string(ModeActive), info["mode"])
}
