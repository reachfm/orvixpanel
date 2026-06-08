// Package update implements the OrvixPanel self-update engine.
//
// v0.7.1 Production Update Engine features:
//   - cPanel-style CLI with check/dry-run/channel/version/rollback flags
//   - Preflight checks (root, OS, disk, Go, Node, nginx, env, systemd)
//   - Backup system with manifest, sha256sums, and git state
//   - Git-based fetch with proper branch/tag handling
//   - Cross-compile build (bin/orvixpanel.linux for installer compatibility)
//   - Frontend build with pnpm
//   - Env self-healing (ORVIX_FRONTEND_DIST, ORVIX_MASTER_KEY, ORVIX_SERVER_SECRET_KEY)
//   - nginx self-healing (duplicate default_server, conflicting sites)
//   - Health verification with automatic rollback on failure
//   - Rollback with manifest tracking
//   - Structured logging with secret redaction
package update

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InstallBase is the canonical installation directory.
const InstallBase = "/opt/orvixpanel"

// BinaryName is the main binary name.
const BinaryName = "orvixpanel"

// LinuxBinary is the installer-expected binary name (cross-compiled output).
const LinuxBinary = "orvixpanel.linux"

// Version represents a software version with git metadata.
type Version struct {
	Tag    string // e.g., "v0.7.1"
	Commit string // 40-char git SHA
	Date   string // ISO 8601 date
	Dirty  bool   // uncommitted changes
}

// String returns a human-readable version string.
func (v Version) String() string {
	s := v.Tag
	if v.Commit != "" {
		s += " (" + v.Commit[:8] + ")"
	}
	if v.Dirty {
		s += " (dirty)"
	}
	return s
}

// Channel represents an update channel.
type Channel string

const (
	ChannelStable  Channel = "stable"
	ChannelPreview Channel = "preview"
)

// UpdateConfig holds the update operation parameters.
type UpdateConfig struct {
	Check       bool    // Check for updates only, don't install
	DryRun      bool    // Simulate update without making changes
	Channel     Channel // Update channel (stable or preview)
	Version     string  // Specific version/tag/commit to install
	Rollback    bool    // Rollback to previous version
	BackupPath  string  // Custom backup directory (default: /var/backups/orvixpanel)
	SkipBackup  bool    // Skip backup creation
	SkipFetch   bool    // Skip git fetch (use existing checkout)
	Verbose     bool    // Verbose output
	NonInteractive bool // Non-interactive mode (no prompts)
}

// UpdateResult contains the result of an update operation.
type UpdateResult struct {
	Success     bool
	FromVersion Version
	ToVersion   Version
	BackupID    string
	RollbackAvailable bool
	Error       error
	Logs        []string
}

// CheckResult contains the result of an update check.
type CheckResult struct {
	UpdateAvailable bool
	CurrentVersion Version
	LatestVersion  Version
	ReleaseNotes   string
	ChangelogURL   string
}

// RollbackPoint represents a point to which we can rollback.
type RollbackPoint struct {
	ID           string    // Backup ID
	Version      Version   // Version at time of backup
	CreatedAt    time.Time // When the backup was created
	ManifestPath string    // Path to manifest file
	DataSize     int64     // Size of backed up data in bytes
}

// PreflightCheck represents a preflight check result.
type PreflightCheck struct {
	Name        string
	Status      CheckStatus
	Message     string
	Details     string
	Suggestions []string
}

// CheckStatus represents the status of a check.
type CheckStatus string

const (
	CheckPass    CheckStatus = "pass"
	CheckWarn    CheckStatus = "warn"
	CheckFail    CheckStatus = "fail"
	CheckSkip    CheckStatus = "skip"
	CheckUnknown CheckStatus = "unknown"
)

// IsBlocking returns true if this check status blocks the update.
func (s CheckStatus) IsBlocking() bool {
	return s == CheckFail
}

// BackupManifest represents a backup manifest.
type BackupManifest struct {
	ID          string            `json:"id"`
	Version     Version           `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
	BackupPaths map[string]string `json:"backup_paths"` // logical name -> actual path
	SHA256Sums  map[string]string `json:"sha256sums"`  // file path -> checksum
	GitState    GitState          `json:"git_state"`
	EnvVars     map[string]string `json:"env_vars"`    // redacted values
	Status      string            `json:"status"`      // pending, completed, failed
}

// GitState captures the git state at backup time.
type GitState struct {
	Commit    string `json:"commit"`
	Tag       string `json:"tag"`
	Branch    string `json:"branch"`
	IsDirty   bool   `json:"is_dirty"`
	Staged    []string `json:"staged_files"`
	Modified  []string `json:"modified_files"`
	Untracked []string `json:"untracked_files"`
}

// GetInstallPaths returns the standard installation paths.
func GetInstallPaths() Paths {
	return Paths{
		Base:      InstallBase,
		Bin:       filepath.Join(InstallBase, "bin"),
		Etc:       filepath.Join(InstallBase, "etc"),
		Var:       filepath.Join(InstallBase, "var"),
		Lib:       filepath.Join(InstallBase, "lib"),
		Cache:     filepath.Join(InstallBase, "var", "cache"),
		Backup:    filepath.Join(InstallBase, "var", "backups"),
		Log:       filepath.Join(InstallBase, "var", "log"),
		Run:       "/run/orvixpanel",
		EnvFile:   filepath.Join(InstallBase, "etc", "orvixpanel.env"),
		Service:   "/etc/systemd/system/orvixpanel.service",
	}
}

// Paths holds all relevant filesystem paths.
type Paths struct {
	Base      string
	Bin       string
	Etc       string
	Var       string
	Lib       string
	Cache     string
	Backup    string
	Log       string
	Run       string
	EnvFile   string
	Service   string
}

// String implements fmt.Stringer for Paths.
func (p Paths) String() string {
	return fmt.Sprintf("Paths{Base:%s Bin:%s Etc:%s Var:%s}", p.Base, p.Bin, p.Etc, p.Var)
}

// EnsurePathsExist creates required directories.
func EnsurePathsExist() error {
	p := GetInstallPaths()
	dirs := []string{p.Base, p.Bin, p.Etc, p.Var, p.Cache, p.Backup, p.Log}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", d, err)
		}
	}
	return nil
}

// IsInstalled checks if OrvixPanel appears to be installed.
func IsInstalled() bool {
	p := GetInstallPaths()
	// Check for at least one of: the binary, the env file, or the service
	if _, err := os.Stat(filepath.Join(p.Bin, BinaryName)); err == nil {
		return true
	}
	if _, err := os.Stat(p.EnvFile); err == nil {
		return true
	}
	if _, err := os.Stat(p.Service); err == nil {
		return true
	}
	return false
}