// Command orvixpanel is the v1.0 entry point. v1.0 ships the Phase 1
// (Foundation) scope only: auth + license + audit + minimal
// account/tenant CRUD. See RELEASE_NOTES.md and audit.md for the
// explicit v1.1 backlog.
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
	"github.com/orvixpanel/orvixpanel/internal/license"
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
		lic = &license.License{
			Tier:       license.TierSMB,
			MaxServers: 1,
			Features:   license.TierFeatures[license.TierSMB],
		}
		log.Warn().Msg("using dev fallback license (SMB)")
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

	// 6. HTTP server.
	server := api.NewServer(api.Deps{
		Config: cfg,
		DB:     database,
		Auth:   authSvc,
		Audit:  auditor,
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
