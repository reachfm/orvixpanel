/**
 * Mail Audit Log Page
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listAuditLogs, getAuditStats } from "@/lib/api/mail";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Spinner } from "@/lib/ui/Feedback";
import { formatDate } from "@/lib/utils";

const ACTION_COLORS: Record<string, string> = {
  sent: "bg-blue-100 text-blue-800",
  received: "bg-green-100 text-green-800",
  login: "bg-purple-100 text-purple-800",
  failed_login: "bg-red-100 text-red-800",
  bounced: "bg-orange-100 text-orange-800",
  rejected: "bg-red-100 text-red-800",
  deferred: "bg-yellow-100 text-yellow-800",
};

const DIRECTION_COLORS: Record<string, string> = {
  inbound: "bg-blue-500",
  outbound: "bg-green-500",
};

export function MailAuditLogPage() {
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState("");
  const [directionFilter, setDirectionFilter] = useState("");

  // Query audit logs
  const { data, isLoading, error } = useQuery({
    queryKey: ["mail", "audit", page, actionFilter, directionFilter],
    queryFn: () => listAuditLogs({
      page,
      page_size: 20,
      action: actionFilter || undefined,
      direction: directionFilter || undefined,
    }),
  });

  // Query stats
  const { data: stats } = useQuery({
    queryKey: ["mail", "audit", "stats"],
    queryFn: () => getAuditStats(),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center text-red-500">
        Failed to load audit logs. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Mail Audit Log</h1>
        <p className="text-gray-500">
          Track all mail operations and security events
        </p>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <div className="text-sm text-gray-500">Total Events</div>
            <div className="text-2xl font-bold">{stats.total_events}</div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Sent</div>
            <div className="text-2xl font-bold">
              {stats.actions?.find((a) => a.action === "sent")?.count || 0}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Received</div>
            <div className="text-2xl font-bold">
              {stats.actions?.find((a) => a.action === "received")?.count || 0}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Failed Logins</div>
            <div className="text-2xl font-bold text-red-600">
              {stats.actions?.find((a) => a.action === "failed_login")?.count || 0}
            </div>
          </Card>
        </div>
      )}

      {/* Filters */}
      <Card>
        <div className="flex flex-wrap gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Action Type
            </label>
            <select
              className="rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              value={actionFilter}
              onChange={(e) => {
                setActionFilter(e.target.value);
                setPage(1);
              }}
            >
              <option value="">All</option>
              <option value="sent">Sent</option>
              <option value="received">Received</option>
              <option value="login">Login</option>
              <option value="failed_login">Failed Login</option>
              <option value="bounced">Bounced</option>
              <option value="rejected">Rejected</option>
              <option value="deferred">Deferred</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Direction
            </label>
            <select
              className="rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              value={directionFilter}
              onChange={(e) => {
                setDirectionFilter(e.target.value);
                setPage(1);
              }}
            >
              <option value="">All</option>
              <option value="inbound">Inbound</option>
              <option value="outbound">Outbound</option>
            </select>
          </div>
        </div>
      </Card>

      {/* Audit Log Table */}
      <Card>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Time
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Action
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Direction
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  From
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  To
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Subject
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Remote IP
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data?.logs.map((log) => (
                <tr key={log.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {formatDate(log.created_at)}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span
                      className={`px-2 py-1 text-xs rounded-full ${
                        ACTION_COLORS[log.action] || "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {log.action}
                    </span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {log.direction && (
                      <div className="flex items-center gap-1">
                        <div className={`w-2 h-2 rounded-full ${DIRECTION_COLORS[log.direction]}`} />
                        <span className="text-xs text-gray-500">{log.direction}</span>
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900 max-w-[150px] truncate">
                    {log.from_email || "-"}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900 max-w-[150px] truncate">
                    {log.to_email || "-"}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500 max-w-[200px] truncate">
                    {log.subject || "-"}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <Badge
                      tone={log.status === "success" || log.status === "delivered" ? "success" : "neutral"}
                      
                    >
                      {log.status}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {log.remote_ip || "-"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {data && data.total > 20 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200">
            <div className="text-sm text-gray-500">
              Showing {(page - 1) * 20 + 1} to {Math.min(page * 20, data.total)} of {data.total}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 rounded border border-gray-300 disabled:opacity-50"
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page * 20 >= data.total}
                className="px-3 py-1 rounded border border-gray-300 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}