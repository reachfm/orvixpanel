// Package update implements the OrvixPanel self-update engine.
//
// v0.7.2 Autonomous Update Manager features:
//   - Dynamic runtime configuration discovery from env file
//   - Custom port support for health checks (8080, 8443, etc.)
//   - Runtime storage verification
//   - Version tracking with channel info
//   - Automatic update scheduler (systemd timers)
//   - Persistent update history store
//   - Production API routes for update management
package update

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Ensure uuid is used to avoid import error
var _ = uuid.New

// Systemd unit file templates

//go:embed systemd/orvixpanel-update-check.service.tpl
var updateCheckServiceTpl string

//go:embed systemd/orvixpanel-update-check.timer.tpl
var updateCheckTimerTpl string

//go:embed systemd/orvixpanel-auto-update.service.tpl
var autoUpdateServiceTpl string

//go:embed systemd/orvixpanel-auto-update.timer.tpl
var autoUpdateTimerTpl string

// Scheduler manages automatic update scheduling via systemd timers.
type Scheduler struct {
	BinaryPath string
}

// NewScheduler creates a new Scheduler instance.
func NewScheduler(binaryPath string) *Scheduler {
	return &Scheduler{BinaryPath: binaryPath}
}

// InstallTimers installs the systemd timer units for automatic updates.
func (s *Scheduler) InstallTimers() error {
	templates := map[string]string{
		"/etc/systemd/system/orvixpanel-update-check.service": updateCheckServiceTpl,
		"/etc/systemd/system/orvixpanel-update-check.timer":    updateCheckTimerTpl,
		"/etc/systemd/system/orvixpanel-auto-update.service":  autoUpdateServiceTpl,
		"/etc/systemd/system/orvixpanel-auto-update.timer":   autoUpdateTimerTpl,
	}

	for path, tpl := range templates {
		if err := s.writeSystemdUnit(path, tpl); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	// Reload systemd and enable timers
	if err := reloadSystemd(); err != nil {
		return fmt.Errorf("systemd reload: %w", err)
	}

	// Enable timers
	for _, timer := range []string{"orvixpanel-update-check.timer", "orvixpanel-auto-update.timer"} {
		cmd := exec.Command("systemctl", "enable", timer)
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			log.Warn().Err(err).Str("timer", timer).Msg("Failed to enable timer")
		}
	}

	return nil
}

// writeSystemdUnit writes a systemd unit file from template.
func (s *Scheduler) writeSystemdUnit(path, tplStr string) error {
	tpl, err := template.New("unit").Parse(tplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		BinaryPath string
		InstallDir string
	}{
		BinaryPath: s.BinaryPath,
		InstallDir: InstallBase,
	}

	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// UninstallTimers removes the systemd timer units.
func (s *Scheduler) UninstallTimers() error {
	timers := []string{
		"orvixpanel-update-check.timer",
		"orvixpanel-update-check.service",
		"orvixpanel-auto-update.timer",
		"orvixpanel-auto-update.service",
	}

	for _, timer := range timers {
		path := "/etc/systemd/system/" + timer
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Warn().Err(err).Str("path", path).Msg("Failed to remove timer unit")
		}
	}

	reloadSystemd()
	return nil
}

// StartTimers starts the update check and auto-update timers.
func (s *Scheduler) StartTimers() error {
	for _, timer := range []string{"orvixpanel-update-check.timer", "orvixpanel-auto-update.timer"} {
		cmd := exec.Command("systemctl", "start", timer)
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("start %s: %w", timer, err)
		}
	}
	return nil
}

// StopTimers stops the update check and auto-update timers.
func (s *Scheduler) StopTimers() error {
	for _, timer := range []string{"orvixpanel-update-check.timer", "orvixpanel-auto-update.timer"} {
		cmd := exec.Command("systemctl", "stop", timer)
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("stop %s: %w", timer, err)
		}
	}
	return nil
}

// TimerStatus returns the status of the update timers.
func (s *Scheduler) TimerStatus() (map[string]bool, error) {
	status := make(map[string]bool)
	timers := []string{"orvixpanel-update-check.timer", "orvixpanel-auto-update.timer"}

	for _, timer := range timers {
		cmd := exec.Command("systemctl", "is-active", timer)
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		output, _ := cmd.Output()
		status[timer] = bytes.TrimSpace(output) != nil && string(bytes.TrimSpace(output)) == "active"
	}

	return status, nil
}

// RecordUpdateStart records the start of an update operation.
func RecordUpdateStart(fromVersion Version, channel Channel) string {
	id := uuid.New().String()
	entry := UpdateHistory{
		ID:          id,
		FromVersion: fromVersion,
		Timestamp:   time.Now().UTC(),
		Channel:     channel,
		Result:      "in_progress",
	}

	// Save as pending entry
	history, _ := GetUpdateHistory()
	history = append(history, entry)
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	SaveUpdateHistory(history)

	return id
}

