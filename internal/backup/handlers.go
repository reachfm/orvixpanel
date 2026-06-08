/**
 * Backup API handlers.
 */

package backup

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// Handler handles backup API requests.
type Handler struct {
	manager   *BackupManager
	restore   *RestoreManager
	scheduler *Scheduler
	db        *gorm.DB
}

// NewHandler creates a new backup handler.
func NewHandler(database *gorm.DB) *Handler {
	config := DefaultConfig()
	manager := NewBackupManager(config)
	restore := NewRestoreManager(manager)
	scheduler := NewScheduler(manager, database)

	return &Handler{
		manager:   manager,
		restore:   restore,
		scheduler: scheduler,
		db:        database,
	}
}

// StartScheduler starts the backup scheduler.
func (h *Handler) StartScheduler() error {
	return h.scheduler.Start()
}

// StopScheduler stops the backup scheduler.
func (h *Handler) StopScheduler() {
	h.scheduler.Stop()
}

// ListBackupsResponse represents the response for listing backups.
type ListBackupsResponse struct {
	Backups    []models.BackupJob `json:"backups"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ListBackups handles GET /api/v1/backups
func (h *Handler) ListBackups(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)

	// Parse pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Query backups
	var backups []models.BackupJob
	var total int64

	query := h.db.Model(&models.BackupJob{}).Where("tenant_id = ?", tenantID)

	// Apply filters
	if backupType := c.QueryParam("type"); backupType != "" {
		query = query.Where("type = ?", backupType)
	}
	if status := c.QueryParam("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&backups).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "backup_list_failed",
			"message": "Failed to list backups",
		})
	}

	return c.JSON(http.StatusOK, ListBackupsResponse{
		Backups:    backups,
		Total:      int(total),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	})
}

// GetBackup handles GET /api/v1/backups/:id
func (h *Handler) GetBackup(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	backupID := c.Param("id")

	var backup models.BackupJob
	if err := h.db.Where("id = ? AND tenant_id = ?", backupID, tenantID).First(&backup).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "backup_not_found",
			"message": "Backup not found",
		})
	}

	return c.JSON(http.StatusOK, backup)
}

// CreateBackupRequest represents a backup creation request.
type CreateBackupRequest struct {
	AccountID      string            `json:"account_id"`
	DomainID       string            `json:"domain_id"`
	Type           models.BackupType `json:"type"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	StorageBackend string            `json:"storage_backend"`
	RetentionDays  int               `json:"retention_days"`
	SourcePath     string            `json:"source_path"`
}

// CreateBackup handles POST /api/v1/backups
func (h *Handler) CreateBackup(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	userID := c.Get("user_id").(string)

	var req CreateBackupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_body",
			"message": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "type_required",
			"message": "Backup type is required",
		})
	}

	// Create backup job
	job := &models.BackupJob{
		ID:            generateID("bkp"),
		TenantID:      tenantID,
		AccountID:     req.AccountID,
		DomainID:      req.DomainID,
		Type:          req.Type,
		Status:        models.BackupStatusPending,
		Name:          req.Name,
		Description:   req.Description,
		StorageBackend: req.StorageBackend,
		RetentionDays: req.RetentionDays,
		CreatedBy:     userID,
	}

	if job.RetentionDays == 0 {
		job.RetentionDays = 30
	}

	// Save job
	h.db.Create(job)

	// Start async backup (in production, use a job queue)
	go h.runBackupAsync(job, req.SourcePath)

	return c.JSON(http.StatusCreated, job)
}

// runBackupAsync runs a backup asynchronously.
func (h *Handler) runBackupAsync(job *models.BackupJob, sourcePath string) {
	// This would be handled by the scheduler or a job queue in production
}

// GetBackupFiles handles GET /api/v1/backups/:id/files
func (h *Handler) GetBackupFiles(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	backupID := c.Param("id")

	// Verify backup exists and belongs to tenant
	var backup models.BackupJob
	if err := h.db.Where("id = ? AND tenant_id = ?", backupID, tenantID).First(&backup).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "backup_not_found",
			"message": "Backup not found",
		})
	}

	// Get files for backup
	var files []models.BackupFile
	h.db.Where("backup_job_id = ?", backupID).Find(&files)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backup_id": backupID,
		"files":     files,
	})
}

