export interface RecentCall {
  id: string
  endpoint: string
  cost: string
  latency: number
  consumer: string
  time: string
  status: number
}

export interface Provider {
  id: string
  name: string
  baseUrl: string
  endpoints: Endpoint[]
  status: "active" | "inactive"
  createdAt: string
}

export interface Endpoint {
  id: string
  route: string
  method: "GET" | "POST" | "PUT" | "DELETE" | "ANY"
  price: string
  rateLimit: number
  status: "active" | "inactive"
}

export interface Settlement {
  id: string
  date: string
  amount: string
  status: "completed" | "pending" | "failed"
  txHash: string
}

export interface Deposit {
  id: string
  amount: string
  status: "completed" | "pending" | "failed"
  txHash: string
  date: string
}

export interface UsageEvent {
  id: string
  api: string
  endpoint: string
  cost: string
  status: number
  date: string
}

export interface ApiKey {
  id: string
  label: string
  prefix: string
  status: "active" | "revoked"
  createdAt: string
}

export const MOCK_PROVIDER_OVERVIEW = {
  totalEarned: "12580.50",
  requestsThisWeek: 48203,
  activeEndpoints: 12,
  pendingSettlement: "3420.00",
  earningsByDay: [
    { date: "Mon", amount: 1250 },
    { date: "Tue", amount: 2340 },
    { date: "Wed", amount: 1890 },
    { date: "Thu", amount: 3100 },
    { date: "Fri", amount: 2780 },
    { date: "Sat", amount: 1420 },
    { date: "Sun", amount: 980 },
  ],
}

export const MOCK_CONSUMER_OVERVIEW = {
  balance: "1250.75",
  spentThisMonth: "3420.00",
  topProviders: [
    { name: "WeatherAPI", spent: "1200.00", calls: 45200 },
    { name: "Geolocation Service", spent: "980.00", calls: 18200 },
    { name: "AI Text Processing", spent: "740.00", calls: 8900 },
    { name: "Email Service", spent: "500.00", calls: 12500 },
  ],
  isLowBalance: false,
}

export const MOCK_PROVIDERS: Provider[] = [
  {
    id: "prov-1",
    name: "Weather API",
    baseUrl: "https://api.castellan.io/weather",
    status: "active",
    createdAt: "2025-01-15",
    endpoints: [
      { id: "ep-1", route: "/current", method: "GET", price: "0.50", rateLimit: 1000, status: "active" },
      { id: "ep-2", route: "/forecast", method: "GET", price: "1.00", rateLimit: 500, status: "active" },
      { id: "ep-3", route: "/history", method: "GET", price: "2.00", rateLimit: 200, status: "inactive" },
    ],
  },
  {
    id: "prov-2",
    name: "Geolocation Service",
    baseUrl: "https://api.castellan.io/geo",
    status: "active",
    createdAt: "2025-02-20",
    endpoints: [
      { id: "ep-4", route: "/lookup", method: "POST", price: "1.00", rateLimit: 500, status: "active" },
      { id: "ep-5", route: "/reverse", method: "GET", price: "0.75", rateLimit: 800, status: "active" },
    ],
  },
  {
    id: "prov-3",
    name: "AI Text Processing",
    baseUrl: "https://api.castellan.io/ai",
    status: "active",
    createdAt: "2025-03-10",
    endpoints: [
      { id: "ep-6", route: "/analyze", method: "POST", price: "5.00", rateLimit: 100, status: "active" },
      { id: "ep-7", route: "/summarize", method: "POST", price: "3.00", rateLimit: 200, status: "active" },
      { id: "ep-8", route: "/translate", method: "POST", price: "2.00", rateLimit: 300, status: "inactive" },
      { id: "ep-9", route: "/generate", method: "POST", price: "10.00", rateLimit: 50, status: "active" },
    ],
  },
]

export const MOCK_SETTLEMENTS: Settlement[] = [
  { id: "stl-1", date: "2025-05-15", amount: "4520.00", status: "completed", txHash: "a3f8c2d1e9b7a4f6c8d0e2f4a6b8c0d2e4f6a8b0" },
  { id: "stl-2", date: "2025-05-01", amount: "3890.00", status: "completed", txHash: "b4c9d3e2f0a8b5c7d9e1f3a5b7c9d1e3f5a7b9c1" },
  { id: "stl-3", date: "2025-04-15", amount: "5100.00", status: "completed", txHash: "c5d0e4f3a1b9c6d8e0f2a4b6c8d0e2f4a6b8c0d2" },
  { id: "stl-4", date: "2025-04-01", amount: "2750.00", status: "completed", txHash: "d6e1f5a4b2c0d7e9f1a3b5c7d9e1f3a5b7c9d1e3" },
  { id: "stl-5", date: "2025-05-20", amount: "3420.00", status: "pending", txHash: "e7f2a6b5c3d1e8f0a2b4c6d8e0f2a4b6c8d0e2f4" },
]

