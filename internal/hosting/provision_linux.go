//go:build linux

// Linux-only: Linux user provisioning, group creation, home dir,
// permission model, suspend/unsuspend/delete. All system exec
// lives in this file. Pure-Go tests for the generators live in
// nginx_test.go and phpfpm_test.go.
package hosting

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Service is the hosting-layer entry point. Holds the on-disk
// paths and the GORM DB.
type Service struct {
	Paths Paths
}

// NewService returns a Service with default paths.
func NewService() *Service {
	return &Service{Paths: DefaultPaths()}
}

// NewServiceWithPaths returns a Service with the given paths.
func NewServiceWithPaths(p Paths) *Service {
	return &Service{Paths: p}
}

// run executes a command and returns the combined output. The
// command is run with /bin/sh -c so the caller can use shell
// features (pipes, globs). Non-zero exit returns *CommandError.
func (s *Service) run(name string, args ...string) error {
	return s.runCtx(context.Background(), name, args...)
}

func (s *Service) runCtx(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: append([]string{name}, args...),
			Output:  strings.TrimSpace(string(out)),
			Err:     err,
		}
	}
	return nil
}

// runShell is run("sh", "-c", cmdline).
func (s *Service) runShell(cmdline string) error {
	return s.run("sh", "-c", cmdline)
}

// CommandError wraps a failed shell/system command with its
// combined output for the caller to log.
type CommandError struct {
	Command []string
	Output  string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %v failed: %v\noutput:\n%s",
		e.Command, e.Err, e.Output)
}
func (e *CommandError) Unwrap() error { return e.Err }

// -----------------------------------------------------------------------------
// Account provisioning
// -----------------------------------------------------------------------------

// CreateAccount provisions a Linux system user for an account.
// Steps:
//   1. groupadd <username>  (private group, GID = UID)
//   2. useradd --home <homedir> --shell /bin/bash --gid <username> <username>
//   3. chown -R <username>:<username> <homedir>
//   4. mkdir -p <homedir>/public_html (0755)
//
// Returns the created user's UID on success.
func (s *Service) CreateAccount(username string) (int, error) {
	if err := validateUsername(username); err != nil {
		return 0, err
	}
	if _, err := user.Lookup(username); err == nil {
		return 0, fmt.Errorf("account %q: %w", username, ErrAccountExists)
	}

	// 1. groupadd
	if err := s.run(s.Paths.GroupaddBin, "-f", username); err != nil {
		return 0, fmt.Errorf("groupadd %s: %w", username, err)
	}

	// 2. useradd
	home := s.Paths.AccountHome(username)
	if err := s.run(s.Paths.UseraddBin,
		"-m",                         // create home if missing
		"-d", home,                   // home dir
		"-s", "/bin/bash",            // shell
		"-g", username,               // primary group = username
		"-c", "OrvixPanel account",   // GECOS
		username,
	); err != nil {
		return 0, fmt.Errorf("useradd %s: %w", username, err)
	}

	// 3. Set ownership + permissions on home.
	if err := s.runShell(fmt.Sprintf("chown -R %s:%s %q", username, username, home)); err != nil {
		return 0, fmt.Errorf("chown home: %w", err)
	}
	// Home dir must be world-traversable (mode 0751) so the nginx
	// worker (www-data) can serve files from it. Files inside
	// inherit the useradd default 0644 which is world-readable.
	if err := os.Chmod(home, 0o751); err != nil {
		return 0, fmt.Errorf("chmod home 0751: %w", err)
	}

	// 4. public_html.
	pub := s.Paths.AccountPublicHTML(username)
	if err := os.MkdirAll(pub, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir public_html: %w", err)
	}
	if err := s.runShell(fmt.Sprintf("chown %s:%s %q", username, username, pub)); err != nil {
		return 0, fmt.Errorf("chown public_html: %w", err)
	}
	// public_html needs world-traversable so nginx can enter.
	if err := os.Chmod(pub, 0o755); err != nil {
		return 0, fmt.Errorf("chmod public_html: %w", err)
	}

	// 5. Look up the UID for the caller.
	u, err := user.Lookup(username)
	if err != nil {
		return 0, fmt.Errorf("lookup created user: %w", err)
	}
	uid, _ := strconv.Atoi(u.Uid)
	return uid, nil
}

