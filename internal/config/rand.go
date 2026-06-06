package config

import (
	"crypto/rand"
	"encoding/base64"
)

// randRead is a thin shim so this package only needs crypto/rand
// for the dev-secret path. Keeping it tiny makes the file easy to
// audit in a security review.
func randRead(b []byte) (int, error) {
	return rand.Read(b)
}

// base64Encode wraps encoding/base64 with std encoding.
func base64Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
