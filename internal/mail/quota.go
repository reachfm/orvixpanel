package mail

import (
	"context"
	"fmt"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// QuotaManager handles quota management operations
type QuotaManager struct {
	db *gorm.DB
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(db *gorm.DB) *QuotaManager {
	return &QuotaManager{db: db}
}

// GetQuotaStatus returns quota status for a mailbox
func (m *QuotaManager) GetQuotaStatus(ctx context.Context, mailboxID string) (*QuotaStatus, error) {
	mailbox, err := m.getMailboxByID(ctx, mailboxID)
	if err != nil {
		return nil, err
	}

	usedPercent := 0.0
	if mailbox.QuotaMB > 0 {
		usedPercent = float64(mailbox.QuotaUsedMB) / float64(mailbox.QuotaMB) * 100
	}

	status := "ok"
	if usedPercent >= 90 {
		status = "warning"
	}
	if usedPercent >= 100 {
		status = "exceeded"
	}

	return &QuotaStatus{
		MailboxID:    mailboxID,
		UsedMB:       mailbox.QuotaUsedMB,
		LimitMB:      mailbox.QuotaMB,
		UsedPercent:  usedPercent,
		Status:       status,
		AllowSend:    usedPercent < 100,
		AllowReceive: usedPercent < 100,
	}, nil
}

// CheckQuota checks if a mailbox can send/receive
func (m *QuotaManager) CheckQuota(ctx context.Context, mailboxID string) error {
	status, err := m.GetQuotaStatus(ctx, mailboxID)
	if err != nil {
		return err
	}

	if !status.AllowReceive {
		return ErrQuotaExceeded
	}

	return nil
}

// UpdateQuotaUsage updates the quota usage for a mailbox
func (m *QuotaManager) UpdateQuotaUsage(ctx context.Context, mailboxID string, usedMB int) error {
	mailbox, err := m.getMailboxByID(ctx, mailboxID)
	if err != nil {
		return err
	}

	mailbox.QuotaUsedMB = usedMB

	// Auto-suspend if quota exceeded
	if usedMB > mailbox.QuotaMB {
		mailbox.Status = "suspended"
	}

	if err := m.db.Save(mailbox).Error; err != nil {
		return NewMailError("UpdateQuotaUsage", err, mailboxID)
	}

	return nil
}

// GetAllQuotas returns quota status for all mailboxes in a domain
func (m *QuotaManager) GetAllQuotas(ctx context.Context, domainID string) ([]*QuotaStatus, error) {
	var mailboxes []models.Mailbox
	if err := m.db.Where("mail_domain_id = ?", domainID).Find(&mailboxes).Error; err != nil {
		return nil, NewMailError("GetAllQuotas", err, domainID)
	}

	statuses := make([]*QuotaStatus, 0, len(mailboxes))
	for _, mb := range mailboxes {
		status, err := m.GetQuotaStatus(ctx, mb.ID)
		if err != nil {
			continue
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetDomainQuotaSummary returns quota summary for a domain
func (m *QuotaManager) GetDomainQuotaSummary(ctx context.Context, domainID string) (*QuotaSummary, error) {
	var mailboxes []models.Mailbox
	if err := m.db.Where("mail_domain_id = ?", domainID).Find(&mailboxes).Error; err != nil {
		return nil, NewMailError("GetDomainQuotaSummary", err, domainID)
	}

	var totalLimit, totalUsed int
	warningCount := 0
	exceededCount := 0

	for _, mb := range mailboxes {
		totalLimit += mb.QuotaMB
		totalUsed += mb.QuotaUsedMB

		if mb.QuotaMB > 0 {
			percent := float64(mb.QuotaUsedMB) / float64(mb.QuotaMB) * 100
			if percent >= 100 {
				exceededCount++
			} else if percent >= 90 {
				warningCount++
			}
		}
	}

	summary := &QuotaSummary{
		TotalMailboxes: len(mailboxes),
		TotalLimitMB:   totalLimit,
		TotalUsedMB:    totalUsed,
		WarningCount:   warningCount,
		ExceededCount:  exceededCount,
	}

	if totalLimit > 0 {
		summary.OverallPercent = float64(totalUsed) / float64(totalLimit) * 100
	}

	return summary, nil
}

// QuotaStatus represents quota status for a mailbox
type QuotaStatus struct {
	MailboxID    string
	UsedMB       int
	LimitMB      int
	UsedPercent  float64
	Status       string // ok, warning, exceeded
	AllowSend    bool
	AllowReceive bool
}

// QuotaSummary represents quota summary for a domain
type QuotaSummary struct {
	TotalMailboxes int
	TotalLimitMB   int
	TotalUsedMB    int
	OverallPercent float64
	WarningCount   int
	ExceededCount  int
}

// getMailboxByID retrieves a mailbox by ID (internal helper)
func (m *QuotaManager) getMailboxByID(ctx context.Context, mailboxID string) (*models.Mailbox, error) {
	var mailbox models.Mailbox
	if err := m.db.Where("id = ?", mailboxID).First(&mailbox).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMailboxNotFound
		}
		return nil, NewMailError("getMailboxByID", err, mailboxID)
	}
	return &mailbox, nil
}

// CleanupQuotas cleans up stale quota records (called periodically)
func (m *QuotaManager) CleanupQuotas(ctx context.Context) error {
	// This would be called by a scheduler to clean up stale quota data
	// In production, this might recalculate actual mailbox sizes
	return nil
}

// FormatQuotaDisplay formats quota for display
func FormatQuotaDisplay(usedMB, limitMB int) string {
	usedStr := formatMB(usedMB)
	limitStr := formatMB(limitMB)
	return fmt.Sprintf("%s / %s", usedStr, limitStr)
}

// formatMB formats MB for display
func formatMB(mb int) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}

// CalculateQuotaPercent calculates quota usage percentage
func CalculateQuotaPercent(usedMB, limitMB int) float64 {
	if limitMB == 0 {
		return 0
	}
	return float64(usedMB) / float64(limitMB) * 100
}