// SuspendAccount locks the account (usermod -L) so login is
// impossible but home dir + files remain on disk.
func (s *Service) SuspendAccount(username string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	return s.run(s.Paths.UsermodBin, "-L", username)
}

// UnsuspendAccount unlocks the account.
func (s *Service) UnsuspendAccount(username string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	return s.run(s.Paths.UsermodBin, "-U", username)
}

// DeleteAccount removes the system user, group, and home dir.
// userdel -r removes the home dir tree. Idempotent: returns nil
// if the user doesn't exist.
//
// userdel refuses if the user owns running processes (e.g. a
// PHP-FPM worker). We pkill first, then sleep 1s for the
// kernel to reap the zombies. If after 5s the user is still in
// use, userdel fails with a real exit code we surface to the
// caller.
func (s *Service) DeleteAccount(username string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if _, err := user.Lookup(username); err != nil {
		// user not found — treat as success
		if _, ok := err.(user.UnknownUserError); ok {
			return nil
		}
		return err
	}
	// Kill any processes owned by the user (PHP-FPM workers,
	// lingering cron, etc.). SIGKILL — we don't care about
	// graceful shutdown, the user is being deleted.
	_ = s.run("pkill", "-9", "-u", username)
	// Give the kernel a beat to release the in-use flag.
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		if err := s.run(s.Paths.UserdelBin, "-r", username); err == nil {
			return nil
		}
	}
	// Last attempt — surface the real error.
	return s.run(s.Paths.UserdelBin, "-r", username)
}

// validateUsername enforces the rules: 1-32 chars, lowercase
// letters/digits/underscore/dash, must start with a letter.
// This is the same rule Linux uses for usernames (matching
// /etc/adduser.conf NAME_REGEX).
func validateUsername(u string) error {
	if len(u) == 0 || len(u) > 32 {
		return fmt.Errorf("username %q: must be 1-32 chars", u)
	}
	if u[0] < 'a' || u[0] > 'z' {
		return fmt.Errorf("username %q: must start with a lowercase letter", u)
	}
	for _, r := range u {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-'
		if !ok {
			return fmt.Errorf("username %q: only [a-z0-9_-] allowed", u)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Disk / inode / usage probes
// -----------------------------------------------------------------------------

// DiskUsed returns the disk usage in bytes for a path using `du`.
// `du -sb <path>` gives apparent size (no block rounding).
func (s *Service) DiskUsed(path string) (int64, error) {
	out, err := exec.Command(s.Paths.DuBin, "-sb", path).Output()
	if err != nil {
		return 0, fmt.Errorf("du %s: %w", path, err)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return 0, fmt.Errorf("du %s: empty output", path)
	}
	n, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("du %s: parse %q: %w", path, fields[0], err)
	}
	return n, nil
}

// InodeCount returns the inode count for a path via find(1).
// find <path> -xdev | wc -l. For /home/<user> this is the count
// of files + dirs under it.
func (s *Service) InodeCount(path string) (int64, error) {
	out, err := exec.Command("sh", "-c",
		fmt.Sprintf("find %q -xdev | wc -l", path),
	).Output()
	if err != nil {
		return 0, fmt.Errorf("find %s: %w", path, err)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse inode count: %w", err)
	}
	return n, nil
}

// PathInfo returns a small struct with size + inode count for a
// path. Convenience for the API.
func (s *Service) PathInfo(path string) (DiskUsage, error) {
	bytes, err := s.DiskUsed(path)
	if err != nil {
		return DiskUsage{}, err
	}
	inodes, _ := s.InodeCount(path) // best-effort
	return DiskUsage{Bytes: bytes, Inodes: inodes}, nil
}

// ensureDir is a tiny helper used by the deploy path.
func ensureDir(p string, mode os.FileMode) error {
	if err := os.MkdirAll(p, mode); err != nil {
		return err
	}
	// Chown via shell — we don't have syscall.Chown in pure Go for
	// arbitrary uids without cgo. The Service methods handle the
	// chown side-effect.
	return nil
}

// chownR recursively chowns a path to uid:gid using syscall.Chown.
// Uses raw syscalls because the Go stdlib doesn't expose a
// recursive chown without cgo.
func chownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Resolve symlinks (we only chown the target if it's a
		// regular file/dir, not the link itself).
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return syscall.Chown(p, uid, gid)
	})
}
