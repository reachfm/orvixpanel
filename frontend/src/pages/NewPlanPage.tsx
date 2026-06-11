/**
 * New plan form. Single page, single mutation. On success, redirect
 * to the new plan's detail page.
 */

import { useState, type FormEvent, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Input } from "@/lib/ui/Input";
import { Button } from "@/lib/ui/Button";
import { ApiError } from "@/lib/api/client";
import { createPlan } from "@/lib/api/plans";
import { planKeys } from "@/lib/query/keys";

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

const errorMap: Record<string, string> = {
  name_required: "Plan name is required.",
  name_already_exists: "A plan with this name already exists.",
  monthly_price_cannot_be_negative: "Monthly price cannot be negative.",
  invalid_body: "Invalid request. Please review your inputs.",
};

export function NewPlanPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Form state
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [monthlyPrice, setMonthlyPrice] = useState(9.99);
  const [diskQuotaMB, setDiskQuotaMB] = useState(10240);
  const [bandwidthGB, setBandwidthGB] = useState(100);
  const [maxDomains, setMaxDomains] = useState(5);
  const [maxUsers, setMaxUsers] = useState(10);
  const [maxSSL, setMaxSSL] = useState(3);
  const [features, setFeatures] = useState<string[]>([]);
  const [isActive, setIsActive] = useState(false);
  const [isDefault, setIsDefault] = useState(false);

  // Validation state
  const [touched, setTouched] = useState({ name: false, displayName: false });

  // Inline validation
  const nameError = useMemo(() => {
    if (!touched.name || name.length === 0) return null;
    if (name.length < 2) return "Name must be at least 2 characters";
    if (!/^[a-z0-9-]+$/.test(name)) return "Only lowercase letters, digits, and dashes allowed";
    return null;
  }, [name, touched.name]);

  const displayNameError = useMemo(() => {
    if (!touched.displayName || displayName.length === 0) return null;
    if (displayName.length < 3) return "Display name must be at least 3 characters";
    return null;
  }, [displayName, touched.displayName]);

  const isFormValid = useMemo(() => {
    return name.length >= 2 &&
      /^[a-z0-9-]+$/.test(name) &&
      displayName.length >= 3 &&
      monthlyPrice >= 0 &&
      diskQuotaMB >= 0 &&
      bandwidthGB >= 0;
  }, [name, displayName, monthlyPrice, diskQuotaMB, bandwidthGB]);

  // Mutation
  const mut = useMutation({
    mutationFn: () => createPlan({
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
    onSuccess: (plan) => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      navigate({ to: "/hosting/plans/$id", params: { id: plan.id } });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    setTouched({ name: true, displayName: true });
    if (!isFormValid) return;
    mut.mutate();
  };

  const errMsg = mut.error instanceof ApiError
    ? (errorMap[mut.error.code] ?? `Failed (${mut.error.code})`)
    : null;

  const toggleFeature = (feature: string) => {
    setFeatures(prev =>
      prev.includes(feature)
        ? prev.filter(f => f !== feature)
        : [...prev, feature]
    );
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="New plan"
        description="Create a hosting plan with resource limits and pricing."
      />

      <Card>
        <form onSubmit={onSubmit} className="grid max-w-2xl grid-cols-1 gap-6 md:grid-cols-2">
          {/* Name field */}
          <div>
            <Input
              label="Plan name"
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
              onBlur={() => setTouched(t => ({ ...t, name: true }))}
              placeholder="starter"
              required
              error={nameError ?? undefined}
              hint="Lowercase letters, digits, and dashes. Used as the internal identifier."
            />
            {name.length > 0 && !nameError && (
              <div className="mt-1.5 flex items-center gap-1.5 text-xs text-success">
                <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <span>Name format is valid</span>
              </div>
            )}
          </div>

          {/* Display name field */}
          <div>
            <Input
              label="Display name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              onBlur={() => setTouched(t => ({ ...t, displayName: true }))}
              placeholder="Starter Plan"
              required
              error={displayNameError ?? undefined}
              hint="Human-readable name shown to users."
            />
          </div>

          {/* Description field */}
          <div className="md:col-span-2">
            <label className="text-xs font-medium text-ink-2">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Perfect for personal projects and small websites."
              rows={3}
              className="mt-1.5 w-full rounded-md border border-surface-border bg-surface-1 px-3 py-2 text-sm placeholder:text-ink-4 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            />
          </div>

          {/* Price field */}
          <div>
            <Input
              label="Monthly price (USD)"
              type="number"
              min={0}
              step={0.01}
              value={monthlyPrice}
              onChange={(e) => setMonthlyPrice(parseFloat(e.target.value || "0"))}
              hint="Price per month in US dollars."
            />
          </div>

          <div className="md:col-span-2" />

          {/* Resource limits */}
          <div>
            <Input
              label="Disk quota (MB)"
              type="number"
              min={0}
              value={diskQuotaMB}
              onChange={(e) => setDiskQuotaMB(parseInt(e.target.value || "0", 10))}
            />
            <div className="mt-1 text-xs text-ink-3">
              {diskQuotaMB >= 1024 ? `~${(diskQuotaMB / 1024).toFixed(1)} GB` : `${diskQuotaMB} MB`}
            </div>
          </div>

          <div>
            <Input
              label="Bandwidth (GB/month)"
              type="number"
              min={0}
              value={bandwidthGB}
              onChange={(e) => setBandwidthGB(parseInt(e.target.value || "0", 10))}
            />
          </div>

          <div>
            <Input
              label="Max domains"
              type="number"
              min={0}
              value={maxDomains}
              onChange={(e) => setMaxDomains(parseInt(e.target.value || "0", 10))}
            />
          </div>

          <div>
            <Input
              label="Max users"
              type="number"
              min={0}
              value={maxUsers}
              onChange={(e) => setMaxUsers(parseInt(e.target.value || "0", 10))}
            />
          </div>

          <div>
            <Input
              label="Max SSL certificates"
              type="number"
              min={0}
              value={maxSSL}
              onChange={(e) => setMaxSSL(parseInt(e.target.value || "0", 10))}
            />
          </div>

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

          {/* Active/Default toggles */}
          <div className="md:col-span-2 flex gap-6">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
              />
              <span>Active (available for new accounts)</span>
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
              />
              <span>Default plan</span>
            </label>
          </div>

          {/* Error message */}
          {errMsg && (
            <div className="md:col-span-2 rounded-md border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger">
              {errMsg}
            </div>
          )}

          {/* Form actions */}
          <div className="md:col-span-2 flex items-center justify-between">
            <div className="text-xs text-ink-3">
              Review before creating — you can edit these settings later.
            </div>
            <div className="flex gap-2">
              <Button type="button" variant="ghost" onClick={() => navigate({ to: "/hosting/plans" })}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                loading={mut.isPending}
                disabled={!isFormValid}
              >
                Create plan
              </Button>
            </div>
          </div>
        </form>
      </Card>
    </div>
  );
}