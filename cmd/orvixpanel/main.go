// Command orvixpanel is the v0.3.0 entry point.
//
// v0.3.0 Enterprise Edition:
//   - Phase 1 Foundation: auth + license + audit + tenant + account
//   - + Encrypted secrets vault (AES-256-GCM)
//   - + Custom RBAC roles
//   - + API key auth (long-lived automation credentials)
//   - + Audit log search + CEF export
//   - + Tenant-level quotas
//   - + Encrypted license persistence + read-only mode on expiry
//
// See ENTERPRISE_PLAN.md and RELEASE_NOTES.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/api"
	"github.com/orvixpanel/orvixpanel/internal/audit"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/db"
	"github.com/orvixpanel/orvixpanel/internal/hosting"
	"github.com/orvixpanel/orvixpanel/internal/license"
	"github.com/orvixpanel/orvixpanel/internal/quota"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
	"github.com/orvixpanel/orvixpanel/internal/vault"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Pretty console for human operators; structured JSON once we know
	// the desired level.
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("orvixpanel exited with error")
	}
}

func run() error {
	// 0. Ensure /run/orvixpanel exists. /run is a tmpfs and is
	// cleared on reboot / WSL reinit / systemd-managed service
	// restarts; install.sh's one-time mkdir is not durable. The
	// binary is the source of truth for runtime state, so we
	// self-heal on every start. Idempotent, best-effort: a failure
	// here only means the doctor check will WARN — runtime itself
	// is fine because no live code currently writes to this path
	// (intended for future pidfile / Unix socket use).
	if err := ensureRuntimeDir("/run/orvixpanel", 0o755, "orvixpanel"); err != nil {
		log.Warn().Err(err).Msg("ensure /run/orvixpanel failed; continuing")
	}

	// 1. Configuration.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Server.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// 2. License. v1.0 accepts the key if ORVIX_ALLOW_DEV=1 or the
	// key matches the ORVIX-{TIER}-{YEAR}-{HASH}-{SIG} shape.
	lic, err := license.Parse(cfg.License.Key)
	if err != nil {
		// In dev mode, keep going with a default SMB license. In
		// production we fail closed.
		if os.Getenv("ORVIX_ALLOW_DEV") != "1" {
			return fmt.Errorf("license: %w", err)
		}
		// Dev fallback uses the same expiry override as the
		// defaultPayload function. We re-implement it here because
		// we're not going through license.Parse.
		expiresAt := int64(1735689600) // 2025-01-01 UTC sentinel
		issuedAt := int64(1704067200)  // 2024-01-01 UTC
		graceDays := 7
		if v := os.Getenv("ORVIX_DEV_LICENSE_EXPIRES_AT"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				expiresAt = t.UTC().Unix()
			} else if t, err := time.Parse("2006-01-02", v); err == nil {
				expiresAt = t.UTC().Unix()
			}
		}
		lic = &license.License{
			Tier:       license.TierSMB,
			MaxServers: 1,
			ExpiresAt:  expiresAt,
			IssuedAt:   issuedAt,
			GraceDays:  graceDays,
			Features:   license.TierFeatures[license.TierSMB],
		}
		log.Warn().Time("expires_at", time.Unix(expiresAt, 0)).Msg("using dev fallback license (SMB)")
	}
	license.SetGlobal(lic)
	log.Info().Str("tier", lic.Tier).Int("features", len(lic.Features)).Msg("license loaded")

	// 3. Database.
	database, err := db.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// 4. Audit chain bootstrap.
	auditor, err := audit.New(context.Background(), database)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}

	// 5. Auth service.
	authSvc, err := auth.New(database, cfg)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	// 5b. Bootstrap root admin on first boot (v1.0). Logs the generated
	// password so the operator can copy it from journalctl. v1.1 moves
	// this to an interactive `orvixpanel init` wizard.
	if err := bootstrapRootAdmin(context.Background(), database, authSvc, cfg); err != nil {
		log.Warn().Err(err).Msg("bootstrap admin: skipped or failed")
	}

	// 5c. v0.3.0 services.
	masterKey, err := license.MasterKeySource()
	if err != nil {
		log.Warn().Err(err).Msg("master key not configured; license store + vault will fail until ORVIX_MASTER_KEY is set")
	}
	var licenseStore *license.Store
	var vaultSvc *vault.Vault
	if masterKey != nil {
		licenseStore, err = license.NewStore(database, masterKey)
		if err != nil {
			return fmt.Errorf("license store: %w", err)
		}
		vaultSvc, err = vault.New(database, masterKey)
		if err != nil {
			return fmt.Errorf("vault: %w", err)
		}
		// Try to load a persisted license and apply it.
		if persisted, perr := licenseStore.Load(context.Background()); perr == nil {
			license.SetGlobal(persisted)
			log.Info().Str("tier", persisted.Tier).Msg("loaded persisted license")
		} else {
			log.Info().Msg("no persisted license; using in-memory dev key")
		}
	}

	rbacSvc := rbac.New(database)
	quotaSvc := quota.New(database)
	apiKeySvc := auth.NewKeyService(database)

	// 5d. v0.2.0 Core Hosting Engine service. On non-Linux this
	// returns a stub that 501s every hosting call; on Linux it
	// does real useradd / nginx / php-fpm work.
	hostingSvc := hosting.NewService()
	if err := hostingSvc.Paths.EnsureDirs(); err != nil {
		// EnsureDirs can fail on a fresh install (perms on /var).
		// Log and continue — the first ProvisionAccount call will
		// surface the real error.
		log.Warn().Err(err).Msg("hosting ensure-dirs: some paths will fail until /var/lib/orvixpanel is writable")
	}

	// 6. HTTP server.
	server := api.NewServer(api.Deps{
		Config:       cfg,
		DB:           database,
		Auth:         authSvc,
		Audit:        auditor,
		LicenseStore: licenseStore,
		RBAC:         rbacSvc,
		Vault:        vaultSvc,
		Quota:        quotaSvc,
		APIKeys:      apiKeySvc,
		Hosting:      hostingSvc,
	})

	httpErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", cfg.Server.BindAddr).Bool("debug", cfg.Server.Debug).Msg("listening")
		if err := server.Listen(cfg.Server.BindAddr); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			httpErr <- err
		}
		close(httpErr)
	}()

	// 7. Signal handling.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-httpErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("http shutdown error")
	}
	log.Info().Msg("orvixpanel stopped cleanly")
	return nil
}
