/**
 * Mailboxes List Page
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
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { Spinner } from "@/lib/ui/Feedback";
import { formatDate, formatMB } from "@/lib/utils";

export function MailboxesListPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState<string | null>(null);
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [selectedDomain, setSelectedDomain] = useState("");
  const [quotaMB, setQuotaMB] = useState(1024);

  // Query mailboxes
  const { data, isLoading, error } = useQuery({
    queryKey: ["mail", "mailboxes"],
    queryFn: () => listMailboxes({ page_size: 50 }),
  });

  // Query domains
  const { data: domainsData } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 100 }),
  });

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
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteMailbox(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "mailboxes"] });
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

  // Calculate quota percentage
  const getQuotaPercent = (used: number, total: number) => {
    if (total === 0) return 0;
    return Math.round((used / total) * 100);
  };

  // Get quota color
  const getQuotaColor = (percent: number) => {
    if (percent >= 90) return "bg-red-500";
    if (percent >= 75) return "bg-yellow-500";
    return "bg-blue-500";
  };

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
        Failed to load mailboxes. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Mailboxes</h1>
          <p className="text-gray-500">
            Manage email mailboxes for your domains
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          Add Mailbox
        </Button>
      </div>

      {/* Mailbox List */}
      <Card>
        {data?.mailboxes.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <p className="text-lg">No mailboxes configured</p>
            <p className="text-sm">Add a mailbox to start receiving emails</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Email
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Quota
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Protocols
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {data?.mailboxes.map((mailbox) => {
                  const quotaPercent = getQuotaPercent(mailbox.quota_used_mb, mailbox.quota_mb);
                  return (
                    <tr key={mailbox.id}>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="font-medium text-gray-900">{mailbox.email}</div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge
                          tone={
                            mailbox.status === "active"
                              ? "success"
                              : mailbox.status === "suspended"
                              ? "warning"
                              : "neutral"
                          }
                        >
                          {mailbox.status}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="w-32">
                          <div className="flex justify-between text-xs mb-1">
                            <span>{formatMB(mailbox.quota_used_mb)}</span>
                            <span>{formatMB(mailbox.quota_mb)}</span>
                          </div>
                          <div className="w-full bg-gray-200 rounded-full h-2">
                            <div
                              className={`h-2 rounded-full ${getQuotaColor(quotaPercent)}`}
                              style={{ width: `${quotaPercent}%` }}
                            />
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex gap-1">
                          {mailbox.enable_imap && (
                            <Badge tone="info">IMAP</Badge>
                          )}
                          {mailbox.enable_pop3 && (
                            <Badge tone="info">POP3</Badge>
                          )}
                          {mailbox.enable_smtp && (
                            <Badge tone="info">SMTP</Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(mailbox.created_at)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setShowPasswordModal(mailbox.id)}
                        >
                          Password
                        </Button>
                        {mailbox.status === "active" ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => suspendMutation.mutate(mailbox.id)}
                          >
                            Suspend
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => reactivateMutation.mutate(mailbox.id)}
                          >
                            Reactivate
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (confirm(`Delete mailbox ${mailbox.email}?`)) {
                              deleteMutation.mutate(mailbox.id);
                            }
                          }}
                        >
                          Delete
                        </Button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Add Mailbox"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate();
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Domain
            </label>
            <select
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              value={selectedDomain}
              onChange={(e) => setSelectedDomain(e.target.value)}
              required
            >
              <option value="">Select domain...</option>
              {domainsData?.domains.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.domain}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Username (local part)
            </label>
            <Input
              type="text"
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              placeholder="john"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <Input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Strong password"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Quota (MB)
            </label>
            <Input
              type="number"
              value={quotaMB}
              onChange={(e) => setQuotaMB(parseInt(e.target.value) || 1024)}
              min={100}
              max={102400}
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setShowCreateModal(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create Mailbox"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Password Change Modal */}
      <Modal
        open={showPasswordModal !== null}
        onClose={() => setShowPasswordModal(null)}
        title="Change Password"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (showPasswordModal) {
              passwordMutation.mutate({ id: showPasswordModal, password: newPassword });
            }
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700">
              New Password
            </label>
            <Input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Strong password"
              required
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setShowPasswordModal(null)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={passwordMutation.isPending}
            >
              {passwordMutation.isPending ? "Updating..." : "Update Password"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}