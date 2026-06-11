/**
 * SSL Certificates list. Real data from the backend API.
 * Professional cPanel-style SSL certificate management.
 * Routes:
 *   /ssl/certificates              CertificatesListPage
 *   /ssl/certificates/:id          CertificateDetailPage
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, Spinner } from "@/lib/ui/Feedback";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/ui/cn";
import {
  sslKeys,
  listCertificates,
  issueCertificate,
  importCertificate,
  renewCertificate,
  deleteCertificate,
  issueStagingCertificate,
  type SSLCertificate,
  type SSLCertStatus,
  type IssueCertificateRequest,
  type SSLProvider,
  type IssueStagingCertificateResponse,
} from "@/lib/api/ssl";

const PAGE_SIZE = 20;

function getStatusTone(status: SSLCertStatus): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "issued":
      return "success";
    case "expiring_soon":
      return "warning";
    case "expired":
    case "failed":
    case "revoked":
      return "danger";
    default:
      return "neutral";
  }
}

function formatExpiry(dateStr?: string): string {
  if (!dateStr) return "—";
  const date = new Date(dateStr);
  const now = new Date();
  const diffDays = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
  if (diffDays < 0) return `Expired ${Math.abs(diffDays)}d ago`;
  if (diffDays === 0) return "Expires today";
  if (diffDays <= 30) return `Expires in ${diffDays}d`;
  return formatDate(dateStr);
}

export function CertificatesListPage() {
  const qc = useQueryClient();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modal state
  const [issueModalOpen, setIssueModalOpen] = useState(false);
  const [stagingModalOpen, setStagingModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [deleteModal, setDeleteModal] = useState<SSLCertificate | null>(null);
  const [renewModal, setRenewModal] = useState<SSLCertificate | null>(null);
  const [stagingResult, setStagingResult] = useState<IssueStagingCertificateResponse | null>(null);

  // Issue form state
  const [newCertDomain, setNewCertDomain] = useState("");
  const [newCertSANs, setNewCertSANs] = useState("");
  const [newCertProvider, setNewCertProvider] = useState<SSLProvider>("letsencrypt");
  const [newCertAutoRenew, setNewCertAutoRenew] = useState(true);

  // Staging form state
  const [stagingDomain, setStagingDomain] = useState("");
  const [stagingEmail, setStagingEmail] = useState("");
  const [stagingSANs, setStagingSANs] = useState("");

  // Import form state
  const [importDomain, setImportDomain] = useState("");
  const [importCertPEM, setImportCertPEM] = useState("");
  const [importKeyPEM, setImportKeyPEM] = useState("");
  const [importChainPEM, setImportChainPEM] = useState("");

  // Fetch certificates
  const q = useQuery({
    queryKey: sslKeys.certificates(),
    queryFn: listCertificates,
  });

  const certs = q.data ?? [];

  // Issue mutation
  const issueMutation = useMutation({
    mutationFn: (body: IssueCertificateRequest) =>
      issueCertificate(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: sslKeys.all });
      setIssueModalOpen(false);
      resetIssueForm();
    },
  });

  // Import mutation
  const importMutation = useMutation({
    mutationFn: (body: { domain: string; cert_pem: string; key_pem: string; chain_pem?: string }) =>
      importCertificate(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: sslKeys.all });
      setImportModalOpen(false);
      resetImportForm();
    },
  });

  // Staging mutation
  const stagingMutation = useMutation({
    mutationFn: (body: { domain: string; email: string; san_names?: string[] }) =>
      issueStagingCertificate(body),
    onSuccess: (data) => {
      setStagingResult(data);
      qc.invalidateQueries({ queryKey: sslKeys.all });
    },
  });

  // Renew mutation
  const renewMutation = useMutation({
    mutationFn: (id: string) => renewCertificate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: sslKeys.all });
      setRenewModal(null);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteCertificate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: sslKeys.all });
      setDeleteModal(null);
    },
  });

  // Filter certificates
  const filteredCerts = useMemo(() => {
    let result = certs;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter((c) => {
        const matchesDomain = c.common_name.toLowerCase().includes(query);
        const matchesSAN = c.san_names ? c.san_names.split(',').some((s) => s.trim().toLowerCase().includes(query)) : false;
        return matchesDomain || matchesSAN;
      });
    }

    if (statusFilter !== "all") {
      result = result.filter((c) => c.status === statusFilter);
    }

    return result;
  }, [certs, searchQuery, statusFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredCerts.length / PAGE_SIZE);
  const paginatedCerts = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredCerts.slice(start, start + PAGE_SIZE);
  }, [filteredCerts, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  // Form handlers
  const resetIssueForm = () => {
    setNewCertDomain("");
    setNewCertSANs("");
    setNewCertProvider("letsencrypt");
    setNewCertAutoRenew(true);
  };

  const resetImportForm = () => {
    setImportDomain("");
    setImportCertPEM("");
    setImportKeyPEM("");
    setImportChainPEM("");
  };

  const handleIssueCert = () => {
    if (!newCertDomain.trim()) return;
    issueMutation.mutate({
      domain: newCertDomain.trim(),
      san_names: newCertSANs ? newCertSANs.split(",").map((s) => s.trim()).filter(Boolean) : undefined,
      provider: newCertProvider,
      auto_renew: newCertAutoRenew,
    });
  };

  const handleImportCert = () => {
    if (!importDomain.trim() || !importCertPEM.trim() || !importKeyPEM.trim()) return;
    importMutation.mutate({
      domain: importDomain.trim(),
      cert_pem: importCertPEM.trim(),
      key_pem: importKeyPEM.trim(),
      chain_pem: importChainPEM.trim() || undefined,
    });
  };

  const handleIssueStagingCert = () => {
    if (!stagingDomain.trim() || !stagingEmail.trim()) return;
    stagingMutation.mutate({
      domain: stagingDomain.trim(),
      email: stagingEmail.trim(),
      san_names: stagingSANs ? stagingSANs.split(",").map((s) => s.trim()).filter(Boolean) : undefined,
    });
  };

  const resetStagingForm = () => {
    setStagingDomain("");
    setStagingEmail("");
    setStagingSANs("");
    setStagingResult(null);
  };

  const closeStagingModal = () => {
    setStagingModalOpen(false);
    resetStagingForm();
  };

  const isPending = issueMutation.isPending || importMutation.isPending || renewMutation.isPending || deleteMutation.isPending || stagingMutation.isPending;

  // Table columns
  const columns: Column<SSLCertificate>[] = [
    {
      key: "domain",
      header: "Domain",
      cell: (cert) => (
        <Link
          to="/ssl/certificates/$id"
          params={{ id: cert.id }}
          className="font-medium text-brand-600 hover:underline"
        >
          {cert.common_name}
        </Link>
      ),
    },
    {
      key: "provider",
      header: "Provider",
      cell: (cert) => (
        <span className="capitalize text-ink-2">{cert.provider}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (cert) => (
        <StatusPill tone={getStatusTone(cert.status)}>
          {cert.status.replace("_", " ")}
        </StatusPill>
      ),
    },
    {
      key: "expires",
      header: "Expires",
      cell: (cert) => (
        <span className="text-ink-2">{formatExpiry(cert.expires_at)}</span>
      ),
    },
    {
      key: "auto_renew",
      header: "Auto-Renew",
      cell: (cert) => (
        <span className={cert.auto_renew ? "text-success" : "text-ink-3"}>
          {cert.auto_renew ? "Yes" : "No"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (cert) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setRenewModal(cert)}
          >
            Renew
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setDeleteModal(cert)}
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
        title="SSL Certificates"
        description={`${certs.length} certificate${certs.length === 1 ? "" : "s"} managed`}
        actions={
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setImportModalOpen(true)}>
              Import
            </Button>
            <Button variant="secondary" onClick={() => setStagingModalOpen(true)}>
              Issue Staging
            </Button>
            <Button variant="primary" onClick={() => setIssueModalOpen(true)}>
              Issue New
            </Button>
          </div>
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
              placeholder="Search by domain..."
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
              <option value="issued">Active</option>
              <option value="expiring_soon">Expiring Soon</option>
              <option value="expired">Expired</option>
              <option value="pending">Pending</option>
              <option value="failed">Failed</option>
              <option value="revoked">Revoked</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState description="Failed to load certificates" onRetry={() => q.refetch()} />
        ) : q.isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size={24} />
          </div>
        ) : filteredCerts.length === 0 ? (
          <EmptyState
            title={searchQuery || statusFilter !== "all" ? "No certificates match your filters" : "No certificates yet"}
            description={
              searchQuery || statusFilter !== "all"
                ? "Try adjusting your search or filters."
                : "Issue your first SSL certificate to secure your domains."
            }
            action={
              !searchQuery && statusFilter === "all" && (
                <Button variant="primary" onClick={() => setIssueModalOpen(true)}>
                  Issue Certificate
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table
              rows={paginatedCerts}
              columns={columns}
              keyOf={(c) => c.id}
            />
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredCerts.length)} of{" "}
                  {filteredCerts.length} certificates
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

      {/* Issue Certificate Modal */}
      <Modal
        open={issueModalOpen}
        onClose={() => !isPending && setIssueModalOpen(false)}
        title="Issue New Certificate"
        description="Let's Encrypt and other ACME providers supported."
        footer={
          <>
            <Button variant="secondary" onClick={() => setIssueModalOpen(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleIssueCert}
              disabled={!newCertDomain.trim() || issueMutation.isPending}
              loading={issueMutation.isPending}
            >
              Issue Certificate
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Domain *"
            placeholder="example.com"
            value={newCertDomain}
            onChange={(e) => setNewCertDomain(e.target.value)}
          />
          <Input
            label="Additional Domains (SANs)"
            placeholder="www.example.com, api.example.com"
            value={newCertSANs}
            onChange={(e) => setNewCertSANs(e.target.value)}
            hint="Comma-separated list of additional domains"
          />
          <Select
            label="Provider"
            value={newCertProvider}
            onChange={(e) => setNewCertProvider(e.target.value as SSLProvider)}
          >
            <option value="letsencrypt">Let's Encrypt</option>
            <option value="zerossl">ZeroSSL</option>
          </Select>

          {/* Auto-renew toggle */}
          <div
            className={cn(
              "flex items-center gap-3 rounded-md border p-3 cursor-pointer",
              newCertAutoRenew
                ? "border-brand-500 bg-brand-500/5"
                : "border-surface-border hover:border-ink-3",
            )}
            onClick={() => setNewCertAutoRenew(!newCertAutoRenew)}
          >
            <div
              className={cn(
                "flex h-5 w-5 items-center justify-center rounded border",
                newCertAutoRenew
                  ? "border-brand-500 bg-brand-500 text-white"
                  : "border-surface-border bg-surface-1",
              )}
            >
              {newCertAutoRenew && (
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="h-3.5 w-3.5">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              )}
            </div>
            <div className="flex-1">
              <div className="text-sm font-medium text-ink-1">Enable auto-renewal</div>
              <div className="text-xs text-ink-3">Recommended — certificate will auto-renew before expiry</div>
            </div>
          </div>
        </div>
      </Modal>

      {/* Import Certificate Modal */}
      <Modal
        open={importModalOpen}
        onClose={() => !isPending && setImportModalOpen(false)}
        title="Import Certificate"
        description="Import an existing SSL certificate and private key."
        footer={
          <>
            <Button variant="secondary" onClick={() => setImportModalOpen(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleImportCert}
              disabled={!importDomain.trim() || !importCertPEM.trim() || !importKeyPEM.trim() || importMutation.isPending}
              loading={importMutation.isPending}
            >
              Import Certificate
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input
            label="Domain *"
            placeholder="example.com"
            value={importDomain}
            onChange={(e) => setImportDomain(e.target.value)}
          />
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1.5">Certificate (PEM) *</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500/20"
              placeholder="-----BEGIN CERTIFICATE-----"
              value={importCertPEM}
              onChange={(e) => setImportCertPEM(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1.5">Private Key (PEM) *</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500/20"
              placeholder="-----BEGIN PRIVATE KEY-----"
              value={importKeyPEM}
              onChange={(e) => setImportKeyPEM(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1.5">Certificate Chain (PEM)</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500/20"
              placeholder="-----BEGIN CERTIFICATE----- (optional)"
              value={importChainPEM}
              onChange={(e) => setImportChainPEM(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      {/* Issue Staging Certificate Modal */}
      <Modal
        open={stagingModalOpen}
        onClose={() => !isPending && closeStagingModal()}
        title="Issue Staging Certificate"
        description="Test SSL certificate issuance using Let's Encrypt STAGING."
        footer={
          stagingResult ? (
            <>
              <Button variant="secondary" onClick={closeStagingModal}>
                Close
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={closeStagingModal} disabled={isPending}>
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={handleIssueStagingCert}
                disabled={!stagingDomain.trim() || !stagingEmail.trim() || stagingMutation.isPending}
                loading={stagingMutation.isPending}
              >
                Issue Staging Certificate
              </Button>
            </>
          )
        }
      >
        {stagingResult ? (
          <div className="space-y-4">
            <div className="rounded-md border border-success/30 bg-success/5 p-4">
              <div className="flex items-center gap-2 text-success font-medium mb-2">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
                Staging Certificate Issued
              </div>
              <div className="text-sm text-ink-2">
                Certificate ID: <span className="font-mono text-ink-1">{stagingResult.certificate_id}</span>
              </div>
              {stagingResult.expires_at && (
                <div className="text-sm text-ink-2 mt-1">
                  Expires: <span className="text-ink-1">{new Date(stagingResult.expires_at).toLocaleDateString()}</span>
                </div>
              )}
            </div>
            {stagingResult.warnings && stagingResult.warnings.length > 0 && (
              <div className="rounded-md border border-warning/30 bg-warning/5 p-3">
                <div className="text-sm font-medium text-warning mb-2">Important Warnings</div>
                <ul className="text-sm text-ink-2 space-y-1">
                  {stagingResult.warnings.map((warning, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-4 w-4 text-warning mt-0.5 flex-shrink-0">
                        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                        <line x1="12" y1="9" x2="12" y2="13" />
                        <line x1="12" y1="17" x2="12.01" y2="17" />
                      </svg>
                      {warning}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <div className="text-xs text-ink-3">
              Staging certificates are NOT trusted by browsers. Use only for testing purposes.
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Staging warning banner */}
            <div className="rounded-md border border-warning/30 bg-warning/5 p-3">
              <div className="flex items-start gap-3">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5 text-warning flex-shrink-0 mt-0.5">
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <line x1="12" y1="9" x2="12" y2="13" />
                  <line x1="12" y1="17" x2="12.01" y2="17" />
                </svg>
                <div>
                  <div className="text-sm font-medium text-warning">Let's Encrypt Staging Environment</div>
                  <div className="text-xs text-ink-2 mt-1">
                    Certificates issued here are for <strong>testing only</strong>. They will show security warnings in browsers and are not suitable for production use.
                  </div>
                </div>
              </div>
            </div>

            <Input
              label="Domain *"
              placeholder="test.example.com"
              value={stagingDomain}
              onChange={(e) => setStagingDomain(e.target.value)}
            />
            <Input
              label="Email *"
              placeholder="admin@example.com"
              type="email"
              value={stagingEmail}
              onChange={(e) => setStagingEmail(e.target.value)}
              hint="Required for Let's Encrypt account registration"
            />
            <Input
              label="Additional Domains (SANs)"
              placeholder="www.test.example.com, api.test.example.com"
              value={stagingSANs}
              onChange={(e) => setStagingSANs(e.target.value)}
              hint="Comma-separated list of additional domains"
            />

            {stagingMutation.isError && (
              <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
                Failed to issue staging certificate: {stagingMutation.error?.message || "Unknown error"}
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* Renew Confirmation Modal */}
      <Modal
        open={renewModal !== null}
        onClose={() => !isPending && setRenewModal(null)}
        title="Renew Certificate"
        description={`Are you sure you want to renew the certificate for "${renewModal?.common_name}"?`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setRenewModal(null)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => renewModal && renewMutation.mutate(renewModal.id)}
              loading={renewMutation.isPending}
            >
              Renew Certificate
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-surface-border bg-surface-2 p-3 text-sm text-ink-2">
          The certificate will be renewed with the same provider and settings.
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => !isPending && setDeleteModal(null)}
        title="Delete Certificate"
        description={`Are you sure you want to delete the certificate for "${deleteModal?.common_name}"?`}
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
              Delete Certificate
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the certificate. Any sites using this cert will lose HTTPS.
        </div>
      </Modal>
    </div>
  );
}

// Re-export types
export type { SSLCertificate, SSLCertStatus } from "@/lib/api/ssl";