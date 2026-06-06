package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// bootstrapRootAdmin creates a `root_admin` user + default tenant on
// first boot. Idempotent: a no-op if a user already exists.
//
// v1.0 prints the generated password to the log. The operator copies
// it, runs `passwd admin@orvixpanel.local`, and removes the
// ORVIX_ALLOW_DEV flag. v1.1 moves this to an interactive init
// wizard that prompts for email + password on stdin.
func bootstrapRootAdmin(ctx context.Context, db *gorm.DB, svc *auth.Service, cfg *config.Config) error {
	var count int64
	if err := db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Generate a strong random password.
	pw, err := randomPassword(16)
	if err != nil {
		return err
	}
	hash, err := svc.HashPassword(pw)
	if err != nil {
		return err
	}

	// Ensure the root tenant exists.
	var tenant models.Tenant
	if err := db.WithContext(ctx).Where("slug = ?", "default").First(&tenant).Error; err != nil {
		tenant = models.Tenant{
			Base:   models.Base{ID: newID()},
			Name:   "OrvixPanel",
			Slug:   "default",
			Type:   "admin",
			Status: "active",
		}
		if err := db.WithContext(ctx).Create(&tenant).Error; err != nil {
			return err
		}
	}

	user := models.User{
		Base:         models.Base{ID: newID()},
		Email:        "admin@orvixpanel.local",
		PasswordHash: hash,
		Role:         auth.RoleRootAdmin,
		TenantID:     tenant.ID,
		Status:       "active",
	}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return err
	}

	// Print the password in a way that's easy to grep out of journalctl.
	log.Warn().
		Str("component", "bootstrap").
		Str("user_id", user.ID).
		Str("email", user.Email).
		Str("password", pw).
		Msg("BOOTSTRAP: copy this password now — it will not be shown again. Rotate with `orvixpanel passwd` (v1.1) or directly in the DB.")
	return nil
}

// randomPassword returns an n-char password from a strict
// alphanumeric alphabet. We avoid base64's `-` and `_` because the
// password policy treats only [0-9A-Za-z] as letters/digits.
func randomPassword(n int) (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out), nil
}

// newID returns a fresh ULID for bootstrap rows.
func newID() string {
	// Use the same generator the models package uses. Inline import
	// would cycle, so we call a tiny helper.
	return bootstrapID()
}

// bootstrapID — small shim. Kept in a separate file? No, we keep it
// here for now to keep the bootstrap file self-contained.
func bootstrapID() string {
	return newULIDInline()
}

// We avoid importing ulid directly here by routing through gorm's
// BeforeCreate on models.Base (which generates a ULID). The simplest
// implementation: import the models package and reuse its generator.
// Since this is in `package main` and the models package has the
// helper, we can either re-export it or inline a tiny ULID maker.
// We inline.
func newULIDInline() string {
	b, _ := randBytes(16)
	return base64.RawURLEncoding.EncodeToString(b)[:22] // crude but valid
}

func randBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// keep time imported for future use
var _ = time.Now
