// Package dns provides DNS zone and record management with optional
// PowerDNS integration.
//
// v0.4.0 scope:
//   - SQLite-first storage (GORM AutoMigrate)
//   - Optional PowerDNS sync when ORVIX_POWERDNS_URL is set
//   - Supported record types: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
//   - No DNSSEC in v0.4.0
//   - No public DNS queries required
package dns

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Supported record types in v0.4.0.
const (
	RecordTypeA     = "A"
	RecordTypeAAAA  = "AAAA"
	RecordTypeCNAME = "CNAME"
	RecordTypeMX    = "MX"
	RecordTypeTXT   = "TXT"
	RecordTypeNS    = "NS"
	RecordTypeSRV   = "SRV"
	RecordTypeCAA   = "CAA"
)

// ValidRecordTypes contains all supported DNS record types.
var ValidRecordTypes = []string{
	RecordTypeA, RecordTypeAAAA, RecordTypeCNAME,
	RecordTypeMX, RecordTypeTXT, RecordTypeNS,
	RecordTypeSRV, RecordTypeCAA,
}

// Validation errors.
var (
	ErrEmptyDomain       = errors.New("domain cannot be empty")
	ErrInvalidDomain     = errors.New("invalid domain format")
	ErrInvalidRecordType = errors.New("unsupported record type")
	ErrInvalidRecordName = errors.New("invalid record name")
	ErrInvalidContent    = errors.New("invalid record content")
	ErrInvalidTTL        = errors.New("ttl must be between 60 and 86400")
	ErrInvalidPriority   = errors.New("priority must be between 0 and 65535")
)

// DomainRegex matches valid domain names.
var DomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)

// RecordDefinition represents a record to validate.
type RecordDefinition struct {
	Name     string
	Type     string
	Content  string
	TTL      int
	Priority int
}

// ValidateZoneDomain validates a zone domain name.
func ValidateZoneDomain(domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ErrEmptyDomain
	}
	if len(domain) > 253 {
		return fmt.Errorf("%w: domain too long", ErrInvalidDomain)
	}
	// Remove trailing dot for FQDN
	domain = strings.TrimSuffix(domain, ".")
	if !DomainRegex.MatchString(domain) {
		return fmt.Errorf("%w: %s", ErrInvalidDomain, domain)
	}
	return nil
}

// ValidateRecordType checks if the record type is supported.
func ValidateRecordType(recordType string) error {
	for _, t := range ValidRecordTypes {
		if t == strings.ToUpper(recordType) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrInvalidRecordType, recordType)
}

// ValidateRecordName checks if the record name is valid.
func ValidateRecordName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidRecordName)
	}
	// Allow @ for apex, * for wildcards
	if name != "@" && name != "*" {
		// Must be a valid hostname label or subdomain
		if len(name) > 63 {
			return fmt.Errorf("%w: label too long", ErrInvalidRecordName)
		}
		validLabel := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
		if !validLabel.MatchString(strings.ToLower(name)) {
			return fmt.Errorf("%w: %s", ErrInvalidRecordName, name)
		}
	}
	return nil
}

// ValidateTTL checks if TTL is in valid range (60-86400 seconds).
func ValidateTTL(ttl int) error {
	if ttl < 60 || ttl > 86400 {
		return fmt.Errorf("%w: %d (valid: 60-86400)", ErrInvalidTTL, ttl)
	}
	return nil
}

// ValidatePriority checks if priority is valid (0-65535).
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidPriority, priority)
	}
	return nil
}

// ValidateRecordContent validates record content based on type.
func ValidateRecordContent(recordType, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrInvalidContent)
	}
	switch strings.ToUpper(recordType) {
	case RecordTypeA:
		return validateIPv4(content)
	case RecordTypeAAAA:
		return validateIPv6(content)
	case RecordTypeMX:
		return validateMXContent(content)
	case RecordTypeTXT:
		return validateTXTContent(content)
	case RecordTypeCNAME, RecordTypeNS:
		return validateHostname(content)
	case RecordTypeSRV:
		return validateSRVContent(content)
	case RecordTypeCAA:
		return validateCAAContent(content)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidRecordType, recordType)
	}
}

// validateIPv4 checks if content is a valid IPv4 address.
func validateIPv4(ip string) error {
	if net.ParseIP(ip) == nil || strings.Contains(ip, ":") {
		return fmt.Errorf("%w: not a valid IPv4 address: %s", ErrInvalidContent, ip)
	}
	return nil
}

// validateIPv6 checks if content is a valid IPv6 address.
func validateIPv6(ip string) error {
	if net.ParseIP(ip) == nil || !strings.Contains(ip, ":") {
		return fmt.Errorf("%w: not a valid IPv6 address: %s", ErrInvalidContent, ip)
	}
	return nil
}

