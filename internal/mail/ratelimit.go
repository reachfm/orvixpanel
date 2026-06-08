package mail

import (
	"context"
	"sync"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// RateLimitManager handles rate limiting for mail operations
type RateLimitManager struct {
	db         *gorm.DB
	counters   map[string]*rateCounter
	mu         sync.RWMutex
	expiration time.Duration
}

// rateCounter tracks message counts for rate limiting
type rateCounter struct {
	Count     int
	WindowEnd time.Time
	mu        sync.Mutex
}

// NewRateLimitManager creates a new rate limit manager
func NewRateLimitManager(db *gorm.DB) *RateLimitManager {
	return &RateLimitManager{
		db:         db,
		counters:   make(map[string]*rateCounter),
		expiration: 24 * time.Hour,
	}
}

// CreateRateLimit creates a new rate limit rule
func (m *RateLimitManager) CreateRateLimit(ctx context.Context, limit *models.MailRateLimit) error {
	if limit.ID == "" {
		limit.ID = generateID("rl")
	}
	if limit.Status == "" {
		limit.Status = "active"
	}

	if err := m.db.Create(limit).Error; err != nil {
		return NewMailError("CreateRateLimit", err, "database error")
	}

	return nil
}

// GetRateLimit retrieves a rate limit rule by ID
func (m *RateLimitManager) GetRateLimit(ctx context.Context, tenantID, limitID string) (*models.MailRateLimit, error) {
	var limit models.MailRateLimit
	if err := m.db.Where("id = ? AND tenant_id = ?", limitID, tenantID).First(&limit).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRateLimitExceeded
		}
		return nil, NewMailError("GetRateLimit", err, limitID)
	}
	return &limit, nil
}

// ListRateLimits retrieves all rate limit rules for a tenant
func (m *RateLimitManager) ListRateLimits(ctx context.Context, tenantID string) ([]models.MailRateLimit, error) {
	var limits []models.MailRateLimit
	if err := m.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&limits).Error; err != nil {
		return nil, NewMailError("ListRateLimits", err, "list error")
	}
	return limits, nil
}

// DeleteRateLimit deletes a rate limit rule
func (m *RateLimitManager) DeleteRateLimit(ctx context.Context, tenantID, limitID string) error {
	limit, err := m.GetRateLimit(ctx, tenantID, limitID)
	if err != nil {
		return err
	}

	if err := m.db.Delete(limit).Error; err != nil {
		return NewMailError("DeleteRateLimit", err, limitID)
	}

	return nil
}

// rateLimitKey generates the unique key for a rate limit rule
func rateLimitKey(limit *models.MailRateLimit) string {
	if limit.MailboxID != "" {
		return limit.TenantID + ":" + limit.MailboxID + ":" + limit.RateType
	}
	return limit.TenantID + ":" + limit.RateType
}

// CheckRateLimit checks if a mailbox has exceeded rate limits
func (m *RateLimitManager) CheckRateLimit(ctx context.Context, tenantID, mailboxID, rateType string) error {
	// Get rate limit rules
	var limits []models.MailRateLimit
	query := m.db.Where("tenant_id = ? AND status = ?", tenantID, "active")
	if mailboxID != "" {
		query = query.Where("mailbox_id = ? OR mailbox_id = ''", mailboxID)
	}
	query = query.Where("rate_type = ?", rateType)

	if err := query.Find(&limits).Error; err != nil {
		return NewMailError("CheckRateLimit", err, "query error")
	}

	// Check each limit
	for _, limit := range limits {
		key := rateLimitKey(&limit)
		if !m.checkCounter(key, limit.MaxMessages, limit.WindowMinutes) {
			return ErrRateLimitExceeded
		}
	}

	return nil
}

// IncrementCounter increments the rate limit counter
func (m *RateLimitManager) IncrementCounter(ctx context.Context, tenantID, mailboxID, rateType string) error {
	// Get rate limit rules
	var limits []models.MailRateLimit
	query := m.db.Where("tenant_id = ? AND status = ?", tenantID, "active")
	if mailboxID != "" {
		query = query.Where("mailbox_id = ? OR mailbox_id = ''", mailboxID)
	}
	query = query.Where("rate_type = ?", rateType)

	if err := query.Find(&limits).Error; err != nil {
		return NewMailError("IncrementCounter", err, "query error")
	}

	// Increment counters
	for _, limit := range limits {
		key := rateLimitKey(&limit)
		m.incrementCounter(key, limit.WindowMinutes)
	}

	return nil
}

// checkCounter checks if the rate limit is exceeded
func (m *RateLimitManager) checkCounter(key string, maxMessages, windowMinutes int) bool {
	m.mu.RLock()
	counter, exists := m.counters[key]
	m.mu.RUnlock()

	if !exists {
		return true // No counter means no limit exceeded
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	// Check if window has expired
	if time.Now().After(counter.WindowEnd) {
		return true // Window expired, reset
	}

	return counter.Count < maxMessages
}

// incrementCounter increments the counter for a key
func (m *RateLimitManager) incrementCounter(key string, windowMinutes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counter, exists := m.counters[key]
	if !exists {
		counter = &rateCounter{
			WindowEnd: time.Now().Add(time.Duration(windowMinutes) * time.Minute),
		}
		m.counters[key] = counter
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	// Reset if window expired
	if time.Now().After(counter.WindowEnd) {
		counter.Count = 0
		counter.WindowEnd = time.Now().Add(time.Duration(windowMinutes) * time.Minute)
	}

	counter.Count++
}

// cleanupCounters removes expired counters
func (m *RateLimitManager) cleanupCounters() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, counter := range m.counters {
		if now.After(counter.WindowEnd) {
			delete(m.counters, key)
		}
	}
}

// GetRateLimitStatus returns the current rate limit status for a mailbox
func (m *RateLimitManager) GetRateLimitStatus(ctx context.Context, tenantID, mailboxID string) (map[string]interface{}, error) {
	limits, err := m.ListRateLimits(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	status := make(map[string]interface{})
	for _, limit := range limits {
		if limit.MailboxID != "" && limit.MailboxID != mailboxID {
			continue
		}

		key := rateLimitKey(&limit)
		m.mu.RLock()
		counter, exists := m.counters[key]
		m.mu.RUnlock()

		remaining := limit.MaxMessages
		resetAt := time.Now().Add(time.Duration(limit.WindowMinutes) * time.Minute)

		if exists {
			counter.mu.Lock()
			if time.Now().Before(counter.WindowEnd) {
				remaining = limit.MaxMessages - counter.Count
				resetAt = counter.WindowEnd
			}
			counter.mu.Unlock()
		}

		status[limit.RateType] = map[string]interface{}{
			"remaining":    remaining,
			"limit":        limit.MaxMessages,
			"window_minutes": limit.WindowMinutes,
			"reset_at":     resetAt,
		}
	}

	return status, nil
}