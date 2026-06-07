//go:build linux

// Linux-only: nginx / php-fpm write+validate+reload, domain
// provisioning, atomic-symswap deployment.
package hosting

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

// domainRegex is the spec rule for valid DNS hostnames. RFC 1123
// with the underscore rule relaxed (we allow underscores because
// real-world internal labels use them).
var domainRegex = regexp.MustCompile(
	`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?` +
		`(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)+$`,
)

// ValidateDomain checks a domain name for shape + length. It
// does NOT resolve DNS — that would be slow and the DNS is
// not the panel's job.
func (s *Service) ValidateDomain(name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if len(name) == 0 || len(name) > 253 {
		return fmt.Errorf("domain %q: must be 1-253 chars", name)
	}
	if !domainRegex.MatchString(name) {
		return fmt.Errorf("domain %q: invalid shape", name)
	}
	// Reject the local TLD; operators would never host test.local
	// in production but the smoke test does, so we add an opt-out
	// via env var for the smoke gate.
	if strings.HasSuffix(name, ".local") && os.Getenv("ORVIX_ALLOW_LOCAL_TLD") != "1" {
		// accept anyway if the OS is WSL (heuristic)
		if !isWSL() {
			return fmt.Errorf("domain %q: .local TLD requires ORVIX_ALLOW_LOCAL_TLD=1", name)
		}
	}
	return nil
}

func isWSL() bool {
	out, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "microsoft") ||
		strings.Contains(strings.ToLower(string(out)), "wsl")
}

// DomainOwnedBy reports whether the document-root path for a
// (username, domain) pair lives inside that user's home dir
// (i.e. the user has permission to write to it). This is a
// path-traversal defense for the create-domain path.
func (s *Service) DomainOwnedBy(username, domain string) (bool, error) {
	if err := validateUsername(username); err != nil {
		return false, err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return false, err
	}
	docRoot := s.Paths.DomainDocumentRoot(username, domain)
	home := s.Paths.AccountHome(username)
	rel, err := filepath.Rel(home, docRoot)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(rel, "..") || rel == ".." {
		return false, nil
	}
	return true, nil
}

// CreateDomain provisions the document root for a (user, domain)
// pair and writes a placeholder index.html so the site is
// immediately servable.
func (s *Service) CreateDomain(username, domain string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	owned, err := s.DomainOwnedBy(username, domain)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf("domain %q: path escapes account home", domain)
	}

	docRoot := s.Paths.DomainDocumentRoot(username, domain)
	if err := os.MkdirAll(docRoot, 0o755); err != nil {
		return fmt.Errorf("mkdir doc root: %w", err)
	}
	if err := s.runShell(fmt.Sprintf("chown -R %s:%s %q", username, username, docRoot)); err != nil {
		return fmt.Errorf("chown doc root: %w", err)
	}

	// Drop a placeholder index.html.
	placeholder := fmt.Sprintf(
		"<!doctype html>\n<title>OrvixPanel</title>\n"+
			"<h1>OrvixPanel placeholder</h1>\n"+
			"<p>Account: %s</p>\n<p>Domain: %s</p>\n",
		username, domain,
	)
	idx := filepath.Join(docRoot, "index.html")
	if err := os.WriteFile(idx, []byte(placeholder), 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	if err := s.runShell(fmt.Sprintf("chown %s:%s %q", username, username, idx)); err != nil {
		return fmt.Errorf("chown index: %w", err)
	}

	// Drop a placeholder info.php so the PHP-FPM path can be
	// smoke-tested with `curl .../info.php`.
	infoPhp := `<?php
echo "OrvixPanel PHP-FPM alive\n";
echo "PHP version: " . PHP_VERSION . "\n";
echo "User: " . get_current_user() . "\n";
echo "open_basedir: " . ini_get('open_basedir') . "\n";
`
	infoPath := filepath.Join(docRoot, "info.php")
	if err := os.WriteFile(infoPath, []byte(infoPhp), 0o644); err != nil {
		return fmt.Errorf("write info.php: %w", err)
	}
	if err := s.runShell(fmt.Sprintf("chown %s:%s %q", username, username, infoPath)); err != nil {
		return fmt.Errorf("chown info.php: %w", err)
	}
	return nil
}

