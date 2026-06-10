/**
 * Mail Domain Detail Page — View domain details and associated mailboxes.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams, useNavigate } from "@tanstack/react-router";
import {
  getDomain,
  listMailboxes,
  getDNSRecords,
  deleteMailbox,
  suspendMailbox,
  reactivateMailbox,
  type Mailbox,
} from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { Table, type Column } from "@/lib/ui/Table";
import { formatDate } from "@/lib/utils";

export function MailDomainDetailPage() {
  const { id } = useParams({ from: "/app/mail/domains/$id" });
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [showDNSModal, setShowDNSModal] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<Mailbox | null>(null);
  const [confirmSuspend, setConfirmSuspend] = useState<Mailbox | null>(null);

  // Query domain
  const { data: domain, isLoading: domainLoading, error: domainError, refetch: refetchDomain } = useQuery({
    queryKey: ["mail", "domains", id],
    queryFn: () => getDomain(id),
  });

  // Query mailboxes for this domain
  const { data: mailboxesData, isLoading: mailboxesLoading } = useQuery({
    queryKey: ["mail", "mailboxes", { domain_id: id }],
    queryFn: () => listMailboxes({ domain_id: id, page_size: 50 }),
  });

  // Query DNS records
  const { data: dnsRecords, isLoading: dnsLoading } = useQuery({
    queryKey: ["mail", "domains", id, "dns"],
    queryFn: () => getDNSRecords(id),
    enabled: showDNSModal,
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (mailboxId: string) => deleteMailbox(mailboxId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
      setConfirmDelete(null);
    },
  });

  // Suspend mutation
  const suspendMutation = useMutation({
    mutationFn: (mailboxId: string) => suspendMailbox(mailboxId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
      setConfirmSuspend(null);
    },
  });

  // Reactivate mutation
  const reactivateMutation = useMutation({
    mutationFn: (mailboxId: string) => reactivateMailbox(mailboxId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
    },
  });

  const mailboxes = mailboxesData?.mailboxes ?? [];
  const isPending = deleteMutation.isPending || suspendMutation.isPending || reactivateMutation.isPending;

  // Table columns
  const columns: Column<Mailbox>[] = [
    {
      key: "email",
      header: "Email Address",
      cell: (m) => (
        <div className="font-medium text-ink-1">{m.email}</div>
      ),
    },
    {
      key: "quota",
      header: "Quota",
      cell: (m) => (
        <span className="text-sm text-ink-2">
          {m.quota_mb} MB ({((m.quota_used_mb / m.quota_mb) * 100).toFixed(0)}% used)
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (m) => (
        <Badge tone={m.status === "active" ? "success" : m.status === "suspended" ? "warning" : "neutral"}>
          {m.status}
        </Badge>
      ),
    },
    {
      key: "protocols",
      header: "Protocols",
      cell: (m) => (
        <div className="flex gap-1">
          {m.enable_imap && <Badge tone="neutral">IMAP</Badge>}
          {m.enable_pop3 && <Badge tone="neutral">POP3</Badge>}
          {m.enable_smtp && <Badge tone="neutral">SMTP</Badge>}
        </div>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (m) => (
        <span className="font-mono text-xs text-ink-2">{formatDate(m.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (m) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          {m.status === "suspended" ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => reactivateMutation.mutate(m.id)}
              disabled={reactivateMutation.isPending}
            >
              Reactivate
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmSuspend(m)}
            >
              Suspend
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setConfirmDelete(m)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  if (domainLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Spinner size={32} />
      </div>
    );
  }

  if (domainError || !domain) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Domain Not Found"
          description="The requested mail domain could not be found"
        />
        <Card>
          <ErrorState
            description="Failed to load mail domain."
            onRetry={() => refetchDomain()}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={domain.domain}
        description={`Mail domain configuration and mailboxes`}
        actions={
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => navigate({ to: "/mail/domains" })}>
              Back to Domains
            </Button>
            <Button variant="secondary" onClick={() => setShowDNSModal(true)}>
              View DNS Records
            </Button>
          </div>
        }
      />

      {/* Domain Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <div className="text-sm text-ink-3">Status</div>
          <div className="mt-1">
            <Badge tone={domain.status === "active" ? "success" : "warning"}>
              {domain.status}
            </Badge>
          </div>
        </Card>
        <Card>
          <div className="text-sm text-ink-3">DKIM</div>
          <div className="mt-1">
            <Badge tone={domain.dkim_public ? "success" : "neutral"}>
              {domain.dkim_public ? "Configured" : "Not Set"}
            </Badge>
          </div>
        </Card>
        <Card>
          <div className="text-sm text-ink-3">Mailboxes</div>
          <div className="text-3xl font-bold text-ink-1 mt-1">{mailboxes.length}</div>
          <div className="text-xs text-ink-3">of {domain.max_mailboxes} max</div>
        </Card>
      </div>

      {/* SPF/DMARC Info */}
      <Card>
        <h2 className="text-sm font-semibold text-ink-3 uppercase tracking-wide mb-4">Authentication</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <div className="text-xs text-ink-3 mb-1">SPF Record</div>
            <code className="block text-sm bg-surface-2 p-2 rounded text-ink-1">
              {domain.spf_record || "Not configured"}
            </code>
          </div>
          <div>
            <div className="text-xs text-ink-3 mb-1">DMARC Policy</div>
            <code className="block text-sm bg-surface-2 p-2 rounded text-ink-1">
              {domain.dmarc_policy || "v=none"}
            </code>
          </div>
        </div>
      </Card>

      {/* Mailboxes Table */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-ink-1">Mailboxes</h2>
          <span className="text-sm text-ink-2">
            {mailboxes.length} of {domain.max_mailboxes} used
          </span>
        </div>

        {mailboxesLoading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={24} />
          </div>
        ) : mailboxes.length === 0 ? (
          <div className="text-center py-8 text-ink-3">
            <p>No mailboxes configured for this domain.</p>
          </div>
        ) : (
          <Table
            columns={columns}
            rows={mailboxes}
            keyOf={(m) => m.id}
          />
        )}
      </Card>

      {/* DNS Records Modal */}
      <Modal
        open={showDNSModal}
        onClose={() => setShowDNSModal(false)}
        title="DNS Configuration"
        description={`DNS records for ${domain.domain}. Add these to your DNS provider.`}
        footer={
          <Button variant="secondary" onClick={() => setShowDNSModal(false)}>
            Close
          </Button>
        }
      >
        {dnsLoading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : dnsRecords ? (
          <div className="space-y-4">
            <div className="rounded-md border border-surface-border bg-surface-2 p-4">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wide text-ink-3">SPF Record</span>
                <Badge tone="info">TXT</Badge>
              </div>
              <code className="block whitespace-pre-wrap text-xs text-ink-1">{dnsRecords.spf ?? "—"}</code>
            </div>

            <div className="rounded-md border border-surface-border bg-surface-2 p-4">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wide text-ink-3">DMARC Record</span>
                <Badge tone="info">TXT</Badge>
              </div>
              <code className="block whitespace-pre-wrap text-xs text-ink-1">{dnsRecords.dmarc ?? "—"}</code>
            </div>

            <div className="rounded-md border border-surface-border bg-surface-2 p-4">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-semibold uppercase tracking-wide text-ink-3">DKIM Record</span>
                <Badge tone="info">TXT</Badge>
              </div>
              <code className="block whitespace-pre-wrap break-all text-xs text-ink-1">{dnsRecords.dkim ?? "—"}</code>
            </div>

            <p className="text-xs text-ink-3">
              Add these DNS records to your domain's DNS settings to enable mail authentication and prevent spam.
            </p>
          </div>
        ) : (
          <p className="text-sm text-ink-3">DNS records unavailable.</p>
        )}
      </Modal>

      {/* Suspend Confirmation Modal */}
      <Modal
        open={confirmSuspend !== null}
        onClose={() => !isPending && setConfirmSuspend(null)}
        title="Suspend Mailbox"
        description={`Are you sure you want to suspend "${confirmSuspend?.email}"?`}
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setConfirmSuspend(null)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={suspendMutation.isPending}
              onClick={() => confirmSuspend && suspendMutation.mutate(confirmSuspend.id)}
            >
              Suspend
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-warning/30 bg-warning/5 p-3 text-sm text-ink-2">
          The mailbox will be disabled and unable to send or receive emails until reactivated.
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={confirmDelete !== null}
        onClose={() => !isPending && setConfirmDelete(null)}
        title="Delete Mailbox"
        description={`Are you sure you want to delete "${confirmDelete?.email}"?`}
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setConfirmDelete(null)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={deleteMutation.isPending}
              onClick={() => confirmDelete && deleteMutation.mutate(confirmDelete.id)}
            >
              Delete
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the mailbox and all stored emails.
        </div>
      </Modal>
    </div>
  );
}