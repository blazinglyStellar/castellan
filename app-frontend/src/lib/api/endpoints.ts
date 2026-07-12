import type {
  AccountResponse,
  ApiKey,
  BalanceResponse,
  CreateApiKeyResponse,
  CreateEndpointRequest,
  CreateProviderRequest,
  CursorParams,
  DashboardMeResponse,
  DepositListResponse,
  DiscoverResponse,
  EarningsResponse,
  Endpoint,
  EntryParams,
  EntryResponse,
  IntentResponse,
  ListEntriesResponse,
  PayoutAddressResponse,
  PayoutCheckResponse,
  Provider,
  SettlementListResponse,
  SettlementParams,
  SettlementSummary,
  SettlementThreshold,
  UsageListResponse,
  UsageParams,
} from "./types"

export class ApiError extends Error {
  status: number
  body: unknown

  constructor(status: number, body: unknown) {
    super(`API request failed with status ${status}`)
    this.name = "ApiError"
    this.status = status
    this.body = body
  }
}

class ApiClient {
  private baseUrl: string

  constructor() {
    this.baseUrl =
      process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080"
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`
    const options: RequestInit = {
      method,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
    }

    if (body !== undefined) {
      options.body = JSON.stringify(body)
    }

    const res = await fetch(url, options)

    if (!res.ok) {
      const errorBody = await res.json().catch(() => null)
      if (res.status === 401) {
        if (typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("auth:unauthorized"))
        }
      }
      throw new ApiError(res.status, errorBody)
    }

    return res.json()
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path)
  }

  async post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body)
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PATCH", path, body)
  }

  async del<T>(path: string): Promise<T> {
    return this.request<T>("DELETE", path)
  }
}

export const api = new ApiClient()

function buildQueryString(params?: object): string {
  if (!params) return ""
  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== null,
  )
  if (entries.length === 0) return ""
  return (
    "?" +
    new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString()
  )
}

// ── Dashboard ──

export function getDashboardMe(): Promise<DashboardMeResponse> {
  return api.get<DashboardMeResponse>("/api/v1/me")
}

// ── Balance ──

export function getBalance(): Promise<BalanceResponse> {
  return api.get<BalanceResponse>("/api/v1/balance")
}

// ── Usage ──

export function getUsage(params?: UsageParams): Promise<UsageListResponse> {
  const qs = buildQueryString(params)
  return api.get<UsageListResponse>(`/api/v1/usage${qs}`)
}

// ── Earnings ──

interface EarningsParams {
  start_date?: string
  end_date?: string
}

export function getEarnings(
  params?: EarningsParams,
): Promise<EarningsResponse> {
  const qs = params ? buildQueryString(params) : ""
  return api.get<EarningsResponse>(`/api/v1/earnings${qs}`)
}

// ── Settlements ──

export function getSettlements(
  params?: SettlementParams,
): Promise<SettlementListResponse> {
  const qs = buildQueryString(params)
  return api.get<SettlementListResponse>(`/api/v1/settlements${qs}`)
}

interface SettlementSummaryParams {
  start_date?: string
  end_date?: string
}

export function getSettlementSummary(
  params?: SettlementSummaryParams,
): Promise<SettlementSummary> {
  const qs = params ? buildQueryString(params) : ""
  return api.get<SettlementSummary>(`/api/v1/settlements/summary${qs}`)
}

export function getSettlementThreshold(): Promise<SettlementThreshold> {
  return api.get<SettlementThreshold>("/api/v1/settlements/threshold")
}

// ── Providers ──

export function getProviders(): Promise<Provider[]> {
  return api.get<Provider[]>("/api/v1/providers")
}

export function createProvider(
  data: CreateProviderRequest,
): Promise<Provider> {
  return api.post<Provider>("/api/v1/providers", data)
}

export function deleteProvider(id: string): Promise<void> {
  return api.del<void>(`/api/v1/providers/${encodeURIComponent(id)}`)
}

export function updateProviderStatus(
  id: string,
  status: string,
): Promise<Provider> {
  return api.patch<Provider>(
    `/api/v1/providers/${encodeURIComponent(id)}/status`,
    { status },
  )
}

export function getProviderEndpoints(
  providerId: string,
): Promise<Endpoint[]> {
  return api.get<Endpoint[]>(
    `/api/v1/providers/${encodeURIComponent(providerId)}/endpoints`,
  )
}

export function createEndpoint(
  providerId: string,
  data: CreateEndpointRequest,
): Promise<Endpoint> {
  return api.post<Endpoint>(
    `/api/v1/providers/${encodeURIComponent(providerId)}/endpoints`,
    data,
  )
}

export function deleteEndpoint(id: string): Promise<void> {
  return api.del<void>(`/api/v1/endpoints/${encodeURIComponent(id)}`)
}

export function updateEndpointStatus(
  id: string,
  status: string,
): Promise<Endpoint> {
  return api.patch<Endpoint>(
    `/api/v1/endpoints/${encodeURIComponent(id)}/status`,
    { status },
  )
}

// ── Discover ──

export function getDiscoverProviders(): Promise<DiscoverResponse> {
  return api.get<DiscoverResponse>("/api/v1/discover")
}

export function getPublicProviderEndpoints(
  providerId: string,
): Promise<Endpoint[]> {
  return api.get<Endpoint[]>(
    `/api/v1/providers/${encodeURIComponent(providerId)}/endpoints/public`,
  )
}

// ── API Keys ──

export function getApiKeys(): Promise<ApiKey[]> {
  return api.get<ApiKey[]>("/api/v1/keys")
}

export function createApiKey(
  label: string,
  expiresAt?: string,
): Promise<CreateApiKeyResponse> {
  return api.post<CreateApiKeyResponse>("/api/v1/keys", {
    label,
    ...(expiresAt ? { expires_at: expiresAt } : {}),
  })
}

export function updateApiKey(
  id: string,
  data: { label?: string; expires_at?: string | null },
): Promise<ApiKey> {
  return api.patch<ApiKey>(`/api/v1/keys/${encodeURIComponent(id)}`, data)
}

export function revokeApiKey(id: string): Promise<void> {
  return api.post<void>(`/api/v1/keys/${encodeURIComponent(id)}/revoke`)
}

export function rotateApiKey(id: string): Promise<CreateApiKeyResponse> {
  return api.post<CreateApiKeyResponse>(
    `/api/v1/keys/${encodeURIComponent(id)}/rotate`,
  )
}

// ── Account ──

export function checkPayoutAddress(address: string): Promise<PayoutCheckResponse> {
  return api.get<PayoutCheckResponse>(`/api/v1/me/payout/check?address=${encodeURIComponent(address)}`)
}

export function updatePayoutAddress(address: string): Promise<PayoutAddressResponse> {
  return api.patch<PayoutAddressResponse>("/api/v1/me/payout", { payout_stellar_address: address })
}

export function getAccount(): Promise<AccountResponse> {
  return api.get<AccountResponse>("/api/v1/accounts/me")
}

export function getAccountEntries(
  params?: EntryParams,
): Promise<ListEntriesResponse> {
  const qs = buildQueryString(params)
  return api.get<ListEntriesResponse>(`/api/v1/accounts/me/entries${qs}`)
}

export function getAccountEntry(id: string): Promise<EntryResponse> {
  return api.get<EntryResponse>(
    `/api/v1/accounts/me/entries/${encodeURIComponent(id)}`,
  )
}

// ── Deposit Intent ──

export function getDepositIntent(): Promise<IntentResponse> {
  return api.get<IntentResponse>("/api/v1/deposits/intent")
}

export function getDeposits(
  params?: CursorParams,
): Promise<DepositListResponse> {
  const qs = buildQueryString(params)
  return api.get<DepositListResponse>(`/api/v1/deposits${qs}`)
}