export const MOCK_DEPOSITS: Deposit[] = [
  { id: "dep-1", amount: "5000.00", status: "completed", txHash: "f8a3b7c6d4e2f9a1b3c5d7e9f1a3b5c7d9e1f3a5", date: "2025-05-14" },
  { id: "dep-2", amount: "2500.00", status: "completed", txHash: "a9b4c8d7e5f3a0b2c4d6e8f0a2b4c6d8e0f2a4b6", date: "2025-05-10" },
  { id: "dep-3", amount: "1000.00", status: "pending", txHash: "b0c5d9e8f6a4b1c3d5e7f9a1b3c5d7e9f1a3b5c7", date: "2025-05-18" },
]

export const MOCK_USAGE_EVENTS: UsageEvent[] = [
  { id: "ue-1", api: "Weather API", endpoint: "/current", cost: "0.50", status: 200, date: "2025-05-17T10:30:00Z" },
  { id: "ue-2", api: "Weather API", endpoint: "/forecast", cost: "1.00", status: 200, date: "2025-05-17T10:28:00Z" },
  { id: "ue-3", api: "Geolocation", endpoint: "/lookup", cost: "1.00", status: 402, date: "2025-05-17T10:25:00Z" },
  { id: "ue-4", api: "AI Text", endpoint: "/analyze", cost: "5.00", status: 200, date: "2025-05-17T10:20:00Z" },
  { id: "ue-5", api: "Weather API", endpoint: "/current", cost: "0.50", status: 200, date: "2025-05-17T10:15:00Z" },
  { id: "ue-6", api: "Email Service", endpoint: "/send", cost: "0.25", status: 200, date: "2025-05-17T10:10:00Z" },
  { id: "ue-7", api: "Geolocation", endpoint: "/reverse", cost: "0.75", status: 200, date: "2025-05-17T10:05:00Z" },
  { id: "ue-8", api: "AI Text", endpoint: "/generate", cost: "10.00", status: 402, date: "2025-05-17T10:00:00Z" },
]

export const MOCK_API_KEYS: ApiKey[] = [
  { id: "ak-1", label: "Production", prefix: "fg_prod_", status: "active", createdAt: "2025-01-15" },
  { id: "ak-2", label: "Staging", prefix: "fg_stag_", status: "active", createdAt: "2025-02-20" },
  { id: "ak-3", label: "Development", prefix: "fg_dev_", status: "revoked", createdAt: "2025-03-10" },
]

export const MOCK_ANALYTICS = {
  requestsOverTime: [
    { date: "Mon", weather: 12500, geo: 8400, ai: 3200, email: 4500 },
    { date: "Tue", weather: 14200, geo: 9100, ai: 3800, email: 5200 },
    { date: "Wed", weather: 13800, geo: 8700, ai: 4100, email: 4900 },
    { date: "Thu", weather: 15100, geo: 9500, ai: 4300, email: 5100 },
    { date: "Fri", weather: 14800, geo: 9200, ai: 3900, email: 4800 },
    { date: "Sat", weather: 8900, geo: 5800, ai: 2100, email: 3200 },
    { date: "Sun", weather: 7600, geo: 5100, ai: 1800, email: 2800 },
  ],
  revenueBreakdown: [
    { endpoint: "/current", requests: 45200, revenue: "22600.00", avgLatency: 45 },
    { endpoint: "/forecast", requests: 18200, revenue: "18200.00", avgLatency: 120 },
    { endpoint: "/lookup", requests: 12500, revenue: "12500.00", avgLatency: 30 },
    { endpoint: "/analyze", requests: 8900, revenue: "44500.00", avgLatency: 850 },
    { endpoint: "/send", requests: 12500, revenue: "3125.00", avgLatency: 65 },
  ],
}

export const MOCK_SETTINGS = {
  email: "user@castellan.io",
  role: "both" as const,
  stellarAddress: "GBRFXKQH45J6X7K8L9M0N1P2Q3R4S5T6U7V8W9X0Y",
  depositMemo: "castellan-deposit-memo-001",
  depositInstructions: "Send any XLM amount to the address above with the memo text. Funds will be credited within 30 seconds of network confirmation.",
}
