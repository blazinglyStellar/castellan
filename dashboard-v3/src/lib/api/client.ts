import type {
  AccountResponse,
  BalanceResponse,
  CursorParams,
  DashboardMeResponse,
  DepositListResponse,
  EarningsResponse,
  Endpoint,
  EntryParams,
  IntentResponse,
  ListEntriesResponse,
  Provider,
  SettlementListResponse,
  UsageListResponse,
  UsageParams,
} from "./types";

export class ApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`API request failed with status ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

class ApiClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl =
      process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const options: RequestInit = {
      method,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
    };

    if (body !== undefined) {
      options.body = JSON.stringify(body);
    }

    const res = await fetch(url, options);

    if (!res.ok) {
      const errorBody = await res.json().catch(() => null);
      if (res.status === 401) {
        if (typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("auth:unauthorized"));
        }
      }
      throw new ApiError(res.status, errorBody);
    }

    return res.json();
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  async post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PATCH", path, body);
  }

  async del<T>(path: string): Promise<T> {
    return this.request<T>("DELETE", path);
  }
}

export const api = new ApiClient();

// ── Dashboard ──

export function getDashboardMe(): Promise<DashboardMeResponse> {
  return api.get<DashboardMeResponse>("/api/v1/me");
}

// ── Balance ──

export function getBalance(): Promise<BalanceResponse> {
  return api.get<BalanceResponse>("/api/v1/balance");
}

// ── Usage ──

export function getUsage(params?: UsageParams): Promise<UsageListResponse> {
  const qs = buildQueryString(params);
  return api.get<UsageListResponse>(`/api/v1/usage${qs}`);
}

// ── Deposits ──

export function getDeposits(params?: CursorParams): Promise<DepositListResponse> {
  const qs = buildQueryString(params);
  return api.get<DepositListResponse>(`/api/v1/deposits${qs}`);
}

// ── Earnings ──

export function getEarnings(): Promise<EarningsResponse> {
  return api.get<EarningsResponse>("/api/v1/earnings");
}

// ── Settlements ──

export function getSettlements(params?: CursorParams): Promise<SettlementListResponse> {
  const qs = buildQueryString(params);
  return api.get<SettlementListResponse>(`/api/v1/settlements${qs}`);
}

// ── Providers ──

export function getProviders(): Promise<Provider[]> {
  return api.get<Provider[]>("/api/v1/providers");
}

export function getProviderEndpoints(providerId: string): Promise<Endpoint[]> {
  return api.get<Endpoint[]>(`/api/v1/providers/${encodeURIComponent(providerId)}/endpoints`);
}

// ── Account ──

export function getAccount(): Promise<AccountResponse> {
  return api.get<AccountResponse>("/api/v1/accounts/me");
}

export function getAccountEntries(params?: EntryParams): Promise<ListEntriesResponse> {
  const qs = buildQueryString(params);
  return api.get<ListEntriesResponse>(`/api/v1/accounts/me/entries${qs}`);
}

// ── Deposit Intent ──

export function getDepositIntent(): Promise<IntentResponse> {
  return api.get<IntentResponse>("/api/v1/deposits/intent");
}

// ── Helpers ──

function buildQueryString(params?: object): string {
  if (!params) return "";
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== null
  );
  if (entries.length === 0) return "";
  return "?" + new URLSearchParams(
    entries.map(([k, v]) => [k, String(v)])
  ).toString();
}
