# Release Notes — v0.4.0-dns-api-preview

**Released:** June 2026
**Tag:** `v0.4.0-dns-api-preview`
**Classification:** PREVIEW ONLY — Not production-ready for DNS resolution

---

## IMPORTANT: Preview Classification

**v0.4.0 is DNS API Preview — SQLite storage and validation only. PowerDNS not live-verified.**

This release provides the DNS REST API layer with SQLite-backed storage. The PowerDNS synchronization code exists but has **NOT been verified against a live PowerDNS server**. No dig queries have been executed. No zone propagation has been tested.

Do NOT claim DNS Engine complete. This is an API layer only.

---

## What Is Included (Working)

- DNS REST API endpoints (zones, records, templates, validate, lookup)
- SQLite storage via GORM AutoMigrate
- Record validation (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)
- 54 unit tests (validator, service, PowerDNS client mocks)
- Smoke test: scripts/smoke-dns-local.sh
- Local-only mode (no PowerDNS required)

---

## What Is NOT Included (Not Verified)

- **No DNS resolution**: dig not installed, no real DNS queries
- **No PowerDNS live integration**: PowerDNS not installed
- **No DNS frontend**: No React UI for DNS management
- **No zone propagation**: DNS not serving real queries
- **No DNSSEC**: Not implemented
- **No zone transfer**: AXFR/IXFR not supported

---

## Why Preview Only

| Gate | Status | Reason |
|------|--------|--------|
| GitHub push | BLOCKED | No credentials in sandbox |
| Frontend build | FAILED | Pre-existing TS errors in SystemHealth.tsx, router.tsx |
| PowerDNS integration | NOT VERIFIED | PowerDNS not installed |
| dig queries | NOT VERIFIED | dig not installed |
| DNS resolution | NOT VERIFIED | No live DNS server |
| Frontend DNS UI | NOT BUILT | v0.4.0 is backend-only |

---

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

---

## Database Migration

Three new tables are created via GORM AutoMigrate:

1. dns_zones — Zone storage with SOA parameters
2. dns_records — Per-zone DNS records
3. dns_zone_templates — Reusable zone templates

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ORVIX_POWERDNS_URL` | No | PowerDNS API URL (optional) |
| `ORVIX_POWERDNS_API_KEY` | No | PowerDNS API key (optional) |
| `ORVIX_POWERDNS_SERVER_ID` | No | PowerDNS server ID (default: localhost) |

When `ORVIX_POWERDNS_URL` and `ORVIX_POWERDNS_API_KEY` are both set, the DNS Engine operates in **sync mode**. When either is missing, it operates in **local-only mode** (preview).

---

## Testing

- 54 unit tests for DNS package (validator, service, PowerDNS client mocks)
- Smoke test: `scripts/smoke-dns-local.sh`
- Tests use httptest for PowerDNS client (mock server only)

**Note**: No integration tests against real PowerDNS server.

---

## Next Releases

### v0.4.1 — DNS Frontend

- DNS navigation in sidebar
- DNS zones page (list/create/delete zones)
- Zone detail view with records table
- Add/edit record modal
- Templates page
- **Prerequisite**: Frontend npm run typecheck must pass

### v0.4.2 — PowerDNS Live Integration

- Install PowerDNS server
- Configure PowerDNS API
- Create zone through Orvix API
- Verify zone appears in PowerDNS
- Create A record
- Verify with dig
- Delete zone
- Verify NXDOMAIN

---

## Upgrade Notes

```bash
# Pull latest code
git pull origin main

# Rebuild binary
go build -o orvixpanel ./cmd/orvixpanel

# Restart service
sudo systemctl restart orvixpanel
```

The GORM AutoMigrate will automatically create the new DNS tables.

---

## Contributors

- OrvixPanel Core Team