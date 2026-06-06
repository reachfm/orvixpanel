# OrvixPanel — Full MVP Document
**Version:** 1.0  
**Status:** Active Build  
**Codename:** Nova  
**Stack:** Go · React · Single Binary · Zero Dependencies  

---

## Table of Contents

1. [Vision & Mission](#1-vision--mission)
2. [Competitive Positioning](#2-competitive-positioning)
3. [Tech Stack — Final Decisions](#3-tech-stack--final-decisions)
4. [System Architecture](#4-system-architecture)
5. [License & Monetization Model](#5-license--monetization-model)
6. [Security Architecture](#6-security-architecture)
7. [AI Layer Architecture](#7-ai-layer-architecture)
8. [Phase 1 — Foundation (Weeks 1–2)](#8-phase-1--foundation-weeks-12)
9. [Phase 2 — Core Hosting Engine (Weeks 3–4)](#9-phase-2--core-hosting-engine-weeks-34)
10. [Phase 3 — DNS · Mail · SSL (Weeks 5–6)](#10-phase-3--dns--mail--ssl-weeks-56)
11. [Phase 4 — Security Engine (Weeks 7–8)](#11-phase-4--security-engine-weeks-78)
12. [Phase 5 — Database · Files · Backups (Weeks 9–10)](#12-phase-5--database--files--backups-weeks-910)
13. [Phase 6 — AI Guardian Layer (Weeks 11–12)](#13-phase-6--ai-guardian-layer-weeks-1112)
14. [Phase 7 — Reseller & White-Label Engine (Weeks 13–14)](#14-phase-7--reseller--white-label-engine-weeks-1314)
15. [Phase 8 — Polish · Hardening · v1 Release (Weeks 15–16)](#15-phase-8--polish--hardening--v1-release-weeks-1516)
16. [API Specification](#16-api-specification)
17. [Database Schema](#17-database-schema)
18. [Frontend Architecture](#18-frontend-architecture)
19. [DevOps & Installer](#19-devops--installer)
20. [Testing Strategy](#20-testing-strategy)

---

## 1. Vision & Mission

### What is OrvixPanel?

OrvixPanel is a **next-generation server control panel** built from scratch in Go, designed to replace cPanel/Plesk/DirectAdmin for modern hosting companies, ISPs, data centers, and enterprise infrastructure teams.

It ships as a **single deployable binary** with an embedded React frontend, zero external runtime dependencies, and a three-tier commercial license model. Every feature — from web hosting management to AI-driven auto-healing — is built in-house to avoid AGPL complications, vendor lock-in, and per-seat pricing traps.

### Core Principles

- **Single binary, zero Docker required** — install in under 60 seconds on any Linux server
- **AI-native from day one** — not bolted on later; Guardian Agent is a first-class system
- **Security-first architecture** — eBPF firewall, Zero Trust internal comms, full audit logging
- **White-label ready** — resellers get a fully branded product, not just a logo swap
- **API-first** — every UI action has a corresponding REST endpoint; no hidden state

### Who Is It For?

| Persona | Description | Pain with cPanel |
|---|---|---|
| Hosting company | Sells shared/VPS hosting to end users | cPanel pricing jumped 700% in 2019 |
| ISP | Manages hundreds of servers for customers | No bulk provisioning API |
| Data center | Provides managed infrastructure | No white-label, no modern UI |
| Developer agency | Deploys client sites at scale | No Git deploy, no Docker, no modern DX |
| Enterprise IT | Manages internal servers | No Zero Trust, no SIEM export |

---

## 2. Competitive Positioning

### Direct Competitors

| Feature | cPanel | Plesk | DirectAdmin | **OrvixPanel** |
|---|---|---|---|---|
| Price | $$$$ (per account) | $$$ (per domain) | $$ | Flat per-server |
| UI Age | 1990s design | Okay | Very dated | 2025 modern |
| Single binary | ❌ | ❌ | ❌ | ✅ |
| AI Guardian | ❌ | ❌ | ❌ | ✅ |
| eBPF Firewall | ❌ | ❌ | ❌ | ✅ |
| White-label | Partial | Partial | ❌ | ✅ Full |
| Git deploy | ❌ | Plugin | ❌ | ✅ Native |
| Zero Trust internal | ❌ | ❌ | ❌ | ✅ |
| Web Terminal | Basic | Basic | ❌ | ✅ xterm.js |
| Real-time metrics | Polling | Polling | ❌ | ✅ WebSocket |
| Postlane Mail bridge | ❌ | ❌ | ❌ | ✅ Native |
| WHMCS plugin | ✅ | ✅ | ✅ | ✅ |
| Open API spec | Partial | Partial | ❌ | ✅ Full OpenAPI 3.1 |

### Killer Differentiators

1. **Flat per-server pricing** — no per-account, no per-domain surprises
2. **AI Guardian Agent** — proactive auto-healing, not reactive alerts
3. **eBPF kernel-level firewall** — 10x faster than iptables, zero packet overhead
4. **Native Postlane bridge** — full mail server management without third-party dependencies
5. **True white-label** — custom domain, custom logo, custom color scheme, custom login page
6. **Zero Trust by default** — every internal service communicates over mTLS

---

## 3. Tech Stack — Final Decisions

### Backend

| Component | Technology | Reason |
|---|---|---|
| Language | Go 1.22+ | Single binary, goroutines, low memory, fast compile |
| Web framework | Fiber v3 | Fasthttp under the hood, middleware-rich, WebSocket support |
| ORM | GORM | Familiar, supports SQLite + PostgreSQL seamlessly |
| Primary DB (small) | SQLite (WAL mode) | Zero-dependency, perfect for single-server installs |
| Primary DB (large) | PostgreSQL 15+ | ISP/Enterprise tier, multi-server sync |
| Cache / pub-sub | Redis 7 | Session store, real-time event bus, rate limiting |
| Search | Bleve | Pure Go, embedded full-text search |
| Internal RPC | gRPC + protobuf | Typed inter-service communication |
| Task queue | Asynq (Redis-backed) | Cron jobs, async tasks, retry logic |
| Metrics | Prometheus client_golang | Exposes /metrics endpoint |
| Config | Viper + TOML | Hot-reload without restart |
| Logging | Zerolog | Structured JSON logs, zero allocation |
| Auth tokens | golang-jwt/jwt v5 | JWT + refresh token rotation |
| SSH/Terminal | golang.org/x/crypto/ssh | Server-side SSH exec piped to WebSocket |
| Process management | Custom supervisor in Go | Replaces systemd dependency for child processes |
| Mail bridge | Postlane HTTP API client | Native OrvixPanel ↔ Postlane integration |

### Frontend

| Component | Technology | Reason |
|---|---|---|
| Framework | React 19 | Server components, concurrent mode |
| Build | Vite 6 | Sub-second HMR, optimized production builds |
| Styling | Tailwind CSS v4 | JIT, zero runtime, design tokens |
| Component library | Radix UI + shadcn/ui | Accessible primitives, zero style lock-in |
| State management | Zustand | Lightweight, no boilerplate |
| Server state | TanStack Query v5 | Cache, optimistic updates, WebSocket integration |
| Routing | TanStack Router | Type-safe routes |
| Terminal | xterm.js + fit addon | Full VT100 terminal in browser |
| Charts | Recharts | React-native, responsive |
| Real-time | Native WebSocket + custom hook | Live metrics, log streaming, terminal |
| Icons | Lucide React | Consistent, tree-shakeable |
| Forms | React Hook Form + Zod | Type-safe form validation |
| Code editor | Monaco Editor (lazy-loaded) | nginx/Apache config editing in browser |
| i18n | react-i18next | Multi-language support for white-label |

### Infrastructure & Security

| Component | Technology |
|---|---|
| Firewall | eBPF via Cilium/BPF libraries (pure Go bindings) |
| IDS/IPS | CrowdSec Go client + custom rules engine |
| WAF | Coraza WAF (pure Go ModSecurity-compatible) |
| SSL/TLS | lego library (Let's Encrypt, ZeroSSL ACME) |
| 2FA | TOTP (RFC 6238) + Passkey/WebAuthn |
| Encryption at rest | AES-256-GCM for secrets, Argon2id for passwords |
| Audit log | Append-only log + PostgreSQL write-ahead |
| SIEM export | CEF format over syslog UDP/TCP |
| Internal mTLS | crypto/tls with per-service certificates |
| Secret storage | Custom vault with memory encryption |

### Deployment

| Component | Technology |
|---|---|
| Build | `go build -ldflags="-s -w"` + embedded FS |
| Frontend embedding | `//go:embed dist/*` |
| Target OS | Linux (Debian 11+, Ubuntu 20+, Rocky 8+, AlmaLinux 8+) |
| Architectures | x86_64, ARM64 |
| Installer | Single bash script (curl | bash) |
| Upgrade | Self-update via signed binary download |
| Init system | systemd unit (auto-generated on install) |
| Containerization | Optional Docker image for dev environments only |

---

## 4. System Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    USER LAYER                              │
│  Browser (React PWA) · CLI (orvix-cli) · REST API clients │
│  WHMCS Plugin · Billing system webhooks                    │
└───────────────────┬────────────────────────────────────────┘
                    │ HTTPS / WSS
┌───────────────────▼────────────────────────────────────────┐
│                API GATEWAY (Fiber v3)                      │
│  JWT Auth · API Key Auth · Rate Limiter · CORS             │
│  WebSocket Hub · OpenAPI 3.1 · Multi-tenant router         │
│  Request audit logger · IP allowlist/denylist              │
└──┬──────────┬──────────┬──────────┬──────────┬────────────┘
   │          │          │          │          │
   ▼          ▼          ▼          ▼          ▼
[Hosting]  [DNS]    [Mail]    [Database]  [Security]
 Service   Service  Service    Service     Engine
   │          │          │          │          │
   └──────────┴──────────┴──────────┴──────────┘
                    │ gRPC (mTLS)
┌───────────────────▼────────────────────────────────────────┐
│              INFRASTRUCTURE LAYER                          │
│  SQLite / PostgreSQL · Redis · Bleve · Event Bus           │
│  Prometheus metrics · Zerolog · Config (Viper/TOML)        │
└────────────────────────────────────────────────────────────┘
                    │
┌───────────────────▼────────────────────────────────────────┐
│            AI GUARDIAN LAYER                               │
│  Anomaly detection · Auto-heal engine · LLM insights       │
│  Predictive alerts · Performance advisor · Log analyzer    │
└────────────────────────────────────────────────────────────┘
                    │
┌───────────────────▼────────────────────────────────────────┐
│         RESELLER / LICENSE ENGINE                          │
│  License key validation · White-label config               │
│  Provisioning API · WHMCS bridge · Usage metering          │
└────────────────────────────────────────────────────────────┘
```

### Internal Service Communication

- All inter-service calls use **gRPC over mTLS**
- Each service has a unique TLS certificate issued by the internal CA
- No service trusts another by default (Zero Trust principle)
- External-facing API uses standard HTTPS with Let's Encrypt certificates

### Multi-Tenancy Model

```
Server (physical or VPS)
└── OrvixPanel instance
    ├── Reseller A (white-label: hostingco.com)
    │   ├── Account: alice.com
    │   ├── Account: bob.net
    │   └── Account: carol.org
    ├── Reseller B (white-label: cloudhost.io)
    │   └── Account: dave.com
    └── Admin (OrvixPanel itself)
```

---

## 5. License & Monetization Model

### License Tiers

| Tier | Servers | Accounts | Price | Target |
|---|---|---|---|---|
| **SMB** | 1–10 | Unlimited | $29/month | Small hosting companies |
| **ISP** | Up to 100 | Unlimited | $199/month | ISPs, regional hosters |
| **Enterprise** | Unlimited | Unlimited | $999/month | Data centers |
| **White-Label** | Unlimited | Unlimited | $1,499/month | Resellers with own brand |

### License Key System

```
ORVIX-{TIER}-{YEAR}-{HASH}-{SIGNATURE}

Example: ORVIX-ISP-2025-A3F7B-X9K2M1P
```

**Validation flow:**

1. On startup, OrvixPanel reads license key from `/etc/orvixpanel/license.key`
2. Key is validated against Orvix License Server (HTTPS, with offline grace period of 7 days)
3. License payload contains: tier, expiry date, max servers, feature flags
4. Features are gated at the Go middleware level — not just in the UI
5. License violations lock the panel to read-only mode (no service disruption to existing sites)

### Revenue Projections (Conservative)

| Month | SMB | ISP | Enterprise | MRR |
|---|---|---|---|---|
| 3 | 20 | 5 | 1 | $2,579 |
| 6 | 80 | 20 | 5 | $9,295 |
| 12 | 300 | 80 | 20 | $36,500 |
| 24 | 1000 | 200 | 50 | $98,800 |

---

## 6. Security Architecture

### Defense in Depth — 7 Layers

```
Layer 7: Application — WAF (Coraza), input validation, CSRF, XSS headers
Layer 6: Authentication — JWT, API keys, 2FA TOTP, Passkey/WebAuthn
Layer 5: Authorization — RBAC with 12 default roles, attribute-based policies
Layer 4: Audit — Append-only audit log, tamper detection via SHA-256 chaining
Layer 3: Network — eBPF firewall, CrowdSec IPS, rate limiting per IP/user
Layer 2: Transport — mTLS for internal services, TLS 1.3 minimum for external
Layer 1: Host — Automatic OS hardening script on install, SELinux/AppArmor profiles
```

### eBPF Firewall

```go
// Kernel-space packet filtering — runs at XDP hook before kernel network stack
// 10x faster than iptables, zero copy, zero context switch overhead

type FirewallRule struct {
    ID          uint64
    Priority    uint8
    Protocol    uint8  // TCP=6, UDP=17, ICMP=1
    SrcIP       net.IP
    SrcCIDR     *net.IPNet
    DstPort     uint16
    Action      FirewallAction // ALLOW, DROP, REJECT, LOG
    RateLimit   *RateLimitSpec
    CreatedAt   time.Time
    Description string
}

// Rules are compiled to BPF bytecode and loaded into kernel
// Changes take effect in <1ms with no traffic interruption
```

**Default firewall rules on install:**

```
ALLOW  TCP  22    (SSH — rate limited: 5 attempts/minute per IP)
ALLOW  TCP  80    (HTTP — redirects to HTTPS)
ALLOW  TCP  443   (HTTPS)
ALLOW  TCP  8443  (OrvixPanel admin — IP whitelist recommended)
ALLOW  UDP  53    (DNS if enabled)
DROP   ALL  *     (Default deny everything else)
```

### CrowdSec IPS Integration

```go
type CrowdSecEngine struct {
    LocalAPI    string        // http://localhost:8080
    APIKey      string
    Scenarios   []string      // ssh-bf, http-bf, http-scan, etc.
    BanDuration time.Duration // Default: 4 hours
    Whitelist   []net.IPNet
}

// OrvixPanel runs as a CrowdSec bouncer
// Decisions from CrowdSec → automatic eBPF rule insertion
// Community threat intel: 150,000+ malicious IPs blocked in real-time
```

### Coraza WAF

```toml
# /etc/orvixpanel/waf.toml
[waf]
enabled = true
mode = "detection"  # or "prevention"
ruleset = "OWASP-CRS-3.3"
custom_rules_dir = "/etc/orvixpanel/waf/custom/"
paranoia_level = 2   # 1=low, 4=paranoid
anomaly_threshold = 5

[waf.exclusions]
# Per-account WAF rule exclusions
enabled = true
```

### RBAC — Role Definitions

```go
const (
    RoleRootAdmin      = "root_admin"      // Full system access
    RoleResellerAdmin  = "reseller_admin"  // Reseller management
    RoleResellerAgent  = "reseller_agent"  // View-only reseller
    RoleAccountOwner   = "account_owner"   // Full account control
    RoleAccountDev     = "account_dev"     // No billing, no DNS
    RoleAccountViewer  = "account_viewer"  // Read-only
    RoleMailAdmin      = "mail_admin"      // Mail management only
    RoleDBAdmin        = "db_admin"        // Database management only
    RoleSecurityAdmin  = "security_admin"  // Firewall, WAF, IDS
    RoleMonitor        = "monitor"         // Metrics and logs only
    RoleSupport        = "support"         // Impersonate + read-only
    RoleBilling        = "billing"         // Billing and license only
)
```

### Two-Factor Authentication

```go
type TwoFactorMethod string
const (
    TOTP    TwoFactorMethod = "totp"    // Google Authenticator, Authy
    Passkey TwoFactorMethod = "passkey" // WebAuthn, FIDO2
    Backup  TwoFactorMethod = "backup"  // 10 one-time backup codes
)

// Passkey (WebAuthn) support:
// - Registration: navigator.credentials.create()
// - Authentication: navigator.credentials.get()
// - Stored: credentialID + publicKey in DB
// - Supported authenticators: YubiKey, Touch ID, Face ID, Windows Hello
```

### Audit Log Format

```json
{
  "id": "01J3K...",
  "timestamp": "2025-06-15T14:23:01Z",
  "actor": {
    "user_id": "usr_abc123",
    "email": "admin@example.com",
    "ip": "1.2.3.4",
    "session_id": "sess_xyz",
    "role": "account_owner"
  },
  "action": "domain.create",
  "resource": {
    "type": "domain",
    "id": "dom_789",
    "name": "example.com"
  },
  "result": "success",
  "duration_ms": 42,
  "prev_hash": "sha256:a3f7...",
  "hash": "sha256:b9c2..."
}
```

Each audit entry hashes the previous entry — tampering with historical records is detectable.

---

## 7. AI Layer Architecture

### Guardian Agent — Overview

The Guardian Agent is a background goroutine pool that continuously monitors system health, predicts failures, and executes automated remediation actions.

```
┌─────────────────────────────────────────────────────┐
│                  GUARDIAN AGENT                     │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ Collector│  │Anomaly   │  │  Heal Engine     │  │
│  │ goroutine│→ │Detector  │→ │  (action runner) │  │
│  │ (1s tick)│  │(ML model)│  │                  │  │
│  └──────────┘  └──────────┘  └──────────────────┘  │
│         │                            │              │
│  ┌──────▼──────┐            ┌────────▼───────────┐  │
│  │ Time-series │            │  Alert Manager     │  │
│  │ in-memory   │            │  (WebSocket push)  │  │
│  │ (60m window)│            │                    │  │
│  └─────────────┘            └────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### Metrics Collected (Every 1 Second)

```go
type SystemSnapshot struct {
    Timestamp   time.Time
    CPU         CPUMetrics
    Memory      MemoryMetrics
    Disk        []DiskMetrics
    Network     []NetworkMetrics
    LoadAvg     [3]float64
    Processes   ProcessMetrics
    Nginx       NginxMetrics
    MySQL       []MySQLMetrics
    Redis       RedisMetrics
    Mail        MailMetrics
    Custom      map[string]float64
}
```

### Anomaly Detection

```go
// Uses a lightweight statistical model (no Python, no external ML)
// Algorithm: Modified Z-score with adaptive baseline
// Baseline: rolling 7-day median with seasonal decomposition

type AnomalyDetector struct {
    Metric      string
    Baseline    float64       // 7-day rolling median
    StdDev      float64       // Rolling standard deviation
    Threshold   float64       // Default: 3.5 (modified Z-score)
    MinSamples  int           // Minimum samples before alerting
    Cooldown    time.Duration // Suppress duplicate alerts
}

func (d *AnomalyDetector) IsAnomaly(value float64) (bool, float64) {
    mad := math.Abs(value - d.Baseline)
    modifiedZ := 0.6745 * mad / d.StdDev
    return modifiedZ > d.Threshold, modifiedZ
}
```

### Auto-Heal Actions

```go
type HealAction string
const (
    HealRestartService   HealAction = "restart_service"
    HealReloadConfig     HealAction = "reload_config"
    HealClearCache       HealAction = "clear_cache"
    HealKillProcess      HealAction = "kill_process"
    HealFreeMemory       HealAction = "free_memory"
    HealRotateLogs       HealAction = "rotate_logs"
    HealRepairPermissions HealAction = "repair_permissions"
    HealBlockIP          HealAction = "block_ip"
    HealScaleWorkers     HealAction = "scale_workers"
    HealNotifyAdmin      HealAction = "notify_admin"
)

// Example auto-heal rule:
// IF nginx.status != "running" FOR 30s → restart nginx → notify admin
// IF cpu.usage > 95% FOR 5m → kill top memory consumer → notify admin
// IF disk.usage > 90% → rotate logs → compress old backups → notify admin
// IF mysql.connections > max_connections * 0.9 → restart connection pool → notify
```

### LLM Insights (Optional — API key required)

```go
type InsightRequest struct {
    SystemState SystemSnapshot
    RecentLogs  []LogEntry
    AuditEvents []AuditEntry
    Question    string // e.g. "Why is my site slow right now?"
}

// Calls configured LLM API (OpenAI, Anthropic, local Ollama)
// Returns: plain English explanation + recommended actions
// Used for: alert explanations, performance analysis, security event summaries
```

---

## 8. Phase 1 — Foundation (Weeks 1–2)

**Goal:** Solid base that all other phases build on. Auth, multi-tenancy, license engine, and the core data model.

### Week 1: Project Scaffold & Auth System

#### Directory Structure

```
orvixpanel/
├── cmd/
│   └── orvixpanel/
│       └── main.go              # Entry point
├── internal/
│   ├── api/                     # HTTP handlers
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── rbac.go
│   │   │   ├── ratelimit.go
│   │   │   ├── audit.go
│   │   │   └── tenant.go
│   │   ├── v1/
│   │   │   ├── auth.go
│   │   │   ├── accounts.go
│   │   │   ├── domains.go
│   │   │   ├── hosting.go
│   │   │   ├── dns.go
│   │   │   ├── mail.go
│   │   │   ├── databases.go
│   │   │   ├── files.go
│   │   │   ├── ssl.go
│   │   │   ├── firewall.go
│   │   │   ├── metrics.go
│   │   │   ├── terminal.go
│   │   │   ├── ai.go
│   │   │   └── reseller.go
│   │   └── router.go
│   ├── config/
│   │   └── config.go            # Viper + TOML config
│   ├── db/
│   │   ├── db.go                # GORM setup, migrations
│   │   ├── models/              # All GORM models
│   │   └── migrations/         # SQL migration files
│   ├── auth/
│   │   ├── jwt.go               # Token generation + validation
│   │   ├── totp.go              # TOTP 2FA
│   │   ├── passkey.go           # WebAuthn/FIDO2
│   │   └── session.go           # Redis session store
│   ├── license/
│   │   └── license.go           # License key validation
│   ├── hosting/                 # Web hosting management
│   ├── dns/                     # DNS zone management
│   ├── mail/                    # Postlane bridge
│   ├── ssl/                     # Certificate management
│   ├── database/                # MySQL/PG management
│   ├── files/                   # File manager
│   ├── firewall/                # eBPF + rules engine
│   ├── waf/                     # Coraza WAF
│   ├── ids/                     # CrowdSec client
│   ├── guardian/                # AI Guardian Agent
│   ├── reseller/                # White-label engine
│   ├── backup/                  # Backup system
│   ├── terminal/                # WebSocket terminal
│   ├── metrics/                 # Prometheus + collectors
│   ├── audit/                   # Audit log system
│   ├── notifications/           # Email/Slack/webhook alerts
│   ├── scheduler/               # Asynq task queue
│   └── embed/
│       └── embed.go             # go:embed frontend dist
├── frontend/                    # React app (Vite)
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── stores/
│   │   ├── hooks/
│   │   ├── lib/
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── scripts/
│   ├── install.sh               # One-command installer
│   └── upgrade.sh               # In-place upgrade script
├── configs/
│   └── orvixpanel.example.toml
├── Makefile
├── Dockerfile                   # Dev only
└── go.mod
```

#### Main Entry Point

```go
// cmd/orvixpanel/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/orvixpanel/orvixpanel/internal/api"
    "github.com/orvixpanel/orvixpanel/internal/config"
    "github.com/orvixpanel/orvixpanel/internal/db"
    "github.com/orvixpanel/orvixpanel/internal/guardian"
    "github.com/orvixpanel/orvixpanel/internal/license"
    "github.com/orvixpanel/orvixpanel/internal/scheduler"
)

func main() {
    cfg := config.Load()

    if err := license.Validate(cfg.License.Key); err != nil {
        log.Fatalf("License validation failed: %v", err)
    }

    database := db.Initialize(cfg.Database)
    db.RunMigrations(database)

    sched := scheduler.New(cfg.Redis)
    sched.Start()

    agent := guardian.NewAgent(cfg.Guardian)
    agent.Start()

    server := api.NewServer(cfg, database)

    go func() {
        if err := server.Listen(cfg.Server.BindAddr); err != nil {
            log.Fatalf("Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 30)
    defer cancel()

    agent.Shutdown()
    sched.Shutdown()
    server.ShutdownWithContext(ctx)
}
```

#### Config Structure

```toml
# /etc/orvixpanel/orvixpanel.toml

[server]
bind_addr    = "0.0.0.0:8443"
external_url = "https://panel.yourhost.com"
secret_key   = "change-this-to-a-random-64-byte-string"
debug        = false

[license]
key = "ORVIX-SMB-2025-XXXXX-YYYYY"

[database]
driver   = "sqlite"          # or "postgres"
dsn      = "/var/lib/orvixpanel/data.db"
# For postgres: "host=localhost user=orvix dbname=orvix sslmode=require"

[redis]
addr     = "localhost:6379"
password = ""
db       = 0

[ssl]
auto_ssl       = true
acme_email     = "admin@yourhost.com"
acme_directory = "https://acme-v02.api.letsencrypt.org/directory"
cert_dir       = "/etc/orvixpanel/certs"

[guardian]
enabled          = true
collect_interval = "1s"
anomaly_threshold = 3.5
llm_provider     = ""     # "openai", "anthropic", "ollama", or ""
llm_api_key      = ""

[mail]
postlane_url     = "http://localhost:9090"
postlane_api_key = ""

[firewall]
ebpf_enabled     = true
crowdsec_enabled = true
crowdsec_api_key = ""

[waf]
enabled          = true
mode             = "prevention"
paranoia_level   = 2

[backup]
default_retention_days = 30
s3_endpoint      = ""
s3_bucket        = ""
s3_access_key    = ""
s3_secret_key    = ""

[notifications]
smtp_host        = ""
smtp_port        = 587
smtp_user        = ""
smtp_pass        = ""
from_email       = "noreply@yourhost.com"
slack_webhook    = ""
```

#### JWT Auth System

```go
// internal/auth/jwt.go

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresAt    int64  `json:"expires_at"`
}

type Claims struct {
    UserID    string   `json:"uid"`
    Email     string   `json:"email"`
    Role      string   `json:"role"`
    TenantID  string   `json:"tid"`
    SessionID string   `json:"sid"`
    jwt.RegisteredClaims
}

// Access token: 15 minutes
// Refresh token: 30 days (stored in Redis, rotated on use)
// Rotation: old refresh token invalidated on use
// Revocation: session ID checked against Redis blacklist
```

#### User & Account Models

```go
// internal/db/models/user.go

type User struct {
    ID              string    `gorm:"primarykey;type:varchar(26)"`
    Email           string    `gorm:"uniqueIndex;not null"`
    PasswordHash    string    `gorm:"not null"`
    Role            string    `gorm:"not null;default:'account_owner'"`
    TenantID        string    `gorm:"index;not null"`
    TOTPSecret      string
    TOTPEnabled     bool      `gorm:"default:false"`
    PasskeyEnabled  bool      `gorm:"default:false"`
    FailedLogins    int       `gorm:"default:0"`
    LockedUntil     *time.Time
    LastLoginAt     *time.Time
    LastLoginIP     string
    Status          string    `gorm:"default:'active'"` // active, suspended, deleted
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Account struct {
    ID          string    `gorm:"primarykey;type:varchar(26)"`
    Username    string    `gorm:"uniqueIndex;not null"`
    Domain      string    `gorm:"uniqueIndex;not null"`
    ResellerID  string    `gorm:"index"`
    Plan        string    `gorm:"not null"` // basic, pro, unlimited
    DiskQuotaMB int       `gorm:"default:10240"`
    BandwidthGB int       `gorm:"default:100"`
    Status      string    `gorm:"default:'active'"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Week 2: Multi-Tenancy, License Engine & Core Middleware

#### Tenant Middleware

```go
// internal/api/middleware/tenant.go

func TenantMiddleware(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        claims := GetClaims(c)
        
        // Resolve tenant from JWT claim
        var tenant Tenant
        if err := db.Where("id = ?", claims.TenantID).First(&tenant).Error; err != nil {
            return fiber.ErrForbidden
        }

        // Check tenant status
        if tenant.Status != "active" {
            return fiber.NewError(403, "tenant_suspended")
        }

        // Check feature flags from license
        lic := license.Get()
        if !lic.HasFeature(c.Route().Name) {
            return fiber.NewError(403, "feature_not_licensed")
        }

        c.Locals("tenant", tenant)
        return c.Next()
    }
}
```

#### License Engine

```go
// internal/license/license.go

type License struct {
    Tier        string    `json:"tier"`      // smb, isp, enterprise, whitelabel
    MaxServers  int       `json:"max_servers"`
    ExpiresAt   time.Time `json:"expires_at"`
    Features    []string  `json:"features"`
    LicensedTo  string    `json:"licensed_to"`
    IssuedAt    time.Time `json:"issued_at"`
}

func Validate(key string) error {
    // 1. Parse key format: ORVIX-{TIER}-{YEAR}-{HASH}-{SIG}
    // 2. Verify ECDSA signature (public key embedded in binary)
    // 3. Check expiry
    // 4. Phone home to license server (with 7-day offline grace)
    // 5. Cache result in memory
}

func (l *License) HasFeature(feature string) bool {
    for _, f := range l.Features {
        if f == feature || f == "*" {
            return true
        }
    }
    return false
}

// Feature list by tier:
var TierFeatures = map[string][]string{
    "smb": {
        "hosting.*", "dns.*", "mail.basic", "ssl.*",
        "database.*", "files.*", "firewall.basic",
        "guardian.basic", "backup.local",
    },
    "isp": {
        "*", "!guardian.llm", "!whitelabel.*",
    },
    "enterprise": {
        "*", "!whitelabel.*",
    },
    "whitelabel": {
        "*",
    },
}
```

#### RBAC Middleware

```go
// internal/api/middleware/rbac.go

type Permission struct {
    Resource string // "domain", "database", "firewall", etc.
    Action   string // "create", "read", "update", "delete", "execute"
}

var RolePermissions = map[string][]Permission{
    RoleRootAdmin: {{"*", "*"}},
    RoleAccountOwner: {
        {"domain", "*"}, {"hosting", "*"}, {"database", "*"},
        {"files", "*"}, {"mail", "*"}, {"ssl", "*"},
        {"dns", "*"}, {"backup", "*"}, {"metrics", "read"},
    },
    RoleAccountDev: {
        {"domain", "read"}, {"hosting", "*"}, {"database", "*"},
        {"files", "*"}, {"metrics", "read"},
    },
    // ... other roles
}

func RequirePermission(resource, action string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        role := GetClaims(c).Role
        if !hasPermission(role, resource, action) {
            return fiber.ErrForbidden
        }
        return c.Next()
    }
}
```

---

## 9. Phase 2 — Core Hosting Engine (Weeks 3–4)

**Goal:** Full web hosting management — virtual hosts, PHP/Node/Python apps, resource quotas.

### Week 3: Virtual Host Management

#### Domain Model

```go
type Domain struct {
    ID          string    `gorm:"primarykey;type:varchar(26)"`
    AccountID   string    `gorm:"index;not null"`
    Name        string    `gorm:"uniqueIndex;not null"`
    Type        string    `gorm:"default:'main'"` // main, addon, subdomain, parked, redirect
    DocumentRoot string   `gorm:"not null"`
    PHPVersion  string    `gorm:"default:'8.3'"`
    Runtime     string    `gorm:"default:'php'"` // php, node, python, ruby, static
    SSLEnabled  bool      `gorm:"default:false"`
    SSLCertID   string
    HTACCESS    bool      `gorm:"default:true"`
    Status      string    `gorm:"default:'active'"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### Nginx Config Generator

```go
// internal/hosting/nginx.go

type VHostConfig struct {
    Domain      *Domain
    Account     *Account
    SSL         *SSLCert
    PHP         *PHPConfig
}

func GenerateNginxConfig(v *VHostConfig) string {
    // Generates optimized nginx server block
    // Includes: gzip, security headers, PHP-FPM upstream, SSL, rate limiting
    // Writes to /etc/nginx/conf.d/orvix/{account}/{domain}.conf
    // Reloads nginx: nginx -s reload (zero downtime)
}

func GeneratePHPFPMPool(account *Account, domain *Domain) string {
    // Per-domain PHP-FPM pool with:
    // - Isolated user (account username)
    // - Resource limits (memory, max children)
    // - open_basedir restriction
    // Writes to /etc/php/{version}/fpm/pool.d/orvix-{domain}.conf
}
```

#### Multi-PHP Version Support

```go
// Supported PHP versions: 7.4, 8.0, 8.1, 8.2, 8.3
// Each version runs as separate PHP-FPM service
// Per-domain version selection via UI dropdown
// Extensions: common set auto-installed, custom extensions per account

type PHPConfig struct {
    Version     string
    MemoryLimit string // "256M"
    MaxExecTime int    // seconds
    UploadSize  string // "64M"
    OpcacheEnabled bool
    Extensions  []string
    CustomINI   map[string]string
}
```

#### Application Runtime Support

```go
type RuntimeManager struct {
    // Node.js: managed via nvm, per-domain .nvmrc
    // Python: managed via pyenv virtualenv, per-domain
    // Ruby: managed via rbenv, per-domain Gemfile
    // Static: pure nginx, no backend process

    Node   *NodeManager
    Python *PythonManager
    Ruby   *RubyManager
}

// Each app gets:
// - Dedicated Unix socket
// - Systemd unit (auto-generated)
// - Process supervision (restart on crash)
// - Log file (/var/log/orvix/{account}/{domain}.log)
// - Resource limits via systemd cgroups
```

### Week 4: Resource Management & Git Deploy

#### Resource Quotas

```go
type ResourceQuota struct {
    AccountID    string
    DiskUsedMB   int64
    DiskLimitMB  int64
    InodUsed     int64
    InodeLimit   int64
    BandwidthGB  int64
    BandwidthUsedGB int64
    CPUPercent   float64   // cgroup CPU limit
    MemoryMB     int64     // cgroup memory limit
    ProcessLimit int       // max procs per account
}

// Disk quotas: Linux disk quota (quotactl syscall)
// CPU/Memory: systemd cgroups v2
// Bandwidth: collected from nginx access logs (goroutine)
// Inode: df -i per account home directory
```

#### Git Deploy System

```go
type GitDeploy struct {
    ID          string
    DomainID    string
    RepoURL     string    // github.com/user/repo
    Branch      string    // main
    AutoDeploy  bool      // webhook trigger
    BuildCmd    string    // npm run build
    OutputDir   string    // dist/
    EnvVars     map[string]string
    LastDeploy  *time.Time
    LastCommit  string
    Status      string    // idle, deploying, failed, success
}

func (g *GitDeploy) Execute(ctx context.Context) error {
    // 1. git clone or git pull
    // 2. Install dependencies (npm/pip/bundle)
    // 3. Run build command
    // 4. Atomic swap: mv new_release → document_root
    // 5. Reload PHP-FPM / restart Node app
    // 6. Store deployment record (for rollback)
    // Zero-downtime: symlink-based atomic deployment
}

// Webhook endpoint: POST /api/v1/domains/{id}/deploy/webhook
// Verifies GitHub/GitLab webhook secret before deploying
```

#### Cron Job Manager

```go
type CronJob struct {
    ID        string
    AccountID string
    DomainID  string
    Command   string
    Schedule  string // cron expression: "0 */6 * * *"
    User      string // runs as account user (sandboxed)
    Enabled   bool
    LastRun   *time.Time
    LastExit  int
    LogFile   string
}

// Implemented via: system crontab with per-account user isolation
// Resource limits: ulimit applied before execution
// Timeout: configurable max runtime
// Logging: output captured and stored (queryable via API)
```

---

## 10. Phase 3 — DNS · Mail · SSL (Weeks 5–6)

**Goal:** Full DNS zone management, Postlane mail bridge, automated SSL management.

### Week 5: DNS Manager

#### DNS Zone Model

```go
type DNSZone struct {
    ID        string    `gorm:"primarykey;type:varchar(26)"`
    AccountID string    `gorm:"index;not null"`
    Domain    string    `gorm:"uniqueIndex;not null"`
    SOAEmail  string
    TTL       int       `gorm:"default:3600"`
    Status    string    `gorm:"default:'active'"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Records   []DNSRecord `gorm:"foreignKey:ZoneID"`
}

type DNSRecord struct {
    ID       string `gorm:"primarykey;type:varchar(26)"`
    ZoneID   string `gorm:"index;not null"`
    Type     string `gorm:"not null"` // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, PTR
    Name     string `gorm:"not null"` // "@", "www", "mail"
    Value    string `gorm:"not null"`
    Priority int    // For MX, SRV records
    TTL      int    `gorm:"default:3600"`
    DNSSEC   bool   `gorm:"default:false"`
}
```

#### PowerDNS Integration

```go
// OrvixPanel manages PowerDNS via its HTTP API
// PowerDNS is installed separately as a dependency

type PowerDNSClient struct {
    BaseURL string // http://localhost:8081
    APIKey  string
}

func (c *PowerDNSClient) CreateZone(zone *DNSZone) error { ... }
func (c *PowerDNSClient) AddRecord(zoneID string, rec *DNSRecord) error { ... }
func (c *PowerDNSClient) DeleteRecord(zoneID, recordID string) error { ... }
func (c *PowerDNSClient) ExportZone(domain string) (string, error) { ... } // BIND format

// DNSSEC: auto-keyed per zone
// Zone transfer: AXFR/IXFR supported
// Import: BIND zone file parser built-in
```

#### Auto DNS Templates

```go
// When a domain is added, auto-create standard DNS records:
var DefaultDNSTemplate = []DNSRecord{
    {Type: "A",    Name: "@",    Value: "{server_ip}",   TTL: 3600},
    {Type: "A",    Name: "www",  Value: "{server_ip}",   TTL: 3600},
    {Type: "A",    Name: "mail", Value: "{server_ip}",   TTL: 3600},
    {Type: "MX",   Name: "@",    Value: "mail.{domain}", Priority: 10, TTL: 3600},
    {Type: "TXT",  Name: "@",    Value: "v=spf1 ip4:{server_ip} ~all", TTL: 3600},
    {Type: "TXT",  Name: "_dmarc", Value: "v=DMARC1; p=quarantine; rua=mailto:dmarc@{domain}", TTL: 3600},
    {Type: "CNAME", Name: "webmail", Value: "mail.{domain}", TTL: 3600},
    {Type: "CAA",  Name: "@",    Value: "0 issue \"letsencrypt.org\"", TTL: 3600},
}
```

### Week 6: Mail Bridge & SSL

#### Postlane Bridge

```go
// internal/mail/postlane.go
// OrvixPanel talks to a running Postlane instance via HTTP API

type PostlaneClient struct {
    BaseURL string
    APIKey  string
    Timeout time.Duration
}

// Operations managed via OrvixPanel UI:
func (p *PostlaneClient) CreateMailbox(accountID, email, password string) error
func (p *PostlaneClient) DeleteMailbox(email string) error
func (p *PostlaneClient) SetMailboxQuota(email string, quotaMB int) error
func (p *PostlaneClient) CreateAlias(from, to string) error
func (p *PostlaneClient) GetMailboxStats(email string) (*MailboxStats, error)
func (p *PostlaneClient) GetQueueStatus() (*QueueStats, error)
func (p *PostlaneClient) GetDKIMKey(domain string) (string, error)
func (p *PostlaneClient) FlushQueue() error
func (p *PostlaneClient) BlockSender(email string) error

type Mailbox struct {
    ID          string
    AccountID   string
    Email       string
    QuotaMB     int
    UsedMB      int
    Aliases     []string
    AutoReply   *AutoReplyConfig
    SpamFilter  *SpamConfig
    Status      string
    CreatedAt   time.Time
}
```

#### SSL Certificate Management

```go
// internal/ssl/ssl.go

type SSLCert struct {
    ID          string    `gorm:"primarykey;type:varchar(26)"`
    DomainID    string    `gorm:"index;not null"`
    Domain      string    `gorm:"not null"`
    SANs        []string  `gorm:"serializer:json"`
    Provider    string    `gorm:"default:'letsencrypt'"` // letsencrypt, zerossl, custom
    CertPath    string
    KeyPath     string
    ChainPath   string
    IssuedAt    time.Time
    ExpiresAt   time.Time
    AutoRenew   bool      `gorm:"default:true"`
    Status      string    `gorm:"default:'pending'"` // pending, active, expired, failed
}

// Auto-renew scheduler:
// - Checks certificates daily
// - Renews 30 days before expiry
// - Supports wildcard certs (DNS-01 challenge via PowerDNS)
// - Fallback to ZeroSSL if Let's Encrypt rate-limited
// - Sends alert if renewal fails

// Challenge types supported:
// HTTP-01: for standard domains (faster, simpler)
// DNS-01: for wildcard certs (*.example.com)
```

---

## 11. Phase 4 — Security Engine (Weeks 7–8)

**Goal:** Production-grade security that makes OrvixPanel the most secure panel on the market.

### Week 7: WAF, eBPF Firewall, IDS

#### Coraza WAF Setup

```go
// internal/waf/waf.go

type WAFEngine struct {
    waf         coraza.WAF
    mode        string // detection, prevention
    auditLogger *AuditLogger
}

func NewWAFEngine(cfg *config.WAFConfig) (*WAFEngine, error) {
    waf, err := coraza.NewWAF(coraza.NewWAFConfig().
        WithDirectives(fmt.Sprintf(`
            SecRuleEngine %s
            SecRequestBodyAccess On
            SecRequestBodyLimit 134217728
            SecResponseBodyAccess On
            SecAuditEngine On
            Include /etc/orvixpanel/waf/coreruleset/crs-setup.conf
            Include /etc/orvixpanel/waf/coreruleset/rules/*.conf
        `, modeDirective(cfg.Mode))),
    )
    // Coraza runs as Fiber middleware — every request is inspected
}

// Custom rule example:
// SecRule REQUEST_URI "@contains /wp-admin" "id:1001,phase:1,deny,msg:'WordPress admin blocked for non-WP sites'"
```

#### eBPF Firewall Implementation

```go
// internal/firewall/ebpf.go

type EBPFFirewall struct {
    program   *ebpf.Program
    ruleMap   *ebpf.Map
    statsMap  *ebpf.Map
    iface     string
}

func (f *EBPFFirewall) LoadProgram() error {
    // Load pre-compiled BPF bytecode (embedded in binary via go:embed)
    // Attach to XDP hook on network interface
    // XDP = eXpress Data Path: processes packets before kernel network stack
    // Throughput: >10Mpps on modern hardware
}

func (f *EBPFFirewall) AddRule(rule *FirewallRule) error {
    // Compile rule to BPF map entry
    // Changes take effect immediately (<1ms) without program reload
    // No iptables. No netfilter. No kernel context switches.
}

func (f *EBPFFirewall) GetStats() *FirewallStats {
    // Read per-rule packet/byte counters from BPF map
    // Returns: packets allowed, dropped, rate-limited per rule
}
```

#### CrowdSec Integration

```go
// internal/ids/crowdsec.go

type CrowdSecBouncer struct {
    client   *csbouncer.LiveBouncer
    firewall *EBPFFirewall
}

func (b *CrowdSecBouncer) Start() {
    // Poll CrowdSec Local API every 10s for new decisions
    // Decision types: ban, throttle, captcha
    // On ban: add eBPF DROP rule for offending IP
    // On unban: remove eBPF rule
    
    // Community blocklist: auto-subscribed
    // Lists: tor exit nodes, scanners, bruteforcers, botnets
}

// OrvixPanel also acts as a CrowdSec signal emitter:
// Failed logins → signal to CrowdSec
// WAF blocks → signal to CrowdSec
// Port scans → signal to CrowdSec
// Community shares signals = free threat intel
```

### Week 8: Auth Hardening & Security Dashboard

#### Brute Force Protection

```go
// internal/auth/bruteforce.go

type BruteForceProtection struct {
    redis *redis.Client
}

func (b *BruteForceProtection) CheckLogin(ip, email string) error {
    // Per-IP: 10 attempts per 10 minutes → temporary ban
    // Per-email: 5 attempts per 5 minutes → account lockout
    // Global: >1000 failed logins/minute → emergency IP block + alert
    
    ipKey := fmt.Sprintf("bf:ip:%s", ip)
    emailKey := fmt.Sprintf("bf:email:%s", email)
    
    ipCount := b.redis.Incr(ctx, ipKey)
    b.redis.Expire(ctx, ipKey, 10*time.Minute)
    
    if ipCount > 10 {
        b.firewall.TempBanIP(ip, 1*time.Hour)
        return ErrIPBanned
    }
    // ...
}
```

#### Security Score System

```go
// Displayed on dashboard — gamified security hardening
type SecurityScore struct {
    Total         int  // 0-100
    SSL           bool // +15 points
    TwoFA         bool // +20 points
    Firewall      bool // +15 points
    WAF           bool // +15 points
    StrongPasswords bool // +10 points
    NoOldPHP      bool // +10 points
    BackupsEnabled bool // +10 points
    EmailVerified bool // +5 points
}

func CalcSecurityScore(account *Account) SecurityScore {
    // Calculates and displays actionable recommendations
    // e.g. "Enable 2FA to add 20 points and secure your account"
}
```

---

## 12. Phase 5 — Database · Files · Backups (Weeks 9–10)

**Goal:** Complete database management, modern web file manager, and automated backup system.

### Week 9: Database Manager & File Manager

#### MySQL/PostgreSQL Management

```go
type ManagedDatabase struct {
    ID          string
    AccountID   string
    Name        string
    Type        string // mysql, postgresql
    Host        string // localhost
    Port        int
    Username    string
    PasswordHash string
    Charset     string // utf8mb4
    Collation   string // utf8mb4_unicode_ci
    SizeMB      int64
    Status      string
    CreatedAt   time.Time
}

// Operations:
// Create/drop database and user (via privileged DB connection)
// Export: mysqldump / pg_dump → compressed .sql.gz
// Import: streaming upload → pipe to mysql / psql
// phpMyAdmin-free: custom web SQL editor (Monaco Editor + REST API)
// Remote access: toggle per database, with IP whitelist
```

#### Web File Manager

```go
// internal/files/manager.go
// All file operations run as the account's system user (via setuid)

type FileManager struct {
    RootPath  string // /home/{account}/
    User      string // account username
    MaxUpload int64  // bytes
}

// Operations:
func (f *FileManager) List(path string) ([]FileInfo, error)
func (f *FileManager) Read(path string) (io.Reader, error)
func (f *FileManager) Write(path string, content io.Reader) error
func (f *FileManager) Delete(paths []string) error
func (f *FileManager) Move(src, dst string) error
func (f *FileManager) Copy(src, dst string) error
func (f *FileManager) Compress(paths []string, archiveName string) error
func (f *FileManager) Extract(archivePath string) error
func (f *FileManager) SetPermissions(path string, mode fs.FileMode) error
func (f *FileManager) Edit(path string, content string) error // With Monaco Editor

// Frontend: React drag-and-drop file manager
// Upload: chunked upload with progress bar (tus protocol)
// Preview: images, text, HTML
// Code edit: syntax highlighting via Monaco Editor
```

### Week 10: Backup System

#### Backup Configuration

```go
type BackupPolicy struct {
    ID          string
    AccountID   string
    Schedule    string // cron: "0 2 * * *"
    RetainDays  int    // 30
    Target      string // local, s3, sftp
    Compress    bool   // zstd compression
    Encrypt     bool   // AES-256-GCM
    EncryptKey  string // KDF from master key
    IncludeDB   bool
    IncludeFiles bool
    IncludeMail bool
    Status      string
}

type BackupJob struct {
    ID         string
    PolicyID   string
    AccountID  string
    StartedAt  time.Time
    FinishedAt *time.Time
    SizeMB     int64
    Files      int
    Status     string // running, success, failed
    Error      string
    StoragePath string
}

// Backup process:
// 1. Snapshot files (tar + zstd)
// 2. Dump databases (mysqldump / pg_dump)
// 3. Export mail (Postlane API)
// 4. Encrypt archive (AES-256-GCM)
// 5. Upload to target (local/S3/SFTP)
// 6. Verify integrity (SHA-256 checksum)
// 7. Purge old backups (retain policy)
// 8. Record in DB + notify on failure

// Restore: one-click from UI
// Partial restore: files only, DB only, or mail only
```

---

## 13. Phase 6 — AI Guardian Layer (Weeks 11–12)

**Goal:** Production-ready AI Guardian Agent with auto-healing, anomaly detection, and LLM-powered insights.

### Week 11: Guardian Agent Core

#### Metric Collector

```go
// internal/guardian/collector.go

type MetricCollector struct {
    interval time.Duration
    buffer   *RingBuffer // 60-minute in-memory window
    sources  []MetricSource
}

// Metric sources (each runs in its own goroutine):
type MetricSource interface {
    Name() string
    Collect() (map[string]float64, error)
}

// Implemented sources:
// CPUSource      — /proc/stat
// MemorySource   — /proc/meminfo
// DiskSource     — /proc/diskstats
// NetworkSource  — /proc/net/dev
// NginxSource    — nginx status module
// MySQLSource    — SHOW GLOBAL STATUS
// RedisSource    — INFO all
// PHPFPMSource   — php-fpm status page
// PostlaneSource — Postlane /metrics endpoint
// ProcessSource  — /proc/{pid}/stat per managed process
```

#### Alert Manager

```go
type AlertRule struct {
    ID          string
    Name        string
    Metric      string    // "cpu.usage_percent"
    Operator    string    // "gt", "lt", "eq"
    Threshold   float64   // 90.0
    Duration    string    // "5m" — must be sustained for this long
    Severity    string    // "info", "warning", "critical"
    Channels    []string  // "email", "slack", "webhook"
    AutoHeal    bool
    HealActions []HealAction
    Cooldown    string    // "30m" — minimum time between repeat alerts
}

// Default rules shipped with OrvixPanel:
// CPU > 90% for 5m         → Warning + auto-kill top process
// Memory > 90% for 2m      → Warning + flush caches
// Disk > 85%               → Warning + rotate logs
// Disk > 95%               → Critical + emergency alert
// nginx down for 30s       → Critical + auto-restart
// MySQL down for 30s       → Critical + auto-restart
// Any service OOM killed   → Critical + auto-restart + alert
// SSL expiry < 7 days      → Critical + force renewal attempt
// Failed logins > 50/min   → Critical + auto-block + alert
```

### Week 12: LLM Insights & Smart Advisor

#### LLM Integration

```go
// internal/guardian/llm.go

type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

// Implementations:
// OpenAIProvider    — GPT-4o via OpenAI API
// AnthropicProvider — Claude via Anthropic API
// OllamaProvider    — local model (llama3, mistral) for privacy
// DisabledProvider  — fallback when no API key configured

func (g *GuardianAgent) ExplainAlert(alert *Alert) string {
    // Builds prompt with: system state, recent logs, alert details
    // Returns: plain-English explanation + top 3 recommended actions
    // Cached: same alert type within 10 minutes returns cached response
}

func (g *GuardianAgent) AnalyzePerformance(account *Account) *PerformanceReport {
    // Analyzes 7-day metrics trend
    // Identifies: top resource consumers, peak times, bottlenecks
    // Returns: structured report with recommendations
}
```

---

## 14. Phase 7 — Reseller & White-Label Engine (Weeks 13–14)

**Goal:** Complete reseller infrastructure — white-label branding, WHMCS integration, provisioning API.

### Week 13: Reseller Engine

#### Reseller Model

```go
type Reseller struct {
    ID              string    `gorm:"primarykey;type:varchar(26)"`
    ParentID        string    // "" for root reseller (direct OrvixPanel customer)
    CompanyName     string
    Domain          string    // panel.theircompany.com
    LogoURL         string
    FaviconURL      string
    PrimaryColor    string    // hex: #1a73e8
    SecondaryColor  string
    CustomCSS       string
    SupportEmail    string
    SupportPhone    string
    SupportURL      string
    BrandName       string    // replaces "OrvixPanel" in UI
    TermsURL        string
    PrivacyURL      string
    AllowedFeatures []string  `gorm:"serializer:json"`
    MaxAccounts     int
    MaxDiskGB       int
    MaxBandwidthGB  int
    Status          string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

#### White-Label Theming

```go
// Theme is injected into the embedded React frontend at runtime
// Via /api/v1/public/theme endpoint (no auth required)

type ThemeConfig struct {
    BrandName    string `json:"brand_name"`
    LogoURL      string `json:"logo_url"`
    FaviconURL   string `json:"favicon_url"`
    PrimaryColor string `json:"primary_color"`
    SecondaryColor string `json:"secondary_color"`
    CustomCSS    string `json:"custom_css"`
    SupportEmail string `json:"support_email"`
    SupportURL   string `json:"support_url"`
    LoginBgURL   string `json:"login_bg_url"`
    FooterText   string `json:"footer_text"`
    HideCredit   bool   `json:"hide_credit"` // White-label tier only
}

// DNS setup for white-label:
// panel.theirdomain.com → CNAME → server_ip
// OrvixPanel detects hostname → loads reseller theme
// SSL: auto-provisioned for reseller's custom domain
```

#### Provisioning API

```go
// Full REST API for automated account creation (WHMCS, Blesta, etc.)
// All endpoints require API key in X-API-Key header

// POST /api/v1/provision/account
// Creates: system user, home directory, PHP-FPM pool, nginx config, DNS zone
// Returns: account credentials, panel URL

// POST /api/v1/provision/account/{id}/suspend
// POST /api/v1/provision/account/{id}/unsuspend
// DELETE /api/v1/provision/account/{id}

// GET /api/v1/provision/account/{id}/usage
// Returns: disk, bandwidth, inodes, account status

// POST /api/v1/provision/account/{id}/password
// Resets main cPanel account password
```

### Week 14: WHMCS Plugin & Billing Hooks

#### WHMCS Module

```
whmcs-orvixpanel/
├── modules/servers/orvixpanel/
│   ├── orvixpanel.php          # Main module file
│   ├── lib/
│   │   ├── OrvixClient.php     # HTTP client for OrvixPanel API
│   │   └── Formatter.php
│   └── templates/
│       ├── overview.tpl         # Client area widget
│       └── login.tpl            # SSO button
```

```php
// Required WHMCS module functions:
function orvixpanel_CreateAccount($params) { ... }
function orvixpanel_SuspendAccount($params) { ... }
function orvixpanel_UnsuspendAccount($params) { ... }
function orvixpanel_TerminateAccount($params) { ... }
function orvixpanel_ChangePassword($params) { ... }
function orvixpanel_ChangePackage($params) { ... }
function orvixpanel_UsageUpdate($params) { ... }  // disk/bandwidth sync
function orvixpanel_ClientAreaPage($params) { ... } // overview widget
function orvixpanel_AdminLink($params) { ... }      // direct login link
```

---

## 15. Phase 8 — Polish · Hardening · v1 Release (Weeks 15–16)

**Goal:** Production-ready v1.0. Performance, security audit, installer, documentation.

### Week 15: Performance & Polish

#### Performance Targets

| Metric | Target |
|---|---|
| Panel page load | < 800ms (cold) |
| API response (simple) | < 50ms p99 |
| API response (complex) | < 500ms p99 |
| WebSocket latency | < 10ms |
| Binary size | < 30MB |
| RAM usage (idle) | < 80MB |
| RAM usage (100 accounts) | < 256MB |

#### Go Performance Optimizations

```go
// Connection pooling
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

// Redis connection pool
rdb = redis.NewClient(&redis.Options{
    PoolSize:    20,
    MinIdleConns: 5,
})

// Response compression
app.Use(compress.New(compress.Config{
    Level: compress.LevelBestSpeed,
}))

// Prepared statements for hot queries
// Index all foreign keys and commonly queried columns
// Batch inserts for metrics (buffer 1s of data, flush in bulk)
```

### Week 16: Security Audit & v1 Release

#### Pre-Release Security Checklist

```
Authentication:
[ ] JWT secret rotation mechanism
[ ] Session fixation protection
[ ] Refresh token rotation verified
[ ] 2FA bypass tested and blocked
[ ] Password reset flow tested
[ ] Account lockout verified

Authorization:
[ ] RBAC bypass attempts (vertical privilege escalation)
[ ] Tenant isolation verified (cross-tenant data access)
[ ] Feature flag bypass tested
[ ] Direct object reference tested (IDOR)

Input Validation:
[ ] SQL injection (all ORM queries use parameterization)
[ ] XSS in all user-facing inputs
[ ] Path traversal in file manager
[ ] Command injection in nginx/PHP config generation
[ ] SSRF in Git deploy URL
[ ] XML/YAML bomb in config import

Infrastructure:
[ ] eBPF firewall bypass tested
[ ] WAF rule coverage verified
[ ] Rate limiting verified under load
[ ] Backup encryption verified (decrypt test)
[ ] Audit log tamper detection tested

Penetration Testing:
[ ] External pentest by qualified firm
[ ] Bug bounty program launch
[ ] CVD (Coordinated Vulnerability Disclosure) policy published
```

#### Installer Script

```bash
#!/bin/bash
# install.sh — OrvixPanel one-command installer

set -euo pipefail

ORVIX_VERSION="1.0.0"
ORVIX_ARCH=$(uname -m)
INSTALL_DIR="/opt/orvixpanel"
DATA_DIR="/var/lib/orvixpanel"
LOG_DIR="/var/log/orvixpanel"
CONF_DIR="/etc/orvixpanel"
SYSTEMD_UNIT="/etc/systemd/system/orvixpanel.service"

check_os() {
    . /etc/os-release
    case $ID in
        ubuntu|debian|rocky|almalinux|centos) ;;
        *) echo "Unsupported OS: $ID"; exit 1 ;;
    esac
}

install_dependencies() {
    # nginx, mysql-server, redis, php-fpm, powerdns
    # Detected from config — only install what's enabled
}

download_binary() {
    BINARY_URL="https://releases.orvixpanel.com/v${ORVIX_VERSION}/orvixpanel-linux-${ORVIX_ARCH}"
    curl -fsSL "${BINARY_URL}.sha256" -o /tmp/orvixpanel.sha256
    curl -fsSL "${BINARY_URL}" -o /tmp/orvixpanel
    sha256sum -c /tmp/orvixpanel.sha256
    install -m 755 /tmp/orvixpanel "${INSTALL_DIR}/orvixpanel"
}

create_systemd_unit() {
    cat > "${SYSTEMD_UNIT}" << EOF
[Unit]
Description=OrvixPanel Server Control Panel
After=network.target

[Service]
Type=simple
User=orvixpanel
ExecStart=${INSTALL_DIR}/orvixpanel
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment=ORVIX_CONFIG=${CONF_DIR}/orvixpanel.toml

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable orvixpanel
    systemctl start orvixpanel
}

echo "✓ OrvixPanel v${ORVIX_VERSION} installed"
echo "→ Admin panel: https://$(hostname -I | awk '{print $1}'):8443"
echo "→ Config: ${CONF_DIR}/orvixpanel.toml"
echo "→ Logs: journalctl -u orvixpanel -f"
```

---

## 16. API Specification

### Base URL

```
https://{panel_domain}:8443/api/v1
```

### Authentication

```http
Authorization: Bearer {jwt_access_token}
X-API-Key: {api_key}         # For machine-to-machine (WHMCS, CLI)
X-Tenant-ID: {tenant_id}     # Required for reseller context
```

### Core Endpoints

```yaml
# Auth
POST   /auth/login
POST   /auth/refresh
POST   /auth/logout
POST   /auth/2fa/setup
POST   /auth/2fa/verify
POST   /auth/passkey/register
POST   /auth/passkey/authenticate
POST   /auth/password/reset

# Accounts
GET    /accounts
POST   /accounts
GET    /accounts/{id}
PUT    /accounts/{id}
DELETE /accounts/{id}
POST   /accounts/{id}/suspend
POST   /accounts/{id}/unsuspend
POST   /accounts/{id}/login   # Single Sign-On into account

# Domains
GET    /accounts/{id}/domains
POST   /accounts/{id}/domains
GET    /domains/{id}
PUT    /domains/{id}
DELETE /domains/{id}
GET    /domains/{id}/stats

# Hosting
GET    /domains/{id}/vhost
PUT    /domains/{id}/vhost
POST   /domains/{id}/deploy
GET    /domains/{id}/deploy/history
POST   /domains/{id}/deploy/{deployId}/rollback
GET    /domains/{id}/logs
POST   /domains/{id}/php/restart

# DNS
GET    /zones
POST   /zones
GET    /zones/{id}
DELETE /zones/{id}
GET    /zones/{id}/records
POST   /zones/{id}/records
PUT    /zones/{id}/records/{recordId}
DELETE /zones/{id}/records/{recordId}
POST   /zones/{id}/import
GET    /zones/{id}/export

# Mail
GET    /accounts/{id}/mailboxes
POST   /accounts/{id}/mailboxes
PUT    /mailboxes/{id}
DELETE /mailboxes/{id}
GET    /mailboxes/{id}/stats
GET    /mail/queue
POST   /mail/queue/flush

# SSL
GET    /accounts/{id}/certs
POST   /accounts/{id}/certs
DELETE /certs/{id}
POST   /certs/{id}/renew

# Databases
GET    /accounts/{id}/databases
POST   /accounts/{id}/databases
DELETE /databases/{id}
GET    /databases/{id}/stats
POST   /databases/{id}/export
POST   /databases/{id}/import

# Firewall
GET    /firewall/rules
POST   /firewall/rules
DELETE /firewall/rules/{id}
GET    /firewall/stats
GET    /firewall/blocked-ips

# Guardian / Metrics
GET    /metrics/system
GET    /metrics/accounts/{id}
WS     /metrics/live          # WebSocket: 1s real-time stream
GET    /guardian/alerts
GET    /guardian/heal-history
POST   /guardian/alerts/{id}/acknowledge
GET    /guardian/insights/{accountId}

# Backups
GET    /accounts/{id}/backups
POST   /accounts/{id}/backups
POST   /backups/{id}/restore
DELETE /backups/{id}

# Terminal
WS     /terminal/{accountId}  # WebSocket: full xterm.js terminal

# Reseller
GET    /resellers
POST   /resellers
GET    /resellers/{id}
PUT    /resellers/{id}
DELETE /resellers/{id}
GET    /resellers/{id}/theme
PUT    /resellers/{id}/theme

# Admin
GET    /admin/system
GET    /admin/license
POST   /admin/license
GET    /admin/audit-log
GET    /admin/updates
POST   /admin/updates/apply
```

---

## 17. Database Schema

### Core Tables

```sql
-- Users
CREATE TABLE users (
    id           VARCHAR(26) PRIMARY KEY,
    email        VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role         VARCHAR(50) NOT NULL DEFAULT 'account_owner',
    tenant_id    VARCHAR(26) NOT NULL,
    totp_secret  VARCHAR(64),
    totp_enabled BOOLEAN DEFAULT FALSE,
    failed_logins INT DEFAULT 0,
    locked_until TIMESTAMP,
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    status       VARCHAR(20) DEFAULT 'active',
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL,
    INDEX idx_users_email (email),
    INDEX idx_users_tenant (tenant_id)
);

-- Accounts (hosting accounts)
CREATE TABLE accounts (
    id           VARCHAR(26) PRIMARY KEY,
    username     VARCHAR(64) UNIQUE NOT NULL,
    reseller_id  VARCHAR(26),
    plan         VARCHAR(50) NOT NULL,
    disk_quota_mb BIGINT DEFAULT 10240,
    bandwidth_gb INT DEFAULT 100,
    status       VARCHAR(20) DEFAULT 'active',
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL
);

-- Domains
CREATE TABLE domains (
    id            VARCHAR(26) PRIMARY KEY,
    account_id    VARCHAR(26) NOT NULL,
    name          VARCHAR(255) UNIQUE NOT NULL,
    type          VARCHAR(20) DEFAULT 'main',
    document_root VARCHAR(512) NOT NULL,
    php_version   VARCHAR(10) DEFAULT '8.3',
    runtime       VARCHAR(20) DEFAULT 'php',
    ssl_enabled   BOOLEAN DEFAULT FALSE,
    ssl_cert_id   VARCHAR(26),
    status        VARCHAR(20) DEFAULT 'active',
    created_at    TIMESTAMP NOT NULL,
    updated_at    TIMESTAMP NOT NULL,
    INDEX idx_domains_account (account_id)
);

-- DNS Zones
CREATE TABLE dns_zones (
    id         VARCHAR(26) PRIMARY KEY,
    account_id VARCHAR(26) NOT NULL,
    domain     VARCHAR(255) UNIQUE NOT NULL,
    soa_email  VARCHAR(255),
    ttl        INT DEFAULT 3600,
    status     VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE dns_records (
    id       VARCHAR(26) PRIMARY KEY,
    zone_id  VARCHAR(26) NOT NULL,
    type     VARCHAR(10) NOT NULL,
    name     VARCHAR(255) NOT NULL,
    value    TEXT NOT NULL,
    priority INT DEFAULT 0,
    ttl      INT DEFAULT 3600,
    INDEX idx_records_zone (zone_id)
);

-- SSL Certificates
CREATE TABLE ssl_certs (
    id         VARCHAR(26) PRIMARY KEY,
    domain_id  VARCHAR(26) NOT NULL,
    domain     VARCHAR(255) NOT NULL,
    provider   VARCHAR(50) DEFAULT 'letsencrypt',
    cert_path  VARCHAR(512),
    key_path   VARCHAR(512),
    chain_path VARCHAR(512),
    issued_at  TIMESTAMP,
    expires_at TIMESTAMP,
    auto_renew BOOLEAN DEFAULT TRUE,
    status     VARCHAR(20) DEFAULT 'pending'
);

-- Mailboxes
CREATE TABLE mailboxes (
    id          VARCHAR(26) PRIMARY KEY,
    account_id  VARCHAR(26) NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    quota_mb    INT DEFAULT 1024,
    status      VARCHAR(20) DEFAULT 'active',
    created_at  TIMESTAMP NOT NULL,
    INDEX idx_mailboxes_account (account_id)
);

-- Audit Log
CREATE TABLE audit_log (
    id          VARCHAR(26) PRIMARY KEY,
    timestamp   TIMESTAMP NOT NULL,
    user_id     VARCHAR(26),
    user_email  VARCHAR(255),
    user_role   VARCHAR(50),
    actor_ip    VARCHAR(45),
    session_id  VARCHAR(64),
    action      VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(26),
    resource_name VARCHAR(255),
    result      VARCHAR(20) NOT NULL,
    duration_ms INT,
    prev_hash   VARCHAR(64),
    hash        VARCHAR(64),
    INDEX idx_audit_timestamp (timestamp),
    INDEX idx_audit_user (user_id),
    INDEX idx_audit_resource (resource_type, resource_id)
);

-- Firewall Rules
CREATE TABLE firewall_rules (
    id          VARCHAR(26) PRIMARY KEY,
    priority    SMALLINT NOT NULL,
    protocol    SMALLINT,
    src_ip      VARCHAR(45),
    src_cidr    VARCHAR(48),
    dst_port    INT,
    action      VARCHAR(20) NOT NULL,
    description VARCHAR(255),
    enabled     BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMP NOT NULL
);

-- Guardian Alerts
CREATE TABLE guardian_alerts (
    id           VARCHAR(26) PRIMARY KEY,
    rule_name    VARCHAR(100) NOT NULL,
    severity     VARCHAR(20) NOT NULL,
    metric       VARCHAR(100),
    value        DECIMAL(10,4),
    threshold    DECIMAL(10,4),
    message      TEXT NOT NULL,
    account_id   VARCHAR(26),
    acknowledged BOOLEAN DEFAULT FALSE,
    acked_by     VARCHAR(26),
    acked_at     TIMESTAMP,
    healed       BOOLEAN DEFAULT FALSE,
    heal_action  VARCHAR(100),
    fired_at     TIMESTAMP NOT NULL,
    resolved_at  TIMESTAMP
);

-- Resellers
CREATE TABLE resellers (
    id              VARCHAR(26) PRIMARY KEY,
    parent_id       VARCHAR(26),
    company_name    VARCHAR(255) NOT NULL,
    domain          VARCHAR(255) UNIQUE,
    logo_url        VARCHAR(512),
    primary_color   VARCHAR(7),
    secondary_color VARCHAR(7),
    custom_css      TEXT,
    brand_name      VARCHAR(100),
    support_email   VARCHAR(255),
    max_accounts    INT DEFAULT 100,
    max_disk_gb     INT DEFAULT 1000,
    status          VARCHAR(20) DEFAULT 'active',
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
```

---

## 18. Frontend Architecture

### Page Structure

```
src/
├── pages/
│   ├── auth/
│   │   ├── Login.tsx
│   │   ├── TwoFactor.tsx
│   │   └── PasswordReset.tsx
│   ├── dashboard/
│   │   └── Dashboard.tsx         # Main overview: metrics, alerts, quick actions
│   ├── domains/
│   │   ├── DomainList.tsx
│   │   ├── DomainCreate.tsx
│   │   └── DomainSettings.tsx
│   ├── hosting/
│   │   ├── PHPConfig.tsx
│   │   ├── GitDeploy.tsx
│   │   └── CronJobs.tsx
│   ├── dns/
│   │   ├── ZoneList.tsx
│   │   └── ZoneEditor.tsx        # Visual DNS record editor
│   ├── mail/
│   │   ├── MailboxList.tsx
│   │   ├── MailboxCreate.tsx
│   │   └── MailQueue.tsx
│   ├── ssl/
│   │   └── CertificateList.tsx
│   ├── databases/
│   │   ├── DatabaseList.tsx
│   │   └── SQLConsole.tsx        # Monaco-powered SQL editor
│   ├── files/
│   │   └── FileManager.tsx       # Drag-and-drop file manager
│   ├── terminal/
│   │   └── Terminal.tsx          # xterm.js WebSocket terminal
│   ├── firewall/
│   │   ├── RulesList.tsx
│   │   ├── RuleCreate.tsx
│   │   └── SecurityDashboard.tsx
│   ├── backups/
│   │   └── BackupList.tsx
│   ├── guardian/
│   │   ├── AlertsList.tsx
│   │   ├── MetricsDashboard.tsx  # Real-time charts
│   │   └── InsightsPanel.tsx     # LLM insights
│   ├── reseller/
│   │   ├── ResellerList.tsx
│   │   └── WhiteLabelSettings.tsx
│   ├── admin/
│   │   ├── AdminDashboard.tsx
│   │   ├── LicenseSettings.tsx
│   │   └── AuditLog.tsx
│   └── account/
│       ├── SecuritySettings.tsx
│       └── APIKeys.tsx
├── components/
│   ├── ui/                       # shadcn/ui components
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── TopBar.tsx
│   │   └── BreadcrumbNav.tsx
│   ├── metrics/
│   │   ├── CPUChart.tsx          # Recharts, WebSocket-fed
│   │   ├── MemoryChart.tsx
│   │   ├── NetworkChart.tsx
│   │   └── DiskUsage.tsx
│   ├── guardian/
│   │   ├── AlertBanner.tsx
│   │   └── SecurityScore.tsx
│   └── shared/
│       ├── DataTable.tsx         # TanStack Table
│       ├── ConfirmDialog.tsx
│       ├── StatusBadge.tsx
│       └── QuotaBar.tsx
├── stores/
│   ├── authStore.ts              # Zustand: user, tokens, tenant
│   ├── themeStore.ts             # Zustand: white-label theme
│   ├── metricsStore.ts           # Zustand: live metrics buffer
│   └── notificationStore.ts     # Zustand: real-time alerts
├── hooks/
│   ├── useWebSocket.ts           # Generic WS hook with reconnect
│   ├── useLiveMetrics.ts         # Metrics WebSocket + Zustand
│   ├── useTerminal.ts            # xterm.js + WebSocket
│   └── useTheme.ts               # White-label theme loader
└── lib/
    ├── api.ts                    # Axios instance + interceptors
    ├── auth.ts                   # Token refresh logic
    ├── constants.ts
    └── utils.ts
```

### Real-Time Metrics Hook

```typescript
// hooks/useLiveMetrics.ts
export function useLiveMetrics(accountId?: string) {
  const setMetrics = useMetricsStore(s => s.setMetrics)
  
  useEffect(() => {
    const ws = new WebSocket(`${WS_BASE}/metrics/live?account=${accountId ?? ''}`)
    
    ws.onmessage = (e) => {
      const snapshot: SystemSnapshot = JSON.parse(e.data)
      setMetrics(snapshot)
    }
    
    ws.onclose = () => {
      // Exponential backoff reconnect: 1s, 2s, 4s, 8s, max 30s
      setTimeout(() => reconnect(), backoff())
    }
    
    return () => ws.close()
  }, [accountId])
}
```

---

## 19. DevOps & Installer

### Build System

```makefile
# Makefile

VERSION  := $(shell git describe --tags --always)
LDFLAGS  := -ldflags="-s -w -X main.version=$(VERSION)"
TARGETS  := linux/amd64 linux/arm64

build:
	cd frontend && npm run build
	cp -r frontend/dist internal/embed/dist
	go build $(LDFLAGS) -o bin/orvixpanel ./cmd/orvixpanel

build-all:
	@for target in $(TARGETS); do \
		GOOS=$$(echo $$target | cut -d/ -f1) \
		GOARCH=$$(echo $$target | cut -d/ -f2) \
		go build $(LDFLAGS) -o bin/orvixpanel-$$(echo $$target | tr / -) ./cmd/orvixpanel; \
	done

release: build-all
	@for binary in bin/orvixpanel-*; do \
		sha256sum $$binary > $$binary.sha256; \
	done

test:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

docker-dev:
	docker-compose up -d  # Dev environment only
```

### Upgrade System

```go
// internal/updater/updater.go

func CheckForUpdates(currentVersion string) (*Release, error) {
    resp, err := http.Get("https://releases.orvixpanel.com/latest.json")
    // Returns: version, download_url, sha256, changelog, critical (bool)
}

func ApplyUpdate(release *Release) error {
    // 1. Download new binary to /tmp/orvixpanel-new
    // 2. Verify SHA-256
    // 3. Run DB migrations (dry-run first)
    // 4. Atomic swap: /opt/orvixpanel/orvixpanel ← new binary
    // 5. systemctl restart orvixpanel
    // Zero-downtime: Fiber graceful shutdown (30s drain)
}
```

---

## 20. Testing Strategy

### Unit Tests

```
Coverage target: 80% across all packages

Priority packages:
- internal/auth/        — 95% (security critical)
- internal/license/     — 95% (business critical)
- internal/firewall/    — 90% (security critical)
- internal/hosting/     — 85% (config generation)
- internal/dns/         — 85%
- internal/guardian/    — 80%
```

### Integration Tests

```go
// tests/integration/

// Spin up real Fiber server + SQLite + Redis (testcontainers)
// Test complete request flows:
// - Full auth flow: register → login → 2FA → refresh → logout
// - Domain lifecycle: create → deploy → SSL → delete
// - Backup/restore cycle
// - Firewall rule CRUD + eBPF mock
// - Reseller white-label isolation
```

### Load Testing

```yaml
# k6 load test — target performance SLAs

scenarios:
  dashboard_load:
    exec: dashboard
    vus: 100
    duration: 5m
  api_stress:
    exec: api
    vus: 500
    duration: 10m

thresholds:
  http_req_duration: ["p99<500"]
  http_req_failed: ["rate<0.01"]
```

### Security Testing

```
OWASP ZAP — automated scan on staging
Nuclei — CVE template scanning
SQLMap — injection testing (all endpoints)
Gobuster — endpoint enumeration
Manual pentest — auth, RBAC, tenant isolation
```

---

## Appendix A — Milestones & Acceptance Criteria

| Phase | Week | Done When |
|---|---|---|
| Foundation | 2 | Login works, JWT issues, RBAC enforced, license validates |
| Core Hosting | 4 | Domain added, nginx config generated, PHP site live |
| DNS + Mail + SSL | 6 | DNS zone resolves, mailbox works, SSL auto-issued |
| Security | 8 | WAF blocking attacks, firewall rules applying, 2FA working |
| DB + Files + Backup | 10 | DB created, files browsable, backup completes and restores |
| Guardian | 12 | Alert fires, auto-heal restarts service, metrics stream live |
| Reseller | 14 | White-label panel loads at custom domain, WHMCS provisions |
| v1 Release | 16 | Pentest passed, installer works on clean Ubuntu, docs live |

---

## Appendix B — Environment Variables

```bash
ORVIX_CONFIG=/etc/orvixpanel/orvixpanel.toml
ORVIX_SECRET_KEY=<64-random-bytes>
ORVIX_LICENSE_KEY=ORVIX-XXXX-XXXX-XXXXX-XXXXX
ORVIX_DB_DSN=/var/lib/orvixpanel/data.db
ORVIX_REDIS_ADDR=localhost:6379
ORVIX_DEBUG=false
ORVIX_LOG_LEVEL=info   # debug, info, warn, error
```

---

*OrvixPanel MVP v1.0 — Document Version 1.0*  
*Classification: Internal — Build Team*  
*Last updated: June 2026*
