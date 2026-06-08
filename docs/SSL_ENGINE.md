# SSL Engine v0.5.0 - Technical Documentation

## Overview

The SSL Engine provides automated SSL/TLS certificate management for OrvixPanel using the ACME protocol via `github.com/go-acme/lego/v4`. It supports Let's Encrypt and ZeroSSL providers with automatic renewal, nginx integration, and health monitoring.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           SSL Engine                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   API Layer  │───▶│   Manager    │───▶│   Provider (lego)    │  │
│  │  (handlers)  │    │              │    │  Let's Encrypt/ZeroSSL│  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│         │                  │                       │                  │
│         ▼                  ▼                       ▼                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Storage    │    │   Health     │    │  Challenge Handler   │  │
│  │  (PEM files) │    │   Scanner    │    │    (HTTP-01)         │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│                                              │                       │
│                                              ▼                       │
│                                    ┌──────────────────────┐          │
│                                    │  Nginx Integration  │          │
│                                    │  (vhost SSL config) │          │
│                                    └──────────────────────┘          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Database Models (`internal/db/models/ssl.go`)

- **SSLCertificate**: Stores certificate metadata (domain, status, paths, dates)
- **SSLEvent**: Audit log for certificate operations
- **ACMEAccount**: ACME provider account information
- **SSLChallenge**: ACME challenge tracking

### 2. Core Package (`internal/ssl/`)

| File | Purpose |
|------|---------|
| `manager.go` | Main orchestrator for certificate operations |
| `provider.go` | Provider interface and common types |
| `letsencrypt.go` | Let's Encrypt provider implementation |
| `storage.go` | Certificate file storage operations |
| `validator.go` | Certificate validation |
| `health_scanner.go` | Certificate health monitoring |
| `renew_scheduler.go` | Automated renewal scheduling |
| `challenge.go` | HTTP-01 ACME challenge handling |
| `nginx_integration.go` | Nginx vhost SSL configuration |
| `events.go` | Event logging |
| `errors.go` | Error types |
| `config.go` | Configuration management |

### 3. API Handlers (`internal/ssl/handlers.go`)

REST API endpoints for certificate management.

### 4. Frontend (`frontend/src/pages/CertificatesList.tsx`)

React UI for certificate management.

## Certificate Lifecycle

```
pending → issued → expiring_soon → expired
                   ↓
                renewed → issued (loop)
```

### Status Values

- `pending`: Certificate request initiated
- `issued`: Certificate successfully issued
- `expiring_soon`: Within 30 days of expiry
- `expired`: Past expiry date
- `revoked`: Manually revoked
- `failed`: Issuance/renewal failed

## Certificate Storage

Certificates are stored as files (NOT in database):

```
/var/lib/orvixpanel/ssl/certs/
└── example.com/
    ├── cert.pem        (0600)
    ├── privkey.pem     (0600)
    ├── chain.pem       (0644)
    └── fullchain.pem   (0644)
```

## ACME Challenge Flow

1. Client initiates certificate issuance
2. Server creates HTTP-01 challenge file at `.well-known/acme-challenge/`
3. ACME server verifies challenge
4. Certificate issued and stored
5. Nginx vhost updated with SSL configuration

## Renewal Scheduling

- File-based locking to prevent duplicate renewals
- Configurable renewal window (default: 30 days before expiry)
- Maximum retry attempts (default: 3)
- Stale lock detection (24-hour timeout)

## Nginx Integration

The SSL engine can automatically update nginx vhost configurations:

- Adds SSL listen directives
- Configures certificate paths
- Sets TLS protocols (1.2, 1.3)
- Configures secure cipher suites
- Adds security headers (HSTS, etc.)
- Validates config with `nginx -t`
- Reloads nginx on success
- Rollback on failure

## Configuration

```go
type Config struct {
    StorageDir           string  // Base directory for certificates
    ChallengeDir         string  // HTTP-01 challenge directory
    RenewalWindowDays    int     // Days before expiry to renew
    RenewalLockFile      string  // Lock file path
    MaxRenewalRetries    int     // Max retry attempts
    NginxConfigDir       string  // Nginx config directory
    NginxBackupDir       string  // Backup directory
    LetsEncryptDirectoryURL string  // ACME directory URL
    LetsEncryptEmail     string  // Default email
}
```

## Security Considerations

1. **Private keys**: Stored with 0600 permissions (owner read/write only)
2. **Certificates**: Stored with 0644 permissions (readable)
3. **Challenge directory**: Must be web-accessible
4. **Tenant isolation**: Multi-tenant support via TenantID
5. **Audit logging**: All operations logged to SSLEvents table

## Dependencies

- `github.com/go-acme/lego/v4` - ACME protocol client
- `gorm.io/gorm` - Database ORM
- `github.com/gofiber/fiber/v2` - HTTP framework

## Future Enhancements

- DNS-01 challenge support
- OCSP stapling
- Certificate transparency monitoring
- Wildcard certificate support
- CloudFlare/Route53 DNS provider integration