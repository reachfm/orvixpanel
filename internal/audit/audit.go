// Package audit implements the append-only audit log with SHA-256
// hash chaining (spec §6 / Audit Log Format).
//
// Each row's `hash` is sha256(prev_hash || canonical_json(row_content)).
// Tampering with any historical row invalidates every subsequent
// row's hash, so the chain is self-verifying.
//
// The chain state (last hash) is cached in memory and persisted by
// reading the latest row on Auditor creation. v1.0 keeps the cache
// in-process; v1.1 can shard it across instances via Redis if needed.
package audit

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Event is the input to Record.
type Event struct {
	Timestamp    time.Time
	UserID       string
	UserEmail    string
	UserRole     string
	ActorIP      string
	SessionID    string
	Action       string
	ResourceType string
	ResourceID   string
	ResourceName string
	Result       string
	DurationMS   int
	Detail       string
}

// Auditor is the entry point.
type Auditor struct {
	db    *gorm.DB
	mu    sync.Mutex
	cache string // last hash seen, in-memory
}

// New opens the audit log and seeds the cache with the most recent
// row's hash so the chain resumes correctly.
func New(ctx context.Context, db *gorm.DB) (*Auditor, error) {
	a := &Auditor{db: db}
	if err := a.loadLastHash(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Auditor) loadLastHash(ctx context.Context) error {
	var last models.AuditEntry
	err := a.db.WithContext(ctx).Order("id DESC").First(&last).Error
	if err == gorm.ErrRecordNotFound {
		a.cache = ""
		return nil
	}
	if err != nil {
		return fmt.Errorf("load last audit hash: %w", err)
	}
	a.cache = last.Hash
	return nil
}

// Record persists an event with hash chaining. Safe for concurrent use.
func (a *Auditor) Record(ctx context.Context, ev Event) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Result == "" {
		ev.Result = "success"
	}

	a.mu.Lock()
	prev := a.cache
	a.mu.Unlock()

	hash, err := computeHash(prev, ev)
	if err != nil {
		return fmt.Errorf("compute hash: %w", err)
	}

	row := models.AuditEntry{
		Base:         models.Base{ID: newID()},
		Timestamp:    ev.Timestamp,
		UserID:       ev.UserID,
		UserEmail:    ev.UserEmail,
		UserRole:     ev.UserRole,
		ActorIP:      ev.ActorIP,
		SessionID:    ev.SessionID,
		Action:       ev.Action,
		ResourceType: ev.ResourceType,
		ResourceID:   ev.ResourceID,
		ResourceName: ev.ResourceName,
		Result:      ev.Result,
		DurationMS:  ev.DurationMS,
		Detail:      ev.Detail,
		PrevHash:    prev,
		Hash:        hash,
	}
	if err := a.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}

	a.mu.Lock()
	a.cache = hash
	a.mu.Unlock()
	return nil
}

// VerifyChain walks the entire log and confirms the chain. Returns
// the index of the first tampered row, or -1 if clean.
func (a *Auditor) VerifyChain(ctx context.Context) (int64, error) {
	var rows []models.AuditEntry
	if err := a.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return 0, err
	}
	var prev string
	for i, r := range rows {
		ev := Event{
			Timestamp:    r.Timestamp,
			UserID:       r.UserID,
			UserEmail:    r.UserEmail,
			UserRole:     r.UserRole,
			ActorIP:      r.ActorIP,
			SessionID:    r.SessionID,
			Action:       r.Action,
			ResourceType: r.ResourceType,
			ResourceID:   r.ResourceID,
			ResourceName: r.ResourceName,
			Result:      r.Result,
			DurationMS:  r.DurationMS,
			Detail:      r.Detail,
		}
		expected, err := computeHash(prev, ev)
		if err != nil {
			return int64(i), fmt.Errorf("hash row %d: %w", i, err)
		}
		if r.PrevHash != prev || r.Hash != expected {
			return int64(i), nil
		}
		prev = r.Hash
	}
	return -1, nil
}

// computeHash returns "sha256:" + hex(sha256(prev || canonical_json(ev))).
func computeHash(prev string, ev Event) (string, error) {
	canonical, err := canonicalJSON(ev)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(prev))
	h.Write([]byte{0})
	h.Write(canonical)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// canonicalJSON serializes the event with sorted keys so the hash is
// deterministic across calls.
func canonicalJSON(ev Event) ([]byte, error) {
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return b, nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf := []byte{'{'}
	for i, k := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		kb, _ := json.Marshal(k)
		buf = append(buf, kb...)
		buf = append(buf, ':')
		vb, err := json.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		buf = append(buf, vb...)
	}
	buf = append(buf, '}')
	return buf, nil
}

// newID returns a fresh ULID.
func newID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
}

// Compile-time guards.
var _ = log.Warn
var _ sort.Interface
