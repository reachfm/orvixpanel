package auth

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// newID returns a fresh ULID. Used for session primary keys.
func newID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
}

// Compile-time guard: keep rand referenced even if a future
// refactor drops its only call site.
var _ = rand.Read
