export interface UserProfile {
  id: string;
  email: string;
  role: "provider" | "consumer";
  deposit_memo: string;
  payout_stellar_address: string;
}

export interface Balance {
  balance: string;
  currency: string;
  available_balance: string;
}

export interface UsageEvent {
  id: string;
  timestamp: string;
  endpoint_route: string;
  method: string;
  request_cost: string;
  currency: string;
  status_code: number;
  latency_ms: number;
  response_size: number;
  request_id: string;
}

export interface Deposit {
  id: string;
  amount: string;
  currency: string;
  memo: string;
  tx_hash: string | null;
  status: string;
  created_at: string;
  confirmed_at: string | null;
}

export interface Provider {
  id: string;
  name: string;
  base_url: string;
  status: string;
  created_at: string;
}

export interface Endpoint {
  id: string;
  route: string;
  method: string;
  price_amount: string;
  currency: string;
  rate_limit: number;
  status: string;
  created_at: string;
}

export interface SettlementBatch {
  id: string;
  status: string;
  total_amount: string;
  currency: string;
  entry_count: number;
  tx_hash: string | null;
  created_at: string;
  completed_at: string | null;
  entries: SettlementEntry[];
}

export interface SettlementEntry {
  provider_id: string;
  amount: string;
  wallet_address: string;
  status: string;
}

export interface Earnings {
  total_earnings: string;
  unsettled_earnings: string;
  by_endpoint: { endpoint_id: string; route: string; total: string }[];
  sparkline: { date: string; amount: string }[];
}

export interface LedgerEntry {
  id: string;
  entry_type: string;
  amount: string;
  balance_after: string;
  status: string;
  description: string;
  created_at: string;
}

export interface ApiKey {
  id: string;
  label: string;
  prefix: string;
  status: string;
  expires_at: string | null;
  created_at: string;
}

export interface DepositIntent {
  address: string;
  memo: string;
  uri: string;
  qr_base64: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  next_cursor: string | null;
}
