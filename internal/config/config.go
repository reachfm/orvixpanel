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
	Server      ServerConfig  `mapstructure:"server"`
	License     LicenseConfig `mapstructure:"license"`
	DB          DBConfig      `mapstructure:"database"`
	Redis       RedisConfig   `mapstructure:"redis"`
	Auth        AuthConfig    `mapstructure:"auth"`
	DNS         DNSConfig     `mapstructure:"dns"`
	SSL         SSLConfig     `mapstructure:"ssl"`
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

// DNSConfig holds DNS mode and PowerDNS integration settings.
type DNSConfig struct {
	Mode           string `mapstructure:"mode"`            // "local" or "powerdns"
	PowerDNSURL    string `mapstructure:"powerdns_url"`    // e.g., "http://127.0.0.1:8081"
	PowerDNSAPIKey string `mapstructure:"powerdns_api_key"` // API key for PowerDNS
	PowerDNSServer string `mapstructure:"powerdns_server"` // e.g., "localhost"
}

// SSLConfig holds SSL/ACME configuration settings.
type SSLConfig struct {
	ChallengeDir string `mapstructure:"challenge_dir"` // Directory for ACME HTTP-01 challenges
}

// Load reads configuration from TOML + environment.
//
// v0.2.1 also reads /etc/orvixpanel/orvixpanel.env (the systemd
// EnvironmentFile shipped by the installer) and exports every
// KEY=VALUE line into os.Environ so viper picks them up.
func Load() (*Config, error) {
	v := viper.New()

	// Optional .env-style file. Lines like
	//   ORVIX_SERVER_SECRET_KEY=foo
	//   # comment
	// become os.Setenv before viper.AutomaticEnv runs.
	envFile := os.Getenv("ORVIX_ENV_FILE")
	if envFile == "" {
		envFile = "/etc/orvixpanel/orvixpanel.env"
	}
	if data, err := os.ReadFile(envFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq <= 0 {
				continue
			}
			k := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, val)
			}
		}
	}

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

	// DNS defaults
	v.SetDefault("dns.mode", "local")
	v.SetDefault("dns.powerdns_server", "localhost")

	// SSL defaults
	v.SetDefault("ssl.challenge_dir", "/var/lib/orvixpanel/acme-challenges")

	// File lookup. The production config file is orvixpanel.toml
	// in /etc/orvixpanel (or ./configs, or .). We DO NOT search
	// for the bare name "orvixpanel" because v0.2.1 ships an
	// orvixpanel.env (a systemd EnvironmentFile) in the same dir
	// and viper would try to parse it as TOML.
	if p := os.Getenv("ORVIX_CONFIG"); p != "" {
		v.SetConfigFile(p)
	} else {
		v.SetConfigName("orvixpanel.toml")
		v.SetConfigType("toml")
		v.AddConfigPath("/etc/orvixpanel")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("ORVIX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// Explicit BindEnv for every key that has both a dot AND an
	// underscore in its struct path (viper's auto-env replace
	// mangles those, so BindEnv is required for unambiguous lookup).
	v.BindEnv("server.bind_addr", "ORVIX_SERVER_BIND_ADDR")
	v.BindEnv("server.external_url", "ORVIX_SERVER_EXTERNAL_URL")
	v.BindEnv("server.secret_key", "ORVIX_SERVER_SECRET_KEY")
	v.BindEnv("license.key", "ORVIX_LICENSE_KEY")
	v.BindEnv("database.dsn", "ORVIX_DATABASE_DSN")
	v.BindEnv("redis.addr", "ORVIX_REDIS_ADDR")
	v.BindEnv("redis.password", "ORVIX_REDIS_PASSWORD")
	v.BindEnv("auth.access_token_ttl", "ORVIX_AUTH_ACCESS_TOKEN_TTL")
	v.BindEnv("auth.refresh_token_ttl", "ORVIX_AUTH_REFRESH_TOKEN_TTL")

	// DNS bindings
	v.BindEnv("dns.mode", "ORVIX_DNS_MODE")
	v.BindEnv("dns.powerdns_url", "ORVIX_POWERDNS_URL")
	v.BindEnv("dns.powerdns_api_key", "ORVIX_POWERDNS_API_KEY")
	v.BindEnv("dns.powerdns_server", "ORVIX_POWERDNS_SERVER_ID")

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
