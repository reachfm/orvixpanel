/**
 * SSL Certificate detail page. Shows certificate info, events, and actions.
 *
 * Routes:
 *   /ssl/certificates/:id          CertificateDetailPage
 */

import { useState } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import {
  sslKeys,
  getCertificate,
  getCertificateEvents,
  renewCertificate,
  deleteCertificate,
  type SSLCertificate,
  type SSLEvent,
} from "@/lib/api/ssl";

function formatDate(dateStr?: string): string {
  if (!dateStr) return "—";
  return new Date(dateStr).toLocaleString();
}

function getStatusTone(status: string): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "issued":
      return "success";
    case "expiring_soon":
      return "warning";
    case "expired":
    case "failed":
    case "revoked":
      return "danger";
    default:
      return "neutral";
  }
}

function getEventTypeTone(eventType: string): "success" | "warning" | "danger" | "neutral" {
  if (eventType.includes("failed") || eventType.includes("error")) return "danger";
  if (eventType.includes("started") || eventType.includes("requested")) return "neutral";
  return "success";
}

export function CertificateDetailPage() {
  const { id } = useParams({ from: "/ssl/certificates/$id" });
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Modal state
  const [renewModalOpen, setRenewModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  // Fetch certificate
  const certQuery = useQuery({
    queryKey: sslKeys.certificate(id),
    queryFn: () => getCertificate(id),
  });

  // Fetch events
  const eventsQuery = useQuery({
    queryKey: sslKeys.certificateEvents(id),
    queryFn: () => getCertificateEvents(id),
  });

  // Renew mutation
  const renewMutation = useMutation({
    mutationFn: () => renewCertificate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: sslKeys.certificate(id) });
      setRenewModalOpen(false);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => deleteCertificate(id),
    onSuccess: () => {
      navigate({ to: "/ssl/certificates" });
    },
  });

  const cert = certQuery.data;
  const events = eventsQuery.data ?? [];

  // Event table columns
  const eventColumns: Column<SSLEvent>[] = [
    {
      key: "created_at",
      header: "Time",
      cell: (event) => (
        <span className="text-sm text-gray-600 dark:text-gray-400">
          {formatDate(event.created_at)}
        </span>
      ),
    },
    {
      key: "event_type",
      header: "Event",
      cell: (event) => (
        <StatusPill tone={getEventTypeTone(event.event_type)}>
          {event.event_type.replace(/_/g, " ")}
        </StatusPill>
      ),
    },
    {
      key: "message",
      header: "Message",
      cell: (event) => (
        <span className="text-sm">{event.message}</span>
      ),
    },
    {
      key: "error_detail",
      header: "Details",
      cell: (event) => (
        <span className="text-sm text-red-600 dark:text-red-400">
          {event.error_detail || "—"}
        </span>
      ),
    },
  ];

  if (certQuery.isLoading) return <LoadingState />;
  if (certQuery.isError) return <ErrorState description="Failed to load certificate" onRetry={() => certQuery.refetch()} />;
  if (!cert) return <ErrorState description="Certificate not found" />;

  return (
    <div className="space-y-6">
      <PageHeader
        title={cert.common_name}
        description={`Certificate ID: ${cert.id}`}
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setRenewModalOpen(true)}>
              Renew
            </Button>
            <Button
              variant="danger"
              onClick={() => setDeleteModalOpen(true)}
            >
              Delete
            </Button>
          </div>
        }
      />

      {/* Certificate Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card className="p-4">
          <h3 className="text-lg font-semibold mb-4">Certificate Details</h3>
          <dl className="space-y-3">
            <div className="flex justify-between">
              <dt className="text-gray-500">Status</dt>
              <dd>
                <StatusPill tone={getStatusTone(cert.status)}>
                  {cert.status.replace(/_/g, " ")}
                </StatusPill>
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Provider</dt>
              <dd className="capitalize">{cert.provider}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Auto-Renew</dt>
              <dd className={cert.auto_renew ? "text-green-600" : "text-gray-400"}>
                {cert.auto_renew ? "Enabled" : "Disabled"}
              </dd>
            </div>
            {cert.serial_number && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Serial Number</dt>
                <dd className="font-mono text-sm truncate max-w-[200px]">{cert.serial_number}</dd>
              </div>
            )}
            {cert.fingerprint && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Fingerprint</dt>
                <dd className="font-mono text-xs truncate max-w-[200px]">{cert.fingerprint}</dd>
              </div>
            )}
          </dl>
        </Card>

        <Card className="p-4">
          <h3 className="text-lg font-semibold mb-4">Validity Period</h3>
          <dl className="space-y-3">
            <div className="flex justify-between">
              <dt className="text-gray-500">Not Before</dt>
              <dd>{formatDate(cert.not_before)}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Not After</dt>
              <dd className={cert.status === "expiring_soon" ? "text-yellow-600 font-medium" : ""}>
                {formatDate(cert.not_after)}
              </dd>
            </div>
            {cert.issuer && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Issuer</dt>
                <dd className="truncate max-w-[200px]">{cert.issuer}</dd>
              </div>
            )}
          </dl>
        </Card>
      </div>

      {/* SANs */}
      {cert.san_names && cert.san_names.length > 0 && (
        <Card className="p-4">
          <h3 className="text-lg font-semibold mb-4">Subject Alternative Names</h3>
          <ul className="space-y-1">
            {cert.san_names.map((san, i) => (
              <li key={i} className="text-sm font-mono">{san}</li>
            ))}
          </ul>
        </Card>
      )}

      {/* Error Message */}
      {cert.error_message && (
        <Card className="p-4 border-red-300 dark:border-red-800">
          <h3 className="text-lg font-semibold mb-2 text-red-600 dark:text-red-400">Error</h3>
          <p className="text-sm text-red-600 dark:text-red-400">{cert.error_message}</p>
        </Card>
      )}

      {/* Events */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Certificate History</h3>
        {eventsQuery.isLoading ? (
          <div className="py-8 text-center text-gray-500">Loading events...</div>
        ) : events.length === 0 ? (
          <EmptyState
            title="No events"
            description="Certificate events will appear here"
          />
        ) : (
          <Table<SSLEvent> rows={events} columns={eventColumns} keyOf={(e) => e.id} />
        )}
      </Card>

      {/* Renew Modal */}
      <Modal
        open={renewModalOpen}
        onClose={() => setRenewModalOpen(false)}
        title="Renew Certificate"
        footer={
          <>
            <Button variant="outline" onClick={() => setRenewModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => renewMutation.mutate()}
              loading={renewMutation.isPending}
            >
              {renewMutation.isPending ? "Renewing..." : "Renew"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to renew the certificate for{" "}
          <strong>{cert.common_name}</strong>?
        </p>
        {renewMutation.isError && (
          <p className="mt-2 text-sm text-red-600 dark:text-red-400">
            Failed to renew certificate. Please try again.
          </p>
        )}
      </Modal>

      {/* Delete Modal */}
      <Modal
        open={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="Delete Certificate"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteMutation.mutate()}
              loading={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete the certificate for{" "}
          <strong>{cert.common_name}</strong>? This action cannot be undone.
        </p>
      </Modal>
    </div>
  );
}