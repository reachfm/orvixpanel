package v1

import "os"

// osGetenvImpl is the real implementation of osGetenv; split out
// so the v1 package can be re-imported from test binaries that
// need to override it.
func osGetenvImpl(k string) string { return os.Getenv(k) }
