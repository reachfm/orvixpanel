// Package ssl provides SSL/TLS certificate management with Let's Encrypt integration.
// Uses github.com/go-acme/lego/v4 for ACME protocol implementation.
package ssl

import "errors"

// Error definitions for SSL operations.
var (
	// ErrCertificateNotFound indicates the certificate does not exist.
	ErrCertificateNotFound = errors.New("certificate not found")

	// ErrCertificateExpired indicates the certificate has expired.
	ErrCertificateExpired = errors.New("certificate expired")

	// ErrChallengeFailed indicates ACME challenge verification failed.
	ErrChallengeFailed = errors.New("acme challenge failed")

	// ErrNginxValidationFailed indicates nginx config validation failed.
	ErrNginxValidationFailed = errors.New("nginx config validation failed")

	// ErrStorageError indicates file storage operation failed.
	ErrStorageError = errors.New("storage error")

	// ErrInvalidProvider indicates an unknown ACME provider.
	ErrInvalidProvider = errors.New("invalid provider")

	// ErrProviderNotConfigured indicates the SSL provider is not configured.
	ErrProviderNotConfigured = errors.New("provider not configured")

	// ErrRenewalInProgress indicates a renewal is already in progress.
	ErrRenewalInProgress = errors.New("renewal already in progress")

	// ErrAccountNotActive indicates the ACME account is not active.
	ErrAccountNotActive = errors.New("acme account not active")

	// ErrDomainNotOwned indicates the domain is not registered in OrvixPanel.
	ErrDomainNotOwned = errors.New("domain not owned by tenant")

	// ErrFileNotFound indicates a certificate file is missing.
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidCertificate indicates the certificate is invalid or corrupt.
	ErrInvalidCertificate = errors.New("invalid certificate")

	// ErrRateLimited indicates ACME rate limit was hit.
	ErrRateLimited = errors.New("acme rate limit exceeded")

	// ErrAlreadyRevoked indicates the certificate was already revoked.
	ErrAlreadyRevoked = errors.New("certificate already revoked")
)

// Error represents an SSL operation error with context.
type Error struct {
	Op   string // Operation that failed
	Err  error   // Underlying error
	Code string // Error code for API responses
}

func (e *Error) Error() string {
	return e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new SSL error with operation context.
func NewError(op string, err error) *Error {
	return &Error{Op: op, Err: err}
}

// WithCode creates a new SSL error with a specific code.
func (e *Error) WithCode(code string) *Error {
	e.Code = code
	return e
}