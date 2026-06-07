//go:build !linux

package main

import "os"

// ensureRuntimeDir is a no-op on non-Linux platforms. /run/orvixpanel
// is a Linux-specific runtime path; the Windows / macOS developer
// builds do not need it. main.go calls this unconditionally so the
// build surface stays simple.
func ensureRuntimeDir(path string, mode os.FileMode, ownerName string) error {
	_ = path
	_ = mode
	_ = ownerName
	return nil
}
