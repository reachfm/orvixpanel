//go:build linux

// Linux executor implementations using exec.Command (never sh -c).
package provisioning

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// LinuxSystemExecutor implements SystemExecutor for Linux.
type LinuxSystemExecutor struct {
	Paths Paths
}

// NewLinuxSystemExecutor creates a new Linux system executor.
func NewLinuxSystemExecutor(p Paths) *LinuxSystemExecutor {
	return &LinuxSystemExecutor{Paths: p}
}

// CreateUser creates a system user with the given username and home dir.
func (e *LinuxSystemExecutor) CreateUser(ctx context.Context, username, homeDir string) error {
	// Use useradd directly - no shell interpretation
	args := []string{
		"-m",              // create home if missing
		"-d", homeDir,     // home dir
		"-s", "/bin/bash", // shell
		"-g", username,    // primary group = username
		"-c", "OrvixPanel account", // GECOS
		username,
	}
	cmd := exec.CommandContext(ctx, "/usr/sbin/useradd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command:    append([]string{"/usr/sbin/useradd"}, args...),
			Output:     string(out),
			Err:        err,
		}
	}
	return nil
}

// DeleteUser removes a system user.
func (e *LinuxSystemExecutor) DeleteUser(ctx context.Context, username string) error {
	cmd := exec.CommandContext(ctx, "/usr/sbin/userdel", "-r", username)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{"userdel", "-r", username},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// CreateDir creates a directory with the given permissions and ownership.
func (e *LinuxSystemExecutor) CreateDir(ctx context.Context, path string, mode int, owner string) error {
	// First create the directory
	if err := os.MkdirAll(path, os.FileMode(mode)); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	// Then chown to the user
	if err := e.Chown(ctx, path, owner); err != nil {
		return fmt.Errorf("chown %s: %w", path, err)
	}

	return nil
}

// RemoveDir removes a directory tree.
func (e *LinuxSystemExecutor) RemoveDir(ctx context.Context, path string) error {
	// Use rm -rf with exec.Command (not shell)
	cmd := exec.CommandContext(ctx, "/bin/rm", "-rf", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "no such file or directory" errors
		if strings.Contains(string(out), "No such file or directory") {
			return nil
		}
		return &CommandError{
			Command: []string{"rm", "-rf", path},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// WriteFile writes content to a file with given permissions.
func (e *LinuxSystemExecutor) WriteFile(ctx context.Context, path string, content []byte, mode int) error {
	if err := os.WriteFile(path, content, os.FileMode(mode)); err != nil {
		return fmt.Errorf("writefile %s: %w", path, err)
	}
	return nil
}

// RemoveFile removes a file.
func (e *LinuxSystemExecutor) RemoveFile(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "/bin/rm", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "no such file" errors
		if strings.Contains(string(out), "No such file or directory") {
			return nil
		}
		return &CommandError{
			Command: []string{"rm", path},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// Chown changes ownership of a file or directory.
func (e *LinuxSystemExecutor) Chown(ctx context.Context, path string, owner string) error {
	// Use chown with exec.Command - no shell interpretation
	cmd := exec.CommandContext(ctx, "/bin/chown", "-R", owner+":"+owner, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{"chown", "-R", owner + ":" + owner, path},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// RunCommand runs an external command and returns its output.
// The command slice must NOT contain shell metacharacters.
func (e *LinuxSystemExecutor) RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", &CommandError{
			Command: append([]string{name}, args...),
			Output:  string(out),
			Err:     err,
		}
	}
	return strings.TrimSpace(string(out)), nil
}

// LinuxWebserverExecutor implements WebserverExecutor for Linux.
type LinuxWebserverExecutor struct {
	NginxBin   string
	NginxCmd   string
	FPMBin     string
	FPMCmd     string
	NginxDir   string
	FPMPoolDir string
}

// NewLinuxWebserverExecutor creates a new Linux webserver executor.
func NewLinuxWebserverExecutor() *LinuxWebserverExecutor {
	return &LinuxWebserverExecutor{
		NginxBin:   "/usr/sbin/nginx",
		NginxCmd:   "nginx -t",
		FPMBin:     "/usr/sbin/php-fpm8.5",
		FPMCmd:     "php-fpm8.5 -t",
		NginxDir:   "/etc/nginx/conf.d/orvix",
		FPMPoolDir: "/etc/php/8.5/fpm/pool.d",
	}
}

// WriteNginxVHost writes the nginx vhost configuration.
func (e *LinuxWebserverExecutor) WriteNginxVHost(ctx context.Context, path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write nginx vhost: %w", err)
	}
	return nil
}

// RemoveNginxVHost removes the nginx vhost configuration.
func (e *LinuxWebserverExecutor) RemoveNginxVHost(ctx context.Context, path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// TestNginx validates the nginx configuration.
func (e *LinuxWebserverExecutor) TestNginx(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, e.NginxBin, "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{e.NginxBin, "-t"},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// ReloadNginx reloads the nginx service.
func (e *LinuxWebserverExecutor) ReloadNginx(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/bin/systemctl", "reload", "nginx")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{"systemctl", "reload", "nginx"},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// WriteFPMPool writes the PHP-FPM pool configuration.
func (e *LinuxWebserverExecutor) WriteFPMPool(ctx context.Context, path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write fpm pool: %w", err)
	}
	return nil
}

// RemoveFPMPool removes the PHP-FPM pool configuration.
func (e *LinuxWebserverExecutor) RemoveFPMPool(ctx context.Context, path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// TestPHP validates the PHP-FPM configuration.
func (e *LinuxWebserverExecutor) TestPHP(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, e.FPMBin, "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{e.FPMBin, "-t"},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// ReloadPHP reloads the PHP-FPM service.
func (e *LinuxWebserverExecutor) ReloadPHP(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/bin/systemctl", "reload", "php8.5-fpm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &CommandError{
			Command: []string{"systemctl", "reload", "php8.5-fpm"},
			Output:  string(out),
			Err:     err,
		}
	}
	return nil
}

// CommandError wraps a failed command with its output.
type CommandError struct {
	Command []string
	Output  string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %v failed: %v\noutput:\n%s", e.Command, e.Err, e.Output)
}

func (e *CommandError) Unwrap() error { return e.Err }

// ChownPath recursively chowns a path to uid:gid using raw syscalls.
// This avoids shell interpretation issues.
func ChownPath(path string, uid, gid int) error {
	return filepathWalk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return syscall.Chown(p, uid, gid)
	})
}

// filepathWalk walks the file tree rooted at root.
// This is a simplified version of filepath.Walk that avoids imports.
func filepathWalk(root string, fn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walk(root, info, fn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

func walk(path string, info os.FileInfo, fn filepath.WalkFunc) error {
	err := fn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := os.ReadDir(path)
	if err != nil {
		return fn(path, info, err)
	}

	for _, name := range names {
		namePath := path + string(os.PathSeparator) + name.Name()
		fileInfo, err := name.Info()
		if err != nil {
			if err := fn(namePath, nil, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			if err := walk(namePath, fileInfo, fn); err != nil {
				if err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// LookupUserUID looks up a user's UID by username.
func LookupUserUID(username string) (int, error) {
	// Use id command directly
	cmd := exec.Command("/usr/bin/id", "-u", username)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("lookup user %s: %w", username, err)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parse uid: %w", err)
	}
	return uid, nil
}

// LookupUserGID looks up a user's primary GID by username.
func LookupUserGID(username string) (int, error) {
	cmd := exec.Command("/usr/bin/id", "-g", username)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("lookup user %s: %w", username, err)
	}
	gid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parse gid: %w", err)
	}
	return gid, nil
}