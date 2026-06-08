/**
 * Mail Aliases Page — Professional email alias management.
 * Features: Create aliases, view destinations, delete aliases.
 * Handles empty states, loading states, error states properly.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listAliases, listDomains, createAlias, deleteAlias } from "@/lib/api/mail";
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
import type { MailAlias } from "@/lib/api/mail";

export function MailAliasesPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<MailAlias | null>(null);
  const [sourceEmail, setSourceEmail] = useState("");
  const [destinations, setDestinations] = useState("");
  const [selectedDomain, setSelectedDomain] = useState("");

  // Query aliases
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["mail", "aliases"],
    queryFn: () => listAliases({ page_size: 50 }),
  });

  // Query domains for dropdown
  const { data: domainsData } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 100 }),
  });

  const domains = domainsData?.domains ?? [];
  const aliases = data?.aliases ?? [];

  // Create mutation
  const createMutation = useMutation({
    mutationFn: () => {
      const destArray = destinations.split(",").map((d) => d.trim()).filter(Boolean);
      return createAlias({
        source_email: sourceEmail,
        destinations: destArray,
        domain_id: selectedDomain,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "aliases"] });
      setShowCreateModal(false);
      setSourceEmail("");
      setDestinations("");
      setSelectedDomain("");
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAlias(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "aliases"] });
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

  const columns: Column<MailAlias>[] = [
    {
      key: "source",
      header: "Source Email",
      cell: (a) => <div className="font-medium text-ink-1">{a.source_email ?? "—"}</div>,
    },
    {
      key: "destinations",
      header: "Destinations",
      cell: (a) => {
        const dests = parseDestinations(a.destinations);
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
      key: "status",
      header: "Status",
      cell: (a) => (
        <Badge tone={a.status === "active" ? "success" : "neutral"}>
          {a.status ?? "unknown"}
        </Badge>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (a) => (
        <span className="font-mono text-xs text-ink-2">{formatDate(a.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (a) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setConfirmDelete(a)}
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
        title="Mail Aliases"
        description={`${aliases.length} alias${aliases.length === 1 ? "" : "es"} configured`}
        actions={
          <Button variant="primary" onClick={() => setShowCreateModal(true)}>
            Add Alias
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
            description="Failed to load aliases."
            onRetry={() => refetch()}
          />
        ) : aliases.length === 0 ? (
          <EmptyState
            title="No aliases yet"
            description="Create an alias to forward emails from one address to another."
            action={
              <Button variant="primary" onClick={() => setShowCreateModal(true)}>
                Add Alias
              </Button>
            }
          />
        ) : (
          <Table
            columns={columns}
            rows={aliases}
            keyOf={(a) => a.id}
          />
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => !isPending && setShowCreateModal(false)}
        title="Add Alias"
        description="Create an email alias that forwards to one or more destinations."
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
              Create Alias
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
            label="Source Email (Alias)"
            value={sourceEmail}
            onChange={(e) => setSourceEmail(e.target.value)}
            placeholder="alias"
            hint={selectedDomain ? `Full address: ${sourceEmail || "alias"}@${domains.find(d => d.id === selectedDomain)?.domain}` : undefined}
          />

          <Input
            label="Destinations"
            value={destinations}
            onChange={(e) => setDestinations(e.target.value)}
            placeholder="user1@example.com, user2@example.com"
            hint="Comma-separated list of destination email addresses"
          />
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={confirmDelete !== null}
        onClose={() => !isPending && setConfirmDelete(null)}
        title="Delete Alias"
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
              Delete Alias
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the alias. Emails sent to this address will no longer be forwarded.
        </div>
      </Modal>
    </div>
  );
}