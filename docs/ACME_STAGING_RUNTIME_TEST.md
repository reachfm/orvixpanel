# ACME Staging Runtime Test Playbook

**Document Version:** 1.0.0
**Created:** 2024-06-13
**Purpose:** Runtime verification of Let's Encrypt Staging implementation
**Mode:** STAGING ONLY — No production certificates

---

## Overview

This playbook provides step-by-step instructions for runtime testing the ACME HTTP-01 challenge implementation on a live VPS with a public domain.

### What This Tests

- ACME HTTP-01 challenge storage and retrieval
- Challenge route serving via OrvixPanel API
- Let's Encrypt staging API communication
- Certificate issuance via Let's Encrypt staging
- Nginx vhost configuration generation
- Nginx configuration validation
- Database record creation
- End-to-end certificate lifecycle

### What This Does NOT Test

- Production Let's Encrypt API (uses staging only)
- Real certificate browser trust (staging certs show warnings)
- Auto-renewal scheduler (manual test only)
- Production deployment workflows

---

## 1. Prerequisites

### 1.1 Environment Requirements

| Requirement | Value | Notes |
|-------------|-------|-------|
| VPS | Fresh or existing OrvixPanel VPS | Root/sudo access required |
| Public IP | Valid IPv4 address | Must be reachable from internet |
| Domain | Registered domain with DNS control | Example: `acme-test.example.com` |
| DNS A Record | Points domain to VPS IP | Propagation may take time |
| Port 80 | Open for HTTP | Let's Encrypt validation |
| Port 443 | Open for HTTPS | Certificate serving |
| nginx | Installed and running | Required for vhost configs |
| OrvixPanel | Service running | Latest build deployed |

### 1.2 Verify Ports

```bash
# Check if ports are open from outside
nc -zv your-vps-ip 80
nc -zv your-vps-ip 443

# Or use an external checker
curl -I http://your-vps-ip:80
```

### 1.3 Verify nginx

```bash
nginx -v
systemctl status nginx
```

### 1.4 Verify OrvixPanel

```bash
systemctl status orvixpanel
```

---

## 2. DNS Configuration Checklist

### 2.1 Example Domain Setup

Use a dedicated subdomain for testing:

```
acme-test.example.com  A  203.0.113.50
```

### 2.2 DNS Validation Commands

```bash
# Verify DNS has propagated
dig +short acme-test.example.com

# Expected output: your VPS IP address (e.g., 203.0.113.50)

# Verify DNS resolves globally
nslookup acme-test.example.com

# Test from outside (if you have external access)
curl -I http://acme-test.example.com/.well-known/
```

### 2.3 DNS Checklist

- [ ] Domain registered and active
- [ ] A record created pointing to VPS IP
- [ ] A record propagated (allow up to 24 hours)
- [ ] `dig +short` returns correct IP
- [ ] `nslookup` succeeds
- [ ] HTTP request to domain returns response (even if 404)

---

## 3. OrvixPanel Service Checks

### 3.1 Service Status

```bash
# Check OrvixPanel service
systemctl status orvixpanel

# Verify it's running and enabled
systemctl is-enabled orvixpanel
systemctl is-active orvixpanel
```

### 3.2 Health Endpoints

```bash
# Check health endpoint
curl -s http://127.0.0.1:8443/healthz

# Expected: {"status":"ok"} or similar

# Check readiness endpoint
curl -s http://127.0.0.1:8443/readyz

# Expected: {"status":"ready"} or similar
```

### 3.3 Service Checklist

- [ ] `systemctl status orvixpanel` shows active (running)
- [ ] Health endpoint returns 200 OK
- [ ] Ready endpoint returns 200 OK
- [ ] No recent errors in journal logs

---

## 4. Challenge Route Checks

### 4.1 Create Test Challenge File

First, manually create a test challenge file:

```bash
# Create challenge directory if not exists
sudo mkdir -p /var/lib/orvixpanel/acme-challenges

# Create test challenge token
TEST_TOKEN="test-token-12345"
TEST_VALUE="test-challenge-value-abcdef"
echo -n "$TEST_VALUE" | sudo tee "/var/lib/orvixpanel/acme-challenges/$TEST_TOKEN"

# Set permissions
sudo chmod 644 "/var/lib/orvixpanel/acme-challenges/$TEST_TOKEN"
sudo chmod 755 /var/lib/orvixpanel/acme-challenges

# Verify file exists
ls -la /var/lib/orvixpanel/acme-challenges/
cat "/var/lib/orvixpanel/acme-challenges/$TEST_TOKEN"
```

### 4.2 Test Challenge Retrieval (Local)

```bash
# Test via OrvixPanel API endpoint (local)
curl -s "http://127.0.0.1:8443/.well-known/acme-challenge/$TEST_TOKEN"

# Expected: "test-challenge-value-abcdef"
```

### 4.3 Test Challenge Retrieval (Public)

```bash
# Test via public domain
curl -s "http://acme-test.example.com/.well-known/acme-challenge/$TEST_TOKEN"

# Expected: "test-challenge-value-abcdef"
```

### 4.4 Test Path Traversal Rejection

```bash
# This should return 404 or empty (not the content of /etc/passwd)
curl -s "http://127.0.0.1:8443/.well-known/acme-challenge/../../../etc/passwd"

# Verify it doesn't contain /etc/passwd content
curl -s "http://127.0.0.1:8443/.well-known/acme-challenge/../../../etc/passwd" | grep -q "root:" && echo "FAIL: Path traversal worked!" || echo "PASS: Path traversal blocked"
```

### 4.5 Challenge Route Checklist

- [ ] Challenge directory exists at `/var/lib/orvixpanel/acme-challenges`
- [ ] Test file created successfully
- [ ] Local challenge URL returns correct value
- [ ] Public challenge URL returns correct value
- [ ] Path traversal attempts are rejected
- [ ] nginx serves challenge files (if configured)

---

## 5. API Login and Authentication

### 5.1 Obtain Bearer Token

```bash
# Login to OrvixPanel API
# Replace with your actual credentials and domain

curl -s -X POST "https://acme-test.example.com/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your-admin-password"
  }' | jq .

# Expected response:
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "expires_at": "2024-06-14T..."
# }
```

### 5.2 Extract Bearer Token

```bash
# Save token to variable
TOKEN=$(curl -s -X POST "https://acme-test.example.com/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your-admin-password"
  }' | jq -r '.token')

echo "Token: $TOKEN"
```

### 5.3 Test Authenticated Request

```bash
# Test authenticated endpoint
curl -s "https://acme-test.example.com/api/v1/me" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 5.4 API Authentication Checklist

- [ ] Login endpoint accepts credentials
- [ ] Valid token returned
- [ ] Token can be used for authenticated requests
- [ ] Invalid token returns 401 Unauthorized

---

## 6. Issue Staging Certificate

### 6.1 Issue Certificate Command

```bash
# Issue staging certificate via Let's Encrypt staging
curl -s -X POST "https://acme-test.example.com/api/v1/ssl/certificates/issue" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "acme-test.example.com",
    "email": "admin@example.com"
  }' | jq .
```

### 6.2 Expected Success Response

```json
{
  "certificate_id": "01HX...",
  "domain": "acme-test.example.com",
  "status": "issued",
  "provider": "letsencrypt_staging",
  "is_staging": true,
  "issued_at": "2024-06-13T12:00:00Z",
  "expires_at": "2024-09-11T12:00:00Z",
  "fingerprint": "SHA256:XX:XX:XX:...",
  "serial_number": "04:XX:XX:...",
  "message": "Certificate issued using Let's Encrypt STAGING. NOT FOR PRODUCTION USE.",
  "warnings": [
    "This is a STAGING certificate issued by Let's Encrypt",
    "Browsers will show security warnings",
    "Do not use in production environments"
  ]
}
```

### 6.3 Issue Certificate Checklist

- [ ] API request returns 201 Created
- [ ] Response contains certificate_id
- [ ] Response indicates staging provider
- [ ] Response contains warnings about staging
- [ ] Certificate ID is not empty

---

## 7. Expected Success Verification

### 7.1 Database Certificate Record

```bash
# Query database for certificate record
# Adjust based on your database setup

# For SQLite
sqlite3 /var/lib/orvixpanel/orvixpanel.db "SELECT * FROM ssl_certificates WHERE common_name='acme-test.example.com';"

# For PostgreSQL
psql -U orvixpanel -d orvixpanel -c "SELECT * FROM ssl_certificates WHERE common_name='acme-test.example.com';"
```

### 7.2 Certificate Files on Disk

```bash
# Check certificate storage directory
ls -la /var/lib/orvixpanel/ssl/certs/acme-test.example.com/

# Expected files:
# - cert.pem (server certificate)
# - privkey.pem (private key)
# - chain.pem (intermediate certificates)
# - fullchain.pem (cert + chain)

# Verify certificate contents
openssl x509 -in /var/lib/orvixpanel/ssl/certs/acme-test.example.com/cert.pem -text -noout | head -20

# Verify it's a staging certificate (Issuer should mention "Staging")
openssl x509 -in /var/lib/orvixpanel/ssl/certs/acme-test.example.com/cert.pem -text -noout | grep -i issuer
```

### 7.3 Nginx Vhost Configuration

```bash
# Check nginx config directory
ls -la /etc/nginx/conf.d/orvix/

# Check if vhost config was created
cat /etc/nginx/conf.d/orvix/acme-test.example.com.conf 2>/dev/null || echo "No vhost config found"

# Verify nginx -t passes
nginx -t

# Expected output:
# nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
# nginx: configuration file /etc/nginx/nginx.conf test is successful
```

### 7.4 SSL Event Records

```bash
# Check SSL events table
sqlite3 /var/lib/orvixpanel/orvixpanel.db "SELECT * FROM ssl_events WHERE resource_name='acme-test.example.com' ORDER BY created_at DESC;"

# Or for PostgreSQL
psql -U orvixpanel -d orvixpanel -c "SELECT * FROM ssl_events WHERE resource_name='acme-test.example.com' ORDER BY created_at DESC;"
```

### 7.5 Service Health After Issuance

```bash
# Verify service is still healthy
curl -s http://127.0.0.1:8443/healthz
curl -s http://127.0.0.1:8443/readyz

# Check for any new errors
journalctl -u orvixpanel --since "10 minutes ago" | grep -i error
```

### 7.6 Success Checklist

- [ ] Database record exists in ssl_certificates table
- [ ] cert.pem file exists and is valid
- [ ] privkey.pem file exists
- [ ] chain.pem file exists
- [ ] Certificate issuer mentions "Staging"
- [ ] nginx -t passes
- [ ] Health endpoint still returns OK
- [ ] Ready endpoint still returns OK
- [ ] SSL event recorded in database

---

## 8. Failure Diagnostics

### 8.1 Journal Logs

```bash
# View OrvixPanel logs
journalctl -u orvixpanel -n 100 --no-pager

# Filter for SSL-related logs
journalctl -u orvixpanel -n 100 --no-pager | grep -i ssl
journalctl -u orvixpanel -n 100 --no-pager | grep -i acme
journalctl -u orvixpanel -n 100 --no-pager | grep -i challenge

# View logs since specific time
journalctl -u orvixpanel --since "1 hour ago" --no-pager
```

### 8.2 nginx Error Logs

```bash
# Check nginx error log
tail -100 /var/log/nginx/error.log

# Check nginx access log for challenge requests
tail -100 /var/log/nginx/access.log | grep -i acme
```

### 8.3 ACME Challenge File Paths

```bash
# List all challenge files
ls -laR /var/lib/orvixpanel/acme-challenges/

# Check if challenge directory is readable
sudo -u www-data ls -la /var/lib/orvixpanel/acme-challenges/ 2>/dev/null || echo "Cannot read as www-data"

# Check nginx can read the directory
nginx -T 2>&1 | grep -i "acme-challenge"
```

### 8.4 Let's Encrypt Staging API Direct

```bash
# Test Let's Encrypt staging directory directly
curl -s https://acme-staging-v02.api.letsencrypt.org/directory | jq .

# Test new nonce endpoint
curl -s -I https://acme-staging-v02.api.letsencrypt.org/acme/new-nonce
```

### 8.5 Database Queries

```bash
# For SQLite
sqlite3 /var/lib/orvixpanel/orvixpanel.db << 'EOF'
-- Check certificate records
SELECT id, common_name, provider, status, created_at FROM ssl_certificates;

-- Check SSL events
SELECT id, action, resource_name, result, created_at FROM ssl_events ORDER BY created_at DESC LIMIT 20;

-- Check for errors
SELECT * FROM ssl_events WHERE result = 'failure';
EOF

# For PostgreSQL
psql -U orvixpanel -d orvixpanel << 'EOF'
-- Check certificate records
SELECT id, common_name, provider, status, created_at FROM ssl_certificates;

-- Check SSL events
SELECT id, action, resource_name, result, created_at FROM ssl_events ORDER BY created_at DESC LIMIT 20;

-- Check for errors
SELECT * FROM ssl_events WHERE result = 'failure';
EOF
```

### 8.6 Common Failure Scenarios

| Failure | Symptom | Diagnosis | Fix |
|---------|---------|-----------|-----|
| DNS not propagated | curl fails | `dig` returns wrong IP | Wait for DNS propagation |
| Port blocked | Connection refused | External curl fails | Open firewall ports |
| Challenge not served | 404 from challenge URL | Local curl works, public fails | Check nginx routing |
| LE API unreachable | Connection timeout | Direct curl to LE fails | Check network/firewall |
| Invalid token | 400 Bad Request | Response mentions token | Check token format |
| Rate limited | 429 Too Many Requests | Response mentions rate | Wait and retry |

---

## 9. Cleanup

### 9.1 Remove Test Domain Vhost

```bash
# Remove nginx vhost config
sudo rm -f /etc/nginx/conf.d/orvix/acme-test.example.com.conf

# Validate nginx config
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
```

### 9.2 Remove Certificate Files

```bash
# Remove certificate files
sudo rm -rf /var/lib/orvixpanel/ssl/certs/acme-test.example.com

# Remove ACME account files (optional)
sudo rm -rf /var/lib/orvixpanel/ssl/accounts/system

# Remove challenge files for the domain
sudo find /var/lib/orvixpanel/acme-challenges -name "*acme-test*" -delete
```

### 9.3 Remove Database Records

```bash
# For SQLite
sqlite3 /var/lib/orvixpanel/orvixpanel.db << 'EOF'
-- Delete certificate record
DELETE FROM ssl_certificates WHERE common_name='acme-test.example.com';

-- Delete related events
DELETE FROM ssl_events WHERE resource_name='acme-test.example.com';

-- Verify deletion
SELECT * FROM ssl_certificates WHERE common_name='acme-test.example.com';
EOF

# For PostgreSQL
psql -U orvixpanel -d orvixpanel << 'EOF'
-- Delete certificate record
DELETE FROM ssl_certificates WHERE common_name='acme-test.example.com';

-- Delete related events
DELETE FROM ssl_events WHERE resource_name='acme-test.example.com';

-- Verify deletion
SELECT * FROM ssl_certificates WHERE common_name='acme-test.example.com';
EOF
```

### 9.4 Verify Cleanup

```bash
# Verify files removed
ls -la /var/lib/orvixpanel/ssl/certs/acme-test.example.com 2>&1

# Verify nginx config removed
ls -la /etc/nginx/conf.d/orvix/acme-test.example.com.conf 2>&1

# Verify database is clean
sqlite3 /var/lib/orvixpanel/orvixpanel.db "SELECT COUNT(*) as cert_count FROM ssl_certificates WHERE common_name='acme-test.example.com';"
```

### 9.5 Cleanup Checklist

- [ ] nginx vhost config removed
- [ ] nginx -t passes after removal
- [ ] nginx reloaded successfully
- [ ] Certificate files deleted
- [ ] Database certificate record deleted
- [ ] Database event records deleted
- [ ] No remaining references to test domain

---

## 10. Pass/Fail Criteria

### 10.1 PASS Criteria

The test is considered **PASS** only if ALL of the following are true:

- [ ] **Public challenge URL works**: `curl http://acme-test.example.com/.well-known/acme-challenge/test` returns correct content
- [ ] **Let's Encrypt staging returns certificate**: API response contains valid certificate data
- [ ] **Cert row stored**: Database contains record in `ssl_certificates` table
- [ ] **Files exist**: `/var/lib/orvixpanel/ssl/certs/acme-test.example.com/` contains all required files
- [ ] **nginx -t passes**: Configuration test succeeds
- [ ] **healthz passes**: `curl http://127.0.0.1:8443/healthz` returns 200
- [ ] **readyz passes**: `curl http://127.0.0.1:8443/readyz` returns 200
- [ ] **Service stable**: No crashes or errors after certificate issuance

### 10.2 FAIL Indicators

Any of the following indicates FAIL:

- Public challenge URL returns 404 or wrong content
- Let's Encrypt API returns error
- No database record created
- Certificate files missing
- nginx -t fails
- Service crashes or becomes unhealthy
- Path traversal vulnerability detected

---

## 11. Copy-Paste Command Block

### Complete Test Sequence

```bash
#!/bin/bash
# ACME Staging Runtime Test Script
# Run on VPS with public domain

set -e

DOMAIN="acme-test.example.com"
EMAIL="admin@example.com"
PASSWORD="your-admin-password"
PANEL_URL="https://$DOMAIN"

echo "=== ACME Staging Runtime Test ==="
echo "Domain: $DOMAIN"
echo ""

# Step 1: Check services
echo "[1/8] Checking services..."
systemctl is-active nginx || { echo "FAIL: nginx not active"; exit 1; }
systemctl is-active orvixpanel || { echo "FAIL: orvixpanel not active"; exit 1; }
echo "PASS: Services running"
echo ""

# Step 2: Check health endpoints
echo "[2/8] Checking health endpoints..."
curl -sf http://127.0.0.1:8443/healthz || { echo "FAIL: healthz failed"; exit 1; }
curl -sf http://127.0.0.1:8443/readyz || { echo "FAIL: readyz failed"; exit 1; }
echo "PASS: Health endpoints OK"
echo ""

# Step 3: Get auth token
echo "[3/8] Getting auth token..."
TOKEN_RESPONSE=$(curl -sf -X POST "$PANEL_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
[ -z "$TOKEN" ] && { echo "FAIL: Could not get token"; exit 1; }
echo "PASS: Auth token obtained"
echo ""

# Step 4: Issue staging certificate
echo "[4/8] Issuing staging certificate..."
CERT_RESPONSE=$(curl -sf -X POST "$PANEL_URL/api/v1/ssl/certificates/issue" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"domain\":\"$DOMAIN\",\"email\":\"$EMAIL\"}")
echo "$CERT_RESPONSE" | jq .
CERT_ID=$(echo "$CERT_RESPONSE" | grep -o '"certificate_id":"[^"]*"' | cut -d'"' -f4)
[ -z "$CERT_ID" ] && { echo "FAIL: No certificate_id returned"; exit 1; }
echo "PASS: Certificate issued (ID: $CERT_ID)"
echo ""

# Step 5: Verify files
echo "[5/8] Verifying certificate files..."
[ -f "/var/lib/orvixpanel/ssl/certs/$DOMAIN/cert.pem" ] || { echo "FAIL: cert.pem missing"; exit 1; }
[ -f "/var/lib/orvixpanel/ssl/certs/$DOMAIN/privkey.pem" ] || { echo "FAIL: privkey.pem missing"; exit 1; }
[ -f "/var/lib/orvixpanel/ssl/certs/$DOMAIN/chain.pem" ] || { echo "FAIL: chain.pem missing"; exit 1; }
echo "PASS: All certificate files exist"
echo ""

# Step 6: Verify nginx config
echo "[6/8] Verifying nginx configuration..."
nginx -t || { echo "FAIL: nginx -t failed"; exit 1; }
echo "PASS: nginx configuration valid"
echo ""

# Step 7: Check database record
echo "[7/8] Checking database record..."
# Adjust for your database
COUNT=$(sqlite3 /var/lib/orvixpanel/orvixpanel.db "SELECT COUNT(*) FROM ssl_certificates WHERE common_name='$DOMAIN';")
[ "$COUNT" -lt 1 ] && { echo "FAIL: No database record"; exit 1; }
echo "PASS: Database record exists"
echo ""

# Step 8: Final health check
echo "[8/8] Final health check..."
curl -sf http://127.0.0.1:8443/healthz || { echo "FAIL: healthz failed"; exit 1; }
curl -sf http://127.0.0.1:8443/readyz || { echo "FAIL: readyz failed"; exit 1; }
echo "PASS: Service still healthy"
echo ""

echo "========================================="
echo "ALL TESTS PASSED!"
echo "========================================="
```

---

## 12. Verification Commands

After running the playbook, execute these verification commands:

### Local Build Verification

```bash
cd /workspace

# Run SSL tests
go test ./internal/ssl/... -v

# Run all tests
go test ./... -v

# Build the binary
go build -buildvcs=false ./cmd/orvixpanel
```

### Expected Output

```
=== RUN   TestNewChallengeStore
--- PASS: TestNewChallengeStore
=== RUN   TestChallengeStoreStoreAndRead
--- PASS: TestChallengeStoreStoreAndRead
... (more tests)
PASS
ok  	github.com/orvixpanel/orvixpanel/internal/ssl	2.379s

=== RUN   TestStagingProviderCreation
--- PASS: TestStagingProviderCreation
... (more staging tests)
PASS
ok  	github.com/orvixpanel/orvixpanel/internal/ssl/staging	0.500s

# Build
go build -buildvcs=false ./cmd/orvixpanel
# (no output = success)
```

---

## Appendix A: Challenge Store Configuration

### Default Paths

| Path | Default Value | Purpose |
|------|---------------|---------|
| ChallengeDir | `/var/lib/orvixpanel/acme-challenges` | HTTP-01 challenge files |
| StorageDir | `/var/lib/orvixpanel/ssl/certs` | Certificate storage |
| NginxConfigDir | `/etc/nginx/conf.d/orvix` | Nginx vhost configs |

### Permissions Required

| Path | Owner | Permissions |
|------|-------|-------------|
| `/var/lib/orvixpanel/acme-challenges` | root:www-data | 0755 |
| Challenge files | root:www-data | 0644 |
| `/var/lib/orvixpanel/ssl/certs` | root:root | 0700 |
| Private keys | root:root | 0600 |
| `/etc/nginx/conf.d/orvix` | root:root | 0755 |

---

## Appendix B: ACME Endpoints

### Let's Encrypt Staging

| Endpoint | URL |
|----------|-----|
| Directory | `https://acme-staging-v02.api.letsencrypt.org/directory` |
| New Nonce | `https://acme-staging-v02.api.letsencrypt.org/acme/new-nonce` |
| New Account | `https://acme-staging-v02.api.letsencrypt.org/acme/new-account` |
| New Order | `https://acme-staging-v02.api.letsencrypt.org/acme/new-order` |

### OrvixPanel ACME Route

| Route | Method | Purpose |
|-------|--------|---------|
| `/.well-known/acme-challenge/:token` | GET | Serve challenge file |
| `/api/v1/ssl/certificates/issue` | POST | Issue staging certificate |

---

## Appendix C: Error Codes

| Code | Meaning | Likely Cause |
|------|---------|--------------|
| `domain_required` | Missing domain | Request body missing `domain` |
| `email_required` | Missing email | Request body missing `email` |
| `invalid_body` | Malformed JSON | Request body not valid JSON |
| `staging_issue_failed` | ACME failure | Let's Encrypt communication error |
| `tls` | TLS error | ACME server unreachable |
| `connection_refused` | Network error | Firewall blocking port 80/443 |

---

## Appendix D: Safety Notes

1. **STAGING ONLY**: This playbook uses Let's Encrypt staging API only
2. **No Production Impact**: Staging certificates are not trusted by browsers
3. **Rate Limits**: Let's Encrypt staging has higher rate limits than production
4. **Cleanup**: Always clean up test certificates after verification
5. **Security**: Never commit real credentials to version control
6. **Logs**: Check logs for any unexpected errors during testing

---

**Document End**