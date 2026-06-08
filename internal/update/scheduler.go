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
func CheckForUpdates(channel Channel) (*CheckResult, error) {
	result := &CheckResult{}

	// Get installed version
	current := InstalledVersion()
	result.CurrentVersion = current

	// Clone/fetch the repo temporarily to check latest
	p := GetInstallPaths()
	buildDir := p.Cache + "/update-check-" + uuid.New().String()
	defer os.RemoveAll(buildDir)

	// Clone shallow
	cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/orvixpanel/orvixpanel.git", buildDir)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone: %s: %w", string(out), err)
	}

	// Get latest tag
	cmd = exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = buildDir
	output, err := cmd.Output()
	if err != nil {
		// No tags found
		result.UpdateAvailable = false
		return result, nil
	}

	latestTag := string(bytes.TrimSpace(output))

	// Compare versions
	if latestTag != current.Tag {
		result.UpdateAvailable = true
		result.LatestVersion = Version{
			Tag: latestTag,
			Date: time.Now().UTC().Format(time.RFC3339),
		}
	}

	return result, nil
}