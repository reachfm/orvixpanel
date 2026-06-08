/**
 * Update Center page - cPanel-style autonomous update manager.
 * Displays current version, checks for updates, manages scheduled updates, and shows history.
 * All data comes from real API endpoints.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Button } from "@/lib/ui/Button";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { useNotification } from "@/lib/ui/Notification";
import { formatDate } from "@/lib/utils";
import {
  updateStatus,
  checkForUpdates,
  installUpdate,
  getUpdateHistory,
  enableScheduler,
  disableScheduler,
  systemHealth,
  type UpdateHistoryEntry,
  type UpdateCheckResult,
  type PreflightCheck,
} from "@/lib/api/system";
import { updateKeys } from "@/lib/query/keys";

export function UpdateCenterPage() {
  const qc = useQueryClient();
  const notify = useNotification();
  const [checking, setChecking] = useState(false);

  // Queries
  const status = useQuery({ queryKey: updateKeys.status(), queryFn: updateStatus, refetchInterval: 30_000 });
  const history = useQuery({ queryKey: updateKeys.history(), queryFn: getUpdateHistory });
  const health = useQuery({ queryKey: updateKeys.health(), queryFn: systemHealth, refetchInterval: 60_000 });

  // Mutations
  const checkMutation = useMutation({
    mutationFn: (channel: string) => checkForUpdates(channel),
    onSuccess: (data: UpdateCheckResult) => {
      setChecking(false);
      if (data.update_available) {
        notify("info", "Update Available", `Version ${data.latest_version} is available. You are on ${data.current_version}.`);
      } else {
        notify("success", "Up to Date", `You are running the latest version: ${data.current_version}`);
      }
      qc.invalidateQueries({ queryKey: updateKeys.all() });
    },
    onError: (err: Error) => {
      setChecking(false);
      notify("error", "Check Failed", err.message);
    },
  });

  const installMutation = useMutation({
    mutationFn: (channel: string) => installUpdate(channel),
    onSuccess: (data) => {
      notify("info", "Update Started", data.message);
      qc.invalidateQueries({ queryKey: updateKeys.all() });
    },
    onError: (err: Error) => {
      notify("error", "Install Failed", err.message);
    },
  });

  const enableSchedulerMutation = useMutation({
    mutationFn: enableScheduler,
    onSuccess: (data) => {
      notify("success", "Scheduler Enabled", data.message);
      qc.invalidateQueries({ queryKey: updateKeys.status() });
    },
    onError: (err: Error) => {
      notify("error", "Scheduler Error", err.message);
    },
  });

  const disableSchedulerMutation = useMutation({
    mutationFn: disableScheduler,
    onSuccess: (data) => {
      notify("success", "Scheduler Disabled", data.message);
      qc.invalidateQueries({ queryKey: updateKeys.status() });
    },
    onError: (err: Error) => {
      notify("error", "Scheduler Error", err.message);
    },
  });

  const handleCheck = (channel = "stable") => {
    setChecking(true);
    checkMutation.mutate(channel);
  };

  const handleInstall = (channel = "stable") => {
    installMutation.mutate(channel);
  };

  const statusTone = (check: PreflightCheck): "success" | "warning" | "danger" | "neutral" => {
    switch (check.status) {
      case "pass": return "success";
      case "warn": return "warning";
      case "fail": return "danger";
      default: return "neutral";
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Update Center"
        description="Autonomous update management. Check for updates, install new versions, and manage automatic updates."
      />

      {/* Version Info Card */}
      <Card>
        <CardHeader
          title="Current Installation"
          description="Running version and channel information"
          actions={
            <StatusPill tone={status.data ? "success" : "neutral"}>
              {status.data ? "Online" : status.isLoading ? "Loading" : "Offline"}
            </StatusPill>
          }
        />
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          <Field label="Version" value={status.data?.current_version ?? "—"} mono />
          <Field label="Commit" value={status.data?.current_commit ? status.data.current_commit.slice(0, 8) : "—"} mono />
          <Field label="Channel" value={status.data?.channel ? capitalize(status.data.channel) : "—"} />
          <Field label="Build Date" value={status.data?.build_date ? formatDate(status.data.build_date) : "—"} />
        </div>
        <div className="mt-4 grid grid-cols-2 gap-4 md:grid-cols-4">
          <Field label="Health Endpoint" value={status.data?.health_endpoint ?? "—"} mono small />
          <Field label="Ready Endpoint" value={status.data?.ready_endpoint ?? "—"} mono small />
          <Field label="Update Check" value={status.data?.update_check_enabled ? "Enabled" : "Disabled"} />
          <Field label="Auto Update" value={status.data?.auto_update_enabled ? "Enabled" : "Disabled"} />
        </div>
      </Card>

      {/* Action Buttons */}
      <Card>
        <CardHeader
          title="Update Actions"
          description="Check for updates or install the latest version"
        />
        <div className="flex flex-wrap gap-3">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleCheck("stable")}
            disabled={checking || checkMutation.isPending}
            leftIcon={checking || checkMutation.isPending ? <Spinner size={14} /> : <IconRefresh />}
          >
            Check for Updates
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={() => handleInstall("stable")}
            disabled={installMutation.isPending}
            leftIcon={installMutation.isPending ? <Spinner size={14} /> : <IconDownload />}
          >
            Install Update
          </Button>
          <div className="ml-auto flex items-center gap-2">
            <span className="text-sm text-ink-2">Auto-update scheduler:</span>
            {status.data?.auto_update_enabled ? (
              <Button
                variant="danger"
                size="sm"
                onClick={() => disableSchedulerMutation.mutate()}
                disabled={disableSchedulerMutation.isPending}
              >
                Disable
              </Button>
            ) : (
              <Button
                variant="primary"
                size="sm"
                onClick={() => enableSchedulerMutation.mutate()}
                disabled={enableSchedulerMutation.isPending}
              >
                Enable
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* System Health / Preflight Checks */}
      {health.data && (
        <Card>
          <CardHeader
            title="System Health Checks"
            description="Pre-flight checks before update installation"
            actions={
              <span className="text-xs text-ink-3">
                {health.data.checks.filter(c => c.status === "pass").length} / {health.data.checks.length} passing
              </span>
            }
          />
          <div className="space-y-2">
            {health.data.checks.map((check: PreflightCheck) => (
              <div key={check.name} className="flex items-start gap-3 rounded-md bg-surface-2 p-3">
                <StatusPill tone={statusTone(check)} dot>
                  {check.status.toUpperCase()}
                </StatusPill>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm text-ink-1">{check.name}</span>
                  </div>
                  <p className="text-xs text-ink-2 mt-0.5">{check.message}</p>
                  {check.suggestions && check.suggestions.length > 0 && (
                    <ul className="mt-1 space-y-0.5">
                      {check.suggestions.map((s, i) => (
                        <li key={i} className="text-xs text-ink-3 flex items-start gap-1">
                          <span className="text-warning-500">•</span>
                          {s}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Update History */}
      <Card>
        <CardHeader
          title="Update History"
          description="Recent update operations and their results"
          actions={
            <Button
              variant="ghost"
              size="sm"
              onClick={() => history.refetch()}
              disabled={history.isFetching}
            >
              Refresh
            </Button>
          }
        />
        {history.isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : history.isError ? (
          <ErrorState title="Failed to load history" onRetry={() => history.refetch()} />
        ) : history.data?.history && history.data.history.length > 0 ? (
          <div className="space-y-2">
            {history.data.history.map((entry: UpdateHistoryEntry) => (
              <HistoryEntry key={entry.id} entry={entry} />
            ))}
          </div>
        ) : (
          <div className="py-8 text-center text-sm text-ink-3">
            No update history yet
          </div>
        )}
      </Card>
    </div>
  );
}

function HistoryEntry({ entry }: { entry: UpdateHistoryEntry }) {
  const resultTone = (result: string): "success" | "danger" | "warning" | "neutral" => {
    switch (result) {
      case "success": return "success";
      case "failed": return "danger";
      case "rolled_back": return "warning";
      case "in_progress": return "neutral";
      default: return "neutral";
    }
  };

  return (
    <div className="flex items-start gap-4 rounded-md border border-surface-border bg-surface-1 p-4">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <StatusPill tone={resultTone(entry.result)} dot>
            {entry.result.replace("_", " ")}
          </StatusPill>
          <span className="text-xs text-ink-3">{capitalize(entry.channel)} channel</span>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <span className="text-ink-2">
            <span className="text-ink-3">from</span>{" "}
            <span className="font-mono text-ink-1">{entry.from_version.tag || "—"}</span>
          </span>
          <span className="text-ink-3">→</span>
          <span className="text-ink-2">
            <span className="text-ink-3">to</span>{" "}
            <span className="font-mono text-ink-1">{entry.to_version.tag || "—"}</span>
          </span>
        </div>
        {entry.error_message && (
          <p className="mt-2 text-xs text-danger-500">{entry.error_message}</p>
        )}
      </div>
      <div className="text-right shrink-0">
        <div className="text-xs text-ink-3">{formatDate(entry.timestamp)}</div>
        <div className="text-xs text-ink-3 mt-0.5">{entry.duration_seconds}s</div>
      </div>
    </div>
  );
}

function Field({ label, value, mono, small }: { label: string; value: string; mono?: boolean; small?: boolean }) {
  return (
    <div>
      <dt className={`${small ? "text-[10px]" : "text-[11px]"} uppercase tracking-wider text-ink-3`}>{label}</dt>
      <dd className={`${small ? "text-xs" : "text-sm"} ${mono ? "font-mono" : ""} text-ink-1 mt-0.5 truncate`}>{value}</dd>
    </div>
  );
}

function capitalize(s: string): string {
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
}

function IconRefresh() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
      <path d="M3 21v-5h5" />
    </svg>
  );
}

function IconDownload() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <polyline points="7 10 12 15 17 10" />
      <line x1="12" y1="15" x2="12" y2="3" />
    </svg>
  );
}