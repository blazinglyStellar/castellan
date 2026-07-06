export type Role = "provider" | "consumer" | "both"

export type Theme = "dark" | "light"

export interface ProviderOverview {
  totalEarned: number
  requestsThisWeek: number
  activeEndpoints: number
  totalEndpoints: number
  pendingSettlement: number
  earningsByDay: { date: string; amount: number }[]
}

export interface ConsumerOverview {
  balance: number
  spentThisMonth: number
  topProviders: { name: string; spent: number; calls: number }[]
  isLowBalance: boolean
}

export interface RecentCall {
  id: string
  endpoint: string
  cost: number
  latency: number
  consumer: string
  time: number
  status: number
}

export interface ConsumerUsage {
  id: string
  api: string
  endpoint: string
  cost: number
  time: number
}

export interface UsageTrend {
  date: string
  requests: number
}

export type APIStatus = "active" | "inactive" | "rate_limited"

export interface ProviderAPI {
  id: string
  name: string
  endpoint: string
  description: string
  status: APIStatus
  totalRequests: number
  totalEarnings: number
  avgLatency: number
  errorRate: number
  createdAt: string
}

export interface AnalyticsSummary {
  totalRequests: number
  successfulRequests: number
  failedRequests: number
  avgLatency: number
  totalEarnings: number
  errorRate: number
}

export interface AnalyticsDataPoint {
  date: string
  requests: number
  latencyP50: number
  latencyP95: number
  latencyP99: number
}

export interface TopEndpoint {
  id: string
  endpoint: string
  method: string
  requests: number
  avgLatency: number
  errorRate: number
}

export interface ErrorBreakdown {
  statusCode: number
  count: number
  label: string
}

export type DepositStatus = "completed" | "pending" | "failed"

export interface Deposit {
  id: string
  amount: number
  status: DepositStatus
  txHash: string
  date: string
}

export interface DiscoverableAPI {
  id: string
  name: string
  description: string
  endpoint: string
  price: number
  category: string
  status: APIStatus
}

export interface UsageEvent {
  id: string
  api: string
  endpoint: string
  method: string
  cost: number
  status: number
  date: string
}

export interface ApiKey {
  id: string
  label: string
  prefix: string
  status: "active" | "revoked"
  createdAt: string
  expiresAt?: string
}

export interface Settlement {
  id: string
  date: string
  amount: number
  status: "completed" | "pending" | "failed"
  txHash: string
}
