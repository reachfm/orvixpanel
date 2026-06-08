/**
 * DNS Zones list. Real data from the backend API.
 *
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
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import { dnsZoneKeys } from "@/lib/query/keys";
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

  // Table columns
  const columns: Column<DNSZone>[] = [
    {
      key: "domain",
      header: "Domain",
      cell: (zone) => (
        <Link
          to="/dns/zones/$id"
          params={{ id: zone.id }}
          className="font-medium text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
        >
          {zone.domain}
        </Link>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (zone) => (
        <span className="capitalize text-gray-600 dark:text-gray-400">{zone.type}</span>
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
      cell: () => <span className="text-gray-500">—</span>,
    },
    {
      key: "created_at",
      header: "Created",
      cell: (zone) => new Date(zone.created_at).toLocaleDateString(),
    },
    {
      key: "actions",
      header: "Actions",
      cell: (zone) => (
        <div className="flex items-center gap-2">
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
            className="text-red-600 hover:text-red-800 dark:text-red-400"
            onClick={() => setDeleteModal(zone)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  if (q.isLoading) return <LoadingState />;
  if (q.isError) return <ErrorState description="Failed to load zones" onRetry={() => q.refetch()} />;

  return (
    <div className="space-y-6">
      <PageHeader
        title="DNS Zones"
        description="Manage your DNS zones and records"
        actions={
          <Button onClick={() => setCreateModalOpen(true)}>
            Create Zone
          </Button>
        }
      />

      {/* Filters */}
      <Card className="p-4">
        <div className="flex flex-wrap gap-4">
          <div className="flex-1 min-w-[200px]">
            <Input
              placeholder="Search zones..."
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
            />
          </div>
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setCurrentPage(1);
            }}
            className="w-40"
          >
            <option value="all">All Status</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="pending">Pending</option>
          </Select>
        </div>
      </Card>

      {/* Table */}
      {paginatedZones.length === 0 ? (
        <EmptyState
          title="No zones found"
          description={
            searchQuery || statusFilter !== "all"
              ? "Try adjusting your filters"
              : "Create your first DNS zone to get started"
          }
          action={
            !searchQuery && statusFilter === "all" ? (
              <Button onClick={() => setCreateModalOpen(true)}>Create Zone</Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Table<DNSZone> rows={paginatedZones} columns={columns} keyOf={(z) => z.id} />
          {totalPages > 1 && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === 1}
                onClick={() => setCurrentPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
                Page {currentPage} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === totalPages}
                onClick={() => setCurrentPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}

      {/* Create Zone Modal */}
      <Modal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Create DNS Zone"
        footer={
          <>
            <Button variant="outline" onClick={() => setCreateModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreateZone}
              disabled={!newZoneDomain.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Domain</label>
            <Input
              placeholder="example.com"
              value={newZoneDomain}
              onChange={(e) => setNewZoneDomain(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreateZone()}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Type</label>
            <Select
              value={newZoneType}
              onChange={(e) => setNewZoneType(e.target.value as DNSZoneType)}
            >
              <option value="native">Native</option>
              <option value="master">Master</option>
              <option value="slave">Slave</option>
            </Select>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Failed to create zone. Please check the domain name.
            </p>
          )}
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        title="Delete DNS Zone"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModal(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteModal && deleteMutation.mutate(deleteModal.id)}
              loading={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete the zone{" "}
          <strong>{deleteModal?.domain}</strong>? This action cannot be undone and
          will remove all records in this zone.
        </p>
      </Modal>
    </div>
  );
}

// Re-export types for use in other components
export type { DNSZone } from "@/lib/api/dns";