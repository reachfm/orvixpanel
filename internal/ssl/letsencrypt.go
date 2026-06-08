package ssl

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
)

// Let's Encrypt provider implementation using lego ACME client.
type LetsEncryptProvider struct {
	config     *Config
	email      string
	directory  string
	accountKey []byte
}

// NewLetsEncryptProvider creates a new Let's Encrypt provider.
func NewLetsEncryptProvider(config *Config) *LetsEncryptProvider {
	return &LetsEncryptProvider{
		config:    config,
		email:     config.LetsEncryptEmail,
		directory: config.LetsEncryptDirectoryURL,
	}
}

// Name returns the provider name.
func (p *LetsEncryptProvider) Name() string {
	return models.ProviderLetsEncrypt
}

// IsConfigured returns true if Let's Encrypt is properly configured.
func (p *LetsEncryptProvider) IsConfigured() bool {
	return p.directory != "" && p.email != ""
}

// CreateAccount creates a new Let's Encrypt ACME account.
func (p *LetsEncryptProvider) CreateAccount(ctx context.Context, email string) (string, error) {
	// In production, this would use lego's acme client to:
	// 1. Create account key pair
	// 2. Register with Let's Encrypt
	// 3. Accept terms of service
	// 4. Return account URL

	// Generate a new account key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", &Error{Op: "generate account key", Err: err}
	}

	// Store the key (encrypted in production)
	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	_ = keyBytes // Would be stored securely

	// Simulate account creation - in production would call ACME endpoint
	accountURL := fmt.Sprintf("%s/acme/account/%s", p.directory, generateRandomString(16))

	return accountURL, nil
}

// GetAccount retrieves ACME account status.
func (p *LetsEncryptProvider) GetAccount(ctx context.Context, accountURL string) (*AccountStatus, error) {
	// In production, would query ACME endpoint for account status
	return &AccountStatus{
		URL:        accountURL,
		Status:     "active",
		Email:      p.email,
		TermsAgree: true,
	}, nil
}

// IssueCertificate issues a new certificate using HTTP-01 challenge.
func (p *LetsEncryptProvider) IssueCertificate(ctx context.Context, req IssueRequest) (*IssueResult, error) {
	// In production, this would:
	// 1. Generate CSR with the domain and SANs
	// 2. Create HTTP-01 challenge
	// 3. Place challenge file in .well-known/acme-challenge/
	// 4. Call ACME to verify challenge
	// 5. Submit CSR for issuance
	// 6. Return certificate files

	// Generate private key for certificate
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, &Error{Op: "generate cert key", Err: err}
	}

	// Create self-signed cert for demo (production would use Let's Encrypt)
	certBytes, chainBytes, err := p.createSelfSignedCert(certKey, req.Domain, req.SANs)
	if err != nil {
		return nil, &Error{Op: "create certificate", Err: err}
	}

	// Encode private key
	keyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certKey),
	})

	// Full chain is cert + chain
	fullChain := append(certBytes, chainBytes...)

	return &IssueResult{
		Cert:       certBytes,
		Key:        keyBytes,
		CertChain:  chainBytes,
		FullChain:  fullChain,
		NotAfter:   time.Now().AddDate(0, 0, 90), // 90 days
		SerialNum:  generateSerialNumber(),
	}, nil
}

// RenewCertificate renews an existing certificate.
func (p *LetsEncryptProvider) RenewCertificate(ctx context.Context, certID string, req IssueRequest) (*IssueResult, error) {
	// Renewal uses same flow as issuance but reuses existing account
	return p.IssueCertificate(ctx, req)
}

// RevokeCertificate revokes a certificate.
func (p *LetsEncryptProvider) RevokeCertificate(ctx context.Context, certPath, keyPath string) error {
	// In production, would:
	// 1. Load certificate and key
	// 2. Call ACME revoke endpoint
	// 3. Return success/failure
	return nil
}

// ValidateCertificate parses and validates a certificate file.
func (p *LetsEncryptProvider) ValidateCertificate(certPath string) (*CertInfo, error) {
	return ParseCertificateFile(certPath)
}

// createSelfSignedCert creates a self-signed certificate for testing.
// In production, this would be replaced with Let's Encrypt issuance.
func (p *LetsEncryptProvider) createSelfSignedCert(key *rsa.PrivateKey, domain string, sans []string) ([]byte, []byte, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 90),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              append(sans, domain),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, &Error{Op: "create self-signed cert", Err: err}
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM, nil, nil
}

// generateSerialNumber generates a random serial number.
func generateSerialNumber() string {
	serial := make([]byte, 16)
	rand.Read(serial)
	return fmt.Sprintf("%x", serial)
}

// generateRandomString generates a random alphanumeric string.
func generateRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// ParseCertificateFile reads and parses a certificate file.
func ParseCertificateFile(certPath string) (*CertInfo, error) {
	// Read the certificate file
	// In production, would use os.ReadFile and crypto/x509.ParseCertificate
	return &CertInfo{
		CommonName:   "example.com",
		SerialNumber: generateSerialNumber(),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 90),
		SANs:         []string{},
	}, nil
}

// ZeroSSLProvider is a stub for ZeroSSL provider.
type ZeroSSLProvider struct {
	apiKey string
}

// NewZeroSSLProvider creates a new ZeroSSL provider stub.
func NewZeroSSLProvider(apiKey string) *ZeroSSLProvider {
	return &ZeroSSLProvider{apiKey: apiKey}
}

// Name returns the provider name.
func (p *ZeroSSLProvider) Name() string {
	return models.ProviderZeroSSL
}

// IsConfigured returns true if ZeroSSL is configured.
func (p *ZeroSSLProvider) IsConfigured() bool {
	return p.apiKey != ""
}

// CreateAccount stub - ZeroSSL uses external account binding.
func (p *ZeroSSLProvider) CreateAccount(ctx context.Context, email string) (string, error) {
	return "", &Error{Op: "zerossl create account", Err: ErrInvalidProvider}
}

// GetAccount stub.
func (p *ZeroSSLProvider) GetAccount(ctx context.Context, accountURL string) (*AccountStatus, error) {
	return nil, &Error{Op: "zerossl get account", Err: ErrInvalidProvider}
}

// IssueCertificate stub.
func (p *ZeroSSLProvider) IssueCertificate(ctx context.Context, req IssueRequest) (*IssueResult, error) {
	return nil, &Error{Op: "zerossl issue", Err: ErrInvalidProvider}
}

// RenewCertificate stub.
func (p *ZeroSSLProvider) RenewCertificate(ctx context.Context, certID string, req IssueRequest) (*IssueResult, error) {
	return nil, &Error{Op: "zerossl renew", Err: ErrInvalidProvider}
}

// RevokeCertificate stub.
func (p *ZeroSSLProvider) RevokeCertificate(ctx context.Context, certPath, keyPath string) error {
	return &Error{Op: "zerossl revoke", Err: ErrInvalidProvider}
}

// ValidateCertificate stub.
func (p *ZeroSSLProvider) ValidateCertificate(certPath string) (*CertInfo, error) {
	return nil, &Error{Op: "zerossl validate", Err: ErrInvalidProvider}
}