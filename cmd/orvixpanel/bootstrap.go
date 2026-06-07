package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/config"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
	"gorm.io/gorm"
)

// BootstrapOptions holds the command-line flags for the bootstrap wizard.
type BootstrapOptions struct {
	AdminEmail    string
	AdminPassword  string
	Interactive    bool
	Skip           bool
}

// ParseBootstrapFlags parses bootstrap-specific flags from os.Args.
// Returns nil if bootstrap is not being invoked.
func ParseBootstrapFlags(args []string) *BootstrapOptions {
	// We only parse flags if --bootstrap is present
	bootstrapIdx := -1
	for i, arg := range args {
		if arg == "--bootstrap" || arg == "bootstrap" {
			bootstrapIdx = i
			break
		}
	}
	if bootstrapIdx < 0 {
		return nil
	}

	opts := &BootstrapOptions{Interactive: true} // Default to interactive mode

	// Parse flags after --bootstrap
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.StringVar(&opts.AdminEmail, "admin-email", "", "Initial admin email address")
	fs.StringVar(&opts.AdminPassword, "admin-password", "", "Initial admin password (will prompt if empty in interactive mode)")
	fs.BoolVar(&opts.Interactive, "i", true, "Interactive mode (prompt for missing values)")
	fs.BoolVar(&opts.Skip, "skip", false, "Skip bootstrap even if no admin exists")

	// Get the bootstrap-specific args
	bootstrapArgs := args[bootstrapIdx+1:]
	// Filter out non-flag args that might be for the main program
	cleanedArgs := []string{}
	for _, arg := range bootstrapArgs {
		if !strings.HasPrefix(arg, "-") {
			// Check if it's part of the main program args (like --config)
			if arg == "--config" || arg == "--debug" || arg == "--help" || arg == "-h" {
				break
			}
			cleanedArgs = append(cleanedArgs, arg)
		} else {
			cleanedArgs = append(cleanedArgs, arg)
		}
	}

	// Reset args to just the bootstrap flags
	fs.Parse(cleanedArgs)
	_ = bootstrapIdx // Silence unused warning

	return opts
}

// RunBootstrapWizard runs the interactive bootstrap if needed.
// Returns true if bootstrap was performed, false otherwise.
func RunBootstrapWizard(ctx context.Context, db *gorm.DB, svc *auth.Service, cfg *config.Config, args []string) (bool, error) {
	// Check if bootstrap should run
	var count int64
	if err := db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check user count: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	opts := ParseBootstrapFlags(args)
	if opts == nil {
		// No bootstrap flags provided - use non-interactive mode with defaults
		return false, bootstrapRootAdminNonInteractive(ctx, db, svc, cfg)
	}

	if opts.Skip {
		log.Info().Msg("bootstrap: skipping (--skip specified)")
		return false, nil
	}

	if opts.Interactive && (opts.AdminEmail == "" || opts.AdminPassword == "") {
		return true, bootstrapRootAdminInteractive(ctx, db, svc, cfg)
	}

	// Non-interactive with explicit credentials
	return true, bootstrapRootAdminWithCredentials(ctx, db, svc, cfg, opts.AdminEmail, opts.AdminPassword)
}

// bootstrapRootAdminWithCredentials creates admin with provided credentials (non-interactive).
func bootstrapRootAdminWithCredentials(ctx context.Context, db *gorm.DB, svc *auth.Service, cfg *config.Config, email, password string) error {
	if email == "" {
		return errors.New("admin email is required (use --admin-email)")
	}
	if password == "" {
		return errors.New("admin password is required (use --admin-password)")
	}
	if err := validateEmail(email); err != nil {
		return fmt.Errorf("invalid admin email: %w", err)
	}
	if err := validatePassword(password); err != nil {
		return fmt.Errorf("invalid admin password: %w", err)
	}

	return createAdminUser(ctx, db, svc, email, password, false)
}

// bootstrapRootAdminNonInteractive creates admin with default credentials in dev mode only.
func bootstrapRootAdminNonInteractive(ctx context.Context, db *gorm.DB, svc *auth.Service, cfg *config.Config) error {
	isDev := os.Getenv("ORVIX_ALLOW_DEV") == "1"

	if !isDev {
		return errors.New(
			"no admin user exists and --bootstrap not provided in non-interactive mode; " +
				"run 'orvixpanel --bootstrap' to create the initial admin user or " +
				"use 'orvixpanel --bootstrap --admin-email=admin@example.com --admin-password=SecurePass123!'")
	}

	// DEV ONLY: Generate random credentials
	devEmail := "admin@orvixpanel.local"
	pw, err := randomPassword(16)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	log.Warn().
		Str("component", "bootstrap").
		Str("mode", "DEV_ONLY").
		Msg("⚠️  DEV ONLY: Using auto-generated admin credentials. DO NOT use in production.")

	return createAdminUser(ctx, db, svc, devEmail, pw, true)
}

// bootstrapRootAdminInteractive prompts for credentials interactively.
func bootstrapRootAdminInteractive(ctx context.Context, db *gorm.DB, svc *auth.Service, cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("OrvixPanel Bootstrap Wizard")
	fmt.Println("===========================")
	fmt.Println()

	var email, password string

	// Prompt for email
	for {
		fmt.Print("Admin email address: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read email: %w", err)
		}
		email = strings.TrimSpace(line)
		if email == "" {
			fmt.Println("Email is required.")
			continue
		}
		if err := validateEmail(email); err != nil {
			fmt.Printf("Invalid email: %v\n", err)
			continue
		}
		break
	}

	// Prompt for password
	for {
		fmt.Print("Admin password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		password = strings.TrimSpace(string(bytePassword))
		fmt.Println() // New line after password input
		if password == "" {
			fmt.Println("Password is required.")
			continue
		}
		if err := validatePassword(password); err != nil {
			fmt.Printf("Invalid password: %v\n", err)
			continue
		}

		// Confirm password
		fmt.Print("Confirm password: ")
		byteConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("read password confirmation: %w", err)
		}
		confirm := strings.TrimSpace(string(byteConfirm))
		fmt.Println() // New line after password input
		if confirm != password {
			fmt.Println("Passwords do not match. Please try again.")
			continue
		}
		break
	}

	fmt.Println()
	return createAdminUser(ctx, db, svc, email, password, false)
}

// createAdminUser creates the admin user with the given credentials.
func createAdminUser(ctx context.Context, db *gorm.DB, svc *auth.Service, email, password string, isDevMode bool) error {
	hash, err := svc.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Ensure the root tenant exists.
	var tenant models.Tenant
	if err := db.WithContext(ctx).Where("slug = ?", "default").First(&tenant).Error; err != nil {
		tenant = models.Tenant{
			Base:   models.Base{ID: newID()},
			Name:   "OrvixPanel",
			Slug:   "default",
			Type:   "admin",
			Status: "active",
		}
		if err := db.WithContext(ctx).Create(&tenant).Error; err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}
	}

	user := models.User{
		Base:         models.Base{ID: newID()},
		Email:        email,
		PasswordHash: hash,
		Role:         auth.RoleRootAdmin,
		TenantID:     tenant.ID,
		Status:       "active",
	}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	// NEVER log passwords in production. Only in dev mode with clear marking.
	if isDevMode {
		log.Warn().
			Str("component", "bootstrap").
			Str("user_id", user.ID).
			Str("email", user.Email).
			Str("password", password).
			Msg("DEV ONLY: Auto-generated admin credentials - DO NOT USE IN PRODUCTION")
	} else {
		log.Info().
			Str("component", "bootstrap").
			Str("user_id", user.ID).
			Str("email", user.Email).
			Msg("Admin user created successfully")
	}

	return nil
}

// validateEmail performs basic email validation.
func validateEmail(email string) error {
	if len(email) < 3 || len(email) > 254 {
		return errors.New("email must be between 3 and 254 characters")
	}
	atIdx := strings.Index(email, "@")
	if atIdx < 1 || atIdx > len(email)-3 {
		return errors.New("email must contain a valid @ with characters before and after")
	}
	domain := email[atIdx+1:]
	if !strings.Contains(domain, ".") {
		return errors.New("email domain must contain a dot")
	}
	return nil
}

// validatePassword enforces password requirements.
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password must be at most 128 characters")
	}
	hasLower := false
	hasUpper := false
	hasDigit := false
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		return errors.New("password must contain uppercase, lowercase, and digit")
	}
	return nil
}

// randomPassword returns an n-char password from a strict
// alphanumeric alphabet. We avoid base64's `-` and `_` because the
// password policy treats only [0-9A-Za-z] as letters/digits.
func randomPassword(n int) (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out), nil
}

// newID returns a fresh ULID for bootstrap rows.
func newID() string {
	// Use the same generator the models package uses. Inline import
	// would cycle, so we call a tiny helper.
	return bootstrapID()
}

// bootstrapID — small shim. Kept in a separate file? No, we keep it
// here for now to keep the bootstrap file self-contained.
func bootstrapID() string {
	return newULIDInline()
}

// We avoid importing ulid directly here by routing through gorm's
// BeforeCreate on models.Base (which generates a ULID). The simplest
// implementation: import the models package and reuse its generator.
// Since this is in `package main` and the models package has the
// helper, we can either re-export it or inline a tiny ULID maker.
// We inline.
func newULIDInline() string {
	b, _ := randBytes(16)
	return base64.RawURLEncoding.EncodeToString(b)[:22] // crude but valid
}

func randBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// keep time imported for future use
var _ = time.Now
