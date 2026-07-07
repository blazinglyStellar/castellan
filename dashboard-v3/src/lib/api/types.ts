// ── Auth ──

export interface User {
  id: string;
  email: string;
  created_at: string;
}

export interface DashboardMeResponse {
  id: string;
  email: string;
  role: string;
  deposit_memo: string;
  payout_stellar_address?: string;
}

// ── Balance & Account ──

export interface Balance {
  balance: string;
  currency: string;
  available_balance: string;
}

export interface AccountResponse {
  id: string;
  balance: string;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface EntryResponse {
  id: string;
  entry_type: string;
  amount: string;
  balance_after: string;
  currency: string;
  reference_type?: string;
  status: string;
  description?: string;
  created_at: string;
}

export interface ListEntriesResponse {
  entries: EntryResponse[];
  total: number;
  limit: number;
  offset: number;
}

// ── Usage ──

export interface UsageEvent {
  id: string;
  timestamp: string;
  route: string;
  method: string;
  request_cost: string;
  currency: string;
  status_code?: number;
  latency_ms?: number;
  response_size?: number;
  request_id: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  next_cursor: string | null;
}

// ── Earnings ──

export interface EndpointEarning {
  endpoint_id: string;
  route: string;
  total: string;
}

export interface DailyEarning {
  date: string;
  amount: string;
}

export interface Earnings {
  total_earnings: string;
  unsettled_earnings: string;
  by_endpoint: EndpointEarning[];
  sparkline: DailyEarning[];
}

// ── Deposits ──

export interface Deposit {
  id: string;
  amount: string;
  currency: string;
  memo?: string;
  tx_hash: string;
  status: string;
  from_address: string;
  created_at: string;
  confirmed_at?: string;
}

export interface IntentResponse {
  sep7_uri: string;
  qr_code: string;
  memo: string;
  destination: string;
  minimum_amount: string;
  asset: string;
}

// ── Settlements ──

export interface SettlementEntry {
  id: string;
  provider_id: string;
  amount: string;
  currency: string;
  wallet_address: string;
  status: string;
  created_at: string;
}

export interface SettlementBatch {
  id: string;
  status: string;
  total_amount: string;
  currency: string;
  entry_count: number;
  created_at: string;
  completed_at?: string;
  entries: SettlementEntry[];
}

// ── Providers ──

export interface Provider {
  id: string;
  owner_id: string;
  name: string;
  base_url: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// ── Endpoints ──

export interface Endpoint {
  id: string;
  provider_id: string;
  route: string;
  method: string;
  price_amount: string;
  currency: string;
  rate_limit: number | null;
  status: string;
  created_at: string;
  updated_at: string;
}

// ── API Keys ──

export interface ApiKey {
  id: string;
  label: string | null;
  status: string;
  created_at: string;
  expires_at?: string;
}

export interface CreateApiKeyResponse extends ApiKey {
  key: string;
}

// ── Generic ──

export interface ErrorResponse {
  error: string;
  code?: string;
}

export interface CursorParams {
  cursor?: string;
  limit?: number;
}

export interface UsageParams extends CursorParams {
  role?: "provider" | "consumer";
  start_date?: string;
  end_date?: string;
  endpoint_id?: string;
  status_code?: number;
}

export interface EntryParams {
  limit?: number;
  offset?: number;
  type?: string;
}

export type BalanceResponse = Balance;
export type EarningsResponse = Earnings;
export type UsageListResponse = PaginatedResponse<UsageEvent>;
export type DepositListResponse = PaginatedResponse<Deposit>;
export type SettlementListResponse = PaginatedResponse<SettlementBatch>;
