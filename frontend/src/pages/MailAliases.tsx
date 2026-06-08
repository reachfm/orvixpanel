/**
 * Mail Aliases Page
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listAliases, listDomains, createAlias, deleteAlias } from "@/lib/api/mail";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { Modal } from "@/lib/ui/Modal";
import { Spinner } from "@/lib/ui/Feedback";
import { formatDate } from "@/lib/utils";

export function MailAliasesPage() {
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [sourceEmail, setSourceEmail] = useState("");
  const [destinations, setDestinations] = useState("");
  const [selectedDomain, setSelectedDomain] = useState("");

  // Query aliases
  const { data, isLoading, error } = useQuery({
    queryKey: ["mail", "aliases"],
    queryFn: () => listAliases({ page_size: 50 }),
  });

  // Query domains
  const { data: domainsData } = useQuery({
    queryKey: ["mail", "domains"],
    queryFn: () => listDomains({ page_size: 100 }),
  });

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
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAlias(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mail", "aliases"] });
    },
  });

  // Parse destinations for display
  const parseDestinations = (json: string): string[] => {
    try {
      return JSON.parse(json);
    } catch {
      return [];
    }
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
        Failed to load aliases. Please try again.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Mail Aliases</h1>
          <p className="text-gray-500">
            Create email aliases that forward to mailboxes or external addresses
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          Add Alias
        </Button>
      </div>

      {/* Alias List */}
      <Card>
        {data?.aliases.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <p className="text-lg">No aliases configured</p>
            <p className="text-sm">Add an alias to forward emails</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Source
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Destinations
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
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
                {data?.aliases.map((alias) => {
                  const destArray = parseDestinations(alias.destinations);
                  return (
                    <tr key={alias.id}>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="font-medium text-gray-900">{alias.source_email}</div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex flex-wrap gap-1">
                          {destArray.map((dest, i) => (
                            <Badge key={i}  tone="neutral">
                              {dest}
                            </Badge>
                          ))}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge
                          tone={alias.status === "active" ? "success" : "neutral"}
                        >
                          {alias.status}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(alias.created_at)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <Button
                          variant="ghost"
                          
                          onClick={() => {
                            if (confirm(`Delete alias ${alias.source_email}?`)) {
                              deleteMutation.mutate(alias.id);
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
        title="Add Alias"
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
              Source Email (Alias)
            </label>
            <Input
              type="text"
              value={sourceEmail}
              onChange={(e) => setSourceEmail(e.target.value)}
              placeholder="alias@domain.com"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Destinations (comma-separated)
            </label>
            <Input
              type="text"
              value={destinations}
              onChange={(e) => setDestinations(e.target.value)}
              placeholder="user1@example.com, user2@example.com"
              required
            />
            <p className="mt-1 text-sm text-gray-500">
              Emails to the alias will be forwarded to these addresses
            </p>
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
              {createMutation.isPending ? "Creating..." : "Create Alias"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}