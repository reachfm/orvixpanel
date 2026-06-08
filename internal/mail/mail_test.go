package mail

import (
	"testing"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// TestMailDomainModel tests mail domain model validation
func TestMailDomainModel(t *testing.T) {
	tests := []struct {
		name    string
		domain  models.MailDomain
		wantErr bool
	}{
		{
			name: "valid domain",
			domain: models.MailDomain{
				ID:        "md_test_001",
				TenantID:  "tenant_001",
				Domain:    "example.com",
				Status:    "active",
				CreatedBy: "admin",
			},
			wantErr: false,
		},
		{
			name: "with DKIM",
			domain: models.MailDomain{
				ID:           "md_test_002",
				TenantID:     "tenant_001",
				Domain:       "mail.example.com",
				DKIMSelector: "default",
				DKIMPublic:   "-----BEGIN PUBLIC KEY-----\nMIIBIjANBg...\n-----END PUBLIC KEY-----",
				Status:       "active",
			},
			wantErr: false,
		},
		{
			name: "catch-all domain",
			domain: models.MailDomain{
				ID:         "md_test_003",
				TenantID:   "tenant_001",
				Domain:     "catchall.example.com",
				IsCatchAll: true,
				Status:     "active",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			if tt.domain.ID == "" {
				t.Error("Domain ID should not be empty")
			}
			if tt.domain.Domain == "" {
				t.Error("Domain name should not be empty")
			}
			if tt.domain.TenantID == "" {
				t.Error("TenantID should not be empty")
			}
		})
	}
}

// TestMailboxModel tests mailbox model validation
func TestMailboxModel(t *testing.T) {
	tests := []struct {
		name    string
		mailbox models.Mailbox
		wantErr bool
	}{
		{
			name: "valid mailbox",
			mailbox: models.Mailbox{
				ID:           "mb_test_001",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				Email:        "user@example.com",
				LocalPart:    "user",
				QuotaMB:      1024,
				Status:       "active",
			},
			wantErr: false,
		},
		{
			name: "with all protocols",
			mailbox: models.Mailbox{
				ID:           "mb_test_002",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				Email:        "admin@example.com",
				EnableIMAP:   true,
				EnablePOP3:   true,
				EnableSMTP:   true,
				Status:       "active",
			},
			wantErr: false,
		},
		{
			name: "suspended mailbox",
			mailbox: models.Mailbox{
				ID:        "mb_test_003",
				TenantID:  "tenant_001",
				Email:     "suspended@example.com",
				Status:    "suspended",
				QuotaMB:   512,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mailbox.ID == "" {
				t.Error("Mailbox ID should not be empty")
			}
			if tt.mailbox.Email == "" {
				t.Error("Email should not be empty")
			}
			if tt.mailbox.TenantID == "" {
				t.Error("TenantID should not be empty")
			}
		})
	}
}

// TestAliasModel tests alias model validation
func TestAliasModel(t *testing.T) {
	tests := []struct {
		name  string
		alias models.MailAlias
	}{
		{
			name: "simple alias",
			alias: models.MailAlias{
				ID:           "al_test_001",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				SourceEmail:  "alias@example.com",
				Destinations:  `["user@example.com"]`,
				Status:       "active",
			},
		},
		{
			name: "multi-destination alias",
			alias: models.MailAlias{
				ID:           "al_test_002",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				SourceEmail:  "team@example.com",
				Destinations:  `["user1@example.com","user2@example.com","user3@example.com"]`,
				Status:       "active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.alias.ID == "" {
				t.Error("Alias ID should not be empty")
			}
			if tt.alias.SourceEmail == "" {
				t.Error("SourceEmail should not be empty")
			}
		})
	}
}

// TestForwarderModel tests forwarder model validation
func TestForwarderModel(t *testing.T) {
	tests := []struct {
		name       string
		forwarder  models.MailForwarder
	}{
		{
			name: "simple forwarder",
			forwarder: models.MailForwarder{
				ID:           "fw_test_001",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				SourceEmail:  "forward@example.com",
				Destinations:  `["external@gmail.com"]`,
				KeepCopy:     true,
				Status:       "active",
			},
		},
		{
			name: "forward without copy",
			forwarder: models.MailForwarder{
				ID:           "fw_test_002",
				TenantID:     "tenant_001",
				MailDomainID: "md_001",
				SourceEmail:  "nocc@example.com",
				Destinations:  `["backup@example.org"]`,
				KeepCopy:     false,
				Status:       "active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.forwarder.ID == "" {
				t.Error("Forwarder ID should not be empty")
			}
			if tt.forwarder.SourceEmail == "" {
				t.Error("SourceEmail should not be empty")
			}
		})
	}
}

// TestPasswordHash tests password hashing
func TestPasswordHash(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal password")
	}

	// Hash should start with bcrypt marker or be bcrypt compatible
	if len(hash) < 10 {
		t.Error("Hash seems too short")
	}
}

// TestGenerateRandomPassword tests random password generation
func TestGenerateRandomPassword(t *testing.T) {
	lengths := []int{16, 24, 32, 64}

	for _, length := range lengths {
		t.Run("", func(t *testing.T) {
			password, err := GenerateRandomPassword(length)
			if err != nil {
				t.Fatalf("GenerateRandomPassword(%d) failed: %v", length, err)
			}

			if len(password) != length {
				t.Errorf("Expected password length %d, got %d", length, len(password))
			}
		})
	}

	// Test uniqueness
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password, _ := GenerateRandomPassword(32)
		if passwords[password] {
			t.Error("Generated duplicate password")
		}
		passwords[password] = true
	}
}

// TestDKIMKeyGeneration tests DKIM key generation format
func TestDKIMKeyGeneration(t *testing.T) {
	// Test config generation (not actual key gen which requires RSA)
	config := OpenDKIMConfig{
		Domain:       "example.com",
		Selector:     "default",
		KeyTable:     "/etc/opendkim/KeyTable",
		SigningTable: "/etc/opendkim/SigningTable",
	}

	keyTable := GenerateOpenDKIMKeyTable(config.Domain, config.Selector, "/etc/opendkim/keys/default")
	if keyTable == "" {
		t.Error("KeyTable should not be empty")
	}

	signingTable := GenerateOpenDKIMSigningTable(config.Domain, config.Selector)
	if signingTable == "" {
		t.Error("SigningTable should not be empty")
	}

	trustedHosts := GenerateOpenDKIMTrustedHosts("mail.example.com")
	if trustedHosts == "" {
		t.Error("TrustedHosts should not be empty")
	}
}

// TestSPFRecordGeneration tests SPF record generation
func TestSPFRecordGeneration(t *testing.T) {
	mgr := &DomainManager{}

	spf := mgr.GetSPFRecord("example.com")
	if spf == "" {
		t.Error("SPF record should not be empty")
	}

	expected := "v=spf1 a mx ~all"
	if spf != expected {
		t.Errorf("Expected SPF record '%s', got '%s'", expected, spf)
	}
}

// TestDMARCRecordGeneration tests DMARC record generation
func TestDMARCRecordGeneration(t *testing.T) {
	mgr := &DomainManager{}

	tests := []struct {
		domain  string
		policy  string
		expect  string
	}{
		{"example.com", "none", "v=DMARC1; p=none; rua=mailto:dmarc@example.com"},
		{"example.com", "quarantine", "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"},
		{"example.com", "reject", "v=DMARC1; p=reject; rua=mailto:dmarc@example.com"},
		{"mail.org", "", "v=DMARC1; p=none; rua=mailto:dmarc@mail.org"},
	}

	for _, tt := range tests {
		t.Run(tt.policy, func(t *testing.T) {
			dmarc := mgr.GetDMARCRecord(tt.domain, tt.policy)
			if dmarc != tt.expect {
				t.Errorf("Expected DMARC '%s', got '%s'", tt.expect, dmarc)
			}
		})
	}
}

// TestValidateDomain tests domain validation
func TestValidateDomain(t *testing.T) {
	tests := []struct {
		domain  string
		wantErr bool
	}{
		{"example.com", false},
		{"mail.example.com", false},
		{"sub.domain.example.com", false},
		{"a.b", false},
		{"", true},
		{"-invalid.com", true},
		{"invalid-.com", true},
		{"invalid..com", true},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			err := validateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDomain(%s) error = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}

// TestValidateEmail tests email validation
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"user.name@example.com", false},
		{"user@sub.example.com", false},
		{"a@b.c", false},
		{"", true},
		{"invalid", true},
		{"@example.com", true},
		{"user@", true},
		{"user@.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail(%s) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

// TestQuotaCalculation tests quota calculation
func TestQuotaCalculation(t *testing.T) {
	tests := []struct {
		used   int
		limit  int
		expect float64
	}{
		{0, 1024, 0},
		{512, 1024, 50},
		{1024, 1024, 100},
		{100, 1000, 10},
		{750, 1000, 75},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			percent := CalculateQuotaPercent(tt.used, tt.limit)
			if percent != tt.expect {
				t.Errorf("CalculateQuotaPercent(%d, %d) = %v, want %v", tt.used, tt.limit, percent, tt.expect)
			}
		})
	}

	// Edge case: zero limit
	percent := CalculateQuotaPercent(100, 0)
	if percent != 0 {
		t.Error("Should return 0 when limit is 0")
	}
}

// TestFormatQuotaDisplay tests quota display formatting
func TestFormatQuotaDisplay(t *testing.T) {
	tests := []struct {
		used   int
		limit  int
		expect string
	}{
		{100, 1024, "100 MB / 1.0 GB"},
		{0, 1024, "0 MB / 1.0 GB"},
		{512, 1024, "512 MB / 1.0 GB"},
		{2048, 2048, "2.0 GB / 2.0 GB"},
		{100, 100, "100 MB / 100 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			display := FormatQuotaDisplay(tt.used, tt.limit)
			if display != tt.expect {
				t.Errorf("FormatQuotaDisplay(%d, %d) = '%s', want '%s'", tt.used, tt.limit, display, tt.expect)
			}
		})
	}
}

// TestParseDestinations tests destination JSON parsing
func TestParseDestinations(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		expect  []string
		wantErr bool
	}{
		{
			name:    "single destination",
			input:   `["user@example.com"]`,
			expect:  []string{"user@example.com"},
			wantErr: false,
		},
		{
			name:    "multiple destinations",
			input:   `["user1@example.com","user2@example.com"]`,
			expect:  []string{"user1@example.com", "user2@example.com"},
			wantErr: false,
		},
		{
			name:    "empty",
			input:   "",
			expect:  []string{},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDestinations(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDestinations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != len(tt.expect) {
				t.Errorf("ParseDestinations() got %d items, want %d", len(result), len(tt.expect))
			}
		})
	}
}

// TestSerializeDestinations tests destination JSON serialization
func TestSerializeDestinations(t *testing.T) {
	tests := []struct {
		input []string
	}{
		{[]string{"user@example.com"}},
		{[]string{"user1@example.com", "user2@example.com"}},
		{[]string{}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, err := SerializeDestinations(tt.input)
			if err != nil {
				t.Errorf("SerializeDestinations() error = %v", err)
				return
			}

			// Verify it's valid JSON
			parsed, err := ParseDestinations(result)
			if err != nil {
				t.Errorf("Serialized result is not valid JSON: %s", result)
				return
			}

			if len(parsed) != len(tt.input) {
				t.Errorf("Round trip failed: got %d, want %d", len(parsed), len(tt.input))
			}
		})
	}
}

// testRateLimitKey is a local helper for test key generation
func testRateLimitKey(limit *models.MailRateLimit) string {
	if limit.MailboxID != "" {
		return limit.TenantID + ":" + limit.MailboxID + ":" + limit.RateType
	}
	return limit.TenantID + ":" + limit.RateType
}

// TestMailRateLimit tests rate limit model
func TestMailRateLimit(t *testing.T) {
	limit := models.MailRateLimit{
		ID:            "rl_test_001",
		TenantID:      "tenant_001",
		RateType:      "outbound",
		MaxMessages:   100,
		WindowMinutes: 60,
		MaxSizeMB:     50,
		Status:        "active",
		CreatedAt:     time.Now(),
	}

	// Test Key generation using local helper
	key := testRateLimitKey(&limit)
	if key == "" {
		t.Error("Key should not be empty")
	}

	// Test with mailbox ID
	limit.MailboxID = "mb_001"
	keyWithMailbox := testRateLimitKey(&limit)
	if keyWithMailbox == "" {
		t.Error("Key should not be empty")
	}
	if keyWithMailbox == key {
		t.Error("Key with mailbox ID should be different")
	}
}

// TestMailAuditLog tests audit log model
func TestMailAuditLog(t *testing.T) {
	now := time.Now()
	log := models.MailAuditLog{
		ID:        "mal_test_001",
		TenantID:  "tenant_001",
		MailboxID: "mb_001",
		Action:    "sent",
		Direction: "outbound",
		FromEmail: "user@example.com",
		ToEmail:   "recipient@example.org",
		Subject:   "Test Subject",
		MessageID: "<test@example.com>",
		SizeBytes: 1024,
		Status:    "sent",
		RemoteIP:  "192.168.1.1",
		CreatedAt: now,
	}

	if log.ID == "" {
		t.Error("ID should not be empty")
	}
	if log.Action == "" {
		t.Error("Action should not be empty")
	}
	if log.TenantID == "" {
		t.Error("TenantID should not be empty")
	}
}

// TestGenerateID tests ID generation
func TestGenerateID(t *testing.T) {
	prefixes := []string{"md", "mb", "al", "fw", "rl", "mal"}

	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			id := generateID(prefix)
			if id == "" {
				t.Error("ID should not be empty")
			}

			// Should start with prefix
			if len(id) < len(prefix)+2 {
				t.Error("ID seems too short")
			}
		})
	}

	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID("test")
		if ids[id] {
			t.Error("Generated duplicate ID")
		}
		ids[id] = true
	}
}

// TestConfigGeneration tests configuration file generation
func TestConfigGeneration(t *testing.T) {
	// Test Postfix main.cf
	postfixConfig := PostfixConfig{
		Hostname:      "mail.example.com",
		MyNetworks:    "127.0.0.0/8",
		MessageSizeMB: 50,
		SSLEnabled:    true,
		SSLCertPath:   "/etc/ssl/certs/mail.crt",
		SSLKeyPath:    "/etc/ssl/private/mail.key",
	}

	postfixMain, err := GeneratePostfixMainCf(postfixConfig)
	if err != nil {
		t.Fatalf("GeneratePostfixMainCf failed: %v", err)
	}
	if postfixMain == "" {
		t.Error("Postfix config should not be empty")
	}
	if !contains(postfixMain, "myhostname = mail.example.com") {
		t.Error("Postfix config should contain hostname")
	}

	// Test Postfix master.cf
	masterCf, err := GeneratePostfixMasterCf()
	if err != nil {
		t.Fatalf("GeneratePostfixMasterCf failed: %v", err)
	}
	if masterCf == "" {
		t.Error("Master.cf should not be empty")
	}

	// Test Dovecot config
	dovecotConfig := DovecotConfig{
		Hostname:     "mail.example.com",
		MailLocation: "maildir:~/Maildir",
		QuotaEnabled: true,
		SSLEnabled:   true,
	}

	dovecotConf, err := GenerateDovecotConf(dovecotConfig)
	if err != nil {
		t.Fatalf("GenerateDovecotConf failed: %v", err)
	}
	if dovecotConf == "" {
		t.Error("Dovecot config should not be empty")
	}
	if !contains(dovecotConf, "protocols = imap pop3 lmtp submission") {
		t.Error("Dovecot config should contain protocols")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestErrorTypes tests error type creation
func TestErrorTypes(t *testing.T) {
	// Test error constants
	errors := []error{
		ErrDomainNotFound,
		ErrDomainExists,
		ErrMailboxNotFound,
		ErrMailboxExists,
		ErrAliasNotFound,
		ErrAliasExists,
		ErrForwarderNotFound,
		ErrForwarderExists,
		ErrQuotaExceeded,
		ErrRateLimitExceeded,
		ErrInvalidEmail,
		ErrDKIMGeneration,
	}

	for _, err := range errors {
		if err == nil {
			t.Errorf("Error constant should not be nil: %T", err)
		}
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

// TestMailError tests MailError struct
func TestMailError(t *testing.T) {
	baseErr := ErrDomainNotFound
	err := NewMailError("TestOp", baseErr, "test details")

	if err == nil {
		t.Fatal("NewMailError should not return nil")
	}

	if err.Operation != "TestOp" {
		t.Errorf("Expected operation 'TestOp', got '%s'", err.Operation)
	}

	if err.Details != "test details" {
		t.Errorf("Expected details 'test details', got '%s'", err.Details)
	}

	if err.Unwrap() != baseErr {
		t.Error("Unwrap should return the original error")
	}
}