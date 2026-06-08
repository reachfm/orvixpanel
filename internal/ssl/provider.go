package ssl

import (
	"context"
	"crypto/x509"
	"encoding/pem"
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
		return nil, ErrInvalidCertificate
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, &Error{Op: "parse certificate", Err: err}
	}

	info := &CertInfo{
		CommonName: cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		IsCA:     cert.IsCA,
		Issuer:   cert.Issuer.String(),
	}

	// Extract SANs
	for _, dnsName := range cert.DNSNames {
		info.SANs = append(info.SANs, dnsName)
	}

	// Calculate fingerprint
	// Fingerprint would be calculated here in real implementation

	return info, nil
}

// pemDecode is a helper to decode PEM block.
func pemDecode(data []byte) (*pem.Block, error) {
	block, _ := pem.Decode(data)
	return block, nil
}