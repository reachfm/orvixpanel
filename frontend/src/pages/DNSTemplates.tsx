/**
 * DNS Templates page.
 *
 * Routes:
 *   /dns/zones                    ZonesListPage
 *   /dns/zones/:id                ZoneDetailPage
 *   /dns/templates                DNSTemplatesPage
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import { dnsTemplateKeys } from "@/lib/query/keys";
import {
  listTemplates,
  createTemplate,
  deleteTemplate,
  type DNSZoneTemplate,
  type TemplateRecordDefinition,
} from "@/lib/api/dns";

const RECORD_TYPES = ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"];

interface TemplateRecordInput {
  name: string;
  type: string;
  content: string;
  ttl: string;
  priority: string;
}

export function DNSTemplatesPage() {
  const qc = useQueryClient();

  // State
  const [searchQuery, setSearchQuery] = useState("");
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [deleteModal, setDeleteModal] = useState<DNSZoneTemplate | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formRecords, setFormRecords] = useState<TemplateRecordInput[]>([
    { name: "", type: "A", content: "", ttl: "3600", priority: "0" },
  ]);

  // Query
  const q = useQuery({
    queryKey: dnsTemplateKeys.list(),
    queryFn: listTemplates,
  });

  // Mutations
  const createMutation = useMutation({
    mutationFn: (body: { name: string; description?: string; records: TemplateRecordDefinition[] }) =>
      createTemplate(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsTemplateKeys.all() });
      closeCreateModal();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTemplate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsTemplateKeys.all() });
      setDeleteModal(null);
    },
  });

  // Filter
  const filteredTemplates = useMemo(() => {
    if (!searchQuery) return q.data?.templates ?? [];
    const query = searchQuery.toLowerCase();
    return (q.data?.templates ?? []).filter(
      (t) =>
        t.name.toLowerCase().includes(query) ||
        (t.description && t.description.toLowerCase().includes(query))
    );
  }, [q.data?.templates, searchQuery]);

  // Form helpers
  const closeCreateModal = () => {
    setCreateModalOpen(false);
    setFormName("");
    setFormDescription("");
    setFormRecords([{ name: "", type: "A", content: "", ttl: "3600", priority: "0" }]);
  };

  const addRecord = () => {
    setFormRecords([...formRecords, { name: "", type: "A", content: "", ttl: "3600", priority: "0" }]);
  };

  const removeRecord = (index: number) => {
    setFormRecords(formRecords.filter((_, i) => i !== index));
  };

  const updateRecord = (index: number, field: keyof TemplateRecordInput, value: string) => {
    const updated = [...formRecords];
    updated[index] = { ...updated[index], [field]: value };
    setFormRecords(updated);
  };

  const handleCreate = () => {
    const records: TemplateRecordDefinition[] = formRecords
      .filter((r) => r.name.trim() && r.content.trim())
      .map((r) => ({
        name: r.name,
        type: r.type as TemplateRecordDefinition["type"],
        content: r.content,
        ttl: r.ttl ? parseInt(r.ttl, 10) : undefined,
        priority: r.priority ? parseInt(r.priority, 10) : undefined,
      }));

    createMutation.mutate({
      name: formName.trim(),
      description: formDescription.trim() || undefined,
      records,
    });
  };

  // Table columns
  const columns: Column<DNSZoneTemplate>[] = [
    {
      key: "name",
      header: "Name",
      cell: (t) => <span className="font-medium text-ink-1">{t.name}</span>,
    },
    {
      key: "description",
      header: "Description",
      cell: (t) => (
        <span className="text-gray-500">{t.description || "—"}</span>
      ),
    },
    {
      key: "records_count",
      header: "Records",
      cell: (t) => {
        try {
          const records = JSON.parse(t.records);
          return <span className="text-gray-500">{records.length} record(s)</span>;
        } catch {
          return <span className="text-gray-400">—</span>;
        }
      },
    },
    {
      key: "created_at",
      header: "Created",
      cell: (t) => new Date(t.created_at).toLocaleDateString(),
    },
    {
      key: "actions",
      header: "Actions",
      cell: (t) => (
        <Button
          variant="ghost"
          size="sm"
          className="text-red-600 hover:text-red-800 dark:text-red-400"
          onClick={() => setDeleteModal(t)}
        >
          Delete
        </Button>
      ),
    },
  ];

  if (q.isLoading) return <LoadingState />;
  if (q.isError) return <ErrorState description="Failed to load templates" onRetry={() => q.refetch()} />;

  return (
    <div className="space-y-6">
      <PageHeader
        title="DNS Templates"
        description="Reusable DNS record templates for quick zone setup"
        actions={
          <Button onClick={() => setCreateModalOpen(true)}>Create Template</Button>
        }
      />

      {/* Search */}
      {filteredTemplates.length > 0 && (
        <Card className="p-4">
          <div className="max-w-sm">
            <Input
              placeholder="Search templates..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </Card>
      )}

      {/* Table */}
      {filteredTemplates.length === 0 ? (
        <EmptyState
          title="No templates found"
          description={
            searchQuery
              ? "Try adjusting your search"
              : "Create your first DNS template to speed up zone setup"
          }
          action={
            !searchQuery ? (
              <Button onClick={() => setCreateModalOpen(true)}>Create Template</Button>
            ) : undefined
          }
        />
      ) : (
        <Table<DNSZoneTemplate> rows={filteredTemplates} columns={columns} keyOf={(t) => t.id} />
      )}

      {/* Create Template Modal */}
      <Modal
        open={createModalOpen}
        onClose={closeCreateModal}
        title="Create DNS Template"
        width="lg"
        footer={
          <>
            <Button variant="outline" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!formName.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create Template"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Template Name"
            placeholder="My Template"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
          />
          <Input
            label="Description (optional)"
            placeholder="Brief description of this template"
            value={formDescription}
            onChange={(e) => setFormDescription(e.target.value)}
          />

          <div>
            <div className="mb-2 flex items-center justify-between">
              <label className="text-xs font-medium text-ink-2">Records</label>
              <Button variant="ghost" size="sm" onClick={addRecord}>
                + Add Record
              </Button>
            </div>

            <div className="space-y-3 max-h-80 overflow-y-auto">
              {formRecords.map((record, index) => (
                <div key={index} className="grid grid-cols-12 gap-2 items-end p-3 bg-surface-2 rounded-md">
                  <div className="col-span-3">
                    <Input
                      placeholder="Name"
                      value={record.name}
                      onChange={(e) => updateRecord(index, "name", e.target.value)}
                    />
                  </div>
                  <div className="col-span-2">
                    <select
                      value={record.type}
                      onChange={(e) => updateRecord(index, "type", e.target.value)}
                      className="w-full rounded-md border border-surface-border bg-surface-1 px-2 py-1.5 text-sm"
                    >
                      {RECORD_TYPES.map((t) => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                  </div>
                  <div className="col-span-4">
                    <Input
                      placeholder="Content"
                      value={record.content}
                      onChange={(e) => updateRecord(index, "content", e.target.value)}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      placeholder="TTL"
                      value={record.ttl}
                      onChange={(e) => updateRecord(index, "ttl", e.target.value)}
                    />
                  </div>
                  <div className="col-span-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-red-500"
                      onClick={() => removeRecord(index)}
                      disabled={formRecords.length === 1}
                    >
                      ×
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {createMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Failed to create template. Please check your input.
            </p>
          )}
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        title="Delete Template"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModal(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={deleteMutation.isPending}
              onClick={() => deleteModal && deleteMutation.mutate(deleteModal.id)}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete the template <strong>{deleteModal?.name}</strong>?
          This action cannot be undone.
        </p>
      </Modal>
    </div>
  );
}

// Re-export types
export type { DNSZoneTemplate } from "@/lib/api/dns";