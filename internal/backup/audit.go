/**
 * Backup audit events.
 */

package backup

import (
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// AuditEventType defines the type of audit event.
type AuditEventType string

const (
	// Backup events
	AuditBackupCreated   AuditEventType = "backup_created"
	AuditBackupStarted   AuditEventType = "backup_started"
	AuditBackupCompleted AuditEventType = "backup_completed"
	AuditBackupFailed    AuditEventType = "backup_failed"
	AuditBackupDeleted   AuditEventType = "backup_deleted"

	// Restore events
	AuditRestoreStarted  AuditEventType = "restore_started"
	AuditRestoreCompleted AuditEventType = "restore_completed"
	AuditRestoreFailed    AuditEventType = "restore_failed"
	AuditRestoreRolledBack AuditEventType = "restore_rolled_back"

	// Schedule events
	AuditScheduleCreated AuditEventType = "schedule_created"
	AuditScheduleDeleted AuditEventType = "schedule_deleted"
)

// BackupAuditEntry represents a backup audit log entry.
type BackupAuditEntry struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	EventType    AuditEventType `json:"event_type"`
	BackupID     string         `json:"backup_id,omitempty"`
	RestoreID    string         `json:"restore_id,omitempty"`
	ScheduleID   string         `json:"schedule_id,omitempty"`
	UserID       string         `json:"user_id"`
	Message      string         `json:"message"`
	Details      string         `json:"details,omitempty"`
	IPAddress    string         `json:"ip_address,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	Success      bool           `json:"success"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ToAuditEntry converts a BackupAuditEntry to a models.AuditEntry.
func (e *BackupAuditEntry) ToAuditEntry() *models.AuditEntry {
	entityID := e.BackupID
	if entityID == "" {
		entityID = e.RestoreID
	}
	if entityID == "" {
		entityID = e.ScheduleID
	}

	var result string
	if e.Success {
		result = "success"
	} else {
		result = "failure"
	}

	return &models.AuditEntry{
		Action:       string(e.EventType),
		ResourceType: "backup",
		ResourceID:   entityID,
		UserID:       e.UserID,
		ActorIP:      e.IPAddress,
		Result:       result,
		Detail:       e.Message,
	}
}

// CreateAuditEntry creates a new audit entry for backup operations.
func CreateAuditEntry(tenantID, userID string, eventType AuditEventType, message string, success bool) *BackupAuditEntry {
	return &BackupAuditEntry{
		ID:        generateID("aud"),
		TenantID:  tenantID,
		EventType: eventType,
		UserID:    userID,
		Message:   message,
		Success:   success,
		CreatedAt: time.Now(),
	}
}

// WithBackupID sets the backup ID for the audit entry.
func (e *BackupAuditEntry) WithBackupID(backupID string) *BackupAuditEntry {
	e.BackupID = backupID
	return e
}

// WithRestoreID sets the restore ID for the audit entry.
func (e *BackupAuditEntry) WithRestoreID(restoreID string) *BackupAuditEntry {
	e.RestoreID = restoreID
	return e
}

// WithScheduleID sets the schedule ID for the audit entry.
func (e *BackupAuditEntry) WithScheduleID(scheduleID string) *BackupAuditEntry {
	e.ScheduleID = scheduleID
	return e
}

// WithDetails sets the details for the audit entry.
func (e *BackupAuditEntry) WithDetails(details string) *BackupAuditEntry {
	e.Details = details
	return e
}

// WithClientInfo sets client information for the audit entry.
func (e *BackupAuditEntry) WithClientInfo(ipAddress, userAgent string) *BackupAuditEntry {
	e.IPAddress = ipAddress
	e.UserAgent = userAgent
	return e
}