// validateMXContent validates MX record content (priority + hostname).
func validateMXContent(content string) error {
	parts := strings.Split(content, " ")
	if len(parts) < 2 {
		return fmt.Errorf("%w: MX must be 'priority hostname'", ErrInvalidContent)
	}
	// Check priority is a number
	var priority int
	if _, err := fmt.Sscanf(parts[0], "%d", &priority); err != nil {
		return fmt.Errorf("%w: MX priority must be a number", ErrInvalidContent)
	}
	if priority < 0 || priority > 65535 {
		return fmt.Errorf("%w: MX priority out of range", ErrInvalidContent)
	}
	// Validate hostname
	hostname := strings.Join(parts[1:], " ")
	return validateHostname(hostname)
}

// validateTXTContent validates TXT record content.
func validateTXTContent(content string) error {
	// TXT records can have quoted strings, but we store the raw content
	// Maximum 255 characters per string (RFC 1035)
	// For simplicity, we allow up to 400 characters (with escape handling)
	if len(content) > 400 {
		return fmt.Errorf("%w: TXT content exceeds maximum length", ErrInvalidContent)
	}
	return nil
}

// validateHostname validates a hostname (FQDN).
func validateHostname(hostname string) error {
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" {
		return fmt.Errorf("%w: hostname cannot be empty", ErrInvalidContent)
	}
	if len(hostname) > 253 {
		return fmt.Errorf("%w: hostname too long", ErrInvalidContent)
	}
	labels := strings.Split(hostname, ".")
	if len(labels) < 1 {
		return fmt.Errorf("%w: invalid hostname", ErrInvalidContent)
	}
	for _, label := range labels {
		if len(label) > 63 {
			return fmt.Errorf("%w: label too long: %s", ErrInvalidContent, label)
		}
		validLabel := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
		if !validLabel.MatchString(strings.ToLower(label)) {
			return fmt.Errorf("%w: invalid label: %s", ErrInvalidContent, label)
		}
	}
	return nil
}

// validateSRVContent validates SRV record content.
func validateSRVContent(content string) error {
	// Format: priority weight port target
	parts := strings.Split(content, " ")
	if len(parts) != 4 {
		return fmt.Errorf("%w: SRV must be 'priority weight port target'", ErrInvalidContent)
	}
	var priority, weight, port int
	if _, err := fmt.Sscanf(parts[0], "%d", &priority); err != nil {
		return fmt.Errorf("%w: SRV priority must be a number", ErrInvalidContent)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &weight); err != nil {
		return fmt.Errorf("%w: SRV weight must be a number", ErrInvalidContent)
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &port); err != nil {
		return fmt.Errorf("%w: SRV port must be a number", ErrInvalidContent)
	}
	if priority < 0 || priority > 65535 {
		return fmt.Errorf("%w: SRV priority out of range", ErrInvalidContent)
	}
	if weight < 0 || weight > 65535 {
		return fmt.Errorf("%w: SRV weight out of range", ErrInvalidContent)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: SRV port must be between 1 and 65535", ErrInvalidContent)
	}
	// Validate target hostname
	target := parts[3]
	if target == "." {
		return nil // SRV can point to nothing (.)
	}
	return validateHostname(target)
}

// validateCAAContent validates CAA record content.
func validateCAAContent(content string) error {
	// Format: flags tag value
	parts := strings.Split(content, " ")
	if len(parts) < 3 {
		return fmt.Errorf("%w: CAA must be 'flags tag value'", ErrInvalidContent)
	}
	var flags int
	if _, err := fmt.Sscanf(parts[0], "%d", &flags); err != nil {
		return fmt.Errorf("%w: CAA flags must be a number", ErrInvalidContent)
	}
	if flags < 0 || flags > 255 {
		return fmt.Errorf("%w: CAA flags out of range (0-255)", ErrInvalidContent)
	}
	tag := parts[1]
	if tag != "issue" && tag != "issuewild" && tag != "iodef" {
		return fmt.Errorf("%w: CAA tag must be 'issue', 'issuewild', or 'iodef'", ErrInvalidContent)
	}
	// Value is a string (hostname or URL for iodef)
	value := strings.Join(parts[2:], " ")
	if len(value) > 255 {
		return fmt.Errorf("%w: CAA value too long", ErrInvalidContent)
	}
	return nil
}

// ValidateRecord validates a complete record definition.
func ValidateRecord(record RecordDefinition) error {
	if err := ValidateRecordType(record.Type); err != nil {
		return err
	}
	if err := ValidateRecordName(record.Name); err != nil {
		return err
	}
	if err := ValidateRecordContent(record.Type, record.Content); err != nil {
		return err
	}
	if err := ValidateTTL(record.TTL); err != nil {
		return err
	}
	if err := ValidatePriority(record.Priority); err != nil {
		return err
	}
	return nil
}

// ValidateZone validates a zone for creation/update.
func ValidateZone(domain string) error {
	return ValidateZoneDomain(domain)
}