// RecordUpdateResult records the result of an update operation.
func RecordUpdateResult(id string, toVersion Version, result string, backupID string, errorMsg string, duration int64) {
	history, err := GetUpdateHistory()
	if err != nil {
		return
	}

	for i := range history {
		if history[i].ID == id {
			history[i].ToVersion = toVersion
			history[i].Result = result
			history[i].BackupID = backupID
			history[i].ErrorMessage = errorMsg
			history[i].Duration = duration
			break
		}
	}

	SaveUpdateHistory(history)
}

// CheckForUpdates checks for available updates using git.
// For preview channel: compares with origin/feature branch HEAD.
// For stable channel: compares with latest semver tag.
func CheckForUpdates(channel Channel) (*CheckResult, error) {
	result := &CheckResult{}

	// Use current directory for git operations (works both in dev and production)
	baseDir, _ := os.Getwd()

	// Get installed version - if empty, fall back to local git state
	current := InstalledVersion()
	if current.Commit == "" {
		// In dev mode, use local HEAD as "installed" version
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			current.Commit = strings.TrimSpace(string(out))
		}
		// Get latest v* tag (not nearest tag, but latest semantic version)
		cmd = exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			tags := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(tags) > 0 && tags[0] != "" {
				current.Tag = tags[0] // First tag is the latest v* tag
			}
		}
		// No tag, use short commit
		if current.Tag == "" && len(current.Commit) > 8 {
			current.Tag = current.Commit[:8]
		}
	}
	result.CurrentVersion = current

	// Fetch latest from origin
	cmd := exec.Command("git", "fetch", "--all", "--tags")
	cmd.Dir = baseDir
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	cmd.Run() // Best effort

	var target Version

	if channel == ChannelPreview {
		// Preview: get remote feature branch HEAD
		cmd = exec.Command("git", "rev-parse", "origin/feature/v0.7.0-mail-hosting")
		cmd.Dir = baseDir
		output, err := cmd.Output()
		if err != nil {
			// Fall back to local HEAD
			cmd = exec.Command("git", "rev-parse", "HEAD")
			cmd.Dir = baseDir
			output, err = cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("get HEAD: %w", err)
			}
		}

		target.Commit = strings.TrimSpace(string(output))

		// Get latest v* tag (not nearest tag, but latest semantic version)
		cmd = exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			tags := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(tags) > 0 && tags[0] != "" {
				target.Tag = tags[0] // First tag is the latest v* tag
			}
		}
		// Fall back to short commit if no tag found
		if target.Tag == "" && len(target.Commit) > 8 {
			target.Tag = target.Commit[:8]
		}
	} else {
		// Stable: get latest v* tag
		cmd = exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
		cmd.Dir = baseDir
		output, err := cmd.Output()
		if err != nil {
			result.UpdateAvailable = false
			return result, nil
		}

		tags := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(tags) == 0 || tags[0] == "" {
			result.UpdateAvailable = false
			return result, nil
		}
		target.Tag = tags[0]

		// Get commit for this tag
		cmd = exec.Command("git", "rev-parse", target.Tag)
		cmd.Dir = baseDir
		if commitOut, err := cmd.Output(); err == nil {
			target.Commit = strings.TrimSpace(string(commitOut))
		}
	}

	result.LatestVersion = target

	// Compare versions - update needed if commits differ
	result.UpdateAvailable = CompareVersions(current, target)

	return result, nil
}

// CompareVersions compares two versions and returns true if update is needed.
// Returns false when installed version equals target version.
func CompareVersions(installed, target Version) bool {
	// If we have commits for both, compare them (authoritative)
	if installed.Commit != "" && target.Commit != "" {
		// Compare full commits if both are full SHA
		if len(installed.Commit) == 40 && len(target.Commit) == 40 {
			return installed.Commit != target.Commit
		}
		// For short commits, do prefix comparison safely
		minLen := len(installed.Commit)
		if len(target.Commit) < minLen {
			minLen = len(target.Commit)
		}
		return installed.Commit[:minLen] != target.Commit[:minLen]
	}

	// If we have tags for both, compare them
	if installed.Tag != "" && target.Tag != "" {
		return installed.Tag != target.Tag
	}

	// If only target has commit info, we need update (installed is unknown/empty)
	if installed.Commit == "" && target.Commit != "" {
		return true
	}

	// If only target has tag info, we need update
	if installed.Tag == "" && target.Tag != "" {
		return true
	}

	// No update needed if we get here (same commit or both empty)
	return false
}