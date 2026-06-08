/**
 * Mailboxes List Page — Professional mail management interface.
 * Features: Create, suspend, reactivate, delete mailboxes with quota bars.
 * Handles empty states, loading states, error states properly.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listMailboxes,
  listDomains,
  createMailbox,
  deleteMailbox,
  suspendMailbox,
  reactivateMailbox,
  changeMailboxPassword,
} from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Card, CardHeader } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Spinner, EmptyState, ErrorState } from "@/lib/ui/Feedback";
import { Table, type Column } from "@/lib/ui/Table";
import { formatDate, formatMB } from "@/lib/utils";
import type { Mailbox } from "@/lib/api/mail";

export function MailboxesListPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<Mailbox | null>(null);
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [selectedDomain, setSelectedDomain] = useState("");
  const [quotaMB, setQuotaMB] = useState(1024);

  // Query mailboxes
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["mail", "mailboxes"],
    queryFn: () => listMailboxes({ page_size: 50 }),
  });

  // Query domains
  const { data: domainsData } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 100 }),
  });

  const domains = domainsData?.domains ?? [];
  const mailboxes = data?.mailboxes ?? [];

  // Create mutation
  const createMutation = useMutation({
    mutationFn: () => {
      const email = newEmail.includes("@") ? newEmail : `${newEmail}@${selectedDomain}`;
      return createMailbox({
        email,
        password: newPassword,
        domain_id: selectedDomain,
        quota_mb: quotaMB,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
      setShowCreateModal(false);
      setNewEmail("");
      setNewPassword("");
      setSelectedDomain("");
      setQuotaMB(1024);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteMailbox(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
      setConfirmDelete(null);
    },
  });

  // Suspend mutation
  const suspendMutation = useMutation({
    mutationFn: (id: string) => suspendMailbox(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
    },
  });

  // Reactivate mutation
  const reactivateMutation = useMutation({
    mutationFn: (id: string) => reactivateMailbox(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
    },
  });

  // Password change mutation
  const passwordMutation = useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      changeMailboxPassword(id, password),
    onSuccess: () => {
      setShowPasswordModal(null);
      setNewPassword("");
    },
  });

  // Calculate quota percentage safely
  const getQuotaPercent = (used?: number, total?: number) => {
    if (!total || total === 0) return 0;
    return Math.round(((used ?? 0) / total) * 100);
  };

  // Get quota color
  const getQuotaColor = (percent: number) => {
    if (percent >= 90) return "bg-danger";
    if (percent >= 75) return "bg-warning";
    return "bg-brand-500";
  };

  const columns: Column<Mailbox>[] = [
    {
      key: "email",
      header: "Email Address",
      cell: (m) => (
        <div className="font-medium text-ink-1">{m.email ?? "—"}</div>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (m) => (
        <Badge tone={m.status === "active" ? "success" : m.status === "suspended" ? "warning" : "neutral"}>
          {m.status ?? "unknown"}
        </Badge>
      ),
    },
    {
      key: "quota",
      header: "Storage",
      cell: (m) => {
        const used = m.quota_used_mb ?? 0;
        const total = m.quota_mb ?? 1;
        const percent = getQuotaPercent(used, total);
        return (
          <div className="w-36">
            <div className="mb-1 flex justify-between text-xs text-ink-3">
              <span>{formatMB(used)}</span>
              <span>{formatMB(total)}</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-surface-2">
              <div
                className={`h-full rounded-full transition-all ${getQuotaColor(percent)}`}
                style={{ width: `${percent}%` }}
              />
            </div>
          </div>
        );
      },
    },
    {
      key: "protocols",
      header: "Protocols",
      cell: (m) => (
        <div className="flex flex-wrap gap-1">
          {m.enable_imap && <Badge tone="info" size="sm">IMAP</Badge>}
          {m.enable_pop3 && <Badge tone="info" size="sm">POP3</Badge>}
          {m.enable_smtp && <Badge tone="info" size="sm">SMTP</Badge>}
          {!m.enable_imap && !m.enable_pop3 && !m.enable_smtp && (
            <span className="text-xs text-ink-3">—</span>
          )}
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
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowPasswordModal(m.id)}
          >
            Password
          </Button>
          {m.status === "active" ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => suspendMutation.mutate(m.id)}
            >
              Suspend
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => reactivateMutation.mutate(m.id)}
            >
              Reactivate
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

  const isPending = createMutation.isPending || deleteMutation.isPending || suspendMutation.isPending || reactivateMutation.isPending || passwordMutation.isPending;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Mailboxes"
        description={`${mailboxes.length} mailbox${mailboxes.length === 1 ? "" : "es"} configured`}
        actions={
          <Button variant="primary" onClick={() => setShowCreateModal(true)}>
            Add Mailbox
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
            description="Failed to load mailboxes."
            onRetry={() => refetch()}
          />
        ) : mailboxes.length === 0 ? (
          <EmptyState
            title="No mailboxes yet"
            description="Create your first mailbox to start receiving emails."
            action={
              <Button variant="primary" onClick={() => setShowCreateModal(true)}>
                Add Mailbox
              </Button>
            }
          />
        ) : (
          <Table
            columns={columns}
            rows={mailboxes}
            keyOf={(m) => m.id}
          />
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => !isPending && setShowCreateModal(false)}
        title="Add Mailbox"
        description="Create a new email mailbox for one of your domains."
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
              disabled={!selectedDomain || !newEmail || !newPassword}
            >
              Create Mailbox
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
            label="Username (local part)"
            value={newEmail}
            onChange={(e) => setNewEmail(e.target.value)}
            placeholder="john"
            description={selectedDomain ? `Full address: ${newEmail || "username"}@${domains.find(d => d.id === selectedDomain)?.domain}` : undefined}
          />

          <Input
            label="Password"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="Strong password"
          />

          <Input
            label="Quota (MB)"
            type="number"
            value={quotaMB}
            onChange={(e) => setQuotaMB(parseInt(e.target.value) || 1024)}
            min={100}
            max={102400}
          />
        </div>
      </Modal>

      {/* Password Change Modal */}
      <Modal
        open={showPasswordModal !== null}
        onClose={() => !isPending && setShowPasswordModal(null)}
        title="Change Password"
        description="Update the password for this mailbox."
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setShowPasswordModal(null)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={passwordMutation.isPending}
              onClick={() => {
                if (showPasswordModal && newPassword) {
                  passwordMutation.mutate({ id: showPasswordModal, password: newPassword });
                }
              }}
              disabled={!newPassword}
            >
              Update Password
            </Button>
          </>
        }
      >
        <Input
          label="New Password"
          type="password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          placeholder="Enter new password"
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={confirmDelete !== null}
        onClose={() => !isPending && setConfirmDelete(null)}
        title="Delete Mailbox"
        description={`Are you sure you want to delete "${confirmDelete?.email}"? This action cannot be undone.`}
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
              Delete Mailbox
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