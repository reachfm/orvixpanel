# OrvixPanel WHMCS Server Module

Drop this directory into your WHMCS installation:

```
/path/to/whmcs/modules/servers/orvixpanel/
```

WHMCS auto-discovers modules placed in `modules/servers/`. After
copying, the module appears as "OrvixPanel" in
**Setup → Products → Servers → Add New Server**.

## Configuration

For each WHMCS server entry that points at an OrvixPanel instance:

| WHMCS field            | Value                                                    |
|------------------------|----------------------------------------------------------|
| **Name**               | OrvixPanel — `<your company>`                            |
| **Hostname**           | `panel.yourhost.com` (no scheme, no port)                |
| **Type**               | OrvixPanel                                               |
| **Username**           | (any — ignored)                                          |
| **Password**           | API key (generated from OrvixPanel → Account → API Keys) |
| **Access Hash**        | (empty)                                                  |
| **Secure**             | ON                                                       |
| **Port**               | 8443 (used to build the API URL)                         |

## Configuration options per product

The module exposes three config options on the WHMCS product:

- **Plan**: `basic` | `pro` | `unlimited`
- **Disk Quota (MB)**: integer, default 10240
- **Bandwidth (GB)**: integer, default 100

## Tested with

- WHMCS 8.6+
- OrvixPanel v1.0+

## Generating an API key in OrvixPanel

1. Log in as the account that should own the WHMCS key (typically
   a dedicated `whmcs` service account with `reseller_admin` role).
2. Navigate to **Account → API Keys → New Key**.
3. Name it `WHMCS Production`, set scopes if you want
   provisioning-only, copy the key.
4. Paste it into the WHMCS server's **Password** field.

## Webhook notifications (Phase 8)

WHMCS → OrvixPanel one-way (provision). For OrvixPanel → WHMCS
(usage-based billing, suspensions due to overage), use the
`POST /api/v1/admin/hooks` endpoint to register a webhook URL
incoming at WHMCS. Phase 8 ships the dispatcher; for now usage
sync is one-shot via the WHMCS daily cron.
