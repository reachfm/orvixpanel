/**
 * Backup database models.
 */

package models

import (
	"time"

	"gorm.io/gorm"
)

// BackupStatus represents the status of a backup job.
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusCanceled  BackupStatus = "canceled"
)

// BackupType represents the type of backup.
type BackupType string

const (
	BackupTypeFull     BackupType = "full"
	BackupTypeFiles    BackupType = "files"
	BackupTypeDatabase BackupType = "database"
)

// BackupJob represents a backup job record.
type BackupJob struct {
	ID             string         `gorm:"primaryKey;size:26" json:"id"`
	TenantID       string         `gorm:"size:26;index;not null" json:"tenant_id"`
	AccountID      string         `gorm:"size:26;index" json:"account_id"`
	DomainID       string         `gorm:"size:26;index" json:"domain_id"`
	Type           BackupType     `gorm:"size:20;not null" json:"type"`
	Status         BackupStatus   `gorm:"size:20;not null;index" json:"status"`
	Name           string         `gorm:"size:255" json:"name"`
	Description    string         `gorm:"size:500" json:"description"`
	StorageBackend string         `gorm:"size:50;default:local" json:"storage_backend"`
	StoragePath    string         `gorm:"size:500" json:"storage_path"`
	FileSize       int64          `gorm:"default:0" json:"file_size"`
	FileCount      int            `gorm:"default:0" json:"file_count"`
	Checksum       string         `gorm:"size:128" json:"checksum"`
	ChecksumAlgo   string         `gorm:"size:20;default:sha256" json:"checksum_algo"`
	RetentionDays  int            `gorm:"default:30" json:"retention_days"`
	ExpiresAt      *time.Time     `gorm:"index" json:"expires_at"`
	ErrorMessage   string         `gorm:"size:1000" json:"error_message,omitempty"`
	ScheduledAt    *time.Time     `json:"scheduled_at,omitempty"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	CreatedBy      string         `gorm:"size:26" json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for BackupJob.
func (BackupJob) TableName() string {
	return "backup_jobs"
}

// BackupFile represents a file within a backup.
type BackupFile struct {
	ID           string         `gorm:"primaryKey;size:26" json:"id"`
	BackupJobID  string         `gorm:"size:26;index;not null" json:"backup_job_id"`
	OriginalPath string         `gorm:"size:1000;not null" json:"original_path"`
	ArchivePath  string         `gorm:"size:1000;not null" json:"archive_path"`
	Size         int64          `gorm:"default:0" json:"size"`
	Checksum     string         `gorm:"size:128" json:"checksum"`
	IsDirectory  bool           `gorm:"default:false" json:"is_directory"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for BackupFile.
func (BackupFile) TableName() string {
	return "backup_files"
}

// RestorePoint represents a restore operation record.
type RestorePoint struct {
	ID              string         `gorm:"primaryKey;size:26" json:"id"`
	TenantID        string         `gorm:"size:26;index;not null" json:"tenant_id"`
	AccountID       string         `gorm:"size:26;index" json:"account_id"`
	DomainID        string         `gorm:"size:26;index" json:"domain_id"`
	BackupJobID     string         `gorm:"size:26;index;not null" json:"backup_job_id"`
	Status          BackupStatus   `gorm:"size:20;not null" json:"status"`
	StagingDir      string         `gorm:"size:500" json:"staging_dir"`
	TargetDir       string         `gorm:"size:500" json:"target_dir"`
	FilesRestored   int            `gorm:"default:0" json:"files_restored"`
	BytesRestored   int64          `gorm:"default:0" json:"bytes_restored"`
	RollbackEnabled bool           `gorm:"default:true" json:"rollback_enabled"`
	RollbackUsed    bool           `gorm:"default:false" json:"rollback_used"`
	ErrorMessage    string         `gorm:"size:1000" json:"error_message,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	CreatedBy       string         `gorm:"size:26" json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for RestorePoint.
func (RestorePoint) TableName() string {
	return "restore_points"
}

// BackupConfig represents storage backend configuration.
type BackupConfig struct {
	ID             string    `gorm:"primaryKey;size:26" json:"id"`
	TenantID       string    `gorm:"size:26;index;not null" json:"tenant_id"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	Backend        string    `gorm:"size:50;not null" json:"backend"` // local, s3, minio, wasabi
	Endpoint       string    `gorm:"size:255" json:"endpoint"`
	Bucket         string    `gorm:"size:255" json:"bucket"`
	Region         string    `gorm:"size:50" json:"region"`
	AccessKeyID    string    `gorm:"size:255" json:"access_key_id,omitempty"`
	SecretAccessKey string   `gorm:"size:500" json:"secret_access_key,omitempty"` // encrypted
	PathPrefix     string    `gorm:"size:255;default:backups" json:"path_prefix"`
	IsDefault      bool      `gorm:"default:false" json:"is_default"`
	RetentionDays  int       `gorm:"default:30" json:"retention_days"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for BackupConfig.
func (BackupConfig) TableName() string {
	return "backup_configs"
}

// BackupSchedule represents a scheduled backup job.
type BackupSchedule struct {
	ID           string         `gorm:"primaryKey;size:26" json:"id"`
	TenantID     string         `gorm:"size:26;index;not null" json:"tenant_id"`
	AccountID    string         `gorm:"size:26;index" json:"account_id"`
	DomainID     string         `gorm:"size:26;index" json:"domain_id"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	Description  string         `gorm:"size:500" json:"description"`
	BackupType   BackupType     `gorm:"size:20;not null" json:"backup_type"`
	CronExpr     string         `gorm:"size:100;not null" json:"cron_expr"`
	RetentionDays int           `gorm:"default:30" json:"retention_days"`
	StorageBackend string       `gorm:"size:50;default:local" json:"storage_backend"`
	IsEnabled    bool           `gorm:"default:true" json:"is_enabled"`
	LastRunAt    *time.Time     `json:"last_run_at,omitempty"`
	NextRunAt    *time.Time     `json:"next_run_at,omitempty"`
	CreatedBy    string         `gorm:"size:26" json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for BackupSchedule.
func (BackupSchedule) TableName() string {
	return "backup_schedules"
}