// RestoreBackupRequest represents a restore request.
type RestoreBackupRequest struct {
	TargetDir       string   `json:"target_dir"`
	FilesToRestore  []string `json:"files_to_restore"`
	RollbackEnabled bool     `json:"rollback_enabled"`
}

// RestoreBackup handles POST /api/v1/backups/:id/restore
func (h *Handler) RestoreBackup(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	userID := c.Get("user_id").(string)
	backupID := c.Param("id")

	var req RestoreBackupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_body",
			"message": "Invalid request body",
		})
	}

	// Get backup
	var backup models.BackupJob
	if err := h.db.Where("id = ? AND tenant_id = ?", backupID, tenantID).First(&backup).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "backup_not_found",
			"message": "Backup not found",
		})
	}

	// Create restore point record
	restorePoint := &models.RestorePoint{
		ID:              generateID("rst"),
		TenantID:        tenantID,
		AccountID:       backup.AccountID,
		DomainID:        backup.DomainID,
		BackupJobID:     backupID,
		Status:          models.BackupStatusPending,
		TargetDir:       req.TargetDir,
		RollbackEnabled: req.RollbackEnabled,
		CreatedBy:       userID,
	}

	// Create staging directory
	stagingDir, err := h.manager.CreateStagingDir(restorePoint.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "staging_failed",
			"message": "Failed to create staging directory",
		})
	}
	restorePoint.StagingDir = stagingDir

	h.db.Create(restorePoint)

	// Start restore async
	go h.runRestoreAsync(restorePoint, backup.StoragePath, req)

	return c.JSON(http.StatusAccepted, restorePoint)
}

// runRestoreAsync runs a restore operation asynchronously.
func (h *Handler) runRestoreAsync(restorePoint *models.RestorePoint, storagePath string, req RestoreBackupRequest) {
	// Update status to running
	restorePoint.Status = models.BackupStatusRunning
	now := time.Now()
	restorePoint.StartedAt = &now
	h.db.Save(restorePoint)

	// Perform restore
	restoreReq := &RestoreRequest{
		BackupJobID:     restorePoint.BackupJobID,
		TargetAccountID: restorePoint.AccountID,
		TargetDomainID:  restorePoint.DomainID,
		TargetDir:       req.TargetDir,
		StagingDir:      restorePoint.StagingDir,
		RollbackEnabled: req.RollbackEnabled,
		FilesToRestore:  req.FilesToRestore,
	}

	// Get backup file from storage
	ctx := context.Background()
	reader, err := h.manager.storage.Download(ctx, storagePath)
	if err != nil {
		restorePoint.Status = models.BackupStatusFailed
		restorePoint.ErrorMessage = err.Error()
		h.db.Save(restorePoint)
		return
	}
	defer reader.Close()

	// Save to temp file for restore
	tempPath := "/tmp/" + restorePoint.ID + ".tar.gz"
	file, _ := os.Create(tempPath)
	io.Copy(file, reader)
	file.Close()

	result, err := h.restore.Restore(ctx, restoreReq, tempPath)
	os.Remove(tempPath)

	if err != nil {
		restorePoint.Status = models.BackupStatusFailed
		restorePoint.ErrorMessage = err.Error()
	} else {
		restorePoint.Status = models.BackupStatusCompleted
		restorePoint.FilesRestored = result.FilesRestored
		restorePoint.BytesRestored = result.BytesRestored
		restorePoint.RollbackUsed = result.RollbackUsed
	}
	restorePoint.CompletedAt = &now
	h.db.Save(restorePoint)
}

// GetRestorePoints handles GET /api/v1/backups/:id/restores
func (h *Handler) GetRestorePoints(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	backupID := c.Param("id")

	var restorePoints []models.RestorePoint
	h.db.Where("backup_job_id = ? AND tenant_id = ?", backupID, tenantID).Order("created_at DESC").Find(&restorePoints)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backup_id":     backupID,
		"restore_points": restorePoints,
	})
}

// DeleteBackup handles DELETE /api/v1/backups/:id
func (h *Handler) DeleteBackup(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	backupID := c.Param("id")

	var backup models.BackupJob
	if err := h.db.Where("id = ? AND tenant_id = ?", backupID, tenantID).First(&backup).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "backup_not_found",
			"message": "Backup not found",
		})
	}

	// Delete from storage
	ctx := context.Background()
	if err := h.manager.DeleteBackup(ctx, backup.StoragePath); err != nil {
		// Log but continue with database deletion
	}

	// Delete files
	h.db.Where("backup_job_id = ?", backupID).Delete(&models.BackupFile{})

	// Delete backup record
	h.db.Delete(&backup)

	return c.NoContent(http.StatusNoContent)
}

// GetBackupStats handles GET /api/v1/backups/stats
func (h *Handler) GetBackupStats(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)

	var stats struct {
		TotalBackups    int64 `json:"total_backups"`
		ActiveBackups   int64 `json:"active_backups"`
		FailedBackups   int64 `json:"failed_backups"`
		TotalStorageMB  int64 `json:"total_storage_mb"`
	}

	// Count total backups
	h.db.Model(&models.BackupJob{}).Where("tenant_id = ?", tenantID).Count(&stats.TotalBackups)

	// Count by status
	h.db.Model(&models.BackupJob{}).Where("tenant_id = ? AND status = ?", tenantID, models.BackupStatusCompleted).Count(&stats.ActiveBackups)
	h.db.Model(&models.BackupJob{}).Where("tenant_id = ? AND status = ?", tenantID, models.BackupStatusFailed).Count(&stats.FailedBackups)

	// Sum storage
	var totalSize int64
	h.db.Model(&models.BackupJob{}).Where("tenant_id = ? AND status = ?", tenantID, models.BackupStatusCompleted).Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize)
	stats.TotalStorageMB = totalSize / (1024 * 1024)

	return c.JSON(http.StatusOK, stats)
}

// CreateScheduleRequest represents a schedule creation request.
type CreateScheduleRequest struct {
	AccountID     string            `json:"account_id"`
	DomainID      string            `json:"domain_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	BackupType    models.BackupType `json:"backup_type"`
	CronExpr      string            `json:"cron_expr"`
	RetentionDays int               `json:"retention_days"`
	StorageBackend string           `json:"storage_backend"`
}

// CreateSchedule handles POST /api/v1/backups/schedules
func (h *Handler) CreateSchedule(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	userID := c.Get("user_id").(string)

	var req CreateScheduleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_body",
			"message": "Invalid request body",
		})
	}

	// Validate
	if req.Name == "" || req.CronExpr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_body",
			"message": "Name and cron expression are required",
		})
	}

	// Create schedule
	schedule := &models.BackupSchedule{
		ID:             generateID("sch"),
		TenantID:       tenantID,
		AccountID:      req.AccountID,
		DomainID:       req.DomainID,
		Name:           req.Name,
		Description:    req.Description,
		BackupType:     req.BackupType,
		CronExpr:       req.CronExpr,
		RetentionDays:  req.RetentionDays,
		StorageBackend: req.StorageBackend,
		IsEnabled:      true,
		CreatedBy:      userID,
	}

	if schedule.RetentionDays == 0 {
		schedule.RetentionDays = 30
	}

	// Calculate next run
	nextRun := calculateNextRun(schedule.CronExpr)
	schedule.NextRunAt = &nextRun

	h.db.Create(schedule)

	return c.JSON(http.StatusCreated, schedule)
}

// ListSchedules handles GET /api/v1/backups/schedules
func (h *Handler) ListSchedules(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)

	var schedules []models.BackupSchedule
	h.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&schedules)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"schedules": schedules,
	})
}

// DeleteSchedule handles DELETE /api/v1/backups/schedules/:id
func (h *Handler) DeleteSchedule(c echo.Context) error {
	tenantID := c.Get("tenant_id").(string)
	scheduleID := c.Param("id")

	var schedule models.BackupSchedule
	if err := h.db.Where("id = ? AND tenant_id = ?", scheduleID, tenantID).First(&schedule).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error":   "schedule_not_found",
			"message": "Schedule not found",
		})
	}

	h.db.Delete(&schedule)

	return c.NoContent(http.StatusNoContent)
}