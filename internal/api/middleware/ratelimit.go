package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/auth"
)

// RateLimiter is a simple in-process token bucket. v1.1 swaps this
// for a Redis-backed limiter for multi-instance deployments.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rps     float64
	burst   float64
	cleanup time.Duration
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a limiter. rps is sustained rate; burst is
// the max instantaneous tokens.
func NewRateLimiter(rps, burst float64) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rps:     rps,
		burst:   burst,
		cleanup: 5 * time.Minute,
	}
}

// Middleware returns a Fiber handler that 429s on rate excess.
func (r *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.IP()
		if claims, ok := c.Locals(LocalClaims).(*auth.Claims); ok && claims != nil {
			key = key + "|" + claims.UserID
		}
		if !r.allow(key) {
			return fiber.NewError(fiber.StatusTooManyRequests, "rate_limited")
		}
		return c.Next()
	}
}

func (r *RateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{tokens: r.burst, last: now}
		r.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = minF(r.burst, b.tokens+elapsed*r.rps)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
