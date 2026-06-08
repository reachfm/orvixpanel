/**
 * Settings page. The v0.3.0 enterprise UI ships a single
 * read-mostly Settings surface: the license status card. Operators
 * can also upload a new license key (PUT /api/v1/admin/license).
 *
 * Future modules (themes, API keys, RBAC roles, tenant quotas) plug
 * into this page as additional tabs without redesign.
 */

import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Badge } from "@/lib/ui/Badge";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Tabs } from "@/lib/ui/Tabs";
import { ErrorState, LoadingState } from "@/lib/ui/Feedback";
import { systemKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import { license, licenseRenewal, uploadLicense } from "@/lib/api/system";
import { ApiError } from "@/lib/api/client";
import { useAuthStore } from "@/lib/auth/store";

export function SettingsPage() {
  const [tab, setTab] = useState("license");
  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Operator configuration. Each tab is independent and reloadable."
      />
      <Tabs
        active={tab}
        onChange={setTab}
        tabs={[
          { key: "license", label: "License",       panel: <LicenseSettings /> },
          { key: "session", label: "Session",        panel: <SessionInfo /> },
        ]}
      />
    </div>
  );
}

function LicenseSettings() {
  const lic = useQuery({ queryKey: systemKeys.license(),         queryFn: license });
  const ren = useQuery({ queryKey: systemKeys.licenseRenewal(),  queryFn: licenseRenewal });
  const [key, setKey] = useState("");
  const qc = useQueryClient();

  const mut = useMutation({
    mutationFn: () => uploadLicense(key.trim()),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: systemKeys.license() });
      qc.invalidateQueries({ queryKey: systemKeys.licenseRenewal() });
      setKey("");
    },
  });

  if (lic.isLoading || ren.isLoading) return <LoadingState />;
  if (lic.isError || ren.isError) {
    return <ErrorState description="Failed to load license." onRetry={() => { lic.refetch(); ren.refetch(); }} />;
  }

  const errMsg =
    mut.error instanceof ApiError
      ? mut.error.code.startsWith("license_parse_failed:")
        ? `Invalid license key: ${mut.error.code.replace("license_parse_failed:", "")}`
        : mut.error.code
      : null;

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader
          title="License"
          description="From /api/v1/admin/license and /renewal-info."
          actions={
            ren.data && (
              <Badge tone={ren.data.status === "active" ? "success" : ren.data.status === "grace" ? "warning" : "danger"}>
                {ren.data.status}
              </Badge>
            )
          }
        />
        {lic.data && (
          <dl className="grid grid-cols-2 gap-3 text-sm">
            <Field label="Tier" value={lic.data.tier} />
            <Field label="Features" value={String(lic.data.features?.length ?? 0)} />
            <Field label="Max servers" value={String(lic.data.max_servers ?? "—")} />
            <Field label="Grace days" value={ren.data ? String(ren.data.grace_days) : "—"} />
            <Field label="Issued at" value={lic.data.issued_at ? formatDate(new Date(lic.data.issued_at * 1000).toISOString()) : "—"} />
            <Field label="Expires at" value={lic.data.expires_at ? formatDate(new Date(lic.data.expires_at * 1000).toISOString()) : "—"} />
            <Field label="Days remaining" value={ren.data ? String(ren.data.days_remaining) : "—"} />
          </dl>
        )}
      </Card>

      <Card>
        <CardHeader title="Upload new key" description="PUT /api/v1/admin/license" />
        <form
          onSubmit={(e: FormEvent) => { e.preventDefault(); mut.mutate(); }}
          className="space-y-3"
        >
          <Input
            label="License key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="ORVIX-ENT-2026-XXXX-XXXX"
            hint="Format: ORVIX-{TIER}-{YEAR}-{HASH}-{SIG}"
          />
          {errMsg && (
            <div className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
              {errMsg}
            </div>
          )}
          <Button type="submit" variant="primary" loading={mut.isPending} disabled={!key.trim()}>
            Upload
          </Button>
        </form>
      </Card>
    </div>
  );
}

function SessionInfo() {
  const user = useAuthStore((s) => s.user);
  return (
    <Card>
      <CardHeader title="Session" description="Current operator. v0.3.0 has no other session settings." />
      <dl className="grid grid-cols-2 gap-3 text-sm">
        <Field label="Email" value={user?.email ?? "—"} />
        <Field label="Role"  value={user?.role ?? "—"} />
      </dl>
      <p className="mt-4 text-xs text-ink-3">
        Sign out from the top-right user menu. Future: passkey setup, MFA enrollment, API key management.
      </p>
    </Card>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}
