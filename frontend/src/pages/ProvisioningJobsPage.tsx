/**
 * Provisioning Jobs - List all website provisioning jobs.
 */

import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { StatusPill } from "@/lib/ui/StatusPill";
import { listProvisioningJobs, STATUS_CONFIG, type JobStatus } from "@/lib/api/provisioning";
import { provisioningKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";

const STATUS_FILTERS: { value: JobStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "rolled_back", label: "Rolled Back" },
];

export function ProvisioningJobsPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<JobStatus | "all">("all");

  const jobsQuery = useQuery({
    queryKey: [...provisioningKeys.list(), search, statusFilter],
    queryFn: () => listProvisioningJobs({
      search: search || undefined,
      status: statusFilter !== "all" ? statusFilter : undefined,
    }),
    refetchInterval: (query) => {
      // Auto-refresh running jobs every 3 seconds
      const data = query.state.data;
      if (data?.jobs?.some(j => j.status === "running" || j.status === "pending")) {
        return 3000;
      }
      return false;
    },
  });

  const jobs = jobsQuery.data?.jobs || [];
  const total = jobsQuery.data?.total || 0;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Provisioning Jobs"
        description="Track website provisioning progress"
        actions={
          <Link to="/hosting/create">
            <Button variant="primary">
              <svg className="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              New Website
            </Button>
          </Link>
        }
      />

      {/* Filters */}
      <Card>
        <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center">
          <div className="flex-1">
            <Input
              placeholder="Search by domain or account..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full"
            />
          </div>
          <div className="flex gap-2 overflow-x-auto">
            {STATUS_FILTERS.map(filter => (
              <button
                key={filter.value}
                onClick={() => setStatusFilter(filter.value)}
                className={`whitespace-nowrap rounded-md px-3 py-1.5 text-sm transition-colors ${
                  statusFilter === filter.value
                    ? "bg-brand-500 text-white"
                    : "bg-surface-2 text-ink-2 hover:bg-surface-3"
                }`}
              >
                {filter.label}
              </button>
            ))}
          </div>
        </div>
      </Card>

      {/* Jobs List */}
      <Card>
        <CardHeader
          title="Jobs"
          description={`${total} job${total !== 1 ? "s" : ""} found`}
        />
        {jobsQuery.isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Spinner size={24} />
          </div>
        ) : jobsQuery.isError ? (
          <div className="p-6">
            <ErrorState
              description="Failed to load provisioning jobs"
              onRetry={() => jobsQuery.refetch()}
            />
          </div>
        ) : jobs.length === 0 ? (
          <div className="flex flex-col items-center py-16 text-center">
            <svg className="h-12 w-12 text-ink-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <h3 className="mt-4 text-sm font-medium text-ink-1">No jobs found</h3>
            <p className="mt-1 text-sm text-ink-3">
              {search || statusFilter !== "all"
                ? "Try adjusting your filters"
                : "Create a new website to start provisioning"}
            </p>
            {!search && statusFilter === "all" && (
              <Link to="/hosting/create" className="mt-4">
                <Button variant="primary">Create Website</Button>
              </Link>
            )}
          </div>
        ) : (
          <div className="divide-y divide-surface-border">
            {jobs.map((job) => {
              const statusConfig = STATUS_CONFIG[job.status];
              return (
                <Link
                  key={job.id}
                  to="/hosting/provisioning/jobs/$id"
                  params={{ id: job.id }}
                  className="flex items-center justify-between p-4 transition-colors hover:bg-surface-2"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <span className="truncate font-mono text-sm font-medium text-ink-1">
                        {job.domain}
                      </span>
                      <StatusPill tone={statusConfig.tone}>
                        {statusConfig.label}
                      </StatusPill>
                      {job.status === "failed" && job.error_message && (
                        <span className="truncate text-xs text-danger">
                          {job.error_message}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 flex items-center gap-4 text-xs text-ink-3">
                      <span>Account: {job.account_id}</span>
                      <span>PHP {job.php_version}</span>
                      <span>Quota: {job.disk_quota_mb >= 1024
                        ? `${(job.disk_quota_mb / 1024).toFixed(0)} GB`
                        : `${job.disk_quota_mb} MB`}</span>
                      <span>Created {formatDate(job.created_at)}</span>
                      {job.completed_at && (
                        <span>Completed {formatDate(job.completed_at)}</span>
                      )}
                    </div>
                  </div>
                  <svg className="h-5 w-5 flex-shrink-0 text-ink-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </Link>
              );
            })}
          </div>
        )}
      </Card>

      {/* Auto-refresh indicator */}
      {jobsQuery.isFetching && jobs.some(j => j.status === "running") && (
        <div className="fixed bottom-4 right-4 flex items-center gap-2 rounded-full bg-surface-1 px-4 py-2 shadow-lg">
          <Spinner size={14} />
          <span className="text-sm text-ink-2">Refreshing...</span>
        </div>
      )}
    </div>
  );
}