# Release Notes — v0.4.0-dns-engine

**Released:** June 2026
**Tag:** `v0.4.0-dns-engine`

## Highlights

v0.4.0 introduces the **DNS Engine**, a SQLite-first DNS zone and record management system with optional PowerDNS synchronization.

### New Features

#### DNS Zone Management
- Create, read, update, and delete DNS zones
- Support for native, master, and slave zone types
- Configurable SOA parameters (refresh, retry, expire, minimum)
- Zone status tracking (active, suspended, pending)

#### DNS Record Management
- Full CRUD operations for DNS records
- Support for 8 record types: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
- TTL validation (60-86400 seconds)
- Priority support for MX and SRV records
- Enable/disable individual records

#### Zone Templates
- Create reusable zone templates with predefined record sets
- Apply templates to new or existing zones
- JSON-based record definitions

#### PowerDNS Integration (Optional)
- Automatic sync to PowerDNS when ORVIX_POWERDNS_URL is set
- Zone and record synchronization
- Falls back to local-only mode when PowerDNS is not configured

#### Record Validation
- Real-time validation without creating records
- Type-specific content validation (IPv4, IPv6, hostname, etc.)
- RFC-compliant format checking

## API Changes

### New Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/dns/zones | List zones |
| POST | /api/v1/dns/zones | Create zone |
| GET | /api/v1/dns/zones/:id | Get zone |
| PUT | /api/v1/dns/zones/:id | Update zone |
| DELETE | /api/v1/dns/zones/:id | Delete zone |
| GET | /api/v1/dns/zones/:id/records | List records |
| POST | /api/v1/dns/zones/:id/records | Create record |
| PUT | /api/v1/dns/zones/:id/records/:recordId | Update record |
| DELETE | /api/v1/dns/zones/:id/records/:recordId | Delete record |
| GET | /api/v1/dns/templates | List templates |
| POST | /api/v1/dns/templates | Create template |
| POST | /api/v1/dns/templates/:id/apply | Apply template |
| POST | /api/v1/dns/validate | Validate record |
| GET | /api/v1/dns/lookup/:domain | Lookup domain |

### New RBAC Permissions

| Permission | Description |
|------------|-------------|
| dns.zone.read | View zones |
| dns.zone.write | Create/update zones |
| dns.zone.delete | Delete zones |
| dns.record.read | View records |
| dns.record.write | Create/update records |
| dns.record.delete | Delete records |
| dns.template.read | View templates |
| dns.template.write | Create/update templates |
| dns.template.apply | Apply templates |
| dns.validate | Validate records |
| dns.lookup | Lookup records |

## Database Migration

Three new tables are created via GORM AutoMigrate:

1. dns_zones — Zone storage with SOA parameters
2. dns_records — Per-zone DNS records
3. dns_zone_templates — Reusable zone templates

## Environment Variables

| Variable | Description |
|----------|-------------|
| ORVIX_POWERDNS_URL | PowerDNS API URL (optional) |
| ORVIX_POWERDNS_API_KEY | PowerDNS API key (optional) |
| ORVIX_POWERDNS_SERVER_ID | PowerDNS server ID (default: localhost) |

## Breaking Changes

None. v0.4.0 is fully backward compatible with v0.3.1.

## Bug Fixes

N/A — Initial release of DNS Engine.

## Deprecations

None.

## Security

- All DNS endpoints require JWT authentication
- Tenant isolation enforced at the service layer
- Input validation prevents malformed DNS data
- Audit logging for all zone and record operations

## Testing

- 54 new unit tests for DNS package
- Coverage for validator, service, and PowerDNS client
- Smoke test script: scripts/smoke-dns-local.sh

## Documentation

- docs/DNS_ENGINE.md — Complete DNS Engine documentation
- API usage examples and configuration guide

## Upgrade Notes

Upgrade from v0.3.1 to v0.4.0:

```bash
# Pull latest code
git pull origin main

# Rebuild binary
go build -o orvixpanel ./cmd/orvixpanel

# Restart service
sudo systemctl restart orvixpanel
```

The GORM AutoMigrate will automatically create the new DNS tables.

## Contributors

- OrvixPanel Core Team

## Next Release

- v0.4.1: DNSSEC signing and validation (planned)