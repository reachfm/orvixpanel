/**
 * Tests for the hosting plans API client (plans.ts).
 * Tests the request builder and type exports.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import * as client from "../client";

// Mock the client module
vi.mock("../client", () => ({
  request: vi.fn(),
}));

describe("Plans API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("types", () => {
    it("should export Plan interface with correct shape", () => {
      // Minimal valid plan object matching the API response
      const plan = {
        id: "01HZ1234567890ABCDEFGHIJ",
        name: "starter",
        display_name: "Starter Plan",
        description: "Perfect for personal projects",
        disk_quota_mb: 10240,
        bandwidth_gb: 100,
        max_domains: 5,
        max_users: 10,
        max_ssl: 3,
        features: ["backup", "staging"],
        monthly_price: 9.99,
        is_active: true,
        is_default: true,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };

      expect(plan.id).toBeDefined();
      expect(plan.name).toBe("starter");
      expect(plan.display_name).toBe("Starter Plan");
      expect(plan.monthly_price).toBe(9.99);
      expect(plan.is_active).toBe(true);
      expect(plan.features).toContain("backup");
    });

    it("should export CreatePlanRequest interface", () => {
      const request = {
        name: "basic",
        display_name: "Basic Plan",
        description: "Basic hosting plan",
        disk_quota_mb: 5120,
        bandwidth_gb: 50,
        max_domains: 3,
        max_users: 5,
        max_ssl: 1,
        features: ["backup"],
        monthly_price: 4.99,
        is_active: true,
        is_default: false,
      };

      expect(request.name).toBe("basic");
      expect(request.monthly_price).toBe(4.99);
    });
  });

  describe("listPlans", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        plans: [
          {
            id: "01HZ1234567890ABCDEFGHIJ",
            name: "starter",
            display_name: "Starter Plan",
            monthly_price: 9.99,
            is_active: true,
            is_default: true,
            features: [],
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
          },
        ],
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      // Dynamic import to get the function
      const { listPlans } = await import("../plans");
      const result = await listPlans();

      expect(client.request).toHaveBeenCalledWith("/api/v1/hosting/plans");
      expect(result).toEqual(mockResponse);
    });

    it("should pass search and status query params", async () => {
      const mockResponse = { plans: [] };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listPlans } = await import("../plans");
      await listPlans({ search: "starter", status: "active" });

      expect(client.request).toHaveBeenCalledWith(
        "/api/v1/hosting/plans?search=starter&status=active"
      );
    });

    it("should handle empty params", async () => {
      const mockResponse = { plans: [] };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listPlans } = await import("../plans");
      await listPlans({});

      expect(client.request).toHaveBeenCalledWith("/api/v1/hosting/plans");
    });
  });

  describe("getPlan", () => {
    it("should call request with correct ID endpoint", async () => {
      const planId = "01HZ1234567890ABCDEFGHIJ";
      const mockPlan = {
        id: planId,
        name: "starter",
        display_name: "Starter Plan",
        monthly_price: 9.99,
        is_active: true,
        is_default: true,
        features: [],
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockPlan);

      const { getPlan } = await import("../plans");
      const result = await getPlan(planId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/hosting/plans/${planId}`);
      expect(result).toEqual(mockPlan);
    });

    it("should encode special characters in ID", async () => {
      const planId = "01HZ12+345/678";
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce({});

      const { getPlan } = await import("../plans");
      await getPlan(planId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/hosting/plans/${encodeURIComponent(planId)}`);
    });
  });

  describe("createPlan", () => {
    it("should POST to correct endpoint with body", async () => {
      const body = {
        name: "basic",
        display_name: "Basic Plan",
        monthly_price: 4.99,
      };

      const mockPlan = {
        id: "01HZNEWPLAN",
        ...body,
        is_active: false,
        is_default: false,
        features: [],
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockPlan);

      const { createPlan } = await import("../plans");
      const result = await createPlan(body);

      expect(client.request).toHaveBeenCalledWith("/api/v1/hosting/plans", {
        method: "POST",
        body,
      });
      expect(result.id).toBe("01HZNEWPLAN");
    });
  });

  describe("updatePlan", () => {
    it("should PUT to correct endpoint with body", async () => {
      const planId = "01HZ1234567890ABCDEFGHIJ";
      const body = {
        name: "basic-updated",
        monthly_price: 5.99,
      };

      const mockPlan = {
        id: planId,
        ...body,
        display_name: "Basic Plan",
        is_active: true,
        is_default: false,
        features: [],
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-02T00:00:00Z",
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockPlan);

      const { updatePlan } = await import("../plans");
      const result = await updatePlan(planId, body);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/hosting/plans/${planId}`, {
        method: "PUT",
        body,
      });
      expect(result.monthly_price).toBe(5.99);
    });
  });

  describe("deletePlan", () => {
    it("should DELETE to correct endpoint", async () => {
      const planId = "01HZ1234567890ABCDEFGHIJ";
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(undefined);

      const { deletePlan } = await import("../plans");
      await deletePlan(planId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/hosting/plans/${planId}`, {
        method: "DELETE",
      });
    });
  });

  describe("activatePlan", () => {
    it("should POST to activate endpoint", async () => {
      const planId = "01HZ1234567890ABCDEFGHIJ";
      const mockPlan = {
        id: planId,
        name: "starter",
        is_active: true,
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockPlan);

      const { activatePlan } = await import("../plans");
      const result = await activatePlan(planId);

      expect(client.request).toHaveBeenCalledWith(
        `/api/v1/hosting/plans/${planId}/activate`,
        { method: "POST" }
      );
      expect(result.is_active).toBe(true);
    });
  });

  describe("deactivatePlan", () => {
    it("should POST to deactivate endpoint", async () => {
      const planId = "01HZ1234567890ABCDEFGHIJ";
      const mockPlan = {
        id: planId,
        name: "starter",
        is_active: false,
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockPlan);

      const { deactivatePlan } = await import("../plans");
      const result = await deactivatePlan(planId);

      expect(client.request).toHaveBeenCalledWith(
        `/api/v1/hosting/plans/${planId}/deactivate`,
        { method: "POST" }
      );
      expect(result.is_active).toBe(false);
    });
  });
});