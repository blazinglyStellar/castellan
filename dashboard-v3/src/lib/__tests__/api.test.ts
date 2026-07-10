import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  getUsage,
  getSettlements,
  getProviders,
  getBalance,
  getAccount,
  getAccountEntries,
  ApiError,
} from "../api/client";
import type { UsageListResponse, SettlementListResponse, Provider } from "../api/types";

function mockFetch(status: number, body: unknown, headers?: Record<string, string>) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", ...headers },
  });
}

async function expectUnauthorized(fn: () => Promise<unknown>) {
  const spy = vi.fn();
  window.addEventListener("auth:unauthorized", spy);
  const err = await fn().catch((e) => e);
  expect(err).toBeInstanceOf(ApiError);
  expect(err.status).toBe(401);
  expect(spy).toHaveBeenCalledTimes(1);
}

describe("API client contract", () => {
  beforeEach(() => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(mockFetch(200, {}));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("getUsage", () => {
    it("parses cursor pagination shape", async () => {
      const data: UsageListResponse = {
        data: [
          {
            id: "evt-001",
            timestamp: "2026-07-07T12:00:00Z",
            route: "/v1/weather/current",
            method: "GET",
            request_cost: "0.01",
            currency: "XLM",
            status_code: 200,
            latency_ms: 150,
            response_size: 1024,
            request_id: "req-001",
          },
        ],
        next_cursor: "2026-07-07T12:00:00Z|evt-001",
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getUsage();

      expect(result).toBeDefined();
      expect(Array.isArray(result.data)).toBe(true);
      expect(result.data[0].request_cost).toBeTypeOf("string");
      expect(result.data[0].id).toBeTypeOf("string");
      expect(result.data[0].status_code).toBeTypeOf("number");
      expect(result.data[0].latency_ms).toBeTypeOf("number");
      expect(result.data[0].response_size).toBeTypeOf("number");
      expect(result.next_cursor).toBeTypeOf("string");
    });

    it("handles null next_cursor", async () => {
      const data: UsageListResponse = {
        data: [
          {
            id: "evt-001",
            timestamp: "2026-07-07T12:00:00Z",
            route: "/v1/test",
            method: "GET",
            request_cost: "0.05",
            currency: "XLM",
            request_id: "req-001",
          },
        ],
        next_cursor: null,
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getUsage();

      expect(result.next_cursor).toBeNull();
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getUsage);
    });
  });

  describe("getSettlements", () => {
    it("parses nested entries array with string amounts", async () => {
      const data: SettlementListResponse = {
        data: [
          {
            id: "batch-001",
            status: "completed",
            total_amount: "25.00",
            currency: "XLM",
            entry_count: 2,
            created_at: "2026-07-01T12:00:00Z",
            completed_at: "2026-07-01T12:30:00Z",
            entries: [
              {
                id: "entry-001",
                provider_id: "prov-001",
                provider_name: "Provider A",
                amount: "15.00",
                currency: "XLM",
                wallet_address: "GABCD...",
                status: "completed",
                created_at: "2026-07-01T12:00:00Z",
              },
              {
                id: "entry-002",
                provider_id: "prov-002",
                provider_name: "Provider B",
                amount: "10.00",
                currency: "XLM",
                wallet_address: "GEFGH...",
                status: "completed",
                created_at: "2026-07-01T12:00:00Z",
              },
            ],
          },
        ],
        next_cursor: null,
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getSettlements();

      expect(result.data[0].total_amount).toBeTypeOf("string");
      expect(result.data[0].entry_count).toBeTypeOf("number");
      expect(Array.isArray(result.data[0].entries)).toBe(true);
      expect(result.data[0].entries[0].amount).toBeTypeOf("string");
      expect(result.data[0].entries[0].provider_id).toBeTypeOf("string");
      expect(result.data[0].entries[1].amount).toBeTypeOf("string");
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getSettlements);
    });
  });

  describe("getProviders", () => {
    it("parses raw JSON array (no wrapper)", async () => {
      const data: Provider[] = [
        {
          id: "prov-001",
          owner_id: "user-001",
          name: "Weather API",
          base_url: "https://api.weather.example.com",
          status: "active",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
        {
          id: "prov-002",
          owner_id: "user-001",
          name: "AI Inference",
          base_url: "https://inference.ai.example.com",
          status: "active",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ];
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getProviders();

      expect(Array.isArray(result)).toBe(true);
      expect(result.length).toBe(2);
      expect(result[0].id).toBeTypeOf("string");
      expect(result[0].name).toBeTypeOf("string");
      expect(result[0].base_url).toBeTypeOf("string");
      expect(result[0].status).toBeTypeOf("string");
      expect(result[0].owner_id).toBeTypeOf("string");
      expect(result[0].created_at).toBeTypeOf("string");
      expect(result[0].updated_at).toBeTypeOf("string");
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getProviders);
    });
  });

  describe("getBalance", () => {
    it("returns string amounts", async () => {
      const data = {
        balance: "1000.00",
        currency: "XLM",
        available_balance: "699.84",
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getBalance();

      expect(result.balance).toBeTypeOf("string");
      expect(result.available_balance).toBeTypeOf("string");
      expect(result.currency).toBeTypeOf("string");
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getBalance);
    });
  });

  describe("getAccount", () => {
    it("returns balance as string", async () => {
      const data = {
        id: "acct-001",
        balance: "1000.00",
        currency: "XLM",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getAccount();

      expect(result.balance).toBeTypeOf("string");
      expect(result.id).toBeTypeOf("string");
      expect(result.currency).toBeTypeOf("string");
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getAccount);
    });
  });

  describe("getAccountEntries", () => {
    it("parses nested entries with string amounts", async () => {
      const data = {
        entries: [
          {
            id: "entry-001",
            entry_type: "deposit",
            amount: "1000.00",
            balance_after: "1000.00",
            currency: "XLM",
            status: "completed",
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "entry-002",
            entry_type: "deduction",
            amount: "-0.05",
            balance_after: "999.95",
            currency: "XLM",
            status: "completed",
            created_at: "2026-01-01T00:00:05Z",
          },
        ],
        total: 2,
        limit: 50,
        offset: 0,
      };
      vi.mocked(fetch).mockResolvedValue(mockFetch(200, data));

      const result = await getAccountEntries();

      expect(Array.isArray(result.entries)).toBe(true);
      expect(result.entries[0].amount).toBeTypeOf("string");
      expect(result.entries[0].balance_after).toBeTypeOf("string");
      expect(result.entries[1].amount).toBeTypeOf("string");
      expect(result.entries[1].balance_after).toBeTypeOf("string");
      expect(result.total).toBeTypeOf("number");
      expect(result.limit).toBeTypeOf("number");
      expect(result.offset).toBeTypeOf("number");
    });

    it("throws ApiError on 401", async () => {
      vi.mocked(fetch).mockResolvedValue(mockFetch(401, { error: "authentication required" }));
      await expectUnauthorized(getAccountEntries);
    });
  });
});