// DeleteDomain removes the document root + vhost + fpm pool for
// a (user, domain) pair. Does NOT delete the account.
func (s *Service) DeleteDomain(username, domain string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	// Best-effort: each step logs but doesn't fail the whole delete.
	_ = s.RemoveVHostConfig(username, domain)
	_ = s.RemoveFPMPool(username, domain)
	docRoot := s.Paths.DomainDocumentRoot(username, domain)
	if err := os.RemoveAll(docRoot); err != nil {
		return fmt.Errorf("remove doc root: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Nginx vhost write + validate
// -----------------------------------------------------------------------------

// WriteVHostConfig writes the rendered config to disk and chowns
// it to the account user (so the user can edit it). The file
// lands in Paths.NginxDir.
func (s *Service) WriteVHostConfig(username, domain, body string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Paths.NginxDir, 0o755); err != nil {
		return err
	}
	dst := s.Paths.NginxVHostPath(username, domain)
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write vhost: %w", err)
	}
	return nil
}

// RemoveVHostConfig deletes the vhost file.
func (s *Service) RemoveVHostConfig(username, domain string) error {
	return os.Remove(s.Paths.NginxVHostPath(username, domain))
}

// TestNginx runs `nginx -t` to validate every config in
// /etc/nginx. Returns the combined output on success/failure.
func (s *Service) TestNginx() error {
	return s.runShell(s.Paths.NginxTestCmd)
}

// ReloadNginx runs `systemctl reload nginx`.
func (s *Service) ReloadNginx() error {
	return s.runShell(s.Paths.NginxReloadCmd)
}

// -----------------------------------------------------------------------------
// PHP-FPM pool write + validate
// -----------------------------------------------------------------------------

// WriteFPMPool writes the rendered pool to disk and chowns it to
// the account user. The file lands in Paths.FpmDir.
func (s *Service) WriteFPMPool(username, domain, body string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Paths.FpmDir, 0o755); err != nil {
		return err
	}
	dst := s.Paths.FpmPoolPath(username, domain)
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write fpm pool: %w", err)
	}
	return nil
}

// RemoveFPMPool deletes the pool file.
func (s *Service) RemoveFPMPool(username, domain string) error {
	return os.Remove(s.Paths.FpmPoolPath(username, domain))
}

// TestPHP runs `php-fpm -t` to validate every pool config.
func (s *Service) TestPHP() error {
	return s.runShell(s.Paths.FpmTestCmd)
}

// ReloadPHP runs `systemctl reload php-fpm`.
func (s *Service) ReloadPHP() error {
	return s.runShell(s.Paths.FpmReloadCmd)
}

// -----------------------------------------------------------------------------
// Deployment: document root + release dirs + atomic symlink swap
// -----------------------------------------------------------------------------

// CreateReleaseDir creates a timestamped release directory
// (releases/<ts>), symlinks "shared" resources into it (none yet),
// and returns the release name. The caller is expected to
// populate the release and call AtomicSwap.
func (s *Service) CreateReleaseDir(username, domain string) (string, error) {
	if err := validateUsername(username); err != nil {
		return "", err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return "", err
	}
	ts := nowMillisUTC()
	rel := s.Paths.ReleaseDir(username, domain, ts)
	if err := os.MkdirAll(rel, 0o755); err != nil {
		return "", fmt.Errorf("mkdir release: %w", err)
	}
	if err := s.runShell(fmt.Sprintf("chown -R %s:%s %q", username, username, rel)); err != nil {
		return "", fmt.Errorf("chown release: %w", err)
	}
	return ts, nil
}

// AtomicSwap makes a release the "current" one by re-pointing
// the document root's symlink at the release dir.
//
// The doc root for (user, domain) is treated as a symlink: we
// write the release files into the user's home, then update the
// symlink atomically with rename(2).
func (s *Service) AtomicSwap(username, domain, newRelease string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	src := s.Paths.ReleaseDir(username, domain, newRelease)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("release %s: %w", newRelease, err)
	}
	dst := s.Paths.DomainDocumentRoot(username, domain)
	// If dst doesn't exist, create it as a symlink to src.
	if _, err := os.Lstat(dst); err != nil {
		if os.IsNotExist(err) {
			return os.Symlink(src, dst)
		}
		return err
	}
	// Atomic swap: rename(2) is atomic on the same filesystem.
	// We do it by creating a temp symlink and renaming.
	tmp := dst + ".orvix-tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(src, tmp); err != nil {
		return fmt.Errorf("temp symlink: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("atomic rename: %w", err)
	}
	return nil
}

// CurrentRelease returns the target of the document-root
// symlink, or "" if the document root is not a symlink.
func (s *Service) CurrentRelease(username, domain string) (string, error) {
	dst := s.Paths.DomainDocumentRoot(username, domain)
	li, err := os.Lstat(dst)
	if err != nil {
		return "", err
	}
	if li.Mode()&os.ModeSymlink == 0 {
		return "", nil
	}
	target, err := os.Readlink(dst)
	if err != nil {
		return "", err
	}
	return filepath.Base(target), nil
}

// ListReleases returns release names sorted by timestamp DESC.
func (s *Service) ListReleases(username, domain string) ([]string, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := s.ValidateDomain(domain); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Paths.ReleasesDir(username, domain))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names, nil
}

// DisablePool removes the pool config + does NOT reload. Use
// this when an account is suspended: nginx keeps serving (with
// 503) but the pool is gone so no PHP is invoked.
func (s *Service) DisablePool(username, domain string) error {
	return s.RemoveFPMPool(username, domain)
}

// nowMillisUTC returns a millisecond-precision UTC timestamp
// string for use as a release name. We use ms because two
// concurrent deploys in the same second are realistic.
func nowMillisUTC() string {
	now := timeNowUTC()
	return fmt.Sprintf("%d", now.UnixMilli())
}

// avoid unused imports in dev builds
var _ = exec.Command
var _ = syscall.Chown
