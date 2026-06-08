# OrvixPanel v0.5.0 - SSL Engine Preview Release

**Release Date:** 2024-01-15
**Status:** Preview (staging)
**Tag:** `v0.5.0-ssl-engine-preview`

## Release Type

**Feature Release** - Major new functionality

## Summary

This release introduces the SSL Engine, a comprehensive certificate management system for automated SSL/TLS certificate issuance, renewal, and monitoring using the ACME protocol.

## What's New

### SSL Certificate Management

- **Automated Issuance**: Issue certificates from Let's Encrypt and ZeroSSL providers
- **ACME Protocol**: Full ACME RFC 8555 compliance via `github.com/go-acme/lego/v4`
- **HTTP-01 Challenge**: Automatic challenge file creation and cleanup
- **Multi-SAN Support**: Issue certificates with multiple subject alternative names
- **Certificate Import**: Import existing certificates from other CAs

### Certificate Lifecycle

- **Auto-Renewal**: Automatic renewal before expiry (default: 30 days before)
- **File-Based Scheduler**: Coordinated renewal across multiple instances
- **Retry Logic**: Exponential backoff for failed renewals (max 3 attempts)
- **Status Tracking**: Real-time certificate status updates

### Health Monitoring

- **Certificate Scanner**: Automated health checks for all certificates
- **Expiry Alerts**: Track certificates expiring in 7/14/30 days
- **File Validation**: Verify certificate file integrity and permissions
- **Chain Validation**: Ensure certificate chain is complete

### Nginx Integration

- **Automatic Configuration**: Update nginx vhost configs with SSL
- **Backup & Rollback**: Automatic backup before changes
- **Config Validation**: `nginx -t` validation before reload
- **Security Headers**: HSTS, X-Frame-Options, X-Content-Type-Options

### API & UI

- **RESTful API**: Full CRUD operations for certificates
- **Dashboard Widget**: SSL statistics on main dashboard
- **Event Audit**: Complete audit trail for all SSL operations
- **Multi-Tenant**: Tenant isolation for certificate data

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/ssl/certificates` | List all certificates |
| GET | `/api/v1/ssl/certificates/:id` | Get certificate details |
| POST | `/api/v1/ssl/certificates` | Issue new certificate |
| POST | `/api/v1/ssl/certificates/:id/renew` | Renew certificate |
| POST | `/api/v1/ssl/certificates/:id/revoke` | Revoke certificate |
| DELETE | `/api/v1/ssl/certificates/:id` | Delete certificate |
| POST | `/api/v1/ssl/import` | Import existing certificate |
| GET | `/api/v1/ssl/certificates/:id/events` | Get certificate events |
| GET | `/api/v1/ssl/events` | Get all SSL events |
| GET | `/api/v1/ssl/health` | Get health report |
| GET | `/api/v1/ssl/dashboard` | Get dashboard stats |

## New Files

### Backend

```
internal/db/models/ssl.go           # Database models
internal/ssl/
├── errors.go                      # Error types
├── config.go                      # Configuration
├── provider.go                    # Provider interface
├── letsencrypt.go                # Let's Encrypt provider
├── storage.go                    # Certificate storage
├── validator.go                   # Certificate validation
├── health_scanner.go              # Health monitoring
├── renew_scheduler.go             # Renewal scheduler
├── challenge.go                   # ACME challenge handler
├── nginx_integration.go           # Nginx integration
├── manager.go                     # Main orchestrator
├── events.go                      # Event logging
└── handlers.go                    # API handlers
```

### Frontend

```
frontend/src/lib/api/ssl.ts         # SSL API client
frontend/src/pages/CertificatesList.tsx    # Certificate list page
frontend/src/pages/CertificateDetail.tsx    # Certificate detail page
```

### Documentation

```
docs/SSL_ENGINE.md    # Technical documentation
docs/SSL_API.md       # API reference
docs/SSL_RENEWAL.md   # Renewal system docs
```

## Dependencies Added

```go
github.com/go-acme/lego/v4 v4.16.0  // ACME protocol client
```

## Configuration

```yaml
# config.yaml
ssl:
  storage_dir: /var/lib/orvixpanel/ssl/certs
  challenge_dir: /var/www/orvixpanel/.well-known/acme-challenge
  renewal_window_days: 30
  renewal_lock_file: /run/orvixpanel/ssl-renew.lock
  max_renewal_retries: 3
  nginx_config_dir: /etc/nginx/conf.d/orvix
  nginx_backup_dir: /var/lib/orvixpanel/ssl/nginx-backup
  lets_encrypt_directory_url: https://acme-v02.api.letsencrypt.org/directory
  lets_encrypt_email: admin@example.com
```

## Breaking Changes

None - this is an additive feature.

## Known Limitations

1. **HTTP-01 Only**: DNS-01 challenge not yet supported
2. **Let's Encrypt Only**: ZeroSSL provider is a stub
3. **No Wildcard**: Wildcard certificate support not implemented
4. **Preview Status**: Not recommended for production use
5. **Sandbox Testing**: Live certificate issuance not tested in sandbox

## Known Issues

- Certificate files stored on local filesystem only (no S3/etc support)
- No OCSP stapling configuration
- No certificate transparency monitoring

## Upgrade Path

This is a new feature. No database migrations required.

## Testing

- 24 unit tests in `internal/ssl/ssl_test.go`
- All tests passing

## Future Releases

- **v0.5.1**: DNS-01 challenge support, CloudFlare/Route53 integration
- **v0.6.0**: Wildcard certificates, OCSP stapling
- **v0.6.1**: Certificate transparency monitoring, S3 storage backend

## Contributors

OrvixPanel Development Team

## License

Apache 2.0