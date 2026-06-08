/**
 * SSL Certificates list. Real data from the backend API.
 *
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
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import {
  sslKeys,
  listCertificates,
  issueCertificate,
  importCertificate,
  renewCertificate,
  deleteCertificate,
  type SSLCertificate,
  type SSLCertStatus,
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
  if (diffDays < 0) return `Expired ${Math.abs(diffDays)} days ago`;
  if (diffDays === 0) return "Expires today";
  if (diffDays <= 30) return `Expires in ${diffDays} days`;
  return date.toLocaleDateString();
}

export function CertificatesListPage() {
  const qc = useQueryClient();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modal state
  const [issueModalOpen, setIssueModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [deleteModal, setDeleteModal] = useState<SSLCertificate | null>(null);
  const [renewModal, setRenewModal] = useState<SSLCertificate | null>(null);

  // Issue form state
  const [newCertDomain, setNewCertDomain] = useState("");
  const [newCertSANs, setNewCertSANs] = useState("");
  const [newCertProvider, setNewCertProvider] = useState("letsencrypt");
  const [newCertAutoRenew, setNewCertAutoRenew] = useState(true);

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
    mutationFn: (body: { domain: string; san_names?: string[]; provider?: string; auto_renew?: boolean }) =>
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
      result = result.filter(
        (c) =>
          c.common_name.toLowerCase().includes(query) ||
          (c.san_names && c.san_names.some((s) => s.toLowerCase().includes(query)))
      );
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

  // Table columns
  const columns: Column<SSLCertificate>[] = [
    {
      key: "domain",
      header: "Domain",
      cell: (cert) => (
        <Link
          to="/ssl/certificates/$id"
          params={{ id: cert.id }}
          className="font-medium text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
        >
          {cert.common_name}
        </Link>
      ),
    },
    {
      key: "provider",
      header: "Provider",
      cell: (cert) => (
        <span className="capitalize text-gray-600 dark:text-gray-400">{cert.provider}</span>
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
        <span className="text-gray-600 dark:text-gray-400">{formatExpiry(cert.not_after)}</span>
      ),
    },
    {
      key: "auto_renew",
      header: "Auto-Renew",
      cell: (cert) => (
        <span className={cert.auto_renew ? "text-green-600" : "text-gray-400"}>
          {cert.auto_renew ? "Yes" : "No"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "Actions",
      cell: (cert) => (
        <div className="flex items-center gap-2">
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
            className="text-red-600 hover:text-red-800 dark:text-red-400"
            onClick={() => setDeleteModal(cert)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  if (q.isLoading) return <LoadingState />;
  if (q.isError) return <ErrorState description="Failed to load certificates" onRetry={() => q.refetch()} />;

  return (
    <div className="space-y-6">
      <PageHeader
        title="SSL Certificates"
        description="Manage SSL/TLS certificates with Let's Encrypt and other providers"
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setImportModalOpen(true)}>
              Import
            </Button>
            <Button onClick={() => setIssueModalOpen(true)}>Issue New</Button>
          </div>
        }
      />

      {/* Filters */}
      <Card className="p-4">
        <div className="flex flex-wrap gap-4">
          <div className="flex-1 min-w-[200px]">
            <Input
              placeholder="Search domains..."
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
            <option value="issued">Active</option>
            <option value="expiring_soon">Expiring Soon</option>
            <option value="expired">Expired</option>
            <option value="pending">Pending</option>
            <option value="failed">Failed</option>
            <option value="revoked">Revoked</option>
          </Select>
        </div>
      </Card>

      {/* Table */}
      {paginatedCerts.length === 0 ? (
        <EmptyState
          title="No certificates found"
          description={
            searchQuery || statusFilter !== "all"
              ? "Try adjusting your filters"
              : "Issue your first SSL certificate to get started"
          }
          action={
            !searchQuery && statusFilter === "all" ? (
              <Button onClick={() => setIssueModalOpen(true)}>Issue Certificate</Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Table<SSLCertificate> rows={paginatedCerts} columns={columns} keyOf={(c) => c.id} />
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

      {/* Issue Certificate Modal */}
      <Modal
        open={issueModalOpen}
        onClose={() => setIssueModalOpen(false)}
        title="Issue New Certificate"
        footer={
          <>
            <Button variant="outline" onClick={() => setIssueModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleIssueCert}
              disabled={!newCertDomain.trim() || issueMutation.isPending}
            >
              {issueMutation.isPending ? "Issuing..." : "Issue"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Domain *</label>
            <Input
              placeholder="example.com"
              value={newCertDomain}
              onChange={(e) => setNewCertDomain(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleIssueCert()}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Additional Domains (SANs)</label>
            <Input
              placeholder="www.example.com, api.example.com"
              value={newCertSANs}
              onChange={(e) => setNewCertSANs(e.target.value)}
            />
            <p className="mt-1 text-xs text-gray-500">Comma-separated list of additional domains</p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Provider</label>
            <Select
              value={newCertProvider}
              onChange={(e) => setNewCertProvider(e.target.value)}
            >
              <option value="letsencrypt">Let's Encrypt</option>
              <option value="zerossl">ZeroSSL</option>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="auto-renew"
              checked={newCertAutoRenew}
              onChange={(e) => setNewCertAutoRenew(e.target.checked)}
              className="rounded border-gray-300"
            />
            <label htmlFor="auto-renew" className="text-sm">Enable auto-renewal (recommended)</label>
          </div>
          {issueMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Failed to issue certificate. Please check the domain name and try again.
            </p>
          )}
        </div>
      </Modal>

      {/* Import Certificate Modal */}
      <Modal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        title="Import Certificate"
        footer={
          <>
            <Button variant="outline" onClick={() => setImportModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleImportCert}
              disabled={!importDomain.trim() || !importCertPEM.trim() || !importKeyPEM.trim() || importMutation.isPending}
            >
              {importMutation.isPending ? "Importing..." : "Import"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Domain *</label>
            <Input
              placeholder="example.com"
              value={importDomain}
              onChange={(e) => setImportDomain(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Certificate (PEM) *</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder="-----BEGIN CERTIFICATE-----"
              value={importCertPEM}
              onChange={(e) => setImportCertPEM(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Private Key (PEM) *</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder="-----BEGIN PRIVATE KEY-----"
              value={importKeyPEM}
              onChange={(e) => setImportKeyPEM(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Certificate Chain (PEM)</label>
            <textarea
              className="w-full h-24 px-3 py-2 text-sm border rounded-md bg-surface-1 border-surface-border focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder="-----BEGIN CERTIFICATE----- (optional)"
              value={importChainPEM}
              onChange={(e) => setImportChainPEM(e.target.value)}
            />
          </div>
          {importMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Failed to import certificate. Please check the PEM data.
            </p>
          )}
        </div>
      </Modal>

      {/* Renew Confirmation Modal */}
      <Modal
        open={renewModal !== null}
        onClose={() => setRenewModal(null)}
        title="Renew Certificate"
        footer={
          <>
            <Button variant="outline" onClick={() => setRenewModal(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => renewModal && renewMutation.mutate(renewModal.id)}
              loading={renewMutation.isPending}
            >
              {renewMutation.isPending ? "Renewing..." : "Renew"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to renew the certificate for{" "}
          <strong>{renewModal?.common_name}</strong>?
        </p>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        title="Delete Certificate"
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
          Are you sure you want to delete the certificate for{" "}
          <strong>{deleteModal?.common_name}</strong>? This action cannot be undone.
        </p>
      </Modal>
    </div>
  );
}

// Re-export types
export type { SSLCertificate, SSLCertStatus } from "@/lib/api/ssl";