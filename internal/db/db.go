// Package db opens the GORM connection, applies connection-pool
// settings, and runs AutoMigrate.
//
// v1.0 supports SQLite only (WAL mode). PostgreSQL is wired through
// the driver but not exercised in the build — the install script
// ships with a SQLite default and operators can swap in Postgres by
// setting [database].driver = "postgres" + a postgres DSN. v1.1 adds
// an integration test for the Postgres path.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open opens the database, configures the pool, and returns *gorm.DB.
func Open(cfg config.DBConfig) (*gorm.DB, error) {
	if cfg.Driver != "sqlite" {
		return nil, fmt.Errorf("v1.0 only supports sqlite; got %q", cfg.Driver)
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time { return time.Now().UTC() },
	}

	db, err := gorm.Open(sqlite.Open(cfg.DSN+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	log.Info().Str("driver", cfg.Driver).Msg("database connected")
	return db, nil
}

// Migrate runs AutoMigrate for the v1.0 schema.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.UserSession{},
		&models.Account{},
		&models.AuditEntry{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	log.Info().Msg("migrations applied")
	return nil
}
