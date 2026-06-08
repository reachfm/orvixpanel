# DNS Engine — v0.4.0

SQLite-first DNS zone and record management with optional PowerDNS synchronization.

## Overview

The DNS Engine provides a complete DNS management layer built directly into OrvixPanel. It stores all zone and record data in the primary SQLite database (GORM AutoMigrate) and optionally syncs changes to an external PowerDNS server when configured.

**Two modes of operation:**

| Mode | Description |
|------|-------------|
| **Local-only** | All DNS data stored in SQLite. No external dependencies. |
| **PowerDNS sync** | Zone/record changes automatically pushed to PowerDNS API. |

## Architecture

```
+-------------------------------------------------------------+
|                     OrvixPanel v0.4.0                       |
+-------------------------------------------------------------+
|                                                             |
|  +-------------+   +--------------+   +-----------------+   |
|  | API Handler |-->| DNS Service  |-->|  SQLite Store   |   |
|  |  (v1/dns)   |   | (service.go) |   |  (GORM models)  |   |
|  +-------------+   +--------------+   +-----------------+   |
|                          |                                   |
|                          v                                   |
|                   +--------------+                          |
|                   |PowerDNS Client|  (optional)             |
|                   |(powerdns_client.go)|                    |
|                   +------+---------+                          |
|                          | HTTP API                         |
|                          v                                  |
|                   +--------------+                          |
|                   |  PowerDNS    |                          |
|                   |   Server     |                          |
|                   +--------------+                          |
+-------------------------------------------------------------+
```

## Database Schema

Three new tables added by v0.4.0:

```sql
-- DNS zones
CREATE TABLE dns_zones (
  id              VARCHAR(26) PRIMARY KEY,
  account_id      VARCHAR(26) NOT NULL,
  tenant_id       VARCHAR(26) NOT NULL,
  domain          VARCHAR(253) UNIQUE NOT NULL,
  type            VARCHAR(20) DEFAULT 'native',
  masters         TEXT,
  soa_refresh     INT DEFAULT 10800,
  soa_retry       INT DEFAULT 7200,
  soa_expire      INT DEFAULT 604800,
  soa_minimum     INT DEFAULT 3600,
  status          VARCHAR(20) DEFAULT 'active',
  created_at      TIMESTAMP,
  updated_at      TIMESTAMP,
  deleted_at      TIMESTAMP
);

-- DNS records
CREATE TABLE dns_records (
  id              VARCHAR(26) PRIMARY KEY,
  zone_id         VARCHAR(26) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  type            VARCHAR(10) NOT NULL,
  content         TEXT NOT NULL,
  ttl             INT DEFAULT 3600,
  priority        INT DEFAULT 0,
  disabled        BOOLEAN DEFAULT FALSE,
  created_at       TIMESTAMP,
  updated_at       TIMESTAMP,
  deleted_at       TIMESTAMP
);

-- Zone templates
CREATE TABLE dns_zone_templates (
  id              VARCHAR(26) PRIMARY KEY,
  tenant_id       VARCHAR(26) NOT NULL,
  name            VARCHAR(255) UNIQUE NOT NULL,
  description     TEXT,
  records         TEXT NOT NULL,
  created_at       TIMESTAMP,
  updated_at       TIMESTAMP,
  deleted_at       TIMESTAMP
);
```

## Supported Record Types

| Type | Description | Content Format |
|------|-------------|----------------|
| A | IPv4 address | `192.0.2.1` |
| AAAA | IPv6 address | `2001:db8::1` |
| CNAME | Canonical name | `example.com` |
| MX | Mail exchange | `10 mail.example.com` |
| TXT | Text record | `v=spf1 include:_spf.example.com ~all` |
| NS | Nameserver | `ns1.example.com` |
| SRV | Service locator | `0 5 443 sip.example.com` |
| CAA | Certification Authority Authorization | `0 issue "letsencrypt.org"` |

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ORVIX_POWERDNS_URL` | No | PowerDNS API URL (e.g., `http://127.0.0.1:8081`) |
| `ORVIX_POWERDNS_API_KEY` | No | PowerDNS API key |
| `ORVIX_POWERDNS_SERVER_ID` | No | PowerDNS server ID (default: `localhost`) |

When `ORVIX_POWERDNS_URL` and `ORVIX_POWERDNS_API_KEY` are both set, the DNS Engine operates in **sync mode**. When either is missing, it operates in **local-only mode**.

### Example: Local-Only Mode

```bash
# No additional configuration needed
./orvixpanel
# Logs: "dns engine running in local-only mode"
```

### Example: PowerDNS Sync Mode

```bash
export ORVIX_POWERDNS_URL="http://127.0.0.1:8081"
export ORVIX_POWERDNS_API_KEY="your-api-key-here"
export ORVIX_POWERDNS_SERVER_ID="localhost"

./orvixpanel
# Logs: "powerdns sync enabled"
```

## API Endpoints

All DNS endpoints require JWT authentication via the `Authorization: Bearer <token>` header.

### Zones

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/dns/zones` | List all zones for tenant |
| `POST` | `/api/v1/dns/zones` | Create a new zone |
| `GET` | `/api/v1/dns/zones/:id` | Get zone by ID |
| `PUT` | `/api/v1/dns/zones/:id` | Update zone |
| `DELETE` | `/api/v1/dns/zones/:id` | Delete zone |

### Records

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/dns/zones/:id/records` | List records in zone |
| `POST` | `/api/v1/dns/zones/:id/records` | Create record |
| `PUT` | `/api/v1/dns/zones/:id/records/:recordId` | Update record |
| `DELETE` | `/api/v1/dns/zones/:id/records/:recordId` | Delete record |

### Templates

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/dns/templates` | List templates for tenant |
| `POST` | `/api/v1/dns/templates` | Create template |
| `POST` | `/api/v1/dns/templates/:id/apply` | Apply template to zone |

### Utility

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/dns/validate` | Validate record without creating |
| `GET` | `/api/v1/dns/lookup/:domain` | Lookup records for domain |

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `dns.zone.read` | View zones |
| `dns.zone.write` | Create/update zones |
| `dns.zone.delete` | Delete zones |
| `dns.record.read` | View records |
| `dns.record.write` | Create/update records |
| `dns.record.delete` | Delete records |
| `dns.template.read` | View templates |
| `dns.template.write` | Create/update templates |
| `dns.template.apply` | Apply templates |
| `dns.validate` | Validate records |
| `dns.lookup` | Lookup records |

## Usage Examples

### Create a Zone

```bash
curl -X POST http://localhost:8080/api/v1/dns/zones \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "example.com",
    "type": "native"
  }'
```

### Add DNS Records

```bash
# A record
curl -X POST http://localhost:8080/api/v1/dns/zones/{zone_id}/records \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "www",
    "type": "A",
    "content": "192.0.2.1",
    "ttl": 3600
  }'

# MX record
curl -X POST http://localhost:8080/api/v1/dns/zones/{zone_id}/records \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "@",
    "type": "MX",
    "content": "10 mail.example.com",
    "ttl": 3600
  }'
```

### Validate a Record

```bash
curl -X POST http://localhost:8080/api/v1/dns/validate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "www",
    "type": "A",
    "content": "192.0.2.1",
    "ttl": 3600
  }'
```

## Limitations (v0.4.0)

- **No DNSSEC**: DNSSEC signing not implemented
- **No public DNS queries**: Local lookup only (no recursive resolver)
- **No zone transfers**: AXFR/IXFR not supported
- **No secondary zones**: Slave zone support is stub-only

## Future Phases

- Phase 4.1: DNSSEC signing and validation
- Phase 4.2: Zone transfer (AXFR/IXFR)
- Phase 4.3: Secondary DNS support
- Phase 4.4: Anycast DNS integration