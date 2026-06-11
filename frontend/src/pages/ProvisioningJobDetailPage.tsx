/**
 * Provisioning Job Detail - Shows timeline of provisioning steps.
 */

import { useParams, useNavigate } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { StatusPill } from "@/lib/ui/StatusPill";
import { getProvisioningJob, listProvisioningEvents, STATUS_CONFIG } from "@/lib/api/provisioning";
import { provisioningKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";

export function ProvisioningJobDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();

  const jobQuery = useQuery({
    queryKey: provisioningKeys.detail(id),
    queryFn: () => getProvisioningJob(id),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "pending" || status === "running") {
        return 2000;
      }
      return false;
    },
  });

  const eventsQuery = useQuery({
    queryKey: provisioningKeys.events(id),
    queryFn: () => listProvisioningEvents(id),
    enabled: !!jobQuery.data,
  });

  const job = jobQuery.data;
  const events = eventsQuery.data || job?.steps || [];
  const statusConfig = job ? STATUS_CONFIG[job.status] : null;

  if (jobQuery.isLoading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <Spinner size={28} />
      </div>
    );
  }

  if (jobQuery.isError || !job) {
    return (
      <ErrorState
        description="Failed to load provisioning job"
        onRetry={() => jobQuery.refetch()}
      />
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={
          <span className="flex items-center gap-3">
            <span className="font-mono">{job.domain}</span>
            {statusConfig && (
              <StatusPill tone={statusConfig.tone}>
                {statusConfig.label}
              </StatusPill>
            )}
          </span>
        }
        description={`Job ${job.id} · Created ${formatDate(job.created_at)}`}
        actions={
          <Button variant="secondary" onClick={() => navigate({ to: "/hosting/provisioning/jobs" })}>
            Back to Jobs
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Timeline */}
          <Card>
            <CardHeader
              title="Provisioning Timeline"
              description="Step-by-step progress of website creation"
            />
            <div className="p-6">
              {events.length === 0 ? (
                <div className="flex flex-col items-center py-8 text-center">
                  <Spinner size={20} />
                  <p className="mt-3 text-sm text-ink-3">Waiting for steps...</p>
                </div>
              ) : (
                <div className="relative">
                  {/* Timeline line */}
                  <div className="absolute left-4 top-0 h-full w-px bg-surface-border" />

                  <div className="space-y-6">
                    {events.map((event, index) => {
                      const stepStatus = event.status;

                      return (
                        <div key={index} className="relative flex gap-4">
                          {/* Status dot */}
                          <div
                            className={`relative z-10 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full ${
                              stepStatus === "completed"
                                ? "bg-success text-white"
                                : stepStatus === "running"
                                ? "bg-brand-500 text-white"
                                : stepStatus === "failed"
                                ? "bg-danger text-white"
                                : stepStatus === "skipped"
                                ? "bg-ink-3 text-white"
                                : "bg-surface-2 text-ink-3"
                            }`}
                          >
                            {stepStatus === "completed" ? (
                              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                              </svg>
                            ) : stepStatus === "running" ? (
                              <Spinner size={14} />
                            ) : stepStatus === "failed" ? (
                              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            ) : (
                              <span className="text-xs">{index + 1}</span>
                            )}
                          </div>

                          {/* Content */}
                          <div className="flex-1 pb-6">
                            <div className="flex items-center justify-between">
                              <span className="font-medium text-ink-1">{event.step}</span>
                              <span className="text-xs text-ink-3">
                                {event.timestamp ? formatDate(event.timestamp) : ""}
                              </span>
                            </div>
                            <p className="mt-1 text-sm text-ink-2">{event.message}</p>
                            {event.error && (
                              <div className="mt-2 rounded-md border border-danger/30 bg-danger/5 p-3">
                                <div className="text-xs font-medium text-danger">Error</div>
                                <div className="mt-1 font-mono text-xs text-danger">{event.error}</div>
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Failure error message */}
              {job.status === "failed" && job.error_message && (
                <div className="mt-6 rounded-md border border-danger/30 bg-danger/5 p-4">
                  <div className="flex items-center gap-2 text-danger">
                    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <span className="font-medium">Provisioning Failed</span>
                  </div>
                  <p className="mt-2 text-sm text-ink-2">{job.error_message}</p>
                  <p className="mt-3 text-xs text-ink-3">
                    The system has automatically rolled back any changes made during the failed provisioning attempt.
                  </p>
                </div>
              )}

              {/* Success message */}
              {job.status === "completed" && (
                <div className="mt-6 rounded-md border border-success/30 bg-success/5 p-4">
                  <div className="flex items-center gap-2 text-success">
                    <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span className="font-medium">Provisioning Complete</span>
                  </div>
                  <p className="mt-2 text-sm text-ink-2">
                    Website {job.domain} has been successfully provisioned and is ready to use.
                  </p>
                </div>
              )}
            </div>
          </Card>

          {/* Log details */}
          {job.status === "failed" && job.error_message && (
            <Card>
              <CardHeader
                title="Error Details"
                description="Technical details for debugging"
              />
              <div className="p-6">
                <pre className="overflow-x-auto rounded-md bg-surface-3 p-4 font-mono text-xs text-ink-2">
                  {job.error_message}
                </pre>
              </div>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Job Info */}
          <Card>
            <CardHeader title="Job Details" />
            <div className="space-y-4 p-6 text-sm">
              <div className="flex justify-between">
                <span className="text-ink-3">Job ID</span>
                <span className="font-mono text-xs">{job.id}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Domain</span>
                <span className="font-mono">{job.domain}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Account</span>
                <span className="font-mono">{job.account_id}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Status</span>
                {statusConfig && (
                  <StatusPill tone={statusConfig.tone}>
                    {statusConfig.label}
                  </StatusPill>
                )}
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Created</span>
                <span className="font-mono text-xs">{formatDate(job.created_at)}</span>
              </div>
              {job.completed_at && (
                <div className="flex justify-between">
                  <span className="text-ink-3">Completed</span>
                  <span className="font-mono text-xs">{formatDate(job.completed_at)}</span>
                </div>
              )}
            </div>
          </Card>

          {/* Configuration */}
          <Card>
            <CardHeader title="Configuration" />
            <div className="space-y-4 p-6 text-sm">
              <div className="flex justify-between">
                <span className="text-ink-3">PHP Version</span>
                <span className="font-mono">{job.php_version}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Disk Quota</span>
                <span className="font-mono">
                  {job.disk_quota_mb >= 1024
                    ? `${(job.disk_quota_mb / 1024).toFixed(1)} GB`
                    : `${job.disk_quota_mb} MB`}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Access Logs</span>
                <span>{job.enable_logs ? "Enabled" : "Disabled"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-3">Default Index</span>
                <span>{job.create_default_index ? "Created" : "Not created"}</span>
              </div>
            </div>
          </Card>

          {/* Actions */}
          {(job.status === "completed" || job.status === "failed") && (
            <Card>
              <CardHeader title="Actions" />
              <div className="space-y-3 p-6">
                {job.status === "completed" && (
                  <Button
                    variant="secondary"
                    className="w-full"
                    onClick={() => navigate({ to: `/domains?search=${job.domain}` })}
                  >
                    View in Domains
                  </Button>
                )}
                {job.status === "failed" && (
                  <Button
                    variant="secondary"
                    className="w-full"
                    onClick={() => navigate({ to: "/hosting/create" })}
                  >
                    Try Again
                  </Button>
                )}
                <Button
                  variant="ghost"
                  className="w-full"
                  onClick={() => navigate({ to: "/hosting/provisioning/jobs" })}
                >
                  View All Jobs
                </Button>
              </div>
            </Card>
          )}

          {/* Auto-refresh status */}
          {(job.status === "pending" || job.status === "running") && (
            <Card>
              <div className="flex items-center gap-3 p-4">
                <Spinner size={16} />
                <div className="text-sm">
                  <div className="font-medium text-ink-1">Provisioning in progress</div>
                  <div className="text-xs text-ink-3">Auto-refreshing...</div>
                </div>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}