// Package config loads OrvixPanel configuration from TOML + env vars.
//
// In v1.0 the schema is intentionally small: server, license, database,
// redis, auth. Other config sections ([ssl], [mail], [waf], [firewall],
// [guardian], [backup], [notifications], [audit]) live in the example
// file for future phases but the binary does not read them.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration object.
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	License LicenseConfig `mapstructure:"license"`
	DB      DBConfig      `mapstructure:"database"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Auth    AuthConfig    `mapstructure:"auth"`
}

type ServerConfig struct {
	BindAddr    string        `mapstructure:"bind_addr"`
	ExternalURL string        `mapstructure:"external_url"`
	SecretKey   string        `mapstructure:"secret_key"`
	Debug       bool          `mapstructure:"debug"`
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type LicenseConfig struct {
	Key              string `mapstructure:"key"`
	OfflineGraceDays int    `mapstructure:"offline_grace_days"`
}

type DBConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AuthConfig struct {
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	MaxFailedLogins int           `mapstructure:"max_failed_logins"`
	LockoutDuration time.Duration `mapstructure:"lockout_duration"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
}

// Load reads configuration from TOML + environment.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults — keep a fresh install functional.
	v.SetDefault("server.bind_addr", "0.0.0.0:8443")
	v.SetDefault("server.external_url", "https://panel.localhost:8443")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.debug", false)

	v.SetDefault("license.offline_grace_days", 7)
	v.SetDefault("license.key", "")

	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "/tmp/orvixpanel-test.db")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)

	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "720h")
	v.SetDefault("auth.max_failed_logins", 5)
	v.SetDefault("auth.lockout_duration", "15m")
	v.SetDefault("auth.bcrypt_cost", 12)

	// File lookup.
	if p := os.Getenv("ORVIX_CONFIG"); p != "" {
		v.SetConfigFile(p)
	} else {
		v.SetConfigName("orvixpanel")
		v.SetConfigType("toml")
		v.AddConfigPath("/etc/orvixpanel")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("ORVIX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// No config file is fine — defaults + env will be used.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.applyDevDefaults(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// applyDevDefaults fills in dev-mode placeholders so a fresh checkout
// can boot without any setup. The ORVIX_ALLOW_DEV=1 opt-in is mandatory
// so a production install never accidentally ships with auto-generated
// secrets.
func (c *Config) applyDevDefaults() error {
	if os.Getenv("ORVIX_ALLOW_DEV") == "1" {
		if c.Server.SecretKey == "" {
			c.Server.SecretKey = randomSecret(64)
		}
		if c.License.Key == "" {
			c.License.Key = "ORVIX-SMB-2025-DEV-DEVPLACE"
		}
		return nil
	}
	if c.Server.SecretKey == "" {
		return fmt.Errorf("server.secret_key is required (or set ORVIX_ALLOW_DEV=1)")
	}
	if c.License.Key == "" {
		return fmt.Errorf("license.key is required (or set ORVIX_ALLOW_DEV=1)")
	}
	return nil
}

func randomSecret(n int) string {
	b := make([]byte, n)
	if _, err := randRead(b); err != nil {
		return "INSECURE-DEV-SECRET"
	}
	return base64Encode(b)
}
