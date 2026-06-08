/**
 * DNS Zones list. Real data from the backend API.
 * Professional cPanel-style DNS zone management.
 * Routes:
 *   /dns/zones                    ZonesListPage
 *   /dns/zones/:id                ZoneDetailPage
 *   /dns/templates                DNSTemplatesPage
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, Spinner } from "@/lib/ui/Feedback";
import { dnsZoneKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import {
  listZones,
  createZone,
  deleteZone,
  type DNSZone,
  type DNSZoneType,
} from "@/lib/api/dns";

const PAGE_SIZE = 20;

export function ZonesListPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modal state
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [deleteModal, setDeleteModal] = useState<DNSZone | null>(null);
  const [newZoneDomain, setNewZoneDomain] = useState("");
  const [newZoneType, setNewZoneType] = useState<DNSZoneType>("native");

  const q = useQuery({
    queryKey: dnsZoneKeys.list(),
    queryFn: listZones,
  });

  const zones = q.data?.zones ?? [];

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (body: { domain: string; type?: DNSZoneType }) => createZone(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.all() });
      setCreateModalOpen(false);
      setNewZoneDomain("");
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteZone(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.all() });
      setDeleteModal(null);
    },
  });

  // Filter zones
  const filteredZones = useMemo(() => {
    let result = zones;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter((z) => z.domain.toLowerCase().includes(query));
    }

    if (statusFilter !== "all") {
      result = result.filter((z) => z.status === statusFilter);
    }

    return result;
  }, [zones, searchQuery, statusFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredZones.length / PAGE_SIZE);
  const paginatedZones = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredZones.slice(start, start + PAGE_SIZE);
  }, [filteredZones, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  const handleCreateZone = () => {
    if (!newZoneDomain.trim()) return;
    createMutation.mutate({
      domain: newZoneDomain.trim(),
      type: newZoneType,
    });
  };

  const isPending = createMutation.isPending || deleteMutation.isPending;

  // Table columns
  const columns: Column<DNSZone>[] = [
    {
      key: "domain",
      header: "Domain",
      cell: (zone) => (
        <Link
          to="/dns/zones/$id"
          params={{ id: zone.id }}
          className="font-medium text-brand-600 hover:underline"
        >
          {zone.domain}
        </Link>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (zone) => (
        <span className="capitalize text-ink-2">{zone.type}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (zone) => (
        <StatusPill tone={zone.status === "active" ? "success" : zone.status === "suspended" ? "warning" : "neutral"}>
          {zone.status}
        </StatusPill>
      ),
    },
    {
      key: "records",
      header: "Records",
      cell: () => <span className="text-ink-3">—</span>,
    },
    {
      key: "created_at",
      header: "Created",
      cell: (zone) => <span className="font-mono text-xs text-ink-2">{formatDate(zone.created_at)}</span>,
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (zone) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate({ to: "/dns/zones/$id", params: { id: zone.id } })}
          >
            View
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setDeleteModal(zone)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="DNS Zones"
        description={`${zones.length} zone${zones.length === 1 ? "" : "s"} configured`}
        actions={
          <Button variant="primary" onClick={() => setCreateModalOpen(true)}>
            Create Zone
          </Button>
        }
      />

      {/* Filters */}
      <Card>
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1">
            <Input
              label="Search"
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search zones..."
            />
          </div>
          <div className="w-full sm:w-40">
            <Select
              label="Status"
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setCurrentPage(1);
              }}
            >
              <option value="all">All statuses</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="pending">Pending</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState description="Failed to load zones" onRetry={() => q.refetch()} />
        ) : q.isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size={24} />
          </div>
        ) : filteredZones.length === 0 ? (
          <EmptyState
            title={searchQuery || statusFilter !== "all" ? "No zones match your filters" : "No zones yet"}
            description={
              searchQuery || statusFilter !== "all"
                ? "Try adjusting your search or filters."
                : "Create your first DNS zone to get started."
            }
            action={
              !searchQuery && statusFilter === "all" && (
                <Button variant="primary" onClick={() => setCreateModalOpen(true)}>
                  Create Zone
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table
              rows={paginatedZones}
              columns={columns}
              keyOf={(z) => z.id}
            />
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredZones.length)} of{" "}
                  {filteredZones.length} zones
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={currentPage === 1}
                    onClick={() => setCurrentPage((p) => p - 1)}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={currentPage === totalPages}
                    onClick={() => setCurrentPage((p) => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      {/* Create Zone Modal */}
      <Modal
        open={createModalOpen}
        onClose={() => !isPending && setCreateModalOpen(false)}
        title="Create DNS Zone"
        description="Add a new DNS zone for domain management."
        footer={
          <>
            <Button variant="secondary" onClick={() => setCreateModalOpen(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleCreateZone}
              disabled={!newZoneDomain.trim() || createMutation.isPending}
              loading={createMutation.isPending}
            >
              Create Zone
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Domain"
            placeholder="example.com"
            value={newZoneDomain}
            onChange={(e) => setNewZoneDomain(e.target.value)}
            hint="Enter the domain name for the DNS zone"
          />
          <Select
            label="Type"
            value={newZoneType}
            onChange={(e) => setNewZoneType(e.target.value as DNSZoneType)}
          >
            <option value="native">Native</option>
            <option value="master">Master</option>
            <option value="slave">Slave</option>
          </Select>
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => !isPending && setDeleteModal(null)}
        title="Delete DNS Zone"
        description={`Are you sure you want to delete "${deleteModal?.domain}"? This action cannot be undone.`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleteModal(null)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteModal && deleteMutation.mutate(deleteModal.id)}
              loading={deleteMutation.isPending}
            >
              Delete Zone
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the zone and all associated DNS records.
        </div>
      </Modal>
    </div>
  );
}

// Re-export types for use in other components
export type { DNSZone } from "@/lib/api/dns";