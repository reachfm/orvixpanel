/**
 * Backup package error types.
 */

package backup

import "errors"

var (
	// ErrBackupNotFound is returned when a backup is not found.
	ErrBackupNotFound = errors.New("backup not found")

	// ErrBackupFailed is returned when a backup operation fails.
	ErrBackupFailed = errors.New("backup operation failed")

	// ErrRestoreFailed is returned when a restore operation fails.
	ErrRestoreFailed = errors.New("restore operation failed")

	// ErrInvalidBackend is returned when an invalid storage backend is specified.
	ErrInvalidBackend = errors.New("invalid storage backend")

	// ErrChecksumMismatch is returned when backup checksum verification fails.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrStorageUnavailable is returned when storage backend is unavailable.
	ErrStorageUnavailable = errors.New("storage backend unavailable")

	// ErrPermissionDenied is returned when permission is denied.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrStagingFailed is returned when staging directory creation fails.
	ErrStagingFailed = errors.New("staging directory creation failed")

	// ErrRollbackFailed is returned when rollback operation fails.
	ErrRollbackFailed = errors.New("rollback failed")

	// ErrRestoreInProgress is returned when a restore is already in progress.
	ErrRestoreInProgress = errors.New("restore already in progress")
)

// BackupError represents a detailed backup error.
type BackupError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *BackupError) Error() string {
	return e.Message
}

// NewBackupError creates a new backup error.
func NewBackupError(code, message, detail string) *BackupError {
	return &BackupError{Code: code, Message: message, Detail: detail}
}