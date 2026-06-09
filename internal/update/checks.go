package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// PreflightChecks runs all pre-update checks.
func PreflightChecks(cfg *UpdateConfig) ([]PreflightCheck, error) {
	checks := []PreflightCheck{
		checkRootUser(),
		checkOS(),
		checkDiskSpace(),
		checkGoInstalled(),
		checkNodeInstalled(),
		checkGitInstalled(),
		checkNginxInstalled(),
		checkSystemdService(),
		checkEnvFile(),
		checkNginxConfig(),
		checkExistingInstall(),
		// v0.7.2: Runtime storage verification
		checkRuntimeStorage(),
		checkRuntimeHealth(),
		checkRuntimeVersion(),
	}

	// Add suggestions for failing checks
	for i := range checks {
		addSuggestions(&checks[i])
	}

	return checks, nil
}

// RunChecks runs checks and returns error if any fail.
func RunChecks(cfg *UpdateConfig) error {
	checks, err := PreflightChecks(cfg)
	if err != nil {
		return fmt.Errorf("preflight checks: %w", err)
	}

	var failed int
	for _, c := range checks {
		if c.Status.IsBlocking() {
			failed++
			fmt.Printf("  [FAIL] %s: %s\n", c.Name, c.Message)
			if c.Suggestions != nil {
				for _, s := range c.Suggestions {
					fmt.Printf("         Suggestion: %s\n", s)
				}
			}
		} else if c.Status == CheckWarn {
			fmt.Printf("  [WARN] %s: %s\n", c.Name, c.Message)
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d preflight check(s) failed", failed)
	}

	return nil
}

// checkRootUser verifies the process is running as root.
func checkRootUser() PreflightCheck {
	check := PreflightCheck{Name: "Root User"}

	if os.Geteuid() != 0 {
		check.Status = CheckFail
		check.Message = "Update requires root privileges"
		check.Details = fmt.Sprintf("Current UID: %d", os.Geteuid())
		check.Suggestions = []string{
			"Run with sudo: sudo orvixpanel update",
			"Or run as root user",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = "Running as root"
	return check
}

// checkOS verifies the OS is Linux and extracts distribution info.
func checkOS() PreflightCheck {
	check := PreflightCheck{Name: "Operating System"}

	if runtime.GOOS != "linux" {
		check.Status = CheckFail
		check.Message = fmt.Sprintf("Unsupported OS: %s", runtime.GOOS)
		check.Details = "OrvixPanel update is only supported on Linux"
		return check
	}

	// Try to read /etc/os-release
	var distName, distID string
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "NAME=") {
				distName = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
			}
			if strings.HasPrefix(line, "ID=") {
				distID = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}

	// Check for supported distributions
	supported := map[string]bool{
		"ubuntu":      true,
		"debian":      true,
		"centos":      true,
		"rhel":        true,
		"rocky":       true,
		"almalinux":   true,
		"fedora":      true,
		"arch":        true,
	}

	if distID != "" && !supported[distID] {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("Distribution may not be fully tested: %s", distName)
		check.Details = fmt.Sprintf("Detected: %s (%s)", distName, distID)
		check.Suggestions = []string{
			"Ensure you have tested this update in a non-production environment",
			"Report any issues to the OrvixPanel team",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = fmt.Sprintf("OS: %s", distName)
	if distName == "" {
		check.Message = "OS: Linux"
	}
	return check
}

// checkDiskSpace checks available disk space in critical directories.
func checkDiskSpace() PreflightCheck {
	check := PreflightCheck{Name: "Disk Space"}

	type pathCheck struct {
		path  string
		minGB float64
	}

	paths := []pathCheck{
		{"/", 2.0},                 // Root filesystem
		{InstallBase, 1.0},         // Installation directory
		{"/var", 1.0},              // Var directory
		{"/tmp", 0.5},              // Temp directory
	}

	var insufficient []string
	for _, pc := range paths {
		avail, err := getAvailableSpace(pc.path)
		if err != nil {
			insufficient = append(insufficient, fmt.Sprintf("%s: cannot check (%v)", pc.path, err))
			continue
		}

		availGB := float64(avail) / 1e9
		if availGB < pc.minGB {
			insufficient = append(insufficient,
				fmt.Sprintf("%s: %.1fGB available (need %.1fGB)", pc.path, availGB, pc.minGB))
		}
	}

	if len(insufficient) > 0 {
		check.Status = CheckFail
		check.Message = "Insufficient disk space"
		check.Details = strings.Join(insufficient, "; ")
		check.Suggestions = []string{
			"Free up disk space before updating",
			"Consider cleaning old backups: find /var/backups/orvixpanel -mtime +30 -delete",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = "Sufficient disk space available"
	return check
}

func getAvailableSpace(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

// checkGoInstalled verifies Go is installed and checks version.
func checkGoInstalled() PreflightCheck {
	check := PreflightCheck{Name: "Go Compiler"}

	path, err := exec.LookPath("go")
	if err != nil {
		check.Status = CheckFail
		check.Message = "Go not found"
		check.Details = "Go is required to build OrvixPanel"
		check.Suggestions = []string{
			"Install Go: apt install golang-go (Debian/Ubuntu)",
			"Or: yum install golang (RHEL/CentOS)",
			"Minimum version: Go 1.21",
		}
		return check
	}

	output, err := exec.Command("go", "version").Output()
	if err != nil {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("Go found at %s but version check failed", path)
		check.Details = err.Error()
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = CheckPass
	check.Message = version

	// Check minimum version
	versionNum := extractGoVersion(string(output))
	if versionNum < 121 {
		check.Status = CheckWarn
		check.Message = version + " (upgrade recommended)"
		check.Suggestions = []string{
			"OrvixPanel v0.7.1 requires Go 1.21 or later",
			"Consider upgrading: https://go.dev/dl/",
		}
	}

	return check
}

func extractGoVersion(output string) int {
	// Parse "go version go1.21.5 linux/amd64"
	parts := strings.Split(output, "go")
	if len(parts) < 2 {
		return 0
	}
	ver := strings.Split(parts[1], ".")
	if len(ver) < 1 {
		return 0
	}
	major, _ := strconv.Atoi(strings.TrimPrefix(ver[0], "go"))
	return major
}

// checkNodeInstalled verifies Node.js and pnpm are installed.
func checkNodeInstalled() PreflightCheck {
	check := PreflightCheck{Name: "Node.js / pnpm"}

	_, err := exec.LookPath("node")
	if err != nil {
		check.Status = CheckWarn
		check.Message = "Node.js not found (frontend update skipped)"
		check.Details = "Frontend will not be updated"
		return check
	}

	nodeOutput, _ := exec.Command("node", "--version").Output()
	nodeVersion := strings.TrimSpace(string(nodeOutput))

	_, err = exec.LookPath("pnpm")
	if err != nil {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("Node.js found (%s) but pnpm not found", nodeVersion)
		check.Details = "Frontend build requires pnpm"
		check.Suggestions = []string{
			"Install pnpm: npm install -g pnpm",
			"Or: curl -fsSL https://get.pnpm.io/install.sh | sh -",
		}
		return check
	}

	pnpmOutput, _ := exec.Command("pnpm", "--version").Output()
	pnpmVersion := strings.TrimSpace(string(pnpmOutput))

	check.Status = CheckPass
	check.Message = fmt.Sprintf("Node.js %s, pnpm %s", nodeVersion, pnpmVersion)
	return check
}

// checkGitInstalled verifies git is installed.
func checkGitInstalled() PreflightCheck {
	check := PreflightCheck{Name: "Git"}

	_, err := exec.LookPath("git")
	if err != nil {
		check.Status = CheckFail
		check.Message = "Git not found"
		check.Details = "Git is required to fetch updates"
		check.Suggestions = []string{
			"Install git: apt install git",
			"Or: yum install git",
		}
		return check
	}

	output, _ := exec.Command("git", "version").Output()
	version := strings.TrimSpace(string(output))

	check.Status = CheckPass
	check.Message = version

	// Check we're in a git repo if updating from existing checkout
	if IsInstalled() {
		gitDir := filepath.Join(InstallBase, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			// We're in a git repo, check for uncommitted changes
			output2, _ := exec.Command("git", "status", "--porcelain").Output()
			if len(bytes.TrimSpace(output2)) > 0 {
				check.Status = CheckWarn
				check.Message = version + " (uncommitted changes detected)"
				check.Suggestions = []string{
					"Consider committing or stashing changes before update",
					"Run: git stash",
				}
			}
		}
	}

	return check
}

// checkNginxInstalled verifies nginx is installed and running.
func checkNginxInstalled() PreflightCheck {
	check := PreflightCheck{Name: "nginx"}

	_, err := exec.LookPath("nginx")
	if err != nil {
		check.Status = CheckWarn
		check.Message = "nginx not found"
		check.Details = "nginx is required for the web frontend"
		check.Suggestions = []string{
			"Install nginx: apt install nginx",
			"Or: yum install nginx",
		}
		return check
	}

	output, _ := exec.Command("nginx", "-v").CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// Check if nginx is running
	running, _ := isProcessRunning("nginx")
	if !running {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("%s (not running)", outputStr)
		check.Details = "nginx is installed but not currently running"
		check.Suggestions = []string{
			"Start nginx: systemctl start nginx",
			"Enable on boot: systemctl enable nginx",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = outputStr + " (running)"
	return check
}

func isProcessRunning(name string) (bool, error) {
	output, err := exec.Command("pgrep", "-x", name).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil // pgrep returns 1 when no process found
		}
		return false, err
	}
	return len(bytes.TrimSpace(output)) > 0, nil
}

// checkSystemdService verifies the orvixpanel systemd service exists.
func checkSystemdService() PreflightCheck {
	check := PreflightCheck{Name: "Systemd Service"}

	servicePath := "/etc/systemd/system/orvixpanel.service"
	if _, err := os.Stat(servicePath); err != nil {
		check.Status = CheckWarn
		check.Message = "orvixpanel.service not found"
		check.Details = "Systemd service file not found at " + servicePath
		check.Suggestions = []string{
			"The service will be created/updated during installation",
			"Manual restart may be required after update",
		}
		return check
	}

	// Check if service is enabled
	output, _ := exec.Command("systemctl", "is-enabled", "orvixpanel.service").Output()
	enabled := strings.TrimSpace(string(output)) == "enabled"

	// Check if service is active
	output2, _ := exec.Command("systemctl", "is-active", "orvixpanel.service").Output()
	active := strings.TrimSpace(string(output2)) == "active"

	if !active {
		check.Status = CheckWarn
		check.Message = "orvixpanel.service exists but is not active"
		check.Suggestions = []string{
			"Start the service: systemctl start orvixpanel",
		}
		return check
	}

	if enabled {
		check.Status = CheckPass
		check.Message = "orvixpanel.service active and enabled"
	} else {
		check.Status = CheckWarn
		check.Message = "orvixpanel.service active but not enabled"
		check.Suggestions = []string{
			"Enable on boot: systemctl enable orvixpanel",
		}
	}

	return check
}

// checkEnvFile checks the .env file exists and has required variables.
func checkEnvFile() PreflightCheck {
	check := PreflightCheck{Name: "Environment File"}

	envFile := GetInstallPaths().EnvFile
	if _, err := os.Stat(envFile); err != nil {
		check.Status = CheckWarn
		check.Message = ".env file not found at " + envFile
		check.Details = "A new .env file will be created during update"
		return check
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		check.Status = CheckWarn
		check.Message = "Cannot read .env file"
		check.Details = err.Error()
		return check
	}

	// Check for required variables
	required := []string{"ORVIX_SERVER_SECRET_KEY"}
	missing := []string{}

	lines := strings.Split(string(data), "\n")
	envVars := make(map[string]bool)
	for _, line := range lines {
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if strings.HasPrefix(key, "#") {
				continue
			}
			envVars[key] = true
		}
	}

	for _, req := range required {
		if !envVars[req] {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("Missing required env vars: %s", strings.Join(missing, ", "))
		check.Details = "These will be auto-generated if missing"
		check.Suggestions = []string{
			"Add them to " + envFile + " before update",
			"They will be auto-generated if missing",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = ".env file valid"
	return check
}

// checkNginxConfig checks nginx configuration for common issues.
func checkNginxConfig() PreflightCheck {
	check := PreflightCheck{Name: "nginx Configuration"}

	// Test nginx configuration
	output, err := exec.Command("nginx", "-t").CombinedOutput()
	if err != nil {
		check.Status = CheckFail
		check.Message = "nginx configuration has errors"
		check.Details = strings.TrimSpace(string(output))
		check.Suggestions = []string{
			"Fix nginx configuration errors before updating",
			"Run: nginx -t for details",
		}
		return check
	}

	// Check for duplicate default_server
	sitesEnabled := "/etc/nginx/sites-enabled"
	if _, err := os.Stat(sitesEnabled); err == nil {
		entries, _ := os.ReadDir(sitesEnabled)
		defaultCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(sitesEnabled, entry.Name()))
			if err != nil {
				continue
			}
			if bytes.Contains(data, []byte("default_server")) {
				defaultCount++
			}
		}

		if defaultCount > 1 {
			check.Status = CheckWarn
			check.Message = fmt.Sprintf("Found %d sites with default_server", defaultCount)
			check.Details = "Multiple default_server directives may cause conflicts"
			check.Suggestions = []string{
				"Only one site should have 'default_server'",
				"Review sites in /etc/nginx/sites-enabled/",
			}
			return check
		}
	}

	check.Status = CheckPass
	check.Message = "nginx configuration valid"
	return check
}

// checkExistingInstall checks if OrvixPanel is already installed.
func checkExistingInstall() PreflightCheck {
	check := PreflightCheck{Name: "Existing Installation"}

	if !IsInstalled() {
		check.Status = CheckWarn
		check.Message = "OrvixPanel does not appear to be installed"
		check.Details = fmt.Sprintf("Installation directory %s does not exist", InstallBase)
		check.Suggestions = []string{
			"This appears to be a fresh install, not an update",
			"Use the install script instead: curl https://get.orvixpanel.com | bash",
		}
		return check
	}

	// Get current version if possible
	currentVersion := getCurrentVersion()
	if currentVersion.Tag != "" {
		check.Status = CheckPass
		check.Message = fmt.Sprintf("Found OrvixPanel v%s", currentVersion.Tag)
	} else {
		check.Status = CheckPass
		check.Message = "OrvixPanel installation detected"
	}

	return check
}

func getCurrentVersion() Version {
	p := GetInstallPaths()
	versionFile := filepath.Join(p.Base, "VERSION")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return Version{}
	}

	var v Version
	fmt.Sscanf(string(data), "tag=%s\ncommit=%s", &v.Tag, &v.Commit)
	return v
}

func addSuggestions(c *PreflightCheck) {
	if c.Suggestions == nil && c.Status == CheckFail {
		c.Suggestions = []string{
			"Resolve the issue above and try again",
			"Run 'orvixpanel doctor' for more diagnostics",
		}
	}
}

// checkRuntimeStorage verifies runtime storage directories from env file.
func checkRuntimeStorage() PreflightCheck {
	// Try to read runtime config
	runtimeCfg, err := ReadRuntimeConfig()
	if err != nil {
		// Fall back to default paths
		p := GetInstallPaths()
		return checkRuntimeStoragePaths(p.Var, p.Log, p.Backup, filepath.Join(p.Var, "orvixpanel.db"))
	}

	return checkRuntimeStoragePaths(runtimeCfg.DataDir, runtimeCfg.LogDir, GetInstallPaths().Backup, runtimeCfg.DBPath)
}

func checkRuntimeStoragePaths(dataDir, logDir, backupDir, dbPath string) PreflightCheck {
	check := PreflightCheck{Name: "Runtime Storage"}
	var missing []string
	var warnings []string

	// Check data directory
	if _, err := os.Stat(dataDir); err != nil {
		missing = append(missing, "data directory")
	}

	// Check log directory
	if _, err := os.Stat(logDir); err != nil {
		missing = append(missing, "log directory")
	}

	// Check backup directory
	if _, err := os.Stat(backupDir); err != nil {
		missing = append(missing, "backup directory")
	}

	// Check database file
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		warnings = append(warnings, "database file not found (will be created on startup)")
	}

	if len(missing) > 0 {
		check.Status = CheckFail
		check.Message = fmt.Sprintf("Missing required directories: %s", strings.Join(missing, ", "))
		check.Suggestions = []string{
			"Ensure OrvixPanel was installed correctly",
			"Check that installation script ran successfully",
		}
		return check
	}

	if len(warnings) > 0 {
		check.Status = CheckWarn
		check.Message = "Runtime storage partially configured"
		check.Details = strings.Join(warnings, "; ")
		return check
	}

	check.Status = CheckPass
	check.Message = "Runtime storage configured correctly"
	return check
}

// checkRuntimeHealth verifies the service is responding at the correct port.
func checkRuntimeHealth() PreflightCheck {
	// Get runtime config for dynamic port detection
	runtimeCfg, err := ReadRuntimeConfig()
	if err != nil {
		// Fall back to default port 8080
		return checkHealthAtPort("0.0.0.0:8080")
	}

	return checkHealthAtEndpoint(runtimeCfg.HealthEndpoint())
}

func checkHealthAtPort(port string) PreflightCheck {
	return checkHealthAtEndpoint("http://" + port + "/healthz")
}

func checkHealthAtEndpoint(endpoint string) PreflightCheck {
	check := PreflightCheck{Name: "Service Health"}

	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", "orvixpanel")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	output, _ := cmd.Output()
	isActive := strings.TrimSpace(string(output)) == "active"

	if !isActive {
		check.Status = CheckFail
		check.Message = "orvixpanel service is not running"
		check.Suggestions = []string{
			"Start the service: systemctl start orvixpanel",
			"Check logs: journalctl -u orvixpanel -n 50",
		}
		return check
	}

	// Check health endpoint
	cmd = exec.Command("curl", "-sf", "-m", "3", endpoint)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		check.Status = CheckWarn
		check.Message = fmt.Sprintf("Service running but health endpoint not responding at %s", endpoint)
		check.Details = "Service may be starting up or endpoint is misconfigured"
		check.Suggestions = []string{
			"Wait for service to fully start",
			"Check service logs: journalctl -u orvixpanel -f",
		}
		return check
	}

	check.Status = CheckPass
	check.Message = "Service health check passed"
	return check
}

// checkRuntimeVersion displays the currently installed version.
func checkRuntimeVersion() PreflightCheck {
	check := PreflightCheck{Name: "Installed Version"}

	v := InstalledVersion()
	if v.Tag == "" {
		check.Status = CheckWarn
		check.Message = "No version information found"
		check.Details = "VERSION file not found or empty"
		return check
	}

	channel := InstalledChannel()
	check.Status = CheckPass
	check.Message = fmt.Sprintf("v%s (commit: %s, channel: %s)", v.Tag, v.Commit[:8], channel)

	if v.Date != "" {
		check.Details = fmt.Sprintf("Built: %s", v.Date)
	}

	return check
}
// UpdateInfo holds detailed update information for verbose output.
type UpdateInfo struct {
InstalledVersion Version
TargetVersion    Version
Channel          Channel
UpdateNeeded     bool
LocalHEAD        string
RemoteHEAD       string
LatestTag        string
IsDirty          bool
UncommittedFiles []string
}

// GetUpdateInfo returns detailed update information for verbose output.
func GetUpdateInfo(channel Channel, version string) (*UpdateInfo, error) {
	info := &UpdateInfo{Channel: channel}

	// Use current directory for git operations (works both in dev and production)
	baseDir, _ := os.Getwd()

	// Get installed version - if empty, fall back to local git state
	info.InstalledVersion = InstalledVersion()
	if info.InstalledVersion.Commit == "" {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			info.InstalledVersion.Commit = strings.TrimSpace(string(out))
		}
	}
	// Always try to get v* tag for display (prefer semantic version over commit)
	if info.InstalledVersion.Tag == "" {
		cmd := exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			tags := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(tags) > 0 && tags[0] != "" {
				info.InstalledVersion.Tag = tags[0]
			}
		}
	}
	// No tag, use short commit
	if info.InstalledVersion.Tag == "" && len(info.InstalledVersion.Commit) > 8 {
		info.InstalledVersion.Tag = info.InstalledVersion.Commit[:8]
	}

	// Fetch latest from origin first (best effort)
	cmd := exec.Command("git", "fetch", "--all", "--tags")
	cmd.Dir = baseDir
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	cmd.Run() // Best effort

	// Get local HEAD
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = baseDir
	if out, err := cmd.Output(); err == nil {
		info.LocalHEAD = strings.TrimSpace(string(out))
	}

	// Get remote HEAD based on channel
	if channel == ChannelPreview {
		cmd = exec.Command("git", "rev-parse", "origin/feature/v0.7.0-mail-hosting")
	} else {
		cmd = exec.Command("git", "rev-parse", "origin/stable")
	}
	cmd.Dir = baseDir
	if out, err := cmd.Output(); err == nil {
		info.RemoteHEAD = strings.TrimSpace(string(out))
	}

	// Get latest v* tag (not nearest tag)
	cmd = exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
	cmd.Dir = baseDir
	if out, err := cmd.Output(); err == nil {
		tags := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(tags) > 0 && tags[0] != "" {
			info.LatestTag = tags[0]
		}
	}

	// Check if dirty (excluding VERSION and runtime files)
	cmd = exec.Command("git", "status", "--porcelain", "-- ':!.VERSION' ':!/opt/orvixpanel/**'")
	cmd.Dir = baseDir
	if out, err := cmd.Output(); err == nil {
		status := strings.TrimSpace(string(out))
		info.IsDirty = len(status) > 0
		if info.IsDirty {
			for _, line := range strings.Split(status, "\n") {
				if line != "" {
					info.UncommittedFiles = append(info.UncommittedFiles, line)
				}
			}
		}
	}

	// Get target version based on version parameter or channel
	if version != "" {
		// --version flag overrides: try exact tag first, then commit
		cmd = exec.Command("git", "describe", "--tags", "--exact-match", "tags/"+version)
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			info.TargetVersion.Tag = strings.TrimSpace(string(out))
		} else {
			info.TargetVersion.Tag = version
		}
		cmd = exec.Command("git", "rev-parse", info.TargetVersion.Tag)
		cmd.Dir = baseDir
		if out, err := cmd.Output(); err == nil {
			info.TargetVersion.Commit = strings.TrimSpace(string(out))
		}
	} else if channel == ChannelPreview {
		info.TargetVersion, _ = getRemoteHEADFromBase(baseDir)
	} else {
		info.TargetVersion, _ = getLatestStableTagFromBase(baseDir)
	}

	info.UpdateNeeded = CompareVersions(info.InstalledVersion, info.TargetVersion)
	return info, nil
}

func getRemoteHEADFromBase(base string) (Version, error) {
	cmd := exec.Command("git", "rev-parse", "origin/feature/v0.7.0-mail-hosting")
	cmd.Dir = base
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = base
		output, err = cmd.Output()
		if err != nil {
			return Version{}, fmt.Errorf("git rev-parse: %w", err)
		}
	}

	v := Version{Commit: strings.TrimSpace(string(output))}

	// Get latest v* tag (not nearest tag, but latest semantic version)
	cmd = exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
	cmd.Dir = base
	if tagOut, err := cmd.Output(); err == nil {
		tags := strings.Split(strings.TrimSpace(string(tagOut)), "\n")
		if len(tags) > 0 && tags[0] != "" {
			v.Tag = tags[0]
		}
	}
	// Fall back to short commit if no tag
	if v.Tag == "" && len(v.Commit) > 8 {
		v.Tag = v.Commit[:8]
	}
	return v, nil
}

func getLatestStableTagFromBase(base string) (Version, error) {
	// Get latest v* tag (not nearest tag, but latest semantic version)
	cmd := exec.Command("git", "tag", "-l", "v*", "--sort=-version:refname")
	cmd.Dir = base
	output, err := cmd.Output()
	if err != nil {
		return Version{}, fmt.Errorf("git tag: %w", err)
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) == 0 || tags[0] == "" {
		return Version{}, fmt.Errorf("no v* tags found")
	}

	v := Version{Tag: tags[0]}
	cmd = exec.Command("git", "rev-parse", v.Tag)
	cmd.Dir = base
	if commitOut, err := cmd.Output(); err == nil {
		v.Commit = strings.TrimSpace(string(commitOut))
	}
	return v, nil
}
