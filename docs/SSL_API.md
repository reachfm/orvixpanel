# SSL Engine API v0.5.0

## Base URL

```
/api/v1/ssl
```

## Authentication

All endpoints require JWT authentication via the `Authorization: Bearer <token>` header.

## Endpoints

### List Certificates

```
GET /api/v1/ssl/certificates
```

**Response:**
```json
[
  {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "domain_id": "...",
    "account_id": "...",
    "tenant_id": "tenant123",
    "provider": "letsencrypt",
    "common_name": "example.com",
    "san_names": ["www.example.com", "api.example.com"],
    "status": "issued",
    "auto_renew": true,
    "cert_path": "/var/lib/orvixpanel/ssl/certs/example.com/cert.pem",
    "key_path": "/var/lib/orvixpanel/ssl/certs/example.com/privkey.pem",
    "not_before": "2024-01-01T00:00:00Z",
    "not_after": "2024-03-31T23:59:59Z",
    "serial_number": "03:AB:CD:...",
    "fingerprint": "SHA256:...",
    "issuer": "Let's Encrypt",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### Get Certificate

```
GET /api/v1/ssl/certificates/:id
```

**Response:** Single certificate object

### Issue Certificate

```
POST /api/v1/ssl/certificates
```

**Request:**
```json
{
  "domain": "example.com",
  "san_names": ["www.example.com", "api.example.com"],
  "provider": "letsencrypt",
  "auto_renew": true,
  "acme_account_id": "optional-account-id"
}
```

**Response:** Created certificate object

**Status Codes:**
- `201 Created` - Certificate issued successfully
- `400 Bad Request` - Invalid request body
- `500 Internal Server Error` - Issuance failed

### Renew Certificate

```
POST /api/v1/ssl/certificates/:id/renew
```

**Response:** Updated certificate object

**Status Codes:**
- `200 OK` - Certificate renewed
- `404 Not Found` - Certificate not found
- `500 Internal Server Error` - Renewal failed

### Revoke Certificate

```
POST /api/v1/ssl/certificates/:id/revoke
```

**Response:** No content (204 No Content)

**Status Codes:**
- `204 No Content` - Certificate revoked
- `404 Not Found` - Certificate not found
- `500 Internal Server Error` - Revocation failed

### Delete Certificate

```
DELETE /api/v1/ssl/certificates/:id
```

**Response:** No content (204 No Content)

**Status Codes:**
- `204 No Content` - Certificate deleted
- `404 Not Found` - Certificate not found
- `500 Internal Server Error` - Deletion failed

### Import Certificate

```
POST /api/v1/ssl/import
```

**Request:**
```json
{
  "domain": "example.com",
  "cert_pem": "-----BEGIN CERTIFICATE-----\n...",
  "key_pem": "-----BEGIN PRIVATE KEY-----\n...",
  "chain_pem": "-----BEGIN CERTIFICATE-----\n..."
}
```

**Response:** Created certificate object

### Get Certificate Events

```
GET /api/v1/ssl/certificates/:id/events
```

**Response:**
```json
[
  {
    "id": "event123",
    "certificate_id": "cert123",
    "event_type": "issued",
    "message": "Certificate issued successfully",
    "error_detail": null,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Get All Events

```
GET /api/v1/ssl/events?limit=100
```

**Query Parameters:**
- `limit` (optional): Maximum number of events to return (default: 100)

**Response:** Array of event objects

### Get Health Report

```
GET /api/v1/ssl/health
```

**Response:**
```json
{
  "total": 10,
  "healthy": 8,
  "expiring_soon": 1,
  "expired": 0,
  "failed": 1,
  "certs": [
    {
      "id": "cert123",
      "domain": "example.com",
      "status": "issued",
      "expires_at": "2024-03-31T23:59:59Z",
      "days_until_expiry": 45,
      "issues": []
    }
  ]
}
```

### Get Dashboard Stats

```
GET /api/v1/ssl/dashboard
```

**Response:**
```json
{
  "total_active": 8,
  "expiring_soon": 1,
  "failed_renewals": 0,
  "auto_renew_enabled": 7
}
```

## Error Responses

All endpoints may return these error responses:

```json
{
  "error": "ssl_not_found",
  "message": "Certificate not found"
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `ssl_not_found` | Certificate not found |
| `ssl_list_failed` | Failed to list certificates |
| `ssl_get_failed` | Failed to get certificate |
| `ssl_issue_failed` | Failed to issue certificate |
| `ssl_renew_failed` | Failed to renew certificate |
| `ssl_revoke_failed` | Failed to revoke certificate |
| `ssl_delete_failed` | Failed to delete certificate |
| `ssl_import_failed` | Failed to import certificate |
| `ssl_events_failed` | Failed to get events |
| `ssl_health_failed` | Failed to get health report |
| `invalid_body` | Invalid request body |
| `domain_required` | Domain is required |
| `domain_and_pem_required` | Domain and PEM data required |

## RBAC Permissions

| Permission | Required For |
|------------|-------------|
| `ssl.cert.read` | GET endpoints |
| `ssl.cert.write` | POST/DELETE endpoints |
| `ssl.events.read` | Event endpoints |
| `ssl.health.read` | Health endpoint |
| `ssl.dashboard.read` | Dashboard endpoint |