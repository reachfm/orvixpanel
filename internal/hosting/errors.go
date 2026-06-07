// Package hosting — errors that are referenced from
// platform-agnostic code live here so they exist on every build.
package hosting

import "errors"

var (
	// ErrUnsupported is returned by OS-exec methods when the
	// package is built for a non-Linux target.
	ErrUnsupported = errors.New("hosting: operation not supported on this OS")

	// ErrAccountExists is returned by CreateAccount when the
	// system user is already present.
	ErrAccountExists = errors.New("account already exists")
)
