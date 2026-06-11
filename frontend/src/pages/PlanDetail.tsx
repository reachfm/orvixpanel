/**
 * Plan detail. Shows plan information with edit functionality.
 * Allows activating, deactivating, and deleting the plan.
 */

import { useState, useMemo } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Modal } from "@/lib/ui/Modal";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { planKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import { getPlan, updatePlan, activatePlan, deactivatePlan, deletePlan } from "@/lib/api/plans";

const FEATURE_OPTIONS = [
  { value: "backup", label: "Automated Backups" },
  { value: "staging", label: "Staging Environments" },
  { value: "malware_scan", label: "Malware Scanning" },
  { value: "cdn", label: "CDN Integration" },
  { value: "git_deploy", label: "Git Deployment" },
  { value: "ssl_auto", label: "Auto SSL" },
  { value: "email", label: "Email Accounts" },
  { value: "database", label: "Database Support" },
];

export function PlanDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [editMode, setEditMode] = useState(false);
  const [deleteModal, setDeleteModal] = useState(false);

  // Form state (for edit mode)
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [monthlyPrice, setMonthlyPrice] = useState(0);
  const [diskQuotaMB, setDiskQuotaMB] = useState(0);
  const [bandwidthGB, setBandwidthGB] = useState(0);
  const [maxDomains, setMaxDomains] = useState(0);
  const [maxUsers, setMaxUsers] = useState(0);
  const [maxSSL, setMaxSSL] = useState(0);
  const [features, setFeatures] = useState<string[]>([]);
  const [isActive, setIsActive] = useState(false);
  const [isDefault, setIsDefault] = useState(false);

  const q = useQuery({
    queryKey: planKeys.detail(id),
    queryFn: () => getPlan(id),
  });

  // Sync form state with fetched data
  useMemo(() => {
    if (q.data) {
      setName(q.data.name);
      setDisplayName(q.data.display_name);
      setDescription(q.data.description);
      setMonthlyPrice(q.data.monthly_price);
      setDiskQuotaMB(q.data.disk_quota_mb);
      setBandwidthGB(q.data.bandwidth_gb);
      setMaxDomains(q.data.max_domains);
      setMaxUsers(q.data.max_users);
      setMaxSSL(q.data.max_ssl);
      setFeatures(q.data.features);
      setIsActive(q.data.is_active);
      setIsDefault(q.data.is_default);
    }
  }, [q.data]);

  const update = useMutation({
    mutationFn: () => updatePlan(id, {
      name,
      display_name: displayName,
      description,
      monthly_price: monthlyPrice,
      disk_quota_mb: diskQuotaMB,
      bandwidth_gb: bandwidthGB,
      max_domains: maxDomains,
      max_users: maxUsers,
      max_ssl: maxSSL,
      features,
      is_active: isActive,
      is_default: isDefault,
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      setEditMode(false);
    },
  });

  const activate = useMutation({
    mutationFn: () => activatePlan(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: planKeys.all() }),
  });

  const deactivate = useMutation({
    mutationFn: () => deactivatePlan(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: planKeys.all() }),
  });

  const del = useMutation({
    mutationFn: () => deletePlan(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      navigate({ to: "/hosting/plans" });
    },
  });

  const isActionPending = update.isPending || activate.isPending || deactivate.isPending || del.isPending;

  const toggleFeature = (feature: string) => {
    setFeatures(prev =>
      prev.includes(feature)
        ? prev.filter(f => f !== feature)
        : [...prev, feature]
    );
  };

  if (q.isLoading) {
    return <div className="flex min-h-[40vh] items-center justify-center"><Spinner size={28} /></div>;
  }
  if (q.isError || !q.data) {
    return <ErrorState description="Failed to load plan." onRetry={() => q.refetch()} />;
  }

  const p = q.data;
  const actions = editMode ? (
    <>
      <Button variant="ghost" onClick={() => setEditMode(false)} disabled={isActionPending}>
        Cancel
      </Button>
      <Button variant="primary" loading={update.isPending} onClick={() => update.mutate()}>
        Save changes
      </Button>
    </>
  ) : (
    <>
      {p.is_active ? (
        <Button variant="secondary" loading={deactivate.isPending} onClick={() => deactivate.mutate()} disabled={p.is_default}>
          Deactivate
        </Button>
      ) : (
        <Button variant="secondary" loading={activate.isPending} onClick={() => activate.mutate()}>
          Activate
        </Button>
      )}
      <Button variant="secondary" onClick={() => setEditMode(true)}>
        Edit
      </Button>
      {!p.is_default && (
        <Button
          variant="danger"
          loading={del.isPending}
          onClick={() => setDeleteModal(true)}
        >
          Delete
        </Button>
      )}
    </>
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title={
          <span className="flex items-center gap-3">
            <span>{p.display_name || p.name}</span>
            <StatusPill tone={p.is_active ? "success" : "neutral"}>
              {p.is_active ? "Active" : "Inactive"}
            </StatusPill>
            {p.is_default && (
              <span className="rounded bg-brand-500/10 px-2 py-0.5 text-xs font-medium text-brand-600">
                Default
              </span>
            )}
          </span>
        }
        description={`Plan ${p.name} · Created ${formatDate(p.created_at)}`}
        actions={actions}
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader
              title="Plan details"
              description={editMode ? "Edit the plan settings below." : "Resource limits and pricing configuration."}
            />
            <div className="p-6">
              {editMode ? (
                <form onSubmit={(e) => { e.preventDefault(); update.mutate(); }} className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <Input
                    label="Plan name"
                    value={name}
                    onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                    hint="Lowercase letters, digits, and dashes only."
                  />
                  <Input
                    label="Display name"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                  />
                  <div className="md:col-span-2">
                    <label className="text-xs font-medium text-ink-2">Description</label>
                    <textarea
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      rows={2}
                      className="mt-1.5 w-full rounded-md border border-surface-border bg-surface-1 px-3 py-2 text-sm placeholder:text-ink-4 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                    />
                  </div>
                  <Input
                    label="Monthly price (USD)"
                    type="number"
                    min={0}
                    step={0.01}
                    value={monthlyPrice}
                    onChange={(e) => setMonthlyPrice(parseFloat(e.target.value || "0"))}
                  />
                  <Input
                    label="Disk quota (MB)"
                    type="number"
                    min={0}
                    value={diskQuotaMB}
                    onChange={(e) => setDiskQuotaMB(parseInt(e.target.value || "0", 10))}
                  />
                  <Input
                    label="Bandwidth (GB/month)"
                    type="number"
                    min={0}
                    value={bandwidthGB}
                    onChange={(e) => setBandwidthGB(parseInt(e.target.value || "0", 10))}
                  />
                  <Input
                    label="Max domains"
                    type="number"
                    min={0}
                    value={maxDomains}
                    onChange={(e) => setMaxDomains(parseInt(e.target.value || "0", 10))}
                  />
                  <Input
                    label="Max users"
                    type="number"
                    min={0}
                    value={maxUsers}
                    onChange={(e) => setMaxUsers(parseInt(e.target.value || "0", 10))}
                  />
                  <Input
                    label="Max SSL certificates"
                    type="number"
                    min={0}
                    value={maxSSL}
                    onChange={(e) => setMaxSSL(parseInt(e.target.value || "0", 10))}
                  />

                  {/* Features */}
                  <div className="md:col-span-2">
                    <label className="text-xs font-medium text-ink-2">Features</label>
                    <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4">
                      {FEATURE_OPTIONS.map((opt) => (
                        <label
                          key={opt.value}
                          className={`flex items-center gap-2 rounded-md border px-3 py-2 text-sm cursor-pointer transition-all ${
                            features.includes(opt.value)
                              ? "border-brand-500 bg-brand-500/5 text-brand-600"
                              : "border-surface-border bg-surface-1 text-ink-2 hover:bg-surface-2"
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={features.includes(opt.value)}
                            onChange={() => toggleFeature(opt.value)}
                            className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
                          />
                          {opt.label}
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Status toggles */}
                  <div className="md:col-span-2 flex gap-6">
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={isActive}
                        onChange={(e) => setIsActive(e.target.checked)}
                        className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
                      />
                      <span>Active</span>
                    </label>
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={isDefault}
                        onChange={(e) => setIsDefault(e.target.checked)}
                        disabled={p.is_default}
                        className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500 disabled:opacity-50"
                      />
                      <span>Default plan</span>
                    </label>
                  </div>
                </form>
              ) : (
                <dl className="grid grid-cols-2 gap-x-8 gap-y-4">
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Monthly price</dt>
                    <dd className="mt-1 font-mono text-lg font-semibold">${p.monthly_price.toFixed(2)}</dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Disk quota</dt>
                    <dd className="mt-1 font-mono text-sm">
                      {p.disk_quota_mb >= 1024
                        ? `${(p.disk_quota_mb / 1024).toFixed(1)} GB`
                        : `${p.disk_quota_mb} MB`}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Bandwidth</dt>
                    <dd className="mt-1 font-mono text-sm">{p.bandwidth_gb} GB/month</dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Max domains</dt>
                    <dd className="mt-1 font-mono text-sm">{p.max_domains}</dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Max users</dt>
                    <dd className="mt-1 font-mono text-sm">{p.max_users}</dd>
                  </div>
                  <div>
                    <dt className="text-xs font-medium text-ink-3">Max SSL certs</dt>
                    <dd className="mt-1 font-mono text-sm">{p.max_ssl}</dd>
                  </div>
                  {p.description && (
                    <div className="col-span-2">
                      <dt className="text-xs font-medium text-ink-3">Description</dt>
                      <dd className="mt-1 text-sm text-ink-2">{p.description}</dd>
                    </div>
                  )}
                </dl>
              )}
            </div>
          </Card>

          {/* Features card */}
          <Card>
            <CardHeader
              title="Included features"
              description="Features enabled for accounts on this plan."
            />
            <div className="p-6">
              {p.features.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {p.features.map((f) => {
                    const opt = FEATURE_OPTIONS.find(o => o.value === f);
                    return (
                      <span
                        key={f}
                        className="rounded-md border border-brand-500/30 bg-brand-500/5 px-3 py-1.5 text-sm font-medium text-brand-600"
                      >
                        {opt?.label || f}
                      </span>
                    );
                  })}
                </div>
              ) : (
                <p className="text-sm text-ink-3">No features enabled for this plan.</p>
              )}
            </div>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader title="Quick stats" />
            <div className="p-6 space-y-4">
              <div className="flex justify-between text-sm">
                <span className="text-ink-3">Plan ID</span>
                <span className="font-mono text-xs">{p.id}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-ink-3">Status</span>
                <StatusPill tone={p.is_active ? "success" : "neutral"}>
                  {p.is_active ? "Active" : "Inactive"}
                </StatusPill>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-ink-3">Default</span>
                <span>{p.is_default ? "Yes" : "No"}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-ink-3">Created</span>
                <span className="font-mono text-xs">{formatDate(p.created_at)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-ink-3">Updated</span>
                <span className="font-mono text-xs">{formatDate(p.updated_at)}</span>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal
        open={deleteModal}
        onClose={() => !isActionPending && setDeleteModal(false)}
        title="Delete Plan"
        description={`Are you sure you want to delete "${p.display_name || p.name}"? This action cannot be undone.`}
        width="md"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleteModal(false)} disabled={isActionPending}>
              Cancel
            </Button>
            <Button variant="danger" loading={del.isPending} onClick={() => del.mutate()}>
              Delete plan
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This action is irreversible. Accounts using this plan will not be deleted, but they will need to be reassigned to a different plan.
        </div>
      </Modal>
    </div>
  );
}