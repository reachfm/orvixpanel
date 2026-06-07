// Package hosting implements the Phase 2 Core Hosting Engine.
//
// OrvixPanel's hosting layer is responsible for:
//   - Linux system user provisioning (useradd, group, home dir, perms)
//   - Per-account PHP-FPM pool generation
//   - Per-domain nginx vhost generation
//   - Per-account disk + inode quota tracking
//   - Document-root creation + atomic symlink-swap deployment
//
// All OS-exec code is split into *_linux.go (`//go:build linux`) and
// *_other.go (`//go:build !linux`) so the package compiles on every
// platform; the non-Linux build returns ErrUnsupported at the call
// sites. Cross-compile for the production target with:
//
//	GOOS=linux go build -o orvixpanel ./cmd/orvixpanel
//
// The service is stateless except for the configuration paths
// (BaseDir, NginxDir, FpmDir, DocumentRoot). All state lives in the
// GORM database (Tenants/Users/Accounts/Domains models in
// internal/db/models).
package hosting

import (
	"fmt"
	"os"
	"path"
)

// ErrUnsupported is defined in errors.go (platform-agnostic).

// Paths bundles the on-disk locations the service writes to.
//
//	BaseDir       — /var/lib/orvixpanel (parent for homes)
//	HomesDir      — BaseDir/homes
//	NginxDir      — /etc/nginx/conf.d/orvix
//	FpmDir        — /etc/php/8.3/fpm/pool.d (or 8.5 — auto-detected)
//	DocumentRoot  — /var/www (parent for /var/www/<account>/<domain>)
//	ReleasesRoot  — BaseDir/releases (atomic deploys)
//	CurrentLink   — BaseDir/current (the symlink deployments point at)
//	LogRoot       — /var/log/orvixpanel
//	NginxTestCmd  — `nginx -t` (for validation)
//	NginxReloadCmd — `systemctl reload nginx`
//	FpmTestCmd    — `php-fpm8.3 -t` (or version detected at boot)
//	FpmReloadCmd  — `systemctl reload php8.3-fpm`
type Paths struct {
	BaseDir        string
	HomesDir       string
	NginxDir       string
	FpmDir         string
	DocumentRoot   string
	ReleasesRoot   string
	CurrentLink    string
	LogRoot        string
	NginxBin       string
	NginxTestCmd   string
	NginxReloadCmd string
	FpmBin         string
	FpmTestCmd     string
	FpmReloadCmd   string
	UseraddBin     string
	UsermodBin     string
	UserdelBin     string
	GroupaddBin    string
	SetquotaBin    string
	DuBin          string
}

// DefaultPaths returns the production paths. PHP version is probed
// at runtime; if detection fails the function still returns a
// usable struct and the smoke test will report the issue clearly.
func DefaultPaths() Paths {
	p := Paths{
		BaseDir:        "/var/lib/orvixpanel",
		HomesDir:       "/var/lib/orvixpanel/homes",
		NginxDir:       "/etc/nginx/conf.d/orvix",
		FpmDir:         "/etc/php/8.5/fpm/pool.d",
		DocumentRoot:   "/var/www",
		ReleasesRoot:   "/var/lib/orvixpanel/releases",
		CurrentLink:    "/var/lib/orvixpanel/current",
		LogRoot:        "/var/log/orvixpanel",
		NginxBin:       "/usr/sbin/nginx",
		NginxTestCmd:   "nginx -t",
		NginxReloadCmd: "systemctl reload nginx",
		FpmBin:         "/usr/sbin/php-fpm8.5",
		FpmTestCmd:     "php-fpm8.5 -t",
		FpmReloadCmd:   "systemctl reload php8.5-fpm",
		UseraddBin:     "/usr/sbin/useradd",
		UsermodBin:     "/usr/sbin/usermod",
		UserdelBin:     "/usr/sbin/userdel",
		GroupaddBin:    "/usr/sbin/groupadd",
		SetquotaBin:    "/usr/sbin/setquota",
		DuBin:          "/usr/bin/du",
	}
	if v := os.Getenv("ORVIX_FPM_VERSION"); v != "" {
		// Operator override for the PHP version we target. We rewrite
		// every Fpm-related path/cmd so the rest of the code is
		// version-agnostic.
		p.FpmDir = "/etc/php/" + v + "/fpm/pool.d"
		p.FpmBin = "/usr/sbin/php-fpm" + v
		p.FpmTestCmd = "php-fpm" + v + " -t"
		p.FpmReloadCmd = "systemctl reload php" + v + "-fpm"
	}
	return p
}

// AccountHomesDir returns the parent directory for account homes.
func (p Paths) AccountHomesDir() string { return p.HomesDir }

// AccountHome returns the home dir for an account username.
func (p Paths) AccountHome(username string) string {
	return path.Join(p.HomesDir, username)
}

// AccountPublicHTML returns the document root parent for an account.
func (p Paths) AccountPublicHTML(username string) string {
	return path.Join(p.AccountHome(username), "public_html")
}

// DomainDocumentRoot returns the document root for a specific domain.
func (p Paths) DomainDocumentRoot(username, domain string) string {
	return path.Join(p.AccountPublicHTML(username), domain)
}

// NginxVHostPath returns the on-disk path for the vhost config.
func (p Paths) NginxVHostPath(username, domain string) string {
	return path.Join(p.NginxDir, username+"-"+domain+".conf")
}

// FpmPoolPath returns the on-disk path for the php-fpm pool config.
func (p Paths) FpmPoolPath(username, domain string) string {
	return path.Join(p.FpmDir, "orvix-"+username+"-"+domain+".conf")
}

// ReleasesDir returns the releases parent for a (username, domain).
func (p Paths) ReleasesDir(username, domain string) string {
	return path.Join(p.ReleasesRoot, username, domain)
}

// ReleaseDir returns the path for a specific release (timestamped).
func (p Paths) ReleaseDir(username, domain, release string) string {
	return path.Join(p.ReleasesDir(username, domain), release)
}

// DomainLogDir returns the log dir for a (username, domain).
func (p Paths) DomainLogDir(username, domain string) string {
	return path.Join(p.LogRoot, username, domain)
}

// EnsureDirs creates the BaseDir + every subdir the service writes to.
// Idempotent. Returns the first dir-creation error.
func (p Paths) EnsureDirs() error {
	dirs := []string{
		p.BaseDir, p.HomesDir, p.NginxDir, p.FpmDir,
		p.DocumentRoot, p.ReleasesRoot, p.LogRoot,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}
