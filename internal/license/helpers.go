package license

import (
	"crypto/rand"
	"crypto/sha256"
	"time"

	"github.com/oklog/ulid/v2"
)

// sha256Sum returns the SHA-256 of b.
func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

// newLicenseID returns a fresh ULID for the LicenseStore row.
func newLicenseID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
}

// timeNow is the clock. Pulled out so tests can freeze it.
var timeNow = func() time.Time { return time.Now().UTC() }

// timeRFC3339 is the format used for license timestamps in API output.
const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
