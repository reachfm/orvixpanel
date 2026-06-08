/**
 * DNS Zone detail page with records management.
 *
 * Routes:
 *   /dns/zones                    ZonesListPage
 *   /dns/zones/:id                ZoneDetailPage
 *   /dns/templates                DNSTemplatesPage
 */

import { useState, useMemo } from "react";
import { useParams, Link, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { Tabs } from "@/lib/ui/Tabs";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import { dnsZoneKeys, dnsTemplateKeys } from "@/lib/query/keys";
import {
  getZone,
  deleteZone,
  listRecords,
  createRecord,
  updateRecord,
  deleteRecord,
  listTemplates,
  applyTemplate,
  type DNSZone,
  type DNSRecord,
  type DNSRecordType,
} from "@/lib/api/dns";

const RECORD_TYPES: DNSRecordType[] = ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"];

const PAGE_SIZE = 50;

export function ZoneDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [tab, setTab] = useState("records");
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modals
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModal, setEditModal] = useState<DNSRecord | null>(null);
  const [deleteModal, setDeleteModal] = useState<DNSRecord | null>(null);
  const [applyTemplateModal, setApplyTemplateModal] = useState(false);

  // Form state
  const [formName, setFormName] = useState("");
  const [formType, setFormType] = useState<DNSRecordType>("A");
  const [formContent, setFormContent] = useState("");
  const [formTTL, setFormTTL] = useState("3600");
  const [formPriority, setFormPriority] = useState("0");
  const [formDisabled, setFormDisabled] = useState(false);

  // Zone query
  const zoneQuery = useQuery({
    queryKey: dnsZoneKeys.detail(id),
    queryFn: () => getZone(id),
  });

  // Records query
  const recordsQuery = useQuery({
    queryKey: dnsZoneKeys.records(id),
    queryFn: () => listRecords(id),
  });

  // Templates query (for apply template)
  const templatesQuery = useQuery({
    queryKey: dnsTemplateKeys.list(),
    queryFn: listTemplates,
  });

  // Mutations
  const createMutation = useMutation({
    mutationFn: (body: { name: string; type: DNSRecordType; content: string; ttl?: number; priority?: number; disabled?: boolean }) =>
      createRecord(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.records(id) });
      closeModals();
    },
  });

  const updateMutation = useMutation({
    mutationFn: (body: { name: string; type: DNSRecordType; content: string; ttl?: number; priority?: number; disabled?: boolean }) =>
      updateRecord(id, editModal!.id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.records(id) });
      closeModals();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteRecord(id, deleteModal!.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.records(id) });
      setDeleteModal(null);
    },
  });

  const applyTemplateMutation = useMutation({
    mutationFn: (templateId: string) => applyTemplate(templateId, { zone_id: id }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.records(id) });
      setApplyTemplateModal(false);
    },
  });

  const deleteZoneMutation = useMutation({
    mutationFn: () => deleteZone(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: dnsZoneKeys.all() });
      navigate({ to: "/dns/zones" });
    },
  });

  const records = recordsQuery.data?.records ?? [];

  // Filter records
  const filteredRecords = useMemo(() => {
    let result = records;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (r) =>
          r.name.toLowerCase().includes(query) ||
          r.content.toLowerCase().includes(query)
      );
    }

    if (typeFilter !== "all") {
      result = result.filter((r) => r.type === typeFilter);
    }

    return result;
  }, [records, searchQuery, typeFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredRecords.length / PAGE_SIZE);
  const paginatedRecords = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredRecords.slice(start, start + PAGE_SIZE);
  }, [filteredRecords, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  // Modal helpers
  const openEditModal = (record: DNSRecord) => {
    setEditModal(record);
    setFormName(record.name);
    setFormType(record.type);
    setFormContent(record.content);
    setFormTTL(String(record.ttl));
    setFormPriority(String(record.priority));
    setFormDisabled(record.disabled);
  };

  const closeModals = () => {
    setCreateModalOpen(false);
    setEditModal(null);
    setFormName("");
    setFormType("A");
    setFormContent("");
    setFormTTL("3600");
    setFormPriority("0");
    setFormDisabled(false);
  };

  const handleSubmit = () => {
    const body = {
      name: formName,
      type: formType,
      content: formContent,
      ttl: formTTL ? parseInt(formTTL, 10) : undefined,
      priority: formPriority ? parseInt(formPriority, 10) : undefined,
      disabled: formDisabled,
    };

    if (editModal) {
      updateMutation.mutate(body);
    } else {
      createMutation.mutate(body);
    }
  };

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  if (zoneQuery.isLoading) return <LoadingState />;
  if (zoneQuery.isError || !zoneQuery.data) {
    return <ErrorState description="Failed to load zone" onRetry={() => zoneQuery.refetch()} />;
  }

  const zone = zoneQuery.data;

  // Table columns
  const columns: Column<DNSRecord>[] = [
    {
      key: "name",
      header: "Name",
      cell: (r) => (
        <span className={`font-mono text-sm ${r.disabled ? "line-through text-gray-400" : ""}`}>
          {r.name || "@"}
        </span>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (r) => (
        <span className="inline-flex items-center rounded bg-surface-2 px-2 py-0.5 text-xs font-medium text-ink-2">
          {r.type}
        </span>
      ),
    },
    {
      key: "content",
      header: "Content",
      cell: (r) => (
        <span className={`font-mono text-sm ${r.disabled ? "line-through text-gray-400" : ""}`}>
          {r.content}
        </span>
      ),
    },
    {
      key: "ttl",
      header: "TTL",
      cell: (r) => <span className="font-mono text-xs text-gray-500">{r.ttl}s</span>,
    },
    {
      key: "priority",
      header: "Priority",
      cell: (r) => <span className="font-mono text-xs text-gray-500">{r.priority || "—"}</span>,
    },
    {
      key: "disabled",
      header: "Status",
      cell: (r) =>
        r.disabled ? (
          <StatusPill tone="warning">Disabled</StatusPill>
        ) : (
          <StatusPill tone="success">Active</StatusPill>
        ),
    },
    {
      key: "actions",
      header: "Actions",
      cell: (r) => (
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => openEditModal(r)}>
            Edit
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-red-600 hover:text-red-800 dark:text-red-400"
            onClick={() => setDeleteModal(r)}
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
        title={
          <span className="flex items-center gap-3">
            <span>{zone.domain}</span>
            <StatusPill
              tone={
                zone.status === "active"
                  ? "success"
                  : zone.status === "suspended"
                  ? "warning"
                  : "neutral"
              }
            >
              {zone.status}
            </StatusPill>
          </span>
        }
        description={
          <span>
            <Link to="/dns/zones" className="text-brand-600 hover:underline">
              DNS Zones
            </Link>
            <span className="mx-1.5 text-ink-4">/</span>
            <span className="font-mono text-xs">{zone.id}</span>
          </span>
        }
        actions={
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={() => setApplyTemplateModal(true)}>
              Apply Template
            </Button>
            <Button
              variant="danger"
              loading={deleteZoneMutation.isPending}
              onClick={() => {
                if (window.confirm(`Delete zone "${zone.domain}"? This will remove all records.`)) {
                  deleteZoneMutation.mutate();
                }
              }}
            >
              Delete Zone
            </Button>
          </div>
        }
      />

      <Tabs
        active={tab}
        onChange={setTab}
        tabs={[
          {
            key: "records",
            label: `Records (${records.length})`,
            panel: (
              <RecordsTab
                columns={columns}
                paginatedRecords={paginatedRecords}
                filteredRecords={filteredRecords}
                searchQuery={searchQuery}
                typeFilter={typeFilter}
                currentPage={currentPage}
                totalPages={totalPages}
                onSearchChange={handleSearchChange}
                onTypeFilterChange={(v) => {
                  setTypeFilter(v);
                  setCurrentPage(1);
                }}
                onPageChange={setCurrentPage}
                onAddRecord={() => setCreateModalOpen(true)}
              />
            ),
          },
          {
            key: "settings",
            label: "Settings",
            panel: <ZoneSettingsTab zone={zone} />,
          },
        ]}
      />

      {/* Create Record Modal */}
      <Modal
        open={createModalOpen}
        onClose={closeModals}
        title="Add DNS Record"
        footer={
          <>
            <Button variant="outline" onClick={closeModals}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!formName.trim() || !formContent.trim() || isSubmitting}
            >
              {isSubmitting ? "Adding..." : "Add Record"}
            </Button>
          </>
        }
      >
        <RecordForm
          name={formName}
          type={formType}
          content={formContent}
          ttl={formTTL}
          priority={formPriority}
          disabled={formDisabled}
          onNameChange={setFormName}
          onTypeChange={(v) => setFormType(v as DNSRecordType)}
          onContentChange={setFormContent}
          onTTLChange={setFormTTL}
          onPriorityChange={setFormPriority}
          onDisabledChange={setFormDisabled}
          error={createMutation.isError ? "Failed to create record" : undefined}
        />
      </Modal>

      {/* Edit Record Modal */}
      <Modal
        open={editModal !== null}
        onClose={closeModals}
        title="Edit DNS Record"
        footer={
          <>
            <Button variant="outline" onClick={closeModals}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!formName.trim() || !formContent.trim() || isSubmitting}
            >
              {isSubmitting ? "Saving..." : "Save Changes"}
            </Button>
          </>
        }
      >
        <RecordForm
          name={formName}
          type={formType}
          content={formContent}
          ttl={formTTL}
          priority={formPriority}
          disabled={formDisabled}
          onNameChange={setFormName}
          onTypeChange={(v) => setFormType(v as DNSRecordType)}
          onContentChange={setFormContent}
          onTTLChange={setFormTTL}
          onPriorityChange={setFormPriority}
          onDisabledChange={setFormDisabled}
          error={updateMutation.isError ? "Failed to update record" : undefined}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        title="Delete DNS Record"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModal(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={deleteMutation.isPending}
              onClick={() => deleteMutation.mutate()}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete the record{" "}
          <strong>
            {deleteModal?.name || "@"} {deleteModal?.type}
          </strong>
          ? This action cannot be undone.
        </p>
      </Modal>

      {/* Apply Template Modal */}
      <Modal
        open={applyTemplateModal}
        onClose={() => setApplyTemplateModal(false)}
        title="Apply Template"
        footer={
          <>
            <Button variant="outline" onClick={() => setApplyTemplateModal(false)}>
              Cancel
            </Button>
          </>
        }
      >
        <ApplyTemplateForm
          templates={templatesQuery.data?.templates ?? []}
          isLoading={templatesQuery.isLoading}
          onApply={(templateId) => applyTemplateMutation.mutate(templateId)}
          isApplying={applyTemplateMutation.isPending}
        />
      </Modal>
    </div>
  );
}

// Records tab component
function RecordsTab({
  columns,
  paginatedRecords,
  filteredRecords,
  searchQuery,
  typeFilter,
  currentPage,
  totalPages,
  onSearchChange,
  onTypeFilterChange,
  onPageChange,
  onAddRecord,
}: {
  columns: Column<DNSRecord>[];
  paginatedRecords: DNSRecord[];
  filteredRecords: DNSRecord[];
  searchQuery: string;
  typeFilter: string;
  currentPage: number;
  totalPages: number;
  onSearchChange: (value: string) => void;
  onTypeFilterChange: (value: string) => void;
  onPageChange: (page: number) => void;
  onAddRecord: () => void;
}) {
  return (
    <div className="space-y-4">
      {/* Filters */}
      <Card className="p-4">
        <div className="flex flex-wrap gap-4">
          <div className="flex-1 min-w-[200px]">
            <Input
              placeholder="Search records..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </div>
          <Select
            value={typeFilter}
            onChange={(e) => onTypeFilterChange(e.target.value)}
            className="w-32"
          >
            <option value="all">All Types</option>
            {RECORD_TYPES.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
          <Button onClick={onAddRecord}>Add Record</Button>
        </div>
      </Card>

      {/* Table */}
      {paginatedRecords.length === 0 ? (
        <EmptyState
          title="No records found"
          description={
            searchQuery || typeFilter !== "all"
              ? "Try adjusting your filters"
              : "Add your first DNS record to get started"
          }
          action={!searchQuery && typeFilter === "all" ? <Button onClick={onAddRecord}>Add Record</Button> : undefined}
        />
      ) : (
        <>
          <Table<DNSRecord> rows={paginatedRecords} columns={columns} keyOf={(r) => r.id} />
          {totalPages > 1 && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === 1}
                onClick={() => onPageChange(currentPage - 1)}
              >
                Previous
              </Button>
              <span className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
                Page {currentPage} of {totalPages} ({filteredRecords.length} total)
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === totalPages}
                onClick={() => onPageChange(currentPage + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// Zone settings tab component
function ZoneSettingsTab({ zone }: { zone: DNSZone }) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <Card>
        <CardHeader title="Zone Information" />
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <Field label="Domain" value={zone.domain} />
          <Field label="Type" value={zone.type} />
          <Field label="Status" value={zone.status} />
          <Field label="Created" value={new Date(zone.created_at).toLocaleString()} />
          <Field label="Updated" value={new Date(zone.updated_at).toLocaleString()} />
        </dl>
      </Card>
      <Card>
        <CardHeader title="SOA Settings" />
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <Field label="Refresh" value={`${zone.soa_refresh}s`} />
          <Field label="Retry" value={`${zone.soa_retry}s`} />
          <Field label="Expire" value={`${zone.soa_expire}s`} />
          <Field label="Minimum" value={`${zone.soa_minimum}s`} />
        </dl>
      </Card>
    </div>
  );
}

// Field helper
function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}

// Record form component
function RecordForm({
  name,
  type,
  content,
  ttl,
  priority,
  disabled,
  onNameChange,
  onTypeChange,
  onContentChange,
  onTTLChange,
  onPriorityChange,
  onDisabledChange,
  error,
}: {
  name: string;
  type: DNSRecordType;
  content: string;
  ttl: string;
  priority: string;
  disabled: boolean;
  onNameChange: (value: string) => void;
  onTypeChange: (value: string) => void;
  onContentChange: (value: string) => void;
  onTTLChange: (value: string) => void;
  onPriorityChange: (value: string) => void;
  onDisabledChange: (value: boolean) => void;
  error?: string;
}) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Name"
          placeholder="@ or www"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          hint="Use @ for root domain"
        />
        <Select
          label="Type"
          value={type}
          onChange={(e) => onTypeChange(e.target.value)}
        >
          {RECORD_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </Select>
      </div>
      <Input
        label="Content"
        placeholder={
          type === "A"
            ? "192.0.2.1"
            : type === "AAAA"
            ? "2001:db8::1"
            : type === "MX"
            ? "10 mail.example.com"
            : type === "CNAME"
            ? "example.com"
            : type === "TXT"
            ? "v=spf1 mx ~all"
            : "Enter value"
        }
        value={content}
        onChange={(e) => onContentChange(e.target.value)}
      />
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="TTL"
          type="number"
          placeholder="3600"
          value={ttl}
          onChange={(e) => onTTLChange(e.target.value)}
          hint="Time to live in seconds"
        />
        <Input
          label="Priority"
          type="number"
          placeholder="0"
          value={priority}
          onChange={(e) => onPriorityChange(e.target.value)}
          hint="For MX, SRV records"
        />
      </div>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={disabled}
          onChange={(e) => onDisabledChange(e.target.checked)}
          className="rounded border-surface-border"
        />
        <span>Disabled</span>
      </label>
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  );
}

// Apply template form component
function ApplyTemplateForm({
  templates,
  isLoading,
  onApply,
  isApplying,
}: {
  templates: { id: string; name: string; description?: string }[];
  isLoading: boolean;
  onApply: (templateId: string) => void;
  isApplying: boolean;
}) {
  const [selectedTemplate, setSelectedTemplate] = useState<string>("");

  if (isLoading) {
    return <p className="text-sm text-gray-500">Loading templates...</p>;
  }

  if (templates.length === 0) {
    return (
      <div>
        <p className="text-sm text-gray-500 mb-4">No templates available.</p>
        <Link to="/dns/templates" className="text-brand-600 hover:underline text-sm">
          Create a template first
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Select
        label="Select Template"
        value={selectedTemplate}
        onChange={(e) => setSelectedTemplate(e.target.value)}
      >
        <option value="">Choose a template...</option>
        {templates.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </Select>
      {selectedTemplate && (
        <div className="rounded-md bg-surface-2 p-3">
          <p className="text-xs text-gray-500">
            {templates.find((t) => t.id === selectedTemplate)?.description || "No description"}
          </p>
        </div>
      )}
      <Button
        onClick={() => selectedTemplate && onApply(selectedTemplate)}
        disabled={!selectedTemplate || isApplying}
        className="w-full"
      >
        {isApplying ? "Applying..." : "Apply Template"}
      </Button>
    </div>
  );
}

// Re-export types
export type { DNSZone, DNSRecord } from "@/lib/api/dns";