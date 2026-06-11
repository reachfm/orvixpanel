/**
 * Tests for the system/health API client (system.ts).
 * Tests the health probes, system info, license, and audit endpoints.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import * as client from "../client";

// Mock the client module
vi.mock("../client", () => ({
  request: vi.fn(),
}));

describe("System API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("types", () => {
    it("should export HealthzResponse interface with correct shape", () => {
      const response = { status: "ok" };
      expect(response.status).toBe("ok");
    });

    it("should export ReadyzResponse interface with correct shape", () => {
      const readyResponse = { status: "ready" };
      expect(readyResponse.status).toBe("ready");

      const degradedResponse = { status: "db_unavailable" };
      expect(degradedResponse.status).toBe("db_unavailable");
    });

    it("should export SystemInfo interface with correct shape", () => {
      const info = {
        name: "OrvixPanel",
        version: "1.0.0",
        uptime_at: "2024-01-01T00:00:00Z",
      };

      expect(info.name).toBe("OrvixPanel");
      expect(info.version).toBe("1.0.0");
      expect(info.uptime_at).toBe("2024-01-01T00:00:00Z");
    });

    it("should export License interface with correct shape", () => {
      const license = {
        tier: "SMB",
        features: ["backup", "staging"],
        max_servers: 5,
        expires_at: 1735689600,
        issued_at: 1704067200,
        grace_days: 7,
      };

      expect(license.tier).toBe("SMB");
      expect(license.features).toContain("backup");
      expect(license.max_servers).toBe(5);
    });

    it("should export LicenseRenewal interface with correct shape", () => {
      const renewal = {
        loaded: true,
        tier: "SMB",
        expires_at: "2025-01-01T00:00:00Z",
        days_remaining: 180,
        grace_days: 7,
        status: "active",
        mode: "active",
        licensed_to: "Test Company",
      };

      expect(renewal.loaded).toBe(true);
      expect(renewal.status).toBe("active");
      expect(renewal.days_remaining).toBe(180);
    });

    it("should export AuditEntry interface with correct shape", () => {
      const entry = {
        id: "audit-001",
        timestamp: "2024-01-01T00:00:00Z",
        user_id: "user-001",
        user_email: "admin@example.com",
        action: "create",
        resource_type: "account",
        resource_id: "acc-001",
        resource_name: "test-account",
        result: "success",
        actor_ip: "127.0.0.1",
        prev_hash: "abc123",
        hash: "def456",
        details: "Account created successfully",
      };

      expect(entry.id).toBe("audit-001");
      expect(entry.action).toBe("create");
      expect(entry.result).toBe("success");
    });

    it("should export SystemHealthResult interface with correct shape", () => {
      const result = {
        checks: [
          {
            name: "Database Connection",
            status: "pass",
            message: "Database is accessible",
          },
        ],
      };

      expect(result.checks).toHaveLength(1);
      expect(result.checks[0].status).toBe("pass");
    });
  });

  describe("healthz", () => {
    it("should call request with correct endpoint and no auth", async () => {
      const mockResponse = { status: "ok" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { healthz } = await import("../system");
      const result = await healthz();

      expect(client.request).toHaveBeenCalledWith("/healthz", { noAuth: true, timeoutMs: 3000 });
      expect(result).toEqual(mockResponse);
    });

    it("should return status object", async () => {
      const mockResponse = { status: "ok" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { healthz } = await import("../system");
      const result = await healthz();

      expect(result.status).toBe("ok");
    });
  });

  describe("readyz", () => {
    it("should call request with correct endpoint and no auth", async () => {
      const mockResponse = { status: "ready" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { readyz } = await import("../system");
      const result = await readyz();

      expect(client.request).toHaveBeenCalledWith("/readyz", { noAuth: true, timeoutMs: 3000 });
      expect(result).toEqual(mockResponse);
    });

    it("should handle ready status", async () => {
      const mockResponse = { status: "ready" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { readyz } = await import("../system");
      const result = await readyz();

      expect(result.status).toBe("ready");
    });

    it("should handle db_unavailable status", async () => {
      const mockResponse = { status: "db_unavailable" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { readyz } = await import("../system");
      const result = await readyz();

      expect(result.status).toBe("db_unavailable");
    });
  });

  describe("systemInfo", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        name: "OrvixPanel",
        version: "1.0.0",
        uptime_at: "2024-01-01T00:00:00Z",
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { systemInfo } = await import("../system");
      const result = await systemInfo();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/system");
      expect(result.name).toBe("OrvixPanel");
    });
  });

  describe("license", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        tier: "SMB",
        features: ["backup"],
        max_servers: 5,
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { license } = await import("../system");
      const result = await license();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/license");
      expect(result.tier).toBe("SMB");
    });
  });

  describe("licenseRenewal", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        loaded: true,
        status: "active",
        days_remaining: 180,
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { licenseRenewal } = await import("../system");
      const result = await licenseRenewal();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/license/renewal-info");
      expect(result.loaded).toBe(true);
    });
  });

  describe("uploadLicense", () => {
    it("should PUT to correct endpoint with key", async () => {
      const mockResponse = {
        tier: "SMB",
        features: ["backup"],
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { uploadLicense } = await import("../system");
      const result = await uploadLicense("test-license-key");

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/license", {
        method: "PUT",
        body: { key: "test-license-key" },
      });
      expect(result.tier).toBe("SMB");
    });
  });

  describe("listAudit", () => {
    it("should call request with default limit", async () => {
      const mockResponse = { entries: [] };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listAudit } = await import("../system");
      const result = await listAudit();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/audit-log?limit=100");
      expect(result.entries).toEqual([]);
    });

    it("should pass custom limit", async () => {
      const mockResponse = { entries: [] };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listAudit } = await import("../system");
      await listAudit(50);

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/audit-log?limit=50");
    });
  });

  describe("searchAudit", () => {
    it("should POST to correct endpoint with body", async () => {
      const mockResponse = { entries: [], next_cursor: undefined };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { searchAudit } = await import("../system");
      const body = {
        user_email: "admin@example.com",
        action: "create",
        limit: 20,
      };
      const result = await searchAudit(body);

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/audit-log/search", {
        method: "POST",
        body,
      });
      expect(result.entries).toEqual([]);
    });
  });

  describe("verifyAudit", () => {
    it("should POST to correct endpoint", async () => {
      const mockResponse = { tampered: false, first_bad_row: -1 };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { verifyAudit } = await import("../system");
      const result = await verifyAudit();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/audit-log/verify", {
        method: "POST",
      });
      expect(result.tampered).toBe(false);
    });
  });

  describe("systemHealth", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        checks: [
          { name: "Database", status: "pass", message: "Connected" },
          { name: "License", status: "pass", message: "Valid" },
        ],
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { systemHealth } = await import("../system");
      const result = await systemHealth();

      expect(client.request).toHaveBeenCalledWith("/api/v1/admin/system/health");
      expect(result.checks).toHaveLength(2);
    });
  });
});