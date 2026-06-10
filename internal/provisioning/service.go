// Package provisioning implements the website provisioning workflow.
//
// Core types:
//   - ProvisioningJob: tracks the overall provisioning operation
//   - ProvisioningEvent: individual step audit trail
//   - Service: orchestrates the full provisioning workflow
//
// Interface-based design allows for:
//   - Testing with fake executors (no root required)
//   - Mocking OS operations in unit tests
//   - Swapping implementations for different platforms
package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// SystemExecutor defines the interface for OS-level operations.
// This allows tests to use fake executors without requiring root.
type SystemExecutor interface {
	// CreateUser creates a system user with the given username and home dir.
	CreateUser(ctx context.Context, username, homeDir string) error
	// DeleteUser removes a system user.
	DeleteUser(ctx context.Context, username string) error
	// CreateDir creates a directory with the given permissions and ownership.
	CreateDir(ctx context.Context, path string, mode int, owner string) error
	// RemoveDir removes a directory tree.
	RemoveDir(ctx context.Context, path string) error
	// WriteFile writes content to a file with given permissions.
	WriteFile(ctx context.Context, path string, content []byte, mode int) error
	// RemoveFile removes a file.
	RemoveFile(ctx context.Context, path string) error
	// Chown changes ownership of a file or directory.
	Chown(ctx context.Context, path string, owner string) error
	// RunCommand runs an external command and returns its output.
	// The command slice must NOT contain shell metacharacters.
	RunCommand(ctx context.Context, name string, args ...string) (string, error)
}

// WebserverExecutor defines the interface for web server operations.
type WebserverExecutor interface {
	// WriteNginxVHost writes the nginx vhost configuration.
	WriteNginxVHost(ctx context.Context, path string, content string) error
	// RemoveNginxVHost removes the nginx vhost configuration.
	RemoveNginxVHost(ctx context.Context, path string) error
	// TestNginx validates the nginx configuration.
	TestNginx(ctx context.Context) error
	// ReloadNginx reloads the nginx service.
	ReloadNginx(ctx context.Context) error
	// WriteFPMPool writes the PHP-FPM pool configuration.
	WriteFPMPool(ctx context.Context, path string, content string) error
	// RemoveFPMPool removes the PHP-FPM pool configuration.
	RemoveFPMPool(ctx context.Context, path string) error
	// TestPHP validates the PHP-FPM configuration.
	TestPHP(ctx context.Context) error
	// ReloadPHP reloads the PHP-FPM service.
	ReloadPHP(ctx context.Context) error
}

// Paths bundles the on-disk paths for provisioning.
type Paths struct {
	WebRoot      string // /var/www
	HomesDir     string // /var/lib/orvixpanel/homes
	NginxDir     string // /etc/nginx/conf.d/orvix
	FPMPoolDir   string // /etc/php/8.5/fpm/pool.d
	LogDir       string // /var/log/orvixpanel
	PHPSocketDir string // /run/php
}

// DefaultPaths returns production paths.
func DefaultPaths() Paths {
	return Paths{
		WebRoot:      "/var/www",
		HomesDir:     "/var/lib/orvixpanel/homes",
		NginxDir:     "/etc/nginx/conf.d/orvix",
		FPMPoolDir:   "/etc/php/8.5/fpm/pool.d",
		LogDir:       "/var/log/orvixpanel",
		PHPSocketDir: "/run/php",
	}
}

// EventRecorder defines the interface for recording provisioning events.
type EventRecorder interface {
	Record(*ProvisioningEvent) error
}

// JobStore defines the interface for persisting provisioning jobs.
type JobStore interface {
	Create(*ProvisioningJob) error
	Update(*ProvisioningJob) error
	GetByID(string) (*ProvisioningJob, error)
	GetByAccount(string) ([]*ProvisioningJob, error)
}

// Service is the main provisioning orchestrator.
type Service struct {
	Paths      Paths
	System     SystemExecutor
	Webserver  WebserverExecutor
	Jobs       JobStore
	Events     EventRecorder
	Validator  *DomainValidator
	Logger     *slog.Logger
}

// ServiceOption is a functional option for Service.
type ServiceOption func(*Service)

// WithPaths sets the paths.
func WithPaths(p Paths) ServiceOption {
	return func(s *Service) { s.Paths = p }
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) ServiceOption {
	return func(s *Service) { s.Logger = l }
}

// NewService creates a new provisioning service with defaults.
func NewService(opts ...ServiceOption) *Service {
	s := &Service{
		Paths:     DefaultPaths(),
		Validator: DefaultDomainValidator(),
		Logger:    slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ProvisionWebsiteRequest contains the input for provisioning a website.
type ProvisionWebsiteRequest struct {
	AccountID  string
	TenantID   string
	Username   string
	Domain     string
	PHPVersion string
}

// ProvisionWebsite orchestrates the full website provisioning workflow.
//
// Steps:
//  1. Validate domain
//  2. Check for duplicate website
//  3. Create job record
//  4. Create directory structure
//  5. Write nginx vhost
//  6. Write PHP-FPM pool
//  7. Test nginx config
//  8. Test PHP-FPM config
//  9. Reload services
// 10. Mark job complete
//
// On any failure:
//  1. Record failure
//  2. Rollback changes
//  3. Mark job rolled_back
//  4. Return error
func (s *Service) ProvisionWebsite(ctx context.Context, req ProvisionWebsiteRequest) (*ProvisioningJob, error) {
	// Default PHP version
	if req.PHPVersion == "" {
		req.PHPVersion = "8.5"
	}

	// 1. Validate domain
	s.recordEvent(ctx, req.Username, "validate_domain", "started", "Validating domain", "")
	if err := s.Validator.Validate(req.Domain); err != nil {
		s.recordEvent(ctx, req.Username, "validate_domain", "failed", err.Error(), "")
		return nil, fmt.Errorf("domain validation failed: %w", err)
	}
	s.recordEvent(ctx, req.Username, "validate_domain", "completed", "Domain valid", "")

	// 2. Create job
	job := NewJob(req.AccountID, req.TenantID, req.Username, req.Domain)
	if s.Jobs != nil {
		if err := s.Jobs.Create(job); err != nil {
			s.Logger.Error("failed to create provisioning job", "error", err)
		}
	}

	// Start job
	now := time.Now().UTC()
	job.Status = JobStatusRunning
	job.StartedAt = &now
	s.updateJob(ctx, job)

	// 3. Create directory structure
	if err := s.createDirectories(ctx, job); err != nil {
		s.failJob(ctx, job, "create_directories", err)
		s.rollbackDirectories(ctx, job)
		return job, err
	}

	// 4. Write nginx vhost
	if err := s.writeNginxVHost(ctx, job, req.PHPVersion); err != nil {
		s.failJob(ctx, job, "write_nginx_vhost", err)
		s.rollbackAll(ctx, job)
		return job, err
	}

	// 5. Write PHP-FPM pool
	if err := s.writeFPMPool(ctx, job, req.PHPVersion); err != nil {
		s.failJob(ctx, job, "write_fpm_pool", err)
		s.rollbackAll(ctx, job)
		return job, err
	}

	// 6. Test nginx
	s.recordEvent(ctx, job.Username, "test_nginx", "started", "Testing nginx configuration", "")
	if err := s.Webserver.TestNginx(ctx); err != nil {
		s.recordEvent(ctx, job.Username, "test_nginx", "failed", err.Error(), "")
		s.failJob(ctx, job, "test_nginx", err)
		s.rollbackAll(ctx, job)
		return job, fmt.Errorf("nginx test failed: %w", err)
	}
	s.recordEvent(ctx, job.Username, "test_nginx", "completed", "Nginx configuration valid", "")

	// 7. Test PHP-FPM
	s.recordEvent(ctx, job.Username, "test_php", "started", "Testing PHP-FPM configuration", "")
	if err := s.Webserver.TestPHP(ctx); err != nil {
		s.recordEvent(ctx, job.Username, "test_php", "failed", err.Error(), "")
		s.failJob(ctx, job, "test_php", err)
		s.rollbackAll(ctx, job)
		return job, fmt.Errorf("PHP-FPM test failed: %w", err)
	}
	s.recordEvent(ctx, job.Username, "test_php", "completed", "PHP-FPM configuration valid", "")

	// 8. Reload services
	if err := s.reloadServices(ctx, job); err != nil {
		s.failJob(ctx, job, "reload_services", err)
		s.rollbackAll(ctx, job)
		return job, err
	}

	// 9. Complete job
	job.Status = JobStatusCompleted
	job.CompletedAt = ptr(time.Now().UTC())
	s.updateJob(ctx, job)
	s.recordEvent(ctx, job.Username, "provision_complete", "completed", "Website provisioned successfully", "")

	return job, nil
}

// createDirectories creates the required directory structure.
func (s *Service) createDirectories(ctx context.Context, job *ProvisioningJob) error {
	username := job.Username
	domain := job.Domain

	// Directory paths
	publicHTML := fmt.Sprintf("%s/%s/%s/public_html", s.Paths.WebRoot, username, domain)
	logs := fmt.Sprintf("%s/%s/%s/logs", s.Paths.LogDir, username, domain)
	tmp := fmt.Sprintf("%s/%s/%s/tmp", s.Paths.LogDir, username, domain)
	ssl := fmt.Sprintf("%s/%s/%s/ssl", s.Paths.LogDir, username, domain)

	dirs := []struct {
		path  string
		mode  int
		owner string
	}{
		{publicHTML, 0755, username},
		{logs, 0755, username},
		{tmp, 0755, username},
		{ssl, 0750, username},
	}

	for _, d := range dirs {
		s.recordEvent(ctx, username, "create_dir:"+d.path, "started", "Creating directory", "")
		if err := s.System.CreateDir(ctx, d.path, d.mode, d.owner); err != nil {
			s.recordEvent(ctx, username, "create_dir:"+d.path, "failed", err.Error(), "")
			return fmt.Errorf("create dir %s: %w", d.path, err)
		}
		s.recordEvent(ctx, username, "create_dir:"+d.path, "completed", "Directory created", "")
	}

	// Create placeholder index.html
	indexPath := fmt.Sprintf("%s/index.html", publicHTML)
	indexContent := []byte("<!DOCTYPE html>\n<html>\n<head>\n    <title>" + domain + "</title>\n</head>\n<body>\n    <h1>Welcome to " + domain + "</h1>\n    <p>OrvixPanel hosting.</p>\n</body>\n</html>\n")
	if err := s.System.WriteFile(ctx, indexPath, indexContent, 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}
	if err := s.System.Chown(ctx, indexPath, username); err != nil {
		return fmt.Errorf("chown index.html: %w", err)
	}

	return nil
}

// writeNginxVHost generates and writes the nginx vhost configuration.
func (s *Service) writeNginxVHost(ctx context.Context, job *ProvisioningJob, phpVersion string) error {
	username := job.Username
	domain := job.Domain

	vhostPath := fmt.Sprintf("%s/%s-%s.conf", s.Paths.NginxDir, username, domain)
	socketPath := fmt.Sprintf("%s/orvix-%s-%s.sock", s.Paths.PHPSocketDir, username, domain)
	publicHTML := fmt.Sprintf("%s/%s/%s/public_html", s.Paths.WebRoot, username, domain)

	// Generate nginx config
	config := generateNginxConfig(username, domain, publicHTML, socketPath)

	s.recordEvent(ctx, username, "write_nginx_vhost", "started", "Writing nginx vhost", vhostPath)
	if err := s.Webserver.WriteNginxVHost(ctx, vhostPath, config); err != nil {
		s.recordEvent(ctx, username, "write_nginx_vhost", "failed", err.Error(), vhostPath)
		return fmt.Errorf("write nginx vhost: %w", err)
	}
	s.recordEvent(ctx, username, "write_nginx_vhost", "completed", "Nginx vhost written", vhostPath)

	return nil
}

// writeFPMPool generates and writes the PHP-FPM pool configuration.
func (s *Service) writeFPMPool(ctx context.Context, job *ProvisioningJob, phpVersion string) error {
	username := job.Username
	domain := job.Domain

	poolPath := fmt.Sprintf("%s/orvix-%s-%s.conf", s.Paths.FPMPoolDir, username, domain)
	socketPath := fmt.Sprintf("%s/orvix-%s-%s.sock", s.Paths.PHPSocketDir, username, domain)
	publicHTML := fmt.Sprintf("%s/%s/%s/public_html", s.Paths.WebRoot, username, domain)

	// Generate PHP-FPM pool config
	config := generateFPMPoolConfig(username, domain, socketPath, publicHTML, phpVersion)

	s.recordEvent(ctx, username, "write_fpm_pool", "started", "Writing PHP-FPM pool", poolPath)
	if err := s.Webserver.WriteFPMPool(ctx, poolPath, config); err != nil {
		s.recordEvent(ctx, username, "write_fpm_pool", "failed", err.Error(), poolPath)
		return fmt.Errorf("write fpm pool: %w", err)
	}
	s.recordEvent(ctx, username, "write_fpm_pool", "completed", "PHP-FPM pool written", poolPath)

	return nil
}

// reloadServices reloads nginx and PHP-FPM.
func (s *Service) reloadServices(ctx context.Context, job *ProvisioningJob) error {
	s.recordEvent(ctx, job.Username, "reload_nginx", "started", "Reloading nginx", "")
	if err := s.Webserver.ReloadNginx(ctx); err != nil {
		s.recordEvent(ctx, job.Username, "reload_nginx", "failed", err.Error(), "")
		return fmt.Errorf("reload nginx: %w", err)
	}
	s.recordEvent(ctx, job.Username, "reload_nginx", "completed", "Nginx reloaded", "")

	s.recordEvent(ctx, job.Username, "reload_php", "started", "Reloading PHP-FPM", "")
	if err := s.Webserver.ReloadPHP(ctx); err != nil {
		s.recordEvent(ctx, job.Username, "reload_php", "failed", err.Error(), "")
		return fmt.Errorf("reload php: %w", err)
	}
	s.recordEvent(ctx, job.Username, "reload_php", "completed", "PHP-FPM reloaded", "")

	return nil
}

// rollbackDirectories removes created directories (best effort).
func (s *Service) rollbackDirectories(ctx context.Context, job *ProvisioningJob) {
	s.recordEvent(ctx, job.Username, "rollback_directories", "started", "Rolling back directories", "")

	publicHTML := fmt.Sprintf("%s/%s/%s", s.Paths.WebRoot, job.Username, job.Domain)
	logs := fmt.Sprintf("%s/%s/%s", s.Paths.LogDir, job.Username, job.Domain)

	_ = s.System.RemoveDir(ctx, publicHTML)
	_ = s.System.RemoveDir(ctx, logs)

	s.recordEvent(ctx, job.Username, "rollback_directories", "rolled_back", "Directories removed", "")
}

// rollbackAll removes all provisioned resources.
func (s *Service) rollbackAll(ctx context.Context, job *ProvisioningJob) {
	s.recordEvent(ctx, job.Username, "rollback_all", "started", "Starting full rollback", "")

	// Remove nginx vhost
	vhostPath := fmt.Sprintf("%s/%s-%s.conf", s.Paths.NginxDir, job.Username, job.Domain)
	_ = s.Webserver.RemoveNginxVHost(ctx, vhostPath)

	// Remove PHP-FPM pool
	poolPath := fmt.Sprintf("%s/orvix-%s-%s.conf", s.Paths.FPMPoolDir, job.Username, job.Domain)
	_ = s.Webserver.RemoveFPMPool(ctx, poolPath)

	// Remove directories
	s.rollbackDirectories(ctx, job)

	// Update job status
	job.Status = JobStatusRolledBack
	s.updateJob(ctx, job)

	s.recordEvent(ctx, job.Username, "rollback_all", "rolled_back", "Full rollback complete", "")
}

// failJob marks a job as failed and records the error.
func (s *Service) failJob(ctx context.Context, job *ProvisioningJob, step string, err error) {
	job.Status = JobStatusFailed
	job.ErrorMsg = fmt.Sprintf("%s: %v", step, err)
	s.updateJob(ctx, job)
	s.recordEvent(ctx, job.Username, step, "failed", err.Error(), "")
}

// updateJob persists the job update.
func (s *Service) updateJob(ctx context.Context, job *ProvisioningJob) {
	if s.Jobs != nil {
		_ = s.Jobs.Update(job)
	}
}

// recordEvent records a provisioning event.
func (s *Service) recordEvent(ctx context.Context, username, step, status, message, details string) {
	event := NewEvent("", step, status, message)
	event.Details = details
	if s.Events != nil {
		_ = s.Events.Record(event)
	}
	if s.Logger != nil {
		s.Logger.Debug("provisioning event",
			"username", username,
			"step", step,
			"status", status,
			"message", message,
		)
	}
}

// generateNginxConfig generates the nginx vhost configuration.
func generateNginxConfig(username, domain, documentRoot, fpmSocket string) string {
	return fmt.Sprintf(`# OrvixPanel generated vhost
# account=%s domain=%s
server {
    listen 80;
    listen [::]:80;
    server_name %s;

    root %s;
    index index.html index.php;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:%s;
        fastcgi_param PHP_VALUE "open_basedir=%s";
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~ /\. {
        deny all;
    }

    access_log /var/log/nginx/%s-%s.access.log;
    error_log  /var/log/nginx/%s-%s.error.log;
}
`, username, domain, domain, documentRoot, fpmSocket, documentRoot, username, domain, username, domain)
}

// generateFPMPoolConfig generates the PHP-FPM pool configuration.
func generateFPMPoolConfig(username, domain, socketPath, documentRoot, phpVersion string) string {
	return fmt.Sprintf(`; OrvixPanel generated PHP-FPM pool
; account=%s domain=%s
[%s-%s]
user = %s
group = %s

listen = %s
listen.owner = www-data
listen.group = www-data
listen.mode = 0660

pm = ondemand
pm.max_children = 10
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 5
pm.process_idle_timeout = 30s
pm.max_requests = 500

php_admin_value[open_basedir] = %s
php_admin_value[upload_tmp_dir] = %s/%s/tmp
php_admin_value[session.save_path] = %s/%s/tmp
php_admin_value[error_log] = %s/%s/logs/php-error.log

chdir = %s
`, username, domain,
		username, domain,
		username, username,
		socketPath,
		documentRoot,
		documentRoot, username, domain,
		documentRoot, username, domain,
		documentRoot)
}

// ptr returns a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}