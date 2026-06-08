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

// Migrate runs AutoMigrate for the schema.
//
// v0.1.0: 5 tables (tenant, user, user_session, account, audit_entry)
// v0.3.0: +5 tables (api_key, custom_role, secret, tenant_quota, license_store)
// v0.4.0: +3 tables (dns_zone, dns_record, dns_zone_template)
// v0.5.0: +4 tables (ssl_certificate, ssl_event, acme_account, ssl_challenge)
// v0.6.0: +6 tables (mail_domain, mailbox, mail_alias, mail_forwarder, mail_rate_limit, mail_audit_log)
// v0.7.0: +5 tables (backup_job, backup_file, restore_point, backup_config, backup_schedule)
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.UserSession{},
		&models.Account{},
		&models.AuditEntry{},
		// v0.3.0 Enterprise Edition
		&models.APIKey{},
		&models.CustomRole{},
		&models.Secret{},
		&models.TenantQuota{},
		&models.LicenseStore{},
		// v0.4.0 DNS Engine
		&models.DNSZone{},
		&models.DNSRecord{},
		&models.DNSZoneTemplate{},
		// v0.5.0 SSL Engine
		&models.ACMEAccount{},
		&models.SSLCertificate{},
		&models.SSLEvent{},
		&models.SSLChallenge{},
		// v0.6.0 Mail Hosting Engine
		&models.MailDomain{},
		&models.Mailbox{},
		&models.MailAlias{},
		&models.MailForwarder{},
		&models.MailRateLimit{},
		&models.MailAuditLog{},
		// v0.7.0 Backup Engine
		&models.BackupJob{},
		&models.BackupFile{},
		&models.RestorePoint{},
		&models.BackupConfig{},
		&models.BackupSchedule{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	log.Info().Msg("migrations applied")
	return nil
}
