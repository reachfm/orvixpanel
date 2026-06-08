/**
 * Backup manager handles backup creation, verification, and storage.
 */

package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// Config holds backup manager configuration.
type Config struct {
	StorageDir    string   // Base directory for local storage
	TempDir       string   // Temporary directory for staging
	ChecksumAlgo  string   // Checksum algorithm (sha256, sha512)
	MaxFileSize   int64    // Maximum file size to include
	ExcludePatterns []string // Patterns to exclude from backup
}

// DefaultConfig returns the default backup manager configuration.
func DefaultConfig() *Config {
	return &Config{
		StorageDir:   "/var/lib/orvixpanel/backups",
		TempDir:      "/tmp/orvixpanel-backup",
		ChecksumAlgo: "sha256",
		MaxFileSize:  10 * 1024 * 1024 * 1024, // 10GB
		ExcludePatterns: []string{
			"*.tmp",
			"*.log",
			"*.swp",
			"*.bak",
			".git/*",
			"node_modules/*",
			".env",
		},
	}
}

// BackupManager handles backup operations.
type BackupManager struct {
	config  *Config
	storage StorageProvider
	factory *StorageFactory
}

// NewBackupManager creates a new backup manager.
func NewBackupManager(config *Config) *BackupManager {
	if config == nil {
		config = DefaultConfig()
	}

	factory := DefaultStorageFactory()
	provider, _ := factory.Create(BackendLocal)
	provider.Initialize(context.Background(), map[string]string{
		"base_path": config.StorageDir,
	})

	return &BackupManager{
		config:  config,
		storage: provider,
		factory: factory,
	}
}

// ArchiveResult holds the result of a backup archive operation.
type ArchiveResult struct {
	BackupID     string
	ArchivePath  string
	FileCount    int
	TotalSize    int64
	Checksum     string
	ChecksumAlgo string
	Duration     time.Duration
	ErrorMessage string
}

// CreateFileBackup creates a backup of files from a directory.
func (m *BackupManager) CreateFileBackup(ctx context.Context, job *models.BackupJob, sourceDir string) (*ArchiveResult, error) {
	result := &ArchiveResult{
		BackupID:     job.ID,
		ChecksumAlgo: m.config.ChecksumAlgo,
	}

	startTime := time.Now()

	// Validate source directory
	if _, err := os.Stat(sourceDir); err != nil {
		result.ErrorMessage = fmt.Sprintf("source directory not accessible: %v", err)
		return result, fmt.Errorf("%w: %v", ErrBackupFailed, err)
	}

	// Create temporary archive file
	archiveName := fmt.Sprintf("%s_%s.tar.gz", job.ID, time.Now().Format("20060102_150405"))
	tempPath := filepath.Join(m.config.TempDir, archiveName)

	if err := os.MkdirAll(m.config.TempDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create temp directory: %v", err)
		return result, fmt.Errorf("%w: %v", ErrStagingFailed, err)
	}

	// Create the archive
	archiveFile, err := os.Create(tempPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create archive: %v", err)
		return result, fmt.Errorf("%w: %v", ErrBackupFailed, err)
	}
	defer archiveFile.Close()

	// Use gzip writer for compression
	gzipWriter, err := gzip.NewWriterLevel(archiveFile, gzip.BestCompression)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create gzip writer: %v", err)
		return result, fmt.Errorf("%w: %v", ErrBackupFailed, err)
	}
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk the source directory
	var fileCount int
	var totalSize int64
	hashWriter := sha256.New()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded patterns
		relPath, _ := filepath.Rel(sourceDir, path)
		if m.shouldExclude(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip large files
		if info.Size() > m.config.MaxFileSize {
			return nil
		}

		// Open source file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Calculate file checksum
		h := sha256.New()
		written, err := io.Copy(h, srcFile)
		if err != nil {
			return err
		}
		srcFile.Close()

		fileChecksum := hex.EncodeToString(h.Sum(nil))

		// Create tar header
		header := &tar.Header{
			Name:    relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Re-open and write file content
		srcFile, err = os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		written, err = io.Copy(tarWriter, srcFile)
		if err != nil {
			return err
		}

		// Update hash for archive
		hashWriter.Write([]byte(relPath))
		hashWriter.Write([]byte(fileChecksum))

		fileCount++
		totalSize += written

		return nil
	})

	if err != nil {
		os.Remove(tempPath)
		result.ErrorMessage = fmt.Sprintf("failed to create archive: %v", err)
		return result, fmt.Errorf("%w: %v", ErrBackupFailed, err)
	}

	// Finalize tar
	tarWriter.Close()
	gzipWriter.Close()
	archiveFile.Close()

	// Calculate final checksum
	result.Checksum = hex.EncodeToString(hashWriter.Sum(nil))
	result.ArchivePath = tempPath
	result.FileCount = fileCount
	result.TotalSize = totalSize
	result.Duration = time.Since(startTime)

	return result, nil
}

// shouldExclude checks if a path matches exclude patterns.
func (m *BackupManager) shouldExclude(path string) bool {
	for _, pattern := range m.config.ExcludePatterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
		// Check directory pattern
		if strings.HasPrefix(pattern, path+"/") {
			return true
		}
	}
	return false
}

// UploadBackup uploads a local backup to the storage backend.
func (m *BackupManager) UploadBackup(ctx context.Context, localPath string, remoteKey string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if err := m.storage.Upload(ctx, remoteKey, file, info.Size()); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

// VerifyBackup verifies a backup's integrity by recalculating checksum.
func (m *BackupManager) VerifyBackup(ctx context.Context, backupPath string, expectedChecksum string) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedChecksum, actualChecksum)
	}

	return nil
}

// ExtractBackup extracts a backup archive to a target directory.
func (m *BackupManager) ExtractBackup(ctx context.Context, archivePath string, targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open archive
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(targetDir, header.Name)

		// Security: prevent path traversal
		absTarget, _ := filepath.Abs(targetPath)
		absDir, _ := filepath.Abs(targetDir)
		if !strings.HasPrefix(absTarget, absDir) {
			return fmt.Errorf("path traversal detected: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Ensure parent directory
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Write file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()

			// Set permissions
			os.Chmod(targetPath, os.FileMode(header.Mode))
		}
	}

	return nil
}

// ListBackups lists all backups for a tenant.
func (m *BackupManager) ListBackups(ctx context.Context, tenantID string, prefix string) ([]string, error) {
	searchPrefix := filepath.Join(tenantID, prefix)
	return m.storage.List(ctx, searchPrefix)
}

// DeleteBackup deletes a backup from storage.
func (m *BackupManager) DeleteBackup(ctx context.Context, key string) error {
	return m.storage.Delete(ctx, key)
}

// GetBackupSize returns the size of a backup.
func (m *BackupManager) GetBackupSize(ctx context.Context, key string) (int64, error) {
	return m.storage.GetSize(ctx, key)
}

// CreateStagingDir creates a staging directory for restore operations.
func (m *BackupManager) CreateStagingDir(backupID string) (string, error) {
	stagingDir := filepath.Join(m.config.TempDir, "restore", backupID)

	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return "", fmt.Errorf("%w: %v", ErrStagingFailed, err)
	}

	return stagingDir, nil
}

// CleanupStagingDir removes a staging directory.
func (m *BackupManager) CleanupStagingDir(stagingDir string) error {
	return os.RemoveAll(stagingDir)
}

// StorageStats holds storage statistics.
type StorageStats struct {
	TotalBackups   int   `json:"total_backups"`
	TotalSize      int64 `json:"total_size"`
	AvailableSpace int64 `json:"available_space"`
}

// GetStorageStats returns storage statistics.
func (m *BackupManager) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	var stats StorageStats

	// List all backups
	backups, err := m.storage.List(ctx, "")
	if err != nil {
		return nil, err
	}

	stats.TotalBackups = len(backups)

	// Calculate total size
	for _, backup := range backups {
		size, err := m.storage.GetSize(ctx, backup)
		if err == nil {
			stats.TotalSize += size
		}
	}

	// Get available space (local only)
	if local, ok := m.storage.(*LocalStorageProvider); ok {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(local.basePath, &stat); err == nil {
			stats.AvailableSpace = int64(stat.Bavail) * int64(stat.Bsize)
		}
	}

	return &stats, nil
}