package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateBackup creates a backup of the current installation.
func CreateBackup(v Version, cfg *UpdateConfig) (*BackupManifest, error) {
	backupID := uuid.New().String()[:8]
	backupDir := cfg.BackupPath
	if backupDir == "" {
		backupDir = filepath.Join(GetInstallPaths().Backup, backupID)
	}

	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	manifest := &BackupManifest{
		ID:          backupID,
		Version:     v,
		CreatedAt:   time.Now().UTC(),
		BackupPaths: make(map[string]string),
		SHA256Sums:  make(map[string]string),
		GitState:    GitState{},
		EnvVars:     make(map[string]string),
		Status:      "pending",
	}

	p := GetInstallPaths()

	// Backup files
	backupItems := []struct {
		name string
		path string
	}{
		{"binaries", p.Bin},
		{"config", p.Etc},
		{"data", p.Var},
	}

	for _, item := range backupItems {
		if _, err := os.Stat(item.path); err == nil {
			backupPath := filepath.Join(backupDir, item.name)
			if err := copyDir(item.path, backupPath, manifest); err != nil {
				return nil, fmt.Errorf("backup %s: %w", item.name, err)
			}
			manifest.BackupPaths[item.name] = backupPath
		}
	}

	// Capture git state if in git repo
	if gitState, err := captureGitState(p.Base); err == nil {
		manifest.GitState = gitState
	}

	// Capture relevant env vars (without secrets)
	captureEnvVars(manifest)

	// Generate sha256sums
	if err := generateSHA256Sums(backupDir, manifest); err != nil {
		return nil, fmt.Errorf("generate sha256sums: %w", err)
	}

	// Save manifest
	manifestPath := filepath.Join(backupDir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	manifest.Status = "completed"

	// Create symlink to latest
	latestLink := filepath.Join(GetInstallPaths().Backup, "latest")
	os.Remove(latestLink)
	if err := os.Symlink(backupDir, latestLink); err != nil {
		// Non-fatal, continue
	}

	return manifest, nil
}

// copyDir copies a directory recursively, tracking file checksums.
func copyDir(src, dst string, manifest *BackupManifest) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		return copyFile(path, dstPath, manifest)
	})
}

// copyFile copies a single file and tracks its checksum.
func copyFile(src, dst string, manifest *BackupManifest) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy with checksum
	hash := sha256.New()
	writer := io.MultiWriter(dstFile, hash)

	if _, err := io.Copy(writer, srcFile); err != nil {
		return err
	}

	// Set permissions
	info, _ := srcFile.Stat()
	os.Chmod(dst, info.Mode())

	// Track checksum (relative to backup dir)
	relDst, _ := filepath.Rel(filepath.Dir(manifest.BackupPaths["binaries"]), dst)
	manifest.SHA256Sums[relDst] = hex.EncodeToString(hash.Sum(nil))

	return nil
}

// captureGitState captures the current git state.
func captureGitState(repoPath string) (GitState, error) {
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return GitState{}, fmt.Errorf("not a git repo: %w", err)
	}

	runGit := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		output, err := cmd.Output()
		return strings.TrimSpace(string(output)), err
	}

	state := GitState{}

	// Get commit
	if commit, err := runGit("rev-parse", "HEAD"); err == nil {
		state.Commit = commit
	}

	// Get branch
	if branch, err := runGit("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		state.Branch = branch
	}

	// Get current tag if any
	if tag, err := runGit("describe", "--tags", "--exact-match"); err == nil {
		state.Tag = tag
	}

	// Check for uncommitted changes
	if status, err := runGit("status", "--porcelain"); err == nil {
		lines := strings.Split(status, "\n")
		for _, line := range lines {
			if len(line) < 2 {
				continue
			}
			stat := line[:2]
			path := strings.TrimSpace(line[3:])

			if stat[0] != ' ' && stat[0] != '?' {
				state.Staged = append(state.Staged, path)
			}
			if stat[1] != ' ' && stat[1] != '?' {
				state.Modified = append(state.Modified, path)
			}
			if stat[0] == '?' && stat[1] == '?' {
				state.Untracked = append(state.Untracked, path)
			}
		}
		state.IsDirty = len(state.Staged)+len(state.Modified)+len(state.Untracked) > 0
	}

	return state, nil
}

// captureEnvVars captures relevant environment variables.
func captureEnvVars(manifest *BackupManifest) {
	// Only capture variable names, not actual values (security)
	relevantVars := []string{
		"ORVIX_SERVER_SECRET_KEY",
		"ORVIX_MASTER_KEY",
		"ORVIX_DB_PATH",
		"ORVIX_FRONTEND_DIST",
		"ORVIX_LICENSE_KEY",
		"ORVIX_ALLOW_DEV",
		"ORVIX_POWERDNS_URL",
	}

	for _, name := range relevantVars {
		if val := os.Getenv(name); val != "" {
			manifest.EnvVars[name] = "[REDACTED]"
		}
	}
}

// generateSHA256Sums generates sha256sums file.
func generateSHA256Sums(dir string, manifest *BackupManifest) error {
	output, err := exec.Command("find", ".", "-type", "f", "-exec", "sha256sum", "{}", ";").
		Output()
	if err != nil {
		return err
	}

	// Save to file
	sumsFile := filepath.Join(dir, "sha256sums.txt")
	return os.WriteFile(sumsFile, output, 0o644)
}

// ListBackups lists available rollback points.
func ListBackups() ([]RollbackPoint, error) {
	backupDir := GetInstallPaths().Backup

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var points []RollbackPoint
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(backupDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest BackupManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		// Calculate size
		var totalSize int64
		filepath.Walk(filepath.Join(backupDir, entry.Name()), func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})

		points = append(points, RollbackPoint{
			ID:           manifest.ID,
			Version:      manifest.Version,
			CreatedAt:    manifest.CreatedAt,
			ManifestPath: manifestPath,
			DataSize:     totalSize,
		})
	}

	return points, nil
}

// LoadBackupManifest loads a backup manifest.
func LoadBackupManifest(backupID string) (*BackupManifest, error) {
	manifestPath := filepath.Join(GetInstallPaths().Backup, backupID, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

// VerifyBackup verifies backup integrity.
func VerifyBackup(backupID string) error {
	manifest, err := LoadBackupManifest(backupID)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	backupDir := filepath.Join(GetInstallPaths().Backup, backupID)
	sumsFile := filepath.Join(backupDir, "sha256sums.txt")

	data, err := os.ReadFile(sumsFile)
	if err != nil {
		return fmt.Errorf("read sums file: %w", err)
	}

	// Verify checksums
	lines := strings.Split(string(data), "\n")
	errors := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}

		checksum := parts[0]
		filePath := strings.TrimPrefix(parts[1], "./")
		fullPath := filepath.Join(backupDir, filePath)

		if !verifyFileChecksum(fullPath, checksum) {
			errors++
		}
	}

	if errors > 0 {
		return fmt.Errorf("backup verification failed: %d file(s) failed checksum", errors)
	}

	_ = manifest // used for logging
	return nil
}

func verifyFileChecksum(path, expected string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]) == expected
}