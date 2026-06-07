//go:build linux

package main

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

// ensureRuntimeDir makes sure path exists with the requested mode and
// is owned by the named user. It is idempotent and best-effort: any
// failure is returned so the caller can log it, but the caller should
// NOT abort startup on failure — the directory is not on the runtime
// critical path today.
//
// On Linux /run is a tmpfs that gets cleared on reboot, on WSL
// reinit, and on some systemd service restarts. install.sh creates
// this directory once, which is not durable; the binary self-heals
// on every start so the doctor check stays green across all of the
// above lifecycle events.
func ensureRuntimeDir(path string, mode os.FileMode, ownerName string) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	// Best-effort chown. Skip silently if the user cannot be looked
	// up (very minimal images, unusual nss configs) — the directory
	// is still there and the doctor check passes.
	u, err := user.Lookup(ownerName)
	if err != nil {
		// UnknownUserIdError("orvixpanel: unknown user") etc.
		// Don't fail startup on lookup errors.
		return nil
	}
	uid, uidErr := strconv.Atoi(u.Uid)
	gid, gidErr := strconv.Atoi(u.Gid)
	if uidErr != nil || gidErr != nil {
		return nil
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown %s: %w", path, err)
	}
	return nil
}
