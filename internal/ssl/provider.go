package ssl

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

// IssueRequest represents a certificate issuance request.
type IssueRequest struct {
	Domain         string
	SANs           []string // Subject Alternative Names
	ACMEAccountID  string
	Provider       string // letsencrypt, zerossl
}

// CertInfo represents parsed certificate information.
type CertInfo struct {
	CommonName   string
	SerialNumber string
	Fingerprint  string
	NotBefore    time.Time
	NotAfter     time.Time
	SANs         []string
	IsCA         bool
	Issuer       string
}

// Provider defines the interface for ACME certificate providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// IsConfigured returns true if the provider is properly configured.
	IsConfigured() bool

	// CreateAccount creates a new ACME account.
	CreateAccount(ctx context.Context, email string) (string, error) // Returns account URL

	// GetAccount retrieves ACME account status.
	GetAccount(ctx context.Context, accountURL string) (*AccountStatus, error)

	// IssueCertificate issues a new certificate.
	IssueCertificate(ctx context.Context, req IssueRequest) (*IssueResult, error)

	// RenewCertificate renews an existing certificate.
	RenewCertificate(ctx context.Context, certID string, req IssueRequest) (*IssueResult, error)

	// RevokeCertificate revokes a certificate.
	RevokeCertificate(ctx context.Context, certPath, keyPath string) error

	// ValidateCertificate parses and validates a certificate file.
	ValidateCertificate(certPath string) (*CertInfo, error)
}

// IssueResult represents the result of a certificate issuance/renewal.
type IssueResult struct {
	Cert       []byte // PEM encoded certificate
	Key        []byte // PEM encoded private key
	CertChain  []byte // PEM encoded certificate chain (excluding cert)
	FullChain  []byte // PEM encoded full chain (cert + chain)
	NotAfter   time.Time
	SerialNum  string
	Fingerprint string
}

// AccountStatus represents ACME account status information.
type AccountStatus struct {
	URL        string
	Status     string // active, deactivated, revoked
	Email      string
	TermsAgree bool
	RemainingEAB int // External Account Binding remaining
	RateLimits *RateLimits
}

// RateLimits represents ACME rate limit information.
type RateLimits struct {
	LimitRemain   int
	LimitUsed     int
	ResetTime     time.Time
	RetryAfter    time.Duration
}

// ACMEError represents an ACME protocol error.
type ACMEError struct {
	Type    string `json:"type"`
	Detail  string `json:"detail"`
	Subproblems []ACMESubproblem `json:"subproblems,omitempty"`
}

// ACMESubproblem represents a subproblem in an ACME error.
type ACMESubproblem struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
	Identifier *ACMEIdentifier `json:"identifier,omitempty"`
}

// ACMEIdentifier represents an identifier in an ACME error.
type ACMEIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ParseCertificate parses a PEM certificate and extracts information.
func ParseCertificate(certPEM []byte) (*CertInfo, error) {
	block, _ := pemDecode(certPEM)
	if block == nil {
		return nil, ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, &Error{Op: "parse certificate", Err: err}
	}

	info := &CertInfo{
		CommonName:   cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		IsCA:         cert.IsCA,
		Issuer:       cert.Issuer.String(),
	}

	// Extract SANs
	for _, dnsName := range cert.DNSNames {
		info.SANs = append(info.SANs, dnsName)
	}

	// Calculate SHA-256 fingerprint
	info.Fingerprint = calculateFingerprint(block.Bytes)

	return info, nil
}

// ParsePrivateKey parses a PEM private key and returns the RSA private key.
func ParsePrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pemDecode(keyPEM)
	if block == nil {
		return nil, ErrInvalidPEM
	}

	// Try parsing as PKCS8 first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fall back to PKCS1
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, &Error{Op: "parse private key", Err: ErrInvalidPrivateKey}
		}
		return rsaKey, nil
	}

	// Convert to RSA if needed
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, &Error{Op: "parse private key", Err: ErrInvalidPrivateKey}
	}

	return rsaKey, nil
}

// ValidateKeyCertMatch verifies that the private key matches the certificate.
func ValidateKeyCertMatch(certPEM, keyPEM []byte) error {
	// Parse certificate
	certBlock, _ := pemDecode(certPEM)
	if certBlock == nil {
		return ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return &Error{Op: "parse certificate", Err: err}
	}

	// Parse private key
	key, err := ParsePrivateKey(keyPEM)
	if err != nil {
		return err
	}

	// Compare public keys
	if !publicKeysEqual(cert.PublicKey, key.Public()) {
		return ErrKeyCertMismatch
	}

	return nil
}

// publicKeysEqual compares two crypto.PublicKey values for equality.
func publicKeysEqual(a, b crypto.PublicKey) bool {
	// Compare RSA public keys
	aRSA, aOk := a.(*rsa.PublicKey)
	bRSA, bOk := b.(*rsa.PublicKey)

	if aOk && bOk {
		return aRSA.N.Cmp(bRSA.N) == 0 && aRSA.E == bRSA.E
	}

	// For other key types, compare byte representation
	aBytes, errA := x509.MarshalPKIXPublicKey(a)
	bBytes, errB := x509.MarshalPKIXPublicKey(b)
	if errA != nil || errB != nil {
		return false
	}

	if len(aBytes) != len(bBytes) {
		return false
	}

	for i := range aBytes {
		if aBytes[i] != bBytes[i] {
			return false
		}
	}

	return true
}

// calculateFingerprint computes SHA-256 fingerprint of DER-encoded certificate.
func calculateFingerprint(derBytes []byte) string {
	hash := sha256.Sum256(derBytes)
	fingerprint := ""
	for i, b := range hash {
		if i > 0 {
			fingerprint += ":"
		}
		fingerprint += fmt.Sprintf("%02X", b)
	}
	return fingerprint
}

// pemDecode is a helper to decode PEM block.
func pemDecode(data []byte) (*pem.Block, error) {
	block, _ := pem.Decode(data)
	return block, nil
}