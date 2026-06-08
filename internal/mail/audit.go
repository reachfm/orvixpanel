package mail

import (
	"context"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// AuditManager handles audit logging for mail operations
type AuditManager struct {
	db *gorm.DB
}

// NewAuditManager creates a new audit manager
func NewAuditManager(db *gorm.DB) *AuditManager {
	return &AuditManager{db: db}
}

// MailAuditAction represents mail audit action types
type MailAuditAction string

const (
	ActionMailSent          MailAuditAction = "sent"
	ActionMailReceived      MailAuditAction = "received"
	ActionMailLogin         MailAuditAction = "login"
	ActionMailFailedLogin   MailAuditAction = "failed_login"
	ActionMailQuotaExceeded MailAuditAction = "quota_exceeded"
	ActionMailBounced       MailAuditAction = "bounced"
	ActionMailRejected      MailAuditAction = "rejected"
	ActionMailDeferred      MailAuditAction = "deferred"
)

// MailDirection represents mail direction
type MailDirection string

const (
	DirectionInbound  MailDirection = "inbound"
	DirectionOutbound MailDirection = "outbound"
)

// CreateAuditLog creates a new audit log entry
func (m *AuditManager) CreateAuditLog(ctx context.Context, log *models.MailAuditLog) error {
	if log.ID == "" {
		log.ID = generateID("mal")
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	if err := m.db.Create(log).Error; err != nil {
		return NewMailError("CreateAuditLog", err, "database error")
	}

	return nil
}

// LogMailSent logs a sent mail event
func (m *AuditManager) LogMailSent(ctx context.Context, tenantID, mailboxID, from, to, subject, messageID string, sizeBytes int64) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		MailboxID: mailboxID,
		Action:    string(ActionMailSent),
		Direction: string(DirectionOutbound),
		FromEmail: from,
		ToEmail:   to,
		Subject:   subject,
		MessageID: messageID,
		SizeBytes: sizeBytes,
		Status:    "sent",
	}
	return m.CreateAuditLog(ctx, log)
}

// LogMailReceived logs a received mail event
func (m *AuditManager) LogMailReceived(ctx context.Context, tenantID, mailboxID, from, to, subject, messageID string, sizeBytes int64, remoteIP string) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		MailboxID: mailboxID,
		Action:    string(ActionMailReceived),
		Direction: string(DirectionInbound),
		FromEmail: from,
		ToEmail:   to,
		Subject:   subject,
		MessageID: messageID,
		SizeBytes: sizeBytes,
		Status:    "delivered",
		RemoteIP:  remoteIP,
	}
	return m.CreateAuditLog(ctx, log)
}

// LogLogin logs a successful login
func (m *AuditManager) LogLogin(ctx context.Context, tenantID, mailboxID, remoteIP, userAgent string) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		MailboxID: mailboxID,
		Action:    string(ActionMailLogin),
		RemoteIP:  remoteIP,
		UserAgent: userAgent,
		Status:    "success",
	}
	return m.CreateAuditLog(ctx, log)
}

// LogFailedLogin logs a failed login attempt
func (m *AuditManager) LogFailedLogin(ctx context.Context, tenantID, email, remoteIP, errorCode string) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		FromEmail: email,
		Action:    string(ActionMailFailedLogin),
		RemoteIP:  remoteIP,
		Status:    "failed",
		ErrorCode: errorCode,
	}
	return m.CreateAuditLog(ctx, log)
}

// LogBounce logs a bounced mail event
func (m *AuditManager) LogBounce(ctx context.Context, tenantID, mailboxID, messageID, errorCode, errorMessage string) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		MailboxID: mailboxID,
		Action:    string(ActionMailBounced),
		MessageID: messageID,
		Status:    "bounced",
		ErrorCode: errorCode,
	}
	_ = errorMessage // Would be stored in detail in production
	return m.CreateAuditLog(ctx, log)
}

// LogRejected logs a rejected mail event
func (m *AuditManager) LogRejected(ctx context.Context, tenantID, from, to, errorCode string, remoteIP string) error {
	log := &models.MailAuditLog{
		TenantID:  tenantID,
		Action:    string(ActionMailRejected),
		Direction: string(DirectionInbound),
		FromEmail: from,
		ToEmail:   to,
		Status:    "rejected",
		ErrorCode: errorCode,
		RemoteIP:  remoteIP,
	}
	return m.CreateAuditLog(ctx, log)
}

// ListAuditLogs retrieves audit logs with filters
func (m *AuditManager) ListAuditLogs(ctx context.Context, tenantID string, filters *AuditLogFilters, page, pageSize int) ([]models.MailAuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var logs []models.MailAuditLog
	var total int64

	query := m.db.Model(&models.MailAuditLog{}).Where("tenant_id = ?", tenantID)

	if filters != nil {
		if filters.MailboxID != "" {
			query = query.Where("mailbox_id = ?", filters.MailboxID)
		}
		if filters.Action != "" {
			query = query.Where("action = ?", filters.Action)
		}
		if filters.Direction != "" {
			query = query.Where("direction = ?", filters.Direction)
		}
		if filters.FromEmail != "" {
			query = query.Where("from_email LIKE ?", "%"+filters.FromEmail+"%")
		}
		if filters.ToEmail != "" {
			query = query.Where("to_email LIKE ?", "%"+filters.ToEmail+"%")
		}
		if !filters.StartDate.IsZero() {
			query = query.Where("created_at >= ?", filters.StartDate)
		}
		if !filters.EndDate.IsZero() {
			query = query.Where("created_at <= ?", filters.EndDate)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, NewMailError("ListAuditLogs", err, "count error")
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, NewMailError("ListAuditLogs", err, "query error")
	}

	return logs, total, nil
}

// GetAuditStats returns audit statistics
func (m *AuditManager) GetAuditStats(ctx context.Context, tenantID string, startDate, endDate time.Time) (*AuditStats, error) {
	stats := &AuditStats{}

	query := m.db.Model(&models.MailAuditLog{}).Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, startDate, endDate)

	// Count by action
	query.Select("action, COUNT(*) as count").Group("action").Scan(&stats.Actions)

	// Count by status
	query.Select("status, COUNT(*) as count").Group("status").Scan(&stats.Statuses)

	// Total count
	var total int64
	query.Count(&total)
	stats.TotalEvents = total

	return stats, nil
}

// AuditLogFilters represents filters for audit log queries
type AuditLogFilters struct {
	MailboxID string
	Action    string
	Direction string
	FromEmail string
	ToEmail   string
	StartDate time.Time
	EndDate   time.Time
}

// AuditStats represents audit statistics
type AuditStats struct {
	TotalEvents int64
	Actions     []ActionCount
	Statuses    []StatusCount
}

// ActionCount represents action count for statistics
type ActionCount struct {
	Action string
	Count  int64
}

// StatusCount represents status count for statistics
type StatusCount struct {
	Status string
	Count  int64
}

// GetRecentActivity returns recent mail activity for a tenant
func (m *AuditManager) GetRecentActivity(ctx context.Context, tenantID string, limit int) ([]models.MailAuditLog, error) {
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var logs []models.MailAuditLog
	if err := m.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, NewMailError("GetRecentActivity", err, "query error")
	}

	return logs, nil
}