package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// BuildConfig holds build configuration.
type BuildConfig struct {
	Version      string  // Specific version/tag/commit
	Channel      Channel // Update channel
	SkipFetch    bool    // Skip git fetch
	SkipFrontend bool    // Skip frontend build
	Verbose      bool    // Verbose output
}

// BuildResult contains the result of a build operation.
type BuildResult struct {
	Success    bool
	BinaryPath string
	Version    Version
	BuildTime  time.Time
	Error      error
	Warnings   []string
}

// Build fetches source and builds the binary.
func Build(cfg *BuildConfig) (*BuildResult, error) {
	result := &BuildResult{
		BuildTime: time.Now().UTC(),
	}

	p := GetInstallPaths()
	buildDir := filepath.Join(p.Cache, "build")

	// Step 1: Fetch source
	if !cfg.SkipFetch {
		if err := fetchSource(buildDir, cfg.Version, cfg.Channel); err != nil {
			result.Error = fmt.Errorf("fetch source: %w", err)
			return result, result.Error
		}
	}

	// Step 2: Get version info
	v, err := getVersionInfo(buildDir)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not determine version: %v", err))
		v = Version{Tag: "unknown", Commit: "unknown"}
	}
	result.Version = v

	// Step 3: Build backend
	backendBin, err := buildBackend(buildDir, cfg.Verbose)
	if err != nil {
		result.Error = fmt.Errorf("build backend: %w", err)
		return result, result.Error
	}
	result.BinaryPath = backendBin

	// Step 4: Build frontend (optional)
	if !cfg.SkipFrontend {
		if warnings, err := buildFrontend(buildDir, cfg.Verbose); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("frontend build failed: %v", err))
		} else {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}

	result.Success = true
	return result, nil
}

// fetchSource fetches the source code from git.
func fetchSource(buildDir, version string, channel Channel) error {
	// Clone or update the repo
	repoURL := "https://github.com/orvixpanel/orvixpanel.git"

	if _, err := os.Stat(filepath.Join(buildDir, ".git")); err == nil {
		// Repo exists, pull latest
		log.Info().Msg("Updating existing source...")
		cmds := [][]string{
			{"git", "fetch", "--all", "--tags"},
			{"git", "reset", "--hard", "HEAD"},
			{"git", "clean", "-fd"},
		}
		for _, args := range cmds {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = buildDir
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git %s: %s: %w", args[1], string(out), err)
			}
		}
	} else {
		// Fresh clone
		log.Info().Msg("Cloning repository...")
		os.MkdirAll(buildDir, 0o755)

		cmd := exec.Command("git", "clone", "--depth", "1", repoURL, buildDir)
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone: %s: %w", string(out), err)
		}

		// Fetch full history for version detection
		cmd = exec.Command("git", "fetch", "--unshallow")
		cmd.Dir = buildDir
		cmd.Run() // Best effort
	}

	// Checkout specific version if requested
	if version != "" {
		log.Info().Str("version", version).Msg("Checking out version...")

		// Try tag first, then branch, then commit
		for _, refType := range []string{"tag", "branch", "commit"} {
			var cmd *exec.Cmd
			switch refType {
			case "tag":
				cmd = exec.Command("git", "checkout", "tags/"+version)
			case "branch":
				cmd = exec.Command("git", "checkout", "-b", "update-"+version, "origin/"+version)
			case "commit":
				cmd = exec.Command("git", "checkout", version)
			}
			cmd.Dir = buildDir
			if _, err := cmd.CombinedOutput(); err == nil {
				log.Info().Str("ref_type", refType).Str("version", version).Msg("Checked out")
				break
			}
		}
	} else if channel == ChannelPreview {
		// Use main branch for preview
		log.Info().Msg("Using preview channel (main branch)")
		cmd := exec.Command("git", "checkout", "main")
		cmd.Dir = buildDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git checkout main: %s", string(out))
		}
	} else {
		// Use latest stable tag
		cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
		cmd.Dir = buildDir
		output, err := cmd.Output()
		if err == nil {
			tag := strings.TrimSpace(string(output))
			log.Info().Str("tag", tag).Msg("Using latest stable tag")
			cmd = exec.Command("git", "checkout", "tags/"+tag)
			cmd.Dir = buildDir
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git checkout tags/%s: %s", tag, string(out))
			}
		}
	}

	return nil
}

// getVersionInfo extracts version information from git.
func getVersionInfo(dir string) (Version, error) {
	runGit := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		output, err := cmd.Output()
		return strings.TrimSpace(string(output)), err
	}

	v := Version{}

	if commit, err := runGit("rev-parse", "HEAD"); err == nil {
		v.Commit = commit
	}

	if tag, err := runGit("describe", "--tags", "--exact-match"); err == nil {
		v.Tag = tag
	}

	if date, err := runGit("log", "-1", "--format=%aI"); err == nil {
		v.Date = date
	}

	// Check if dirty
	if status, err := runGit("status", "--porcelain"); err == nil {
		v.Dirty = len(strings.TrimSpace(status)) > 0
	}

	return v, nil
}

// buildBackend builds the Go backend binary.
func buildBackend(srcDir string, verbose bool) (string, error) {
	log.Info().Msg("Building backend...")

	p := GetInstallPaths()
	outputBin := filepath.Join(p.Bin, LinuxBinary)

	// Ensure output directory
	os.MkdirAll(p.Bin, 0o755)

	args := []string{
		"build",
		"-ldflags", buildLdflags(),
		"-o", outputBin,
		"./cmd/orvixpanel",
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}

	// Verify output
	if _, err := os.Stat(outputBin); err != nil {
		return "", fmt.Errorf("binary not found at %s", outputBin)
	}

	log.Info().Str("binary", outputBin).Msg("Backend built successfully")
	return outputBin, nil
}

// buildLdflags generates linker flags for version info.
func buildLdflags() string {
	return `-s -w -X main.version=update`
}

// buildFrontend builds the React frontend.
func buildFrontend(srcDir string, verbose bool) ([]string, error) {
	frontendDir := filepath.Join(srcDir, "frontend")
	if _, err := os.Stat(frontendDir); err != nil {
		return []string{"frontend directory not found, skipping"}, nil
	}

	log.Info().Msg("Building frontend...")

	// Install dependencies
	cmd := exec.Command("pnpm", "install", "--frozen-lockfile")
	cmd.Dir = frontendDir
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pnpm install: %w", err)
	}

	// Build
	cmd = exec.Command("pnpm", "build")
	cmd.Dir = frontendDir
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pnpm build: %w", err)
	}

	// Copy dist to installation directory
	p := GetInstallPaths()
	distDir := filepath.Join(frontendDir, "dist")
	frontendDest := p.Var + "/www/orvixpanel"

	if err := os.MkdirAll(frontendDest, 0o755); err != nil {
		return nil, fmt.Errorf("create frontend dir: %w", err)
	}

	// Remove old dist
	os.RemoveAll(frontendDest)

	if err := copyDirSimple(distDir, frontendDest); err != nil {
		return nil, fmt.Errorf("copy frontend dist: %w", err)
	}

	// Set env var
	setFrontendDist(frontendDest)

	log.Info().Str("path", frontendDest).Msg("Frontend built and deployed")

	return []string{"frontend built successfully"}, nil
}

// copyDirSimple copies a directory without checksum tracking.
func copyDirSimple(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFileSimple(path, dstPath)
	})
}

func copyFileSimple(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// setFrontendDist sets the ORVIX_FRONTEND_DIST environment variable.
func setFrontendDist(path string) {
	p := GetInstallPaths()
	envFile := p.EnvFile

	// Read existing env
	var lines []string
	if data, err := os.ReadFile(envFile); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Update or add ORVIX_FRONTEND_DIST
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "ORVIX_FRONTEND_DIST=") {
			lines[i] = "ORVIX_FRONTEND_DIST=" + path
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, "ORVIX_FRONTEND_DIST="+path)
	}

	// Write back
	os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0o644)
}

// InstallResult contains the result of an installation.
type InstallResult struct {
	Success      bool
	BinaryPath   string
	FrontendPath string
	Error        error
}

// Install installs the built binary and restarts the service.
func Install(binaryPath string) (*InstallResult, error) {
	result := &InstallResult{BinaryPath: binaryPath}

	p := GetInstallPaths()

	// Step 1: Stop service
	log.Info().Msg("Stopping orvixpanel service...")
	if err := stopService(); err != nil {
		result.Error = fmt.Errorf("stop service: %w", err)
		return result, result.Error
	}

	// Step 2: Install binary
	installPath := filepath.Join(p.Bin, BinaryName)
	if err := os.Rename(binaryPath, installPath); err != nil {
		// Try copy if rename fails (different filesystem)
		if err := copyFileSimple(binaryPath, installPath); err != nil {
			result.Error = fmt.Errorf("install binary: %w", err)
			return result, result.Error
		}
	}
	os.Chmod(installPath, 0o755)

	// Step 3: Save version file
	versionInfo, _ := getVersionInfo(filepath.Dir(installPath))
	versionFile := filepath.Join(p.Base, "VERSION")
	versionContent := fmt.Sprintf("tag=%s\ncommit=%s\ndate=%s\nbuilt=%s\n",
		versionInfo.Tag, versionInfo.Commit, versionInfo.Date, time.Now().UTC().Format(time.RFC3339))
	os.WriteFile(versionFile, []byte(versionContent), 0o644)

	// Step 4: Start service
	log.Info().Msg("Starting orvixpanel service...")
	if err := startService(); err != nil {
		result.Error = fmt.Errorf("start service: %w", err)
		return result, result.Error
	}

	// Step 5: Self-heal
	if err := SelfHeal(); err != nil {
		log.Warn().Err(err).Msg("Self-heal had warnings")
	}

	result.Success = true
	return result, nil
}

func stopService() error {
	cmd := exec.Command("systemctl", "stop", "orvixpanel")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}

func startService() error {
	cmd := exec.Command("systemctl", "start", "orvixpanel")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}