# SSL Certificate Renewal System v0.5.0

## Overview

The renewal scheduler provides automated certificate renewal using file-based locking to prevent duplicate operations in multi-instance deployments.

## How It Works

### Renewal Window

Certificates are renewed when they enter the configured renewal window (default: 30 days before expiry).

```
Expiry Date: March 31, 2024
Renewal Window: 30 days
Renewal Start: March 1, 2024
```

### File-Based Locking

The scheduler uses a lock file to coordinate renewal operations across multiple OrvixPanel instances:

```
/run/orvixpanel/ssl-renew.lock
```

**Lock File Format:**
```
{
  "instance_id": "instance-abc123",
  "started_at": "2024-03-01T00:00:00Z",
  "pid": 1234
}
```

### Lock Acquisition Algorithm

1. Attempt to create lock file (exclusive)
2. If lock exists:
   - Check if lock is stale (> 24 hours old)
   - If stale, remove and retry
   - If not stale, skip this instance
3. If lock acquired, proceed with renewal
4. Release lock after completion

### Renewal Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Renewal Scheduler                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Acquire Lock                                             │
│     ├─ Check existing lock                                   │
│     ├─ If stale, remove it                                   │
│     └─ Create new lock                                        │
│                                                              │
│  2. Query Expiring Certificates                             │
│     └─ SELECT * FROM ssl_certificates                        │
│        WHERE auto_renew = true                               │
│        AND status IN ('issued', 'expiring_soon')             │
│        AND not_after <= NOW() + renewal_window               │
│                                                              │
│  3. For Each Certificate:                                    │
│     ├─ Update status to 'pending'                            │
│     ├─ Log renewal_started event                             │
│     ├─ Call ACME provider to renew                           │
│     ├─ If success:                                            │
│     │   ├─ Store new certificate files                        │
│     │   ├─ Update database record                             │
│     │   └─ Log renewal_succeeded event                      │
│     └─ If failure:                                           │
│         ├─ Increment retry count                              │
│         ├─ If retries < max_retries: set status back        │
│         └─ If retries >= max_retries: set status to 'failed'│
│                                                              │
│  4. Release Lock                                             │
│     └─ Delete lock file                                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

```go
type Config struct {
    // Renewal settings
    RenewalWindowDays    int    // Days before expiry to start renewal (default: 30)
    RenewalLockFile      string // Path to renewal lock file
    MaxRenewalRetries    int    // Max retry attempts (default: 3)
}
```

### Default Values

| Setting | Default Value | Description |
|---------|--------------|-------------|
| `RenewalWindowDays` | 30 | Days before expiry to start renewal |
| `MaxRenewalRetries` | 3 | Maximum retry attempts |
| `RenewalLockFile` | `/run/orvixpanel/ssl-renew.lock` | Lock file path |

## CLI Commands

### Manual Renewal

```bash
# Renew a specific certificate
orvixpanel ssl renew <certificate-id>

# Renew all expiring certificates
orvixpanel ssl renew-all

# Force renewal (ignore lock)
orvixpanel ssl renew --force <certificate-id>
```

### Scheduler Commands

```bash
# Run renewal check once
orvixpanel ssl scheduler run

# Start background scheduler
orvixpanel ssl scheduler start

# Stop background scheduler
orvixpanel ssl scheduler stop
```

### Lock Management

```bash
# Check lock status
orvixpanel ssl lock status

# Force release lock
orvixpanel ssl lock release
```

## Retry Strategy

### Backoff Algorithm

Each retry uses exponential backoff:

```
Retry 1: Immediate
Retry 2: Wait 5 minutes
Retry 3: Wait 30 minutes
```

### Failure Handling

After max retries, the certificate is marked as `failed` and requires manual intervention:

1. Check logs for failure reason
2. Fix underlying issue
3. Reset retry count: `orvixpanel ssl reset-retries <cert-id>`
4. Manually renew: `orvixpanel ssl renew <cert-id>`

## Event Types

| Event | Description |
|-------|-------------|
| `renewal_started` | Renewal process initiated |
| `renewed` | Certificate renewed successfully |
| `renewal_failed` | Renewal failed |
| `retry_scheduled` | Retry scheduled |

## Monitoring

### Health Check

```bash
orvixpanel ssl health
```

### Certificate Status

```bash
orvixpanel ssl list --filter expiring
```

### Renewal Statistics

```bash
orvixpanel ssl stats
```

Output:
```
Total Certificates:     10
Active:                   8
Expiring Soon:            1
Failed:                   1

Last Renewal Check:       2024-03-01 00:00:00
Next Scheduled Check:     2024-03-02 00:00:00
```

## Troubleshooting

### Lock Not Releasing

If a lock file is not released (e.g., crash during renewal):

1. Check if any renewal process is running:
   ```bash
   ps aux | grep orvixpanel | grep ssl
   ```

2. If not running, force release the lock:
   ```bash
   orvixpanel ssl lock release --force
   ```

3. Verify certificate status:
   ```bash
   orvixpanel ssl status <cert-id>
   ```

### Renewal Stuck in Pending

If a certificate is stuck in `pending` status:

1. Check the lock file timestamp
2. Force release if stale
3. Reset status:
   ```bash
   orvixpanel ssl reset-status <cert-id>
   ```
4. Retry renewal

### Rate Limiting

Let's Encrypt has rate limits. If you hit rate limits:

1. Wait for limit window to reset (usually 1 hour)
2. Disable auto-renewal for affected certificates
3. Use staging endpoint for testing:
   ```yaml
   # config.yaml
   ssl:
     lets_encrypt_directory_url: "https://acme-staging-v02.api.letsencrypt.org/directory"
   ```