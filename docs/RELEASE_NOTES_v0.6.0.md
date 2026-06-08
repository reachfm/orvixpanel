# OrvixPanel v0.6.0 - Backup & Restore Engine

**Release Date:** 2026-06-08
**Status:** Stable

## Overview

v0.6.0 introduces the **Backup & Restore Engine** - a comprehensive system for automated backups with staging-based restore and rollback safety. This release provides tenant-isolated backup operations with checksum verification and multiple storage backend support.

## Features

### Core Backup System

- **File-based Backups**: tar.gz archive creation with SHA256 checksums
- **Multiple Backup Types**: Full, Files-only, Database (abstraction ready)
- **Backup Integrity Verification**: Checksum validation on every backup
- **Retention Policy**: Configurable retention days with automatic cleanup

### Storage Provider Abstraction

- **Local Storage**: Primary storage with filesystem-based operations
- **S3 Compatible**: AWS S3, MinIO, Wasabi provider stubs ready for implementation
- **Extensible Architecture**: Easy to add new storage backends

### Safe Restore Operations

- **Staging Directory**: Extract backups to staging before restoration
- **Automatic Rollback**: If restore fails, automatically restore original files
- **Path Traversal Protection**: Security checks prevent malicious archive extraction

### Backup Scheduling

- **Cron-based Scheduling**: Configure backup schedules with cron expressions
- **Automatic Execution**: Scheduler runs backups at configured intervals
- **Retention Cleanup**: Automatic removal of expired backups

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/backups` | List all backups |
| GET | `/api/v1/backups/:id` | Get backup details |
| POST | `/api/v1/backups` | Create new backup |
| DELETE | `/api/v1/backups/:id` | Delete backup |
| POST | `/api/v1/backups/:id/restore` | Restore from backup |
| GET | `/api/v1/backups/:id/restores` | Get restore history |
| GET | `/api/v1/backups/stats` | Get backup statistics |
| POST | `/api/v1/backups/schedules` | Create backup schedule |
| GET | `/api/v1/backups/schedules` | List backup schedules |
| DELETE | `/api/v1/backups/schedules/:id` | Delete schedule |

### Frontend Dashboard

- **Backup List View**: Paginated list with filters
- **Backup Detail View**: File list, restore history, checksum verification
- **Create Backup Modal**: Select type, configure retention
- **Restore UI**: Target directory selection with rollback toggle
- **Statistics Cards**: Active backups, storage used, failed backups

### Audit Events

All backup operations are logged with audit entries:
- `backup_created`, `backup_started`, `backup_completed`, `backup_failed`, `backup_deleted`
- `restore_started`, `restore_completed`, `restore_failed`, `restore_rolled_back`
- `schedule_created`, `schedule_deleted`

## Files Changed

### Backend (Go)

| File | Description |
|------|-------------|
| `internal/db/models/backup.go` | Database models: BackupJob, BackupFile, RestorePoint, BackupSchedule |
| `internal/backup/errors.go` | Custom error types |
| `internal/backup/storage.go` | Storage provider interface and implementations |
| `internal/backup/manager.go` | Backup creation, verification, extraction |
| `internal/backup/restore.go` | Restore with staging and rollback |
| `internal/backup/scheduler.go` | Cron-based backup scheduler |
| `internal/backup/handlers.go` | Echo API handlers |
| `internal/backup/audit.go` | Audit event types and helpers |
| `internal/backup/backup_test.go` | Unit tests (14 tests passing) |

### Frontend (React/TypeScript)

| File | Description |
|------|-------------|
| `src/lib/api/backup.ts` | TypeScript API client |
| `src/pages/BackupsList.tsx` | Backup list page with filters |
| `src/pages/BackupDetail.tsx` | Backup detail with restore UI |
| `src/router.tsx` | Added /backup and /backup/$id routes |

## Configuration

### Storage Directories (Default)

```yaml
backup:
  storage_dir: "/var/lib/orvixpanel/backups"
  temp_dir: "/tmp/orvixpanel-backup"
  checksum_algo: "sha256"
  max_file_size: 10737418240  # 10GB
  exclude_patterns:
    - "*.tmp"
    - "*.log"
    - "*.swp"
    - "*.bak"
    - ".git/*"
    - "node_modules/*"
    - ".env"
```

### Retention Policy

- Default retention: 30 days
- Configurable per backup job
- Automatic cleanup of expired backups

## Verification Results

### Build Verification
- **Go Build**: ✅ `/workspace/orvixpanel` (22MB ELF 64-bit executable)
- **Frontend Build**: ✅ Built successfully (414KB JS, 25KB CSS)

### Unit Test Results
- **Go Unit Tests**: ✅ 14/14 tests passing
  ```
  === RUN   TestStorageLocal → PASS
  === RUN   TestStorageFactory → PASS
  === RUN   TestDefaultConfig → PASS
  === RUN   TestBackupManagerCreateFileBackup → PASS
  === RUN   TestBackupManagerVerifyBackup → PASS
  === RUN   TestRestoreManagerStaging → PASS
  === RUN   TestBackupError → PASS
  === RUN   TestAuditEntry → PASS
  === RUN   TestGenerateID → PASS
  === RUN   TestBackupJobModel → PASS
  === RUN   TestBackupFileModel → PASS
  === RUN   TestRestorePointModel → PASS
  === RUN   TestBackupScheduleModel → PASS
  === RUN   TestBackupConfigModel → PASS
  ```
- **Frontend Tests**: ✅ 25/25 tests passing (3 test files)

### Type Checking
- **Go vet**: ✅ No issues
- **TypeScript Typecheck**: ✅ No TypeScript errors

### Smoke Test Script
Located at `/workspace/scripts/smoke-backup-local.sh` for operational verification:
- Creates real tar.gz archives with SHA256 checksums
- Tests staging directory and rollback mechanism
- Verifies tenant isolation in handler queries
- Tests scheduler and retention policy

### Tag: v0.6.0-backup-preview

## Rules Enforced

1. **No Fake Backups**: Every backup creates actual tar.gz archives
2. **Checksum Every Backup**: SHA256 checksum on all backup files
3. **Never Overwrite Live Data**: Staging + rollback before any restoration
4. **Tenant Isolation**: All operations scoped to tenant_id

## Breaking Changes

None - this is a pure feature addition.

## Deprecations

None.

## Migration Guide

No migration needed - fresh feature installation only.

## Known Limitations

- S3/MinIO/Wasabi storage providers are stubs (ready for implementation)
- Database backup abstraction interface ready but not fully implemented
- Cron parsing uses simple daily calculation (full cron support planned)

## Roadmap for v0.7.0

- Complete S3/MinIO/Wasabi storage implementations
- Full cron expression parsing for schedules
- Incremental/differential backup support
- Backup encryption at rest
- Backup verification automation