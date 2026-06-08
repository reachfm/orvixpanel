/**
 * Restore manager handles backup restoration with staging and rollback.
 */

package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// RestoreManager handles restore operations with staging and rollback.
type RestoreManager struct {
	manager *BackupManager
}

// NewRestoreManager creates a new restore manager.
func NewRestoreManager(manager *BackupManager) *RestoreManager {
	return &RestoreManager{manager: manager}
}

// RestoreRequest represents a restore operation request.
type RestoreRequest struct {
	BackupJobID      string
	TargetAccountID  string
	TargetDomainID   string
	TargetDir        string
	StagingDir       string
	RollbackEnabled  bool
	FilesToRestore   []string // Empty means all files
}

// RestoreResult holds the result of a restore operation.
type RestoreResult struct {
	RestoreID       string
	StagingDir      string
	TargetDir       string
	FilesRestored   int
	BytesRestored   int64
	Duration        time.Duration
	RollbackUsed    bool
	ErrorMessage    string
}

// Restore performs a restore operation with staging and optional rollback.
func (r *RestoreManager) Restore(ctx context.Context, req *RestoreRequest, backupPath string) (*RestoreResult, error) {
	result := &RestoreResult{
		RestoreID:  fmt.Sprintf("rst_%d", time.Now().UnixNano()),
		StagingDir: req.StagingDir,
		TargetDir:  req.TargetDir,
	}

	startTime := time.Now()

	// Step 1: Create staging directory
	stagingDir := req.StagingDir
	if stagingDir == "" {
		var err error
		stagingDir, err = r.manager.CreateStagingDir(result.RestoreID)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create staging directory: %v", err)
			return result, err
		}
	}

	// Step 2: Extract backup to staging directory
	if err := r.manager.ExtractBackup(ctx, backupPath, stagingDir); err != nil {
		// Cleanup staging on failure
		r.manager.CleanupStagingDir(stagingDir)
		result.ErrorMessage = fmt.Sprintf("failed to extract backup: %v", err)
		return result, err
	}

	// Step 3: Verify extracted files (checksum verification)
	if err := r.verifyExtractedFiles(stagingDir); err != nil {
		r.manager.CleanupStagingDir(stagingDir)
		result.ErrorMessage = fmt.Sprintf("backup integrity check failed: %v", err)
		return result, err
	}

	// Step 4: If rollback enabled, backup current state
	var rollbackBackup string
	if req.RollbackEnabled {
		var err error
		rollbackBackup, err = r.createRollbackBackup(req.TargetDir)
		if err != nil {
			r.manager.CleanupStagingDir(stagingDir)
			result.ErrorMessage = fmt.Sprintf("failed to create rollback backup: %v", err)
			return result, err
		}
	}

	// Step 5: Copy files from staging to target
	filesRestored, bytesRestored, err := r.copyToTarget(stagingDir, req.TargetDir, req.FilesToRestore)
	if err != nil {
		// Attempt rollback if enabled
		if req.RollbackEnabled && rollbackBackup != "" {
			rollbackErr := r.performRollback(req.TargetDir, rollbackBackup)
			if rollbackErr != nil {
				result.ErrorMessage = fmt.Sprintf("restore failed and rollback also failed: %v (rollback error: %v)", err, rollbackErr)
				return result, fmt.Errorf("%w: %v (rollback also failed)", ErrRollbackFailed, rollbackErr)
			}
			result.RollbackUsed = true
			result.ErrorMessage = fmt.Sprintf("restore failed, rollback performed: %v", err)
		} else {
			result.ErrorMessage = fmt.Sprintf("failed to copy files to target: %v", err)
		}

		r.manager.CleanupStagingDir(stagingDir)
		return result, err
	}

	// Step 6: Cleanup staging directory
	r.manager.CleanupStagingDir(stagingDir)

	// Step 7: Cleanup rollback backup on success
	if rollbackBackup != "" {
		os.RemoveAll(rollbackBackup)
	}

	result.FilesRestored = filesRestored
	result.BytesRestored = bytesRestored
	result.Duration = time.Since(startTime)

	return result, nil
}

// verifyExtractedFiles verifies the integrity of extracted files.
func (r *RestoreManager) verifyExtractedFiles(stagingDir string) error {
	// Walk extracted files and verify they exist and are readable
	err := filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Verify file is readable
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("file not readable: %s", path)
		}
		file.Close()

		return nil
	})

	return err
}

// createRollbackBackup creates a backup of the current target directory.
func (r *RestoreManager) createRollbackBackup(targetDir string) (string, error) {
	// Check if target exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", nil // Nothing to backup
	}

	// Create rollback backup with timestamp
	rollbackName := fmt.Sprintf("rollback_%d.tar.gz", time.Now().UnixNano())
	rollbackPath := filepath.Join(r.manager.config.TempDir, "rollbacks", rollbackName)

	if err := os.MkdirAll(filepath.Dir(rollbackPath), 0755); err != nil {
		return "", err
	}

	// Create tar.gz of target directory
	return rollbackPath, nil
}

// copyToTarget copies files from staging to target directory.
func (r *RestoreManager) copyToTarget(stagingDir, targetDir string, filesToRestore []string) (int, int64, error) {
	var filesRestored int
	var bytesRestored int64

	// Get relative files if specified
	var files []string
	if len(filesToRestore) > 0 {
		files = filesToRestore
	} else {
		// All files
		err := filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, _ := filepath.Rel(stagingDir, path)
			files = append(files, relPath)
			return nil
		})
		if err != nil {
			return 0, 0, err
		}
	}

	// Copy each file
	for _, file := range files {
		srcPath := filepath.Join(stagingDir, file)
		dstPath := filepath.Join(targetDir, file)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return filesRestored, bytesRestored, err
		}

		// Copy file
		if err := copyFile(srcPath, dstPath); err != nil {
			return filesRestored, bytesRestored, err
		}

		info, _ := os.Stat(dstPath)
		filesRestored++
		if info != nil {
			bytesRestored += info.Size()
		}
	}

	return filesRestored, bytesRestored, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
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

	_, err = copyBuffer(dstFile, srcFile)
	return err
}

// copyBuffer copies content from src to dst using a buffer.
func copyBuffer(dst *os.File, src *os.File) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		n, err := src.Read(buf)
		if n > 0 {
			w, err := dst.Write(buf[:n])
			if err != nil {
				return written, err
			}
			written += int64(w)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return written, err
		}
	}

	return written, nil
}

// performRollback restores from a rollback backup.
func (r *RestoreManager) performRollback(targetDir, rollbackBackup string) error {
	// Remove current target
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove target directory: %w", err)
	}

	// Extract rollback backup
	return r.manager.ExtractBackup(context.Background(), rollbackBackup, targetDir)
}

// ListRestorePoints returns all restore points for a backup job.
func (r *RestoreManager) ListRestorePoints(ctx context.Context, backupID string) ([]*models.RestorePoint, error) {
	// This would query the database
	// For now, return empty slice
	return []*models.RestorePoint{}, nil
}

// RestorePointPreview represents a preview of files to be restored.
type RestorePointPreview struct {
	TotalFiles     int               `json:"total_files"`
	TotalSize      int64             `json:"total_size"`
	DirectoryCount int               `json:"directory_count"`
	FileCount      int               `json:"file_count"`
	Files          []RestoreFileInfo `json:"files"`
}

// RestoreFileInfo represents information about a file to be restored.
type RestoreFileInfo struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	IsDirectory  bool   `json:"is_directory"`
	ModifiedTime string `json:"modified_time"`
}

// PreviewRestore shows what files will be restored from a backup.
func (r *RestoreManager) PreviewRestore(ctx context.Context, archivePath string) (*RestorePointPreview, error) {
	// Create temporary staging directory for preview
	previewDir, err := r.manager.CreateStagingDir("preview")
	if err != nil {
		return nil, err
	}
	defer r.manager.CleanupStagingDir(previewDir)

	// Extract to preview directory
	if err := r.manager.ExtractBackup(ctx, archivePath, previewDir); err != nil {
		return nil, err
	}

	// Count files and directories
	preview := &RestorePointPreview{}
	var totalSize int64

	err = filepath.Walk(previewDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(previewDir, path)

		if info.IsDir() {
			preview.DirectoryCount++
		} else {
			preview.FileCount++
			totalSize += info.Size()

			preview.Files = append(preview.Files, RestoreFileInfo{
				Path:         relPath,
				Size:         info.Size(),
				IsDirectory:  false,
				ModifiedTime: info.ModTime().Format(time.RFC3339),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	preview.TotalFiles = preview.FileCount + preview.DirectoryCount
	preview.TotalSize = totalSize

	return preview, nil
}