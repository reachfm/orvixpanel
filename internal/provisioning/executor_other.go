//go:build !linux

// Non-Linux stub implementations.
package provisioning

import (
	"context"
	"errors"
)

// ErrUnsupported is returned when provisioning is attempted on a non-Linux platform.
var ErrUnsupported = errors.New("provisioning not supported on this platform")

// LinuxSystemExecutor is a stub on non-Linux.
type LinuxSystemExecutor struct{}

// CreateUser returns ErrUnsupported.
func (*LinuxSystemExecutor) CreateUser(ctx context.Context, username, homeDir string) error {
	return ErrUnsupported
}

// DeleteUser returns ErrUnsupported.
func (*LinuxSystemExecutor) DeleteUser(ctx context.Context, username string) error {
	return ErrUnsupported
}

// CreateDir returns ErrUnsupported.
func (*LinuxSystemExecutor) CreateDir(ctx context.Context, path string, mode int, owner string) error {
	return ErrUnsupported
}

// RemoveDir returns ErrUnsupported.
func (*LinuxSystemExecutor) RemoveDir(ctx context.Context, path string) error {
	return ErrUnsupported
}

// WriteFile returns ErrUnsupported.
func (*LinuxSystemExecutor) WriteFile(ctx context.Context, path string, content []byte, mode int) error {
	return ErrUnsupported
}

// RemoveFile returns ErrUnsupported.
func (*LinuxSystemExecutor) RemoveFile(ctx context.Context, path string) error {
	return ErrUnsupported
}

// Chown returns ErrUnsupported.
func (*LinuxSystemExecutor) Chown(ctx context.Context, path string, owner string) error {
	return ErrUnsupported
}

// RunCommand returns ErrUnsupported.
func (*LinuxSystemExecutor) RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	return "", ErrUnsupported
}

// LinuxWebserverExecutor is a stub on non-Linux.
type LinuxWebserverExecutor struct{}

// WriteNginxVHost returns ErrUnsupported.
func (*LinuxWebserverExecutor) WriteNginxVHost(ctx context.Context, path string, content string) error {
	return ErrUnsupported
}

// RemoveNginxVHost returns ErrUnsupported.
func (*LinuxWebserverExecutor) RemoveNginxVHost(ctx context.Context, path string) error {
	return ErrUnsupported
}

// TestNginx returns ErrUnsupported.
func (*LinuxWebserverExecutor) TestNginx(ctx context.Context) error {
	return ErrUnsupported
}

// ReloadNginx returns ErrUnsupported.
func (*LinuxWebserverExecutor) ReloadNginx(ctx context.Context) error {
	return ErrUnsupported
}

// WriteFPMPool returns ErrUnsupported.
func (*LinuxWebserverExecutor) WriteFPMPool(ctx context.Context, path string, content string) error {
	return ErrUnsupported
}

// RemoveFPMPool returns ErrUnsupported.
func (*LinuxWebserverExecutor) RemoveFPMPool(ctx context.Context, path string) error {
	return ErrUnsupported
}

// TestPHP returns ErrUnsupported.
func (*LinuxWebserverExecutor) TestPHP(ctx context.Context) error {
	return ErrUnsupported
}

// ReloadPHP returns ErrUnsupported.
func (*LinuxWebserverExecutor) ReloadPHP(ctx context.Context) error {
	return ErrUnsupported
}