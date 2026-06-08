package ssl

import (
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"context"
	"time"

	"gorm.io/gorm"
)

// EventLogger handles SSL event logging.
type EventLogger struct {
	db *gorm.DB
}

// NewEventLogger creates a new event logger.
func NewEventLogger(db *gorm.DB) *EventLogger {
	return &EventLogger{db: db}
}

// LogEvent logs an SSL event to the database.
func (l *EventLogger) LogEvent(ctx context.Context, certID, eventType, message, errorDetail string) error {
	event := &models.SSLEvent{
		CertificateID: certID,
		EventType:    eventType,
		Message:     message,
		ErrorDetail: errorDetail,
	}
	return l.db.WithContext(ctx).Create(event).Error
}

// LogIssueStarted logs the start of a certificate issuance.
func (l *EventLogger) LogIssueStarted(ctx context.Context, certID string) error {
	return l.LogEvent(ctx, certID, "issue_started", "Certificate issuance started", "")
}

// LogIssueSucceeded logs successful certificate issuance.
func (l *EventLogger) LogIssueSucceeded(ctx context.Context, certID string, expiresAt time.Time) error {
	return l.LogEvent(ctx, certID, "issued", "Certificate issued successfully", "Expires: "+expiresAt.Format("2006-01-02"))
}

// LogIssueFailed logs failed certificate issuance.
func (l *EventLogger) LogIssueFailed(ctx context.Context, certID string, err error) error {
	return l.LogEvent(ctx, certID, "issue_failed", "Certificate issuance failed", err.Error())
}

// LogRenewalStarted logs the start of a certificate renewal.
func (l *EventLogger) LogRenewalStarted(ctx context.Context, certID string) error {
	return l.LogEvent(ctx, certID, "renewal_started", "Certificate renewal started", "")
}

// LogRenewalSucceeded logs successful certificate renewal.
func (l *EventLogger) LogRenewalSucceeded(ctx context.Context, certID string, expiresAt time.Time) error {
	return l.LogEvent(ctx, certID, "renewed", "Certificate renewed successfully", "Expires: "+expiresAt.Format("2006-01-02"))
}

// LogRenewalFailed logs failed certificate renewal.
func (l *EventLogger) LogRenewalFailed(ctx context.Context, certID string, err error) error {
	return l.LogEvent(ctx, certID, "renewal_failed", "Certificate renewal failed", err.Error())
}

// LogRevocation logs a certificate revocation.
func (l *EventLogger) LogRevocation(ctx context.Context, certID string) error {
	return l.LogEvent(ctx, certID, "revoked", "Certificate revoked", "")
}

// LogDeletion logs certificate deletion.
func (l *EventLogger) LogDeletion(ctx context.Context, certID string) error {
	return l.LogEvent(ctx, certID, "deleted", "Certificate deleted", "")
}

// LogNginxUpdated logs nginx configuration update.
func (l *EventLogger) LogNginxUpdated(ctx context.Context, certID string, domain string) error {
	return l.LogEvent(ctx, certID, "nginx_updated", "Nginx vhost updated with SSL for "+domain, "")
}

// LogNginxFailed logs nginx configuration failure.
func (l *EventLogger) LogNginxFailed(ctx context.Context, certID string, err error) error {
	return l.LogEvent(ctx, certID, "nginx_failed", "Nginx configuration failed", err.Error())
}

// LogChallengeRequested logs ACME challenge request.
func (l *EventLogger) LogChallengeRequested(ctx context.Context, certID, token, domain string) error {
	return l.LogEvent(ctx, certID, "challenge_requested", "ACME HTTP-01 challenge requested for "+domain, "Token: "+token)
}

// LogChallengeVerified logs successful challenge verification.
func (l *EventLogger) LogChallengeVerified(ctx context.Context, certID, token string) error {
	return l.LogEvent(ctx, certID, "challenge_verified", "ACME HTTP-01 challenge verified", "Token: "+token)
}

// LogChallengeFailed logs failed challenge verification.
func (l *EventLogger) LogChallengeFailed(ctx context.Context, certID string, err error) error {
	return l.LogEvent(ctx, certID, "challenge_failed", "ACME HTTP-01 challenge failed", err.Error())
}

// GetCertificateEvents retrieves all events for a certificate.
func (l *EventLogger) GetCertificateEvents(ctx context.Context, certID string) ([]*models.SSLEvent, error) {
	var events []*models.SSLEvent
	err := l.db.WithContext(ctx).
		Where("certificate_id = ?", certID).
		Order("created_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetRecentEvents retrieves recent events across all certificates.
func (l *EventLogger) GetRecentEvents(ctx context.Context, limit int) ([]*models.SSLEvent, error) {
	var events []*models.SSLEvent
	err := l.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetEventsByType retrieves events of a specific type.
func (l *EventLogger) GetEventsByType(ctx context.Context, eventType string, limit int) ([]*models.SSLEvent, error) {
	var events []*models.SSLEvent
	err := l.db.WithContext(ctx).
		Where("event_type = ?", eventType).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetFailedEvents retrieves all failure events.
func (l *EventLogger) GetFailedEvents(ctx context.Context) ([]*models.SSLEvent, error) {
	var events []*models.SSLEvent
	err := l.db.WithContext(ctx).
		Where("event_type IN ?", []string{"issue_failed", "renewal_failed", "challenge_failed", "nginx_failed"}).
		Order("created_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetEventsByDateRange retrieves events within a date range.
func (l *EventLogger) GetEventsByDateRange(ctx context.Context, start, end time.Time) ([]*models.SSLEvent, error) {
	var events []*models.SSLEvent
	err := l.db.WithContext(ctx).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}