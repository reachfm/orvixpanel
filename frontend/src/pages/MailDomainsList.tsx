/**
 * Mail Domains List Page — Professional mail domain management.
 * Features: Create domains, view DNS records (SPF/DKIM/DMARC), delete domains.
 * Handles empty states, loading states, error states properly.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listDomains, createDomain, deleteDomain, getDNSRecords } from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Spinner, EmptyState, ErrorState } from "@/lib/ui/Feedback";
import { Table, type Column } from "@/lib/ui/Table";
import { formatDate } from "@/lib/utils";
import type { MailDomain } from "@/lib/api/mail";

export function MailDomainsListPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDNSModal, setShowDNSModal] = useState<MailDomain | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<MailDomain | null>(null);
  const [newDomain, setNewDomain] = useState("");
  const [maxMailboxes, setMaxMailboxes] = useState(100);

  // Query domains
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 50 }),
  });

  // Create mutation
  const createMutation = useMutation({
    mutationFn: () => createDomain({ domain: newDomain, max_mailboxes: maxMailboxes }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "domains"] });
      setShowCreateModal(false);
      setNewDomain("");
      setMaxMailboxes(100);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteDomain(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "domains"] });
      setConfirmDelete(null);
    },
  });

  // DNS records query
  const { data: dnsRecords, isLoading: dnsLoading } = useQuery({
    queryKey: ["mail", "domains", showDNSModal?.id, "dns"],
    queryFn: () => (showDNSModal ? getDNSRecords(showDNSModal.id) : null),
    enabled: !!showDNSModal,
  });

  const domains = data?.domains ?? [];

  // Parse DKIM status
  const getDKIMStatus = (domain: MailDomain) => {
    if (domain.dkim_public) return { label: "Configured", tone: "success" as const };
    return { label: "Not Set", tone: "neutral" as const };
  };

  const columns: Column<MailDomain>[] = [
    {
      key: "domain",
      header: "Domain",
      cell: (d) => <div className="font-medium text-ink-1">{d.domain ?? "—"}</div>,
    },
    {
      key: "status",
      header: "Status",
      cell: (d) => (
        <Badge tone={d.status === "active" ? "success" : "warning"}>
          {d.status ?? "unknown"}
        </Badge>
      ),
    },
    {
      key: "dkim",
      header: "DKIM",
      cell: (d) => {
        const dk = getDKIMStatus(d);
        return <Badge tone={dk.tone}>{dk.label}</Badge>;
      },
    },
    {
      key: "catch_all",
      header: "Catch All",
      cell: (d) => (
        <span className="text-sm text-ink-2">{d.is_catch_all ? "Yes" : "No"}</span>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (d) => (
        <span className="font-mono text-xs text-ink-2">{formatDate(d.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (d) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowDNSModal(d)}
          >
            DNS Records
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setConfirmDelete(d)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  const isPending = createMutation.isPending || deleteMutation.isPending;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Mail Domains"
        description={`${domains.length} domain${domains.length === 1 ? "" : "s"} configured`}
        actions={
          <Button variant="primary" onClick={() => setShowCreateModal(true)}>
            Add Domain
          </Button>
        }
      />

      <Card>
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size={24} />
          </div>
        ) : error ? (
          <ErrorState
            description="Failed to load mail domains."
            onRetry={() => refetch()}
          />
        ) : domains.length === 0 ? (
          <EmptyState
            title="No mail domains yet"
            description="Add your first mail domain to start managing email for your domains."
            action={
              <Button variant="primary" onClick={() => setShowCreateModal(true)}>
                Add Domain
              </Button>
            }
          />
        ) : (
          <Table
            columns={columns}
            rows={domains}
            keyOf={(d) => d.id}
          />
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => !isPending && setShowCreateModal(false)}
        title="Add Mail Domain"
        description="Add a new mail domain for email hosting."
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setShowCreateModal(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={createMutation.isPending}
              onClick={() => createMutation.mutate()}
              disabled={!newDomain.trim()}
            >
              Create Domain
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Domain Name"
            value={newDomain}
            onChange={(e) => setNewDomain(e.target.value)}
            placeholder="example.com"
            description="Enter the domain name without www"
          />

          <Input
            label="Max Mailboxes"
            type="number"
            value={maxMailboxes}
            onChange={(e) => setMaxMailboxes(parseInt(e.target.value) || 100)}
            min={1}
            max={10000}
          />
        </div>
      </Modal>

      {/* DNS Records Modal */}
      <Modal
        open={showDNSModal !== null}
        onClose={() => setShowDNSModal(null)}
        title="DNS Configuration"
        description={`DNS records for ${showDNSModal?.domain ?? "domain"}. Add these to your DNS provider.`}
        footer={
          <Button variant="secondary" onClick={() => setShowDNSModal(null)}>
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

      {/* Delete Confirmation Modal */}
      <Modal
        open={confirmDelete !== null}
        onClose={() => !isPending && setConfirmDelete(null)}
        title="Delete Mail Domain"
        description={`Are you sure you want to delete "${confirmDelete?.domain}"? This will remove all mailboxes and aliases for this domain.`}
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
              Delete Domain
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the mail domain and all associated mailboxes, aliases, and forwarders.
        </div>
      </Modal>
    </div>
  );
}