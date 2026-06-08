/**
 * Backup scheduler handles automated backup execution and retention.
 */

package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// Scheduler handles scheduled backup jobs.
type Scheduler struct {
	manager  *BackupManager
	db       *gorm.DB
	stopCh   chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewScheduler creates a new backup scheduler.
func NewScheduler(manager *BackupManager, database *gorm.DB) *Scheduler {
	return &Scheduler{
		manager: manager,
		db:      database,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.running = true
	s.wg.Add(1)
	go s.run()

	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.wg.Wait()
	s.running = false
}

// run is the main scheduler loop.
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	s.runScheduledBackups()

	for {
		select {
		case <-ticker.C:
			s.runScheduledBackups()
		case <-s.stopCh:
			return
		}
	}
}

// runScheduledBackups runs all due scheduled backups.
func (s *Scheduler) runScheduledBackups() {
	// Get all enabled schedules that are due
	now := time.Now()
	var schedules []models.BackupSchedule

	// Query schedules that should run now
	s.db.Where("is_enabled = ? AND next_run_at <= ?", true, now).Find(&schedules)

	for _, schedule := range schedules {
		s.runBackup(schedule)
	}
}

// runBackup executes a single scheduled backup.
func (s *Scheduler) runBackup(schedule models.BackupSchedule) {
	log.Printf("[backup] Running scheduled backup: %s (ID: %s)", schedule.Name, schedule.ID)

	// Create backup job
	job := &models.BackupJob{
		ID:             generateID("bkp"),
		TenantID:       schedule.TenantID,
		AccountID:      schedule.AccountID,
		DomainID:       schedule.DomainID,
		Type:           schedule.BackupType,
		Status:         models.BackupStatusPending,
		Name:           schedule.Name,
		ScheduledAt:    schedule.NextRunAt,
		StorageBackend:  schedule.StorageBackend,
		RetentionDays:   schedule.RetentionDays,
		CreatedBy:      schedule.CreatedBy,
	}

	// Calculate next run time
	nextRun := calculateNextRun(schedule.CronExpr)
	schedule.LastRunAt = timePtr(time.Now())
	schedule.NextRunAt = timePtr(nextRun)

	// Update schedule
	s.db.Save(&schedule)

	// Execute backup
	s.executeBackup(job, schedule)
}

// executeBackup performs the actual backup operation.
func (s *Scheduler) executeBackup(job *models.BackupJob, schedule models.BackupSchedule) {
	ctx := context.Background()

	now := time.Now()
	job.StartedAt = timePtr(now)
	job.Status = models.BackupStatusRunning

	// Save job to database
	s.db.Create(job)

	// Determine source directory based on backup type
	var sourceDir string
	switch schedule.BackupType {
	case models.BackupTypeFiles:
		sourceDir = fmt.Sprintf("/var/lib/orvixpanel/domains/%s/public", schedule.DomainID)
	case models.BackupTypeDatabase:
		sourceDir = fmt.Sprintf("/var/lib/orvixpanel/databases/%s", schedule.DomainID)
	case models.BackupTypeFull:
		sourceDir = fmt.Sprintf("/var/lib/orvixpanel/accounts/%s", schedule.AccountID)
	default:
		sourceDir = fmt.Sprintf("/var/lib/orvixpanel/domains/%s", schedule.DomainID)
	}

	// Create backup
	result, err := s.manager.CreateFileBackup(ctx, job, sourceDir)

	if err != nil {
		job.Status = models.BackupStatusFailed
		job.ErrorMessage = err.Error()
		job.CompletedAt = timePtr(time.Now())
		s.db.Save(job)
		log.Printf("[backup] Backup failed: %s - %v", job.ID, err)
		return
	}

	// Upload to storage
	remoteKey := fmt.Sprintf("%s/%s/%s.tar.gz", job.TenantID, schedule.BackupType, job.ID)
	if err := s.manager.UploadBackup(ctx, result.ArchivePath, remoteKey); err != nil {
		job.Status = models.BackupStatusFailed
		job.ErrorMessage = fmt.Sprintf("upload failed: %v", err)
		job.CompletedAt = timePtr(time.Now())
		s.db.Save(job)
		log.Printf("[backup] Backup upload failed: %s - %v", job.ID, err)
		return
	}

	// Update job with success info
	job.Status = models.BackupStatusCompleted
	job.StoragePath = remoteKey
	job.FileCount = result.FileCount
	job.FileSize = result.TotalSize
	job.Checksum = result.Checksum
	job.ChecksumAlgo = result.ChecksumAlgo
	job.CompletedAt = timePtr(time.Now())

	// Set expiration based on retention
	expiresAt := now.AddDate(0, 0, schedule.RetentionDays)
	job.ExpiresAt = timePtr(expiresAt)

	s.db.Save(job)

	// Cleanup temp file
	os.Remove(result.ArchivePath)

	log.Printf("[backup] Backup completed: %s (%d files, %d bytes)", job.ID, result.FileCount, result.TotalSize)
}

// calculateNextRun calculates the next run time from a cron expression.
func calculateNextRun(cronExpr string) time.Time {
	// Simple implementation: daily at midnight
	// In production, use robfig/cron for full cron support
	next := time.Now().Add(24 * time.Hour)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	return next
}

// CleanupExpiredBackups removes backups that have exceeded their retention period.
func (s *Scheduler) CleanupExpiredBackups() error {
	ctx := context.Background()

	var expiredJobs []models.BackupJob
	now := time.Now()

	// Find expired backups
	if err := s.db.Where("expires_at <= ? AND status = ?", now, models.BackupStatusCompleted).Find(&expiredJobs).Error; err != nil {
		return fmt.Errorf("failed to query expired backups: %w", err)
	}

	log.Printf("[backup] Found %d expired backups to clean up", len(expiredJobs))

	for _, job := range expiredJobs {
		// Delete from storage
		if err := s.manager.DeleteBackup(ctx, job.StoragePath); err != nil {
			log.Printf("[backup] Failed to delete storage for %s: %v", job.ID, err)
		}

		// Delete database record
		s.db.Delete(&job)

		log.Printf("[backup] Cleaned up expired backup: %s", job.ID)
	}

	return nil
}

// GetBackupStats returns statistics about backups.
type BackupStats struct {
	TotalBackups    int64 `json:"total_backups"`
	ActiveBackups   int `json:"active_backups"`
	ExpiredBackups  int `json:"expired_backups"`
	FailedBackups   int `json:"failed_backups"`
	TotalStorageMB  int64 `json:"total_storage_mb"`
}

// GetBackupStats returns backup statistics.
func (s *Scheduler) GetBackupStats() (*BackupStats, error) {
	var stats BackupStats

	// Count by status
	s.db.Model(&models.BackupJob{}).Group("status").Count(&stats.TotalBackups)

	var active, expired, failed int64
	s.db.Model(&models.BackupJob{}).Where("status = ?", models.BackupStatusCompleted).Count(&active)
	s.db.Model(&models.BackupJob{}).Where("status = ? AND expires_at <= ?", models.BackupStatusCompleted, time.Now()).Count(&expired)
	s.db.Model(&models.BackupJob{}).Where("status = ?", models.BackupStatusFailed).Count(&failed)

	stats.ActiveBackups = int(active)
	stats.ExpiredBackups = int(expired)
	stats.FailedBackups = int(failed)

	// Sum file sizes
	var totalSize int64
	s.db.Model(&models.BackupJob{}).Where("status = ?", models.BackupStatusCompleted).Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize)
	stats.TotalStorageMB = totalSize / (1024 * 1024)

	return &stats, nil
}

// timePtr returns a pointer to a time value.
func timePtr(t time.Time) *time.Time {
	return &t
}

// generateID generates a unique ID for backup jobs.
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}