/**
 * Mail Domains List Page
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listDomains, createDomain, deleteDomain, getDNSRecords } from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { Spinner } from "@/lib/ui/Feedback";
import { formatDate } from "@/lib/utils";

export function MailDomainsListPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDNSModal, setShowDNSModal] = useState<string | null>(null);
  const [newDomain, setNewDomain] = useState("");
  const [maxMailboxes, setMaxMailboxes] = useState(100);

  // Query domains
  const { data, isLoading, error } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 50 }),
  });

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (domain: string) =>
      createDomain({ domain, max_mailboxes: maxMailboxes }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "domains"] });
      setShowCreateModal(false);
      setNewDomain("");
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteDomain(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "domains"] });
    },
  });

  // DNS records query
  const { data: dnsRecords } = useQuery({
    queryKey: ["mail", "domains", showDNSModal, "dns"],
    queryFn: () => (showDNSModal ? getDNSRecords(showDNSModal) : null),
    enabled: !!showDNSModal,
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
        Failed to load mail domains. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Mail Domains</h1>
          <p className="text-gray-500">
            Manage mail domains, DNS records, and DKIM keys
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          Add Domain
        </Button>
      </div>

      {/* Domain List */}
      <Card>
        {data?.domains.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <p className="text-lg">No mail domains configured</p>
            <p className="text-sm">Add your first mail domain to get started</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Domain
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    DKIM
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Catch All
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
                {data?.domains.map((domain) => (
                  <tr key={domain.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="font-medium text-gray-900">{domain.domain}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Badge
                        tone={domain.status === "active" ? "success" : "warning"}
                      >
                        {domain.status}
                      </Badge>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {domain.dkim_public ? (
                        <Badge tone="info">Configured</Badge>
                      ) : (
                        <Badge tone="neutral">Not Set</Badge>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {domain.is_catch_all ? "Yes" : "No"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatDate(domain.created_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                      <Button
                        variant="ghost"
                        
                        onClick={() => setShowDNSModal(domain.id)}
                      >
                        DNS Records
                      </Button>
                      <Button
                        variant="ghost"
                        
                        onClick={() => {
                          if (confirm(`Delete ${domain.domain}?`)) {
                            deleteMutation.mutate(domain.id);
                          }
                        }}
                      >
                        Delete
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Add Mail Domain"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(newDomain);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Domain Name
            </label>
            <Input
              type="text"
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              placeholder="example.com"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Max Mailboxes
            </label>
            <Input
              type="number"
              value={maxMailboxes}
              onChange={(e) => setMaxMailboxes(Number(e.target.value))}
              min={1}
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
              {createMutation.isPending ? "Creating..." : "Create Domain"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* DNS Records Modal */}
      <Modal
        open={!!showDNSModal}
        onClose={() => setShowDNSModal(null)}
        title="DNS Records"
      >
        {dnsRecords ? (
          <div className="space-y-4">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-1">SPF Record</h3>
              <code className="block bg-gray-100 p-2 rounded text-sm">
                {dnsRecords.spf}
              </code>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-1">DMARC Record</h3>
              <code className="block bg-gray-100 p-2 rounded text-sm">
                {dnsRecords.dmarc}
              </code>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-1">DKIM Record</h3>
              <code className="block bg-gray-100 p-2 rounded text-sm whitespace-pre-wrap">
                {dnsRecords.dkim}
              </code>
            </div>
            <div className="text-sm text-gray-500 mt-4">
              Add these records to your DNS provider to enable mail authentication.
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-center h-32">
            <Spinner />
          </div>
        )}
      </Modal>
    </div>
  );
}