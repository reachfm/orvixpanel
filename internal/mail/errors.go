package mail

import "errors"

// Error types for mail operations
var (
	ErrDomainNotFound     = errors.New("mail domain not found")
	ErrDomainExists       = errors.New("mail domain already exists")
	ErrDomainInvalid      = errors.New("invalid mail domain")
	ErrMailboxNotFound    = errors.New("mailbox not found")
	ErrMailboxExists      = errors.New("mailbox already exists")
	ErrMailboxInvalid     = errors.New("invalid mailbox")
	ErrMailboxSuspended   = errors.New("mailbox is suspended")
	ErrAliasNotFound      = errors.New("alias not found")
	ErrAliasExists        = errors.New("alias already exists")
	ErrForwarderNotFound  = errors.New("forwarder not found")
	ErrForwarderExists    = errors.New("forwarder already exists")
	ErrQuotaExceeded      = errors.New("mailbox quota exceeded")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrDKIMGeneration      = errors.New("DKIM key generation failed")
	ErrConfigGeneration   = errors.New("mail config generation failed")
	ErrSMTPAuthFailed     = errors.New("SMTP authentication failed")
	ErrIMAPAuthFailed     = errors.New("IMAP authentication failed")
	ErrDeliveryFailed     = errors.New("email delivery failed")
)

// MailError represents a mail operation error with context
type MailError struct {
	Operation string
	Err       error
	Details   string
}

func (e *MailError) Error() string {
	if e.Details != "" {
		return e.Operation + ": " + e.Err.Error() + " (" + e.Details + ")"
	}
	return e.Operation + ": " + e.Err.Error()
}

func (e *MailError) Unwrap() error {
	return e.Err
}

// NewMailError creates a new mail error with context
func NewMailError(op string, err error, details string) *MailError {
	return &MailError{
		Operation: op,
		Err:       err,
		Details:   details,
	}
}