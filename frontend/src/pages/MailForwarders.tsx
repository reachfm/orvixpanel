/**
 * Mail Forwarders Page — Professional email forwarder management.
 * Features: Create forwarders, view destinations, keep local copy option, delete.
 * Handles empty states, loading states, error states properly.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listForwarders, listDomains, createForwarder, deleteForwarder } from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Spinner, EmptyState, ErrorState } from "@/lib/ui/Feedback";
import { Table, type Column } from "@/lib/ui/Table";
import { Select } from "@/lib/ui/Select";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/ui/cn";
import type { MailForwarder } from "@/lib/api/mail";

export function MailForwardersPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<MailForwarder | null>(null);
  const [sourceEmail, setSourceEmail] = useState("");
  const [destinations, setDestinations] = useState("");
  const [selectedDomain, setSelectedDomain] = useState("");
  const [keepLocal, setKeepLocal] = useState(false);

  // Query forwarders
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["mail", "forwarders"],
    queryFn: () => listForwarders({ page_size: 50 }),
  });

  // Query domains for dropdown
  const { data: domainsData } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 100 }),
  });

  const domains = domainsData?.domains ?? [];
  const forwarders = data?.forwarders ?? [];

  // Create mutation
  const createMutation = useMutation({
    mutationFn: () => {
      const destArray = destinations.split(",").map((d) => d.trim()).filter(Boolean);
      return createForwarder({
        source_email: sourceEmail,
        destinations: destArray,
        domain_id: selectedDomain,
        keep_copy: keepLocal,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "forwarders"] });
      setShowCreateModal(false);
      setSourceEmail("");
      setDestinations("");
      setSelectedDomain("");
      setKeepLocal(false);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteForwarder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "forwarders"] });
      setConfirmDelete(null);
    },
  });

  // Parse destinations for display
  const parseDestinations = (json: string | null | undefined): string[] => {
    if (!json) return [];
    try {
      return JSON.parse(json);
    } catch {
      return [];
    }
  };

  const columns: Column<MailForwarder>[] = [
    {
      key: "source",
      header: "Source Email",
      cell: (f) => <div className="font-medium text-ink-1">{f.source_email ?? "—"}</div>,
    },
    {
      key: "destinations",
      header: "Destinations",
      cell: (f) => {
        const dests = parseDestinations(f.destinations);
        return (
          <div className="flex flex-wrap gap-1">
            {dests.length > 0 ? (
              dests.map((dest, i) => (
                <Badge key={i} tone="neutral">
                  {dest}
                </Badge>
              ))
            ) : (
              <span className="text-xs text-ink-3">—</span>
            )}
          </div>
        );
      },
    },
    {
      key: "keep_copy",
      header: "Keep Local",
      cell: (f) => (
        <Badge tone={f.keep_copy ? "success" : "neutral"}>
          {f.keep_copy ? "Yes" : "No"}
        </Badge>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (f) => (
        <Badge tone={f.status === "active" ? "success" : "neutral"}>
          {f.status ?? "unknown"}
        </Badge>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (f) => (
        <span className="font-mono text-xs text-ink-2">{formatDate(f.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (f) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setConfirmDelete(f)}
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
        title="Mail Forwarders"
        description={`${forwarders.length} forwarder${forwarders.length === 1 ? "" : "s"} configured`}
        actions={
          <Button variant="primary" onClick={() => setShowCreateModal(true)}>
            Add Forwarder
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
            description="Failed to load forwarders."
            onRetry={() => refetch()}
          />
        ) : forwarders.length === 0 ? (
          <EmptyState
            title="No forwarders yet"
            description="Create a forwarder to redirect emails from one address to another."
            action={
              <Button variant="primary" onClick={() => setShowCreateModal(true)}>
                Add Forwarder
              </Button>
            }
          />
        ) : (
          <Table
            columns={columns}
            rows={forwarders}
            keyOf={(f) => f.id}
          />
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => !isPending && setShowCreateModal(false)}
        title="Add Forwarder"
        description="Forward emails from a source address to one or more destinations."
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
              disabled={!selectedDomain || !sourceEmail.trim() || !destinations.trim()}
            >
              Create Forwarder
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Select
            label="Domain"
            value={selectedDomain}
            onChange={(e) => setSelectedDomain(e.target.value)}
          >
            <option value="">Select domain...</option>
            {domains.map((d) => (
              <option key={d.id} value={d.id}>
                {d.domain}
              </option>
            ))}
          </Select>

          <Input
            label="Source Email"
            value={sourceEmail}
            onChange={(e) => setSourceEmail(e.target.value)}
            placeholder="from@domain.com"
            hint={selectedDomain ? `Full address: ${sourceEmail || "from"}@${domains.find(d => d.id === selectedDomain)?.domain}` : undefined}
          />

          <Input
            label="Destinations"
            value={destinations}
            onChange={(e) => setDestinations(e.target.value)}
            placeholder="dest1@example.com, dest2@example.com"
            hint="Comma-separated list of destination email addresses"
          />

          {/* Keep Local Copy Toggle */}
          <div
            className={cn(
              "flex items-center gap-3 rounded-md border p-3 cursor-pointer",
              keepLocal
                ? "border-brand-500 bg-brand-500/5"
                : "border-surface-border hover:border-ink-3",
            )}
            onClick={() => setKeepLocal(!keepLocal)}
          >
            <div
              className={cn(
                "flex h-5 w-5 items-center justify-center rounded border",
                keepLocal
                  ? "border-brand-500 bg-brand-500 text-white"
                  : "border-surface-border bg-surface-1",
              )}
            >
              {keepLocal && (
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="h-3.5 w-3.5">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              )}
            </div>
            <div className="flex-1">
              <div className="text-sm font-medium text-ink-1">Keep local copy</div>
              <div className="text-xs text-ink-3">Also deliver a copy to the source mailbox</div>
            </div>
          </div>
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={confirmDelete !== null}
        onClose={() => !isPending && setConfirmDelete(null)}
        title="Delete Forwarder"
        description={`Are you sure you want to delete "${confirmDelete?.source_email}"?`}
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
              Delete Forwarder
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the forwarder. Emails sent to this address will no longer be redirected.
        </div>
      </Modal>
    </div>
  );
}