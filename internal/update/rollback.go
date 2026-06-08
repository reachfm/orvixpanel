package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// RollbackResult contains the result of a rollback operation.
type RollbackResult struct {
	Success        bool
	FromVersion    Version
	ToVersion      Version
	BackupID       string
	Error          error
	ServiceRestarted bool
}

// Rollback reverts to a previous backup.
func Rollback(backupID string) (*RollbackResult, error) {
	result := &RollbackResult{BackupID: backupID}

	// Load backup manifest
	manifest, err := LoadBackupManifest(backupID)
	if err != nil {
		result.Error = fmt.Errorf("load backup manifest: %w", err)
		return result, result.Error
	}

	result.FromVersion = Version{Tag: "current"}
	result.ToVersion = manifest.Version

	p := GetInstallPaths()

	// Verify backup integrity
	log.Info().Str("backup_id", backupID).Msg("Verifying backup integrity...")
	if err := VerifyBackup(backupID); err != nil {
		result.Error = fmt.Errorf("backup verification failed: %w", err)
		return result, result.Error
	}

	// Stop service
	log.Info().Msg("Stopping orvixpanel service...")
	if err := stopService(); err != nil {
		result.Error = fmt.Errorf("stop service: %w", err)
		return result, result.Error
	}

	// Restore files
	backupDir := filepath.Join(p.Backup, backupID)

	restoreItems := []struct {
		srcName string
		dstPath string
	}{
		{"binaries", p.Bin},
		{"config", p.Etc},
		{"data", p.Var},
	}

	for _, item := range restoreItems {
		backupPath := filepath.Join(backupDir, item.srcName)
		if _, err := os.Stat(backupPath); err != nil {
			continue // Skip if not in backup
		}

		log.Info().Str("item", item.srcName).Msg("Restoring...")

		// Remove current
		os.RemoveAll(item.dstPath)

		// Restore from backup
		if err := copyDirSimple(backupPath, item.dstPath); err != nil {
			result.Error = fmt.Errorf("restore %s: %w", item.srcName, err)
			return result, result.Error
		}
	}

	// Reload systemd
	exec.Command("systemctl", "daemon-reload").Run()

	// Start service
	log.Info().Msg("Starting orvixpanel service...")
	if err := startService(); err != nil {
		result.Error = fmt.Errorf("start service: %w", err)
		return result, result.Error
	}
	result.ServiceRestarted = true

	result.Success = true
	return result, nil
}

// RollbackToPrevious performs a rollback to the most recent backup.
func RollbackToPrevious() (*RollbackResult, error) {
	backups, err := ListBackups()
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups available for rollback")
	}

	// Find the most recent backup (not the current one)
	var latest *RollbackPoint
	for i := range backups {
		if latest == nil || backups[i].CreatedAt.After(latest.CreatedAt) {
			latest = &backups[i]
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no suitable backup found")
	}

	log.Info().Str("backup_id", latest.ID).Str("version", latest.Version.Tag).
		Msg("Rolling back to previous version...")

	return Rollback(latest.ID)
}

// RestoreBinary restores just the binary from a backup.
func RestoreBinary(backupID string) error {
	p := GetInstallPaths()
	backupDir := filepath.Join(p.Backup, backupID)
	backupBin := filepath.Join(backupDir, "binaries", LinuxBinary)
	installPath := filepath.Join(p.Bin, BinaryName)

	if _, err := os.Stat(backupBin); err != nil {
		return fmt.Errorf("backup binary not found: %w", err)
	}

	// Stop service
	if err := stopService(); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}

	// Copy binary
	if err := copyFileSimple(backupBin, installPath); err != nil {
		return fmt.Errorf("restore binary: %w", err)
	}
	os.Chmod(installPath, 0o755)

	// Start service
	return startService()
}

// GetCurrentBinaryHash returns the SHA256 hash of the current binary.
func GetCurrentBinaryHash() (string, error) {
	p := GetInstallPaths()
	binaryPath := filepath.Join(p.Bin, BinaryName)

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", fmt.Errorf("read binary: %w", err)
	}

	cmd := exec.Command("sha256sum")
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("sha256sum: %w", err)
	}

	return string(output)[:64], nil
}