package provisioning

import (
	"context"
	"strings"
	"testing"
)

// FakeSystemExecutor is a test double for SystemExecutor.
type FakeSystemExecutor struct {
	CreatedDirs  []string
	RemovedDirs  []string
	WrittenFiles map[string][]byte
	RemovedFiles []string
	ChownedPaths []string
	Errors       map[string]error // key is operation name
}

func NewFakeSystemExecutor() *FakeSystemExecutor {
	return &FakeSystemExecutor{
		CreatedDirs:  []string{},
		RemovedDirs:  []string{},
		WrittenFiles: make(map[string][]byte),
		RemovedFiles: []string{},
		ChownedPaths: []string{},
		Errors:       make(map[string]error),
	}
}

func (f *FakeSystemExecutor) CreateUser(ctx context.Context, username, homeDir string) error {
	if err := f.Errors["CreateUser"]; err != nil {
		return err
	}
	return nil
}

func (f *FakeSystemExecutor) DeleteUser(ctx context.Context, username string) error {
	if err := f.Errors["DeleteUser"]; err != nil {
		return err
	}
	return nil
}

func (f *FakeSystemExecutor) CreateDir(ctx context.Context, path string, mode int, owner string) error {
	if err := f.Errors["CreateDir"]; err != nil {
		return err
	}
	f.CreatedDirs = append(f.CreatedDirs, path)
	return nil
}

func (f *FakeSystemExecutor) RemoveDir(ctx context.Context, path string) error {
	if err := f.Errors["RemoveDir"]; err != nil {
		return err
	}
	f.RemovedDirs = append(f.RemovedDirs, path)
	return nil
}

func (f *FakeSystemExecutor) WriteFile(ctx context.Context, path string, content []byte, mode int) error {
	if err := f.Errors["WriteFile"]; err != nil {
		return err
	}
	f.WrittenFiles[path] = content
	return nil
}

func (f *FakeSystemExecutor) RemoveFile(ctx context.Context, path string) error {
	if err := f.Errors["RemoveFile"]; err != nil {
		return err
	}
	f.RemovedFiles = append(f.RemovedFiles, path)
	return nil
}

func (f *FakeSystemExecutor) Chown(ctx context.Context, path string, owner string) error {
	if err := f.Errors["Chown"]; err != nil {
		return err
	}
	f.ChownedPaths = append(f.ChownedPaths, path)
	return nil
}

func (f *FakeSystemExecutor) RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	if err := f.Errors["RunCommand"]; err != nil {
		return "", err
	}
	return "ok", nil
}

// FakeWebserverExecutor is a test double for WebserverExecutor.
type FakeWebserverExecutor struct {
	WrittenVHosts map[string]string
	WrittenPools  map[string]string
	RemovedVHosts []string
	RemovedPools  []string
	NginxTestErr  error
	PHPTestErr    error
	NginxReloadErr error
	PHPReloadErr   error
	CalledOrder   []string
}

func NewFakeWebserverExecutor() *FakeWebserverExecutor {
	return &FakeWebserverExecutor{
		WrittenVHosts: make(map[string]string),
		WrittenPools:  make(map[string]string),
		RemovedVHosts: []string{},
		RemovedPools:  []string{},
		CalledOrder:   []string{},
	}
}

func (f *FakeWebserverExecutor) WriteNginxVHost(ctx context.Context, path string, content string) error {
	if f.NginxTestErr != nil {
		return f.NginxTestErr
	}
	f.WrittenVHosts[path] = content
	f.CalledOrder = append(f.CalledOrder, "WriteNginxVHost:"+path)
	return nil
}

func (f *FakeWebserverExecutor) RemoveNginxVHost(ctx context.Context, path string) error {
	f.RemovedVHosts = append(f.RemovedVHosts, path)
	return nil
}

func (f *FakeWebserverExecutor) TestNginx(ctx context.Context) error {
	f.CalledOrder = append(f.CalledOrder, "TestNginx")
	if f.NginxTestErr != nil {
		return f.NginxTestErr
	}
	return nil
}

func (f *FakeWebserverExecutor) ReloadNginx(ctx context.Context) error {
	f.CalledOrder = append(f.CalledOrder, "ReloadNginx")
	if f.NginxReloadErr != nil {
		return f.NginxReloadErr
	}
	return nil
}

func (f *FakeWebserverExecutor) WriteFPMPool(ctx context.Context, path string, content string) error {
	if f.PHPTestErr != nil {
		return f.PHPTestErr
	}
	f.WrittenPools[path] = content
	f.CalledOrder = append(f.CalledOrder, "WriteFPMPool:"+path)
	return nil
}

func (f *FakeWebserverExecutor) RemoveFPMPool(ctx context.Context, path string) error {
	f.RemovedPools = append(f.RemovedPools, path)
	return nil
}

func (f *FakeWebserverExecutor) TestPHP(ctx context.Context) error {
	f.CalledOrder = append(f.CalledOrder, "TestPHP")
	if f.PHPTestErr != nil {
		return f.PHPTestErr
	}
	return nil
}

func (f *FakeWebserverExecutor) ReloadPHP(ctx context.Context) error {
	f.CalledOrder = append(f.CalledOrder, "ReloadPHP")
	if f.PHPReloadErr != nil {
		return f.PHPReloadErr
	}
	return nil
}

// FakeJobStore is a test double for JobStore.
type FakeJobStore struct {
	Jobs     map[string]*ProvisioningJob
	CreateErr error
	UpdateErr error
}

func NewFakeJobStore() *FakeJobStore {
	return &FakeJobStore{
		Jobs: make(map[string]*ProvisioningJob),
	}
}

func (f *FakeJobStore) Create(job *ProvisioningJob) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	f.Jobs[job.ID] = job
	return nil
}

func (f *FakeJobStore) Update(job *ProvisioningJob) error {
	if f.UpdateErr != nil {
		return f.UpdateErr
	}
	f.Jobs[job.ID] = job
	return nil
}

func (f *FakeJobStore) GetByID(id string) (*ProvisioningJob, error) {
	return f.Jobs[id], nil
}

func (f *FakeJobStore) GetByAccount(accountID string) ([]*ProvisioningJob, error) {
	var result []*ProvisioningJob
	for _, j := range f.Jobs {
		if j.AccountID == accountID {
			result = append(result, j)
		}
	}
	return result, nil
}

// FakeEventRecorder is a test double for EventRecorder.
type FakeEventRecorder struct {
	Events []*ProvisioningEvent
}

func NewFakeEventRecorder() *FakeEventRecorder {
	return &FakeEventRecorder{
		Events: []*ProvisioningEvent{},
	}
}

func (f *FakeEventRecorder) Record(event *ProvisioningEvent) error {
	f.Events = append(f.Events, event)
	return nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestDomainValidator_ValidDomains(t *testing.T) {
	v := DefaultDomainValidator()
	validDomains := []string{
		"example.com",
		"www.example.com",
		"sub.domain.example.com",
		"my-site123.com",
		"a.co",
	}
	for _, d := range validDomains {
		if err := v.Validate(d); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", d, err)
		}
	}
}

func TestDomainValidator_InvalidDomains(t *testing.T) {
	v := DefaultDomainValidator()
	invalidDomains := []struct {
		domain string
		msg    string
	}{
		{"", "empty domain"},
		{"..", "path traversal"},
		{"../etc/passwd", "path traversal"},
		{"/var/www", "path traversal"},
		{"example..com", "empty label"},
		{".example.com", "empty label"},
		{"example.com.", "trailing dot allowed"},
		{"a b.com", "space"},
		{"example.com/path", "path"},
		{"example&.com", "invalid char"},
		{"localhost", "reserved"},
		{"example.com:8080", "port not allowed"},
	}
	for _, tc := range invalidDomains {
		err := v.Validate(tc.domain)
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error (%s)", tc.domain, tc.msg)
		}
	}
}

func TestDomainValidator_PathTraversal(t *testing.T) {
	v := DefaultDomainValidator()
	traversalAttempts := []string{
		"..",
		"../",
		"/..",
		"\\",
		"%2e%2e",
		"%252e",
		"foo..bar.com",
		"foo%2ebar.com",
	}
	for _, d := range traversalAttempts {
		err := v.Validate(d)
		if err == nil {
			t.Errorf("Validate(%q) should reject path traversal", d)
		}
		if !strings.Contains(err.Error(), "path traversal") && !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Validate(%q) = %v, want path traversal error", d, err)
		}
	}
}

func TestDomainValidator_Wildcard(t *testing.T) {
	v := DefaultDomainValidator()

	// Wildcard not allowed by default
	if err := v.Validate("*.example.com"); err == nil {
		t.Error("Validate(*.example.com) should reject wildcard by default")
	}

	// Wildcard allowed when configured
	v.AllowWildcard = true
	if err := v.Validate("*.example.com"); err != nil {
		t.Errorf("Validate(*.example.com) with AllowWildcard=true = %v, want nil", err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		domain  string
		want    string
	}{
		{"example.com", "example_com"},
		{"www.example.com", "www_example_com"},
		{"my-site123.com", "my-site123_com"},
		{"sub.domain.example.com", "sub_domain_example_com"},
	}
	for _, tc := range cases {
		got := SanitizeFilename(tc.domain)
		if got != tc.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tc.domain, got, tc.want)
		}
	}
}

func TestNewJob(t *testing.T) {
	job := NewJob("acc1", "ten1", "user1", "example.com")
	if job.ID == "" {
		t.Error("NewJob() should set ID")
	}
	if job.AccountID != "acc1" {
		t.Errorf("NewJob().AccountID = %q, want %q", job.AccountID, "acc1")
	}
	if job.Status != JobStatusPending {
		t.Errorf("NewJob().Status = %q, want %q", job.Status, JobStatusPending)
	}
}

func TestProvisioningService_CreateDirectories(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	svc := &Service{
		Paths:   DefaultPaths(),
		System:  sys,
		Webserver: ws,
		Jobs:    store,
		Events:  events,
	}

	job := NewJob("acc1", "ten1", "user1", "example.com")
	err := svc.createDirectories(context.Background(), job)
	if err != nil {
		t.Fatalf("createDirectories() = %v, want nil", err)
	}

	// Check directories created
	expectedDirs := []string{
		"/var/www/user1/example.com/public_html",
		"/var/log/orvixpanel/user1/example.com/logs",
		"/var/log/orvixpanel/user1/example.com/tmp",
		"/var/log/orvixpanel/user1/example.com/ssl",
	}
	for _, d := range expectedDirs {
		found := false
		for _, created := range sys.CreatedDirs {
			if created == d {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("createDirectories() did not create dir %q. Created: %v", d, sys.CreatedDirs)
		}
	}

	// Check index.html written
	indexPath := "/var/www/user1/example.com/public_html/index.html"
	if _, ok := sys.WrittenFiles[indexPath]; !ok {
		t.Errorf("createDirectories() did not write index.html at %q", indexPath)
	}
}

func TestProvisioningService_WriteNginxVHost(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	svc := &Service{
		Paths:    DefaultPaths(),
		System:   sys,
		Webserver: ws,
		Jobs:     store,
		Events:   events,
	}

	job := NewJob("acc1", "ten1", "user1", "example.com")
	err := svc.writeNginxVHost(context.Background(), job, "8.5")
	if err != nil {
		t.Fatalf("writeNginxVHost() = %v, want nil", err)
	}

	// Check vhost written
	vhostPath := "/etc/nginx/conf.d/orvix/user1-example.com.conf"
	if content, ok := ws.WrittenVHosts[vhostPath]; !ok {
		t.Errorf("writeNginxVHost() did not write vhost at %q", vhostPath)
	} else {
		// Verify config contains expected values
		if !strings.Contains(content, "server_name example.com") {
			t.Errorf("nginx config missing server_name")
		}
		if !strings.Contains(content, "root /var/www/user1/example.com/public_html") {
			t.Errorf("nginx config missing correct root")
		}
		if !strings.Contains(content, "fastcgi_pass unix:/run/php/orvix-user1-example.com.sock") {
			t.Errorf("nginx config missing correct fastcgi_pass")
		}
	}
}

func TestProvisioningService_WriteFPMPool(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	svc := &Service{
		Paths:    DefaultPaths(),
		System:   sys,
		Webserver: ws,
		Jobs:     store,
		Events:   events,
	}

	job := NewJob("acc1", "ten1", "user1", "example.com")
	err := svc.writeFPMPool(context.Background(), job, "8.5")
	if err != nil {
		t.Fatalf("writeFPMPool() = %v, want nil", err)
	}

	// Check pool written
	poolPath := "/etc/php/8.5/fpm/pool.d/orvix-user1-example.com.conf"
	if content, ok := ws.WrittenPools[poolPath]; !ok {
		t.Errorf("writeFPMPool() did not write pool at %q", poolPath)
	} else {
		// Verify config contains expected values
		if !strings.Contains(content, "[user1-example.com]") {
			t.Errorf("PHP-FPM pool missing correct pool name")
		}
		if !strings.Contains(content, "user = user1") {
			t.Errorf("PHP-FPM pool missing correct user")
		}
		if !strings.Contains(content, "listen = /run/php/orvix-user1-example.com.sock") {
			t.Errorf("PHP-FPM pool missing correct socket path")
		}
	}
}

func TestProvisioningService_Rollback(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	svc := &Service{
		Paths:    DefaultPaths(),
		System:   sys,
		Webserver: ws,
		Jobs:     store,
		Events:   events,
	}

	job := NewJob("acc1", "ten1", "user1", "example.com")
	job.Status = JobStatusFailed
	job.ErrorMsg = "test error"

	svc.rollbackAll(context.Background(), job)

	// Check vhost removed
	if len(ws.RemovedVHosts) != 1 {
		t.Errorf("rollbackAll() removed %d vhosts, want 1", len(ws.RemovedVHosts))
	}

	// Check pool removed
	if len(ws.RemovedPools) != 1 {
		t.Errorf("rollbackAll() removed %d pools, want 1", len(ws.RemovedPools))
	}

	// Check directories removed
	if len(sys.RemovedDirs) < 1 {
		t.Errorf("rollbackAll() did not remove directories")
	}
}

func TestProvisioningService_ProvisionWebsite_NginxFailure(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	// Make nginx test fail
	ws.NginxTestErr = &CommandError{
		Command: []string{"nginx", "-t"},
		Output:  "nginx: configuration file test failed",
		Err:     nil,
	}

	svc := &Service{
		Paths:     DefaultPaths(),
		System:    sys,
		Webserver: ws,
		Jobs:      store,
		Events:    events,
		Validator: DefaultDomainValidator(),
	}

	req := ProvisionWebsiteRequest{
		AccountID: "acc1",
		TenantID:  "ten1",
		Username:  "user1",
		Domain:    "example.com",
	}

	job, err := svc.ProvisionWebsite(context.Background(), req)
	if err == nil {
		t.Fatalf("ProvisionWebsite() = nil, want error on nginx failure")
	}
	if job == nil {
		t.Fatal("ProvisionWebsite() returned nil job on error")
	}
	if job.Status != JobStatusRolledBack {
		t.Errorf("job.Status = %q, want %q", job.Status, JobStatusRolledBack)
	}

	// Verify rollback happened
	if len(ws.RemovedVHosts) != 1 {
		t.Errorf("On nginx failure, rollback should remove vhost. Removed: %v", ws.RemovedVHosts)
	}
}

func TestProvisioningService_ProvisionWebsite_Success(t *testing.T) {
	sys := NewFakeSystemExecutor()
	ws := NewFakeWebserverExecutor()
	store := NewFakeJobStore()
	events := NewFakeEventRecorder()

	svc := &Service{
		Paths:     DefaultPaths(),
		System:    sys,
		Webserver: ws,
		Jobs:      store,
		Events:    events,
		Validator: DefaultDomainValidator(),
	}

	req := ProvisionWebsiteRequest{
		AccountID:  "acc1",
		TenantID:   "ten1",
		Username:   "user1",
		Domain:     "example.com",
		PHPVersion: "8.5",
	}

	job, err := svc.ProvisionWebsite(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionWebsite() = %v, want nil", err)
	}
	if job == nil {
		t.Fatal("ProvisionWebsite() returned nil job")
	}
	if job.Status != JobStatusCompleted {
		t.Errorf("job.Status = %q, want %q", job.Status, JobStatusCompleted)
	}

	// Verify all steps happened in order
	expectedOrder := []string{
		"WriteNginxVHost",
		"WriteFPMPool",
		"TestNginx",
		"TestPHP",
		"ReloadNginx",
		"ReloadPHP",
	}
	for i, expected := range expectedOrder {
		if i >= len(ws.CalledOrder) {
			t.Errorf("Missing step %s in call order. Got: %v", expected, ws.CalledOrder)
			continue
		}
		if !strings.Contains(ws.CalledOrder[i], expected) {
			t.Errorf("Step %d: expected %s, got %s", i, expected, ws.CalledOrder[i])
		}
	}
}

func TestGenerateNginxConfig(t *testing.T) {
	config := generateNginxConfig("user1", "example.com", "/var/www/user1/example.com/public_html", "/run/php/orvix-user1-example.com.sock")

	// Security checks
	if strings.Contains(config, "sh -c") {
		t.Error("nginx config should not contain shell execution")
	}
	if strings.Contains(config, "eval") {
		t.Error("nginx config should not contain eval")
	}

	// Required elements
	checks := []string{
		"server_name example.com",
		"root /var/www/user1/example.com/public_html",
		"fastcgi_pass unix:/run/php/orvix-user1-example.com.sock",
		"open_basedir=/var/www/user1/example.com/public_html",
		"X-Content-Type-Options",
		"deny all",
	}
	for _, check := range checks {
		if !strings.Contains(config, check) {
			t.Errorf("nginx config missing: %s", check)
		}
	}
}

func TestGenerateFPMPoolConfig(t *testing.T) {
	config := generateFPMPoolConfig("user1", "example.com", "/run/php/orvix-user1-example.com.sock", "/var/www/user1/example.com/public_html", "8.5")

	// Security checks
	if strings.Contains(config, "sh -c") {
		t.Error("fpm config should not contain shell execution")
	}

	// Required elements
	checks := []string{
		"[user1-example.com]",
		"user = user1",
		"group = user1",
		"listen = /run/php/orvix-user1-example.com.sock",
		"open_basedir",
	}
	for _, check := range checks {
		if !strings.Contains(config, check) {
			t.Errorf("fpm config missing: %s", check)
		}
	}
}