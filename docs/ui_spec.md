# Castellan — UI Specification

Designed for AI-assisted generation (OpenDesign, Claude Design, v0, etc.).

---

## 1. Product Overview

Castellan is a developer platform that lets API providers charge per request instead of subscriptions. It acts as a reverse proxy handling auth, metering, and prepaid billing backed by Stellar blockchain settlement.

**Two user roles:**
- **Provider** — owns APIs, sets pricing, receives payouts
- **Consumer** — calls APIs, prepays via Stellar, tracks usage

A user can hold both roles simultaneously (e.g., someone who both provides and consumes APIs).

---

## 2. Design System

### Palette

| Token | Hex | Usage |
|---|---|---|
| `--bg` | `#0a0a0b` | Main background (dark theme preferred) |
| `--surface` | `#18181b` | Card/sheet background |
| `--border` | `#27272a` | Borders, dividers |
| `--text-primary` | `#fafafa` | Primary text |
| `--text-secondary` | `#a1a1aa` | Secondary/muted text |
| `--accent` | `#3b82f6` | Primary actions, links |
| `--accent-green` | `#22c55e` | Positive (balance, credits, success) |
| `--accent-red` | `#ef4444` | Negative (insufficient balance, errors) |
| `--accent-amber` | `#f59e0b` | Pending/warnings |

Light theme alternative: swap to a clean white/gray-50 background with gray-900 text, keeping blue accent. Default to dark.

### Typography

- **UI:** Inter or Geist (sans-serif, developer-friendly)
- **Monospace:** JetBrains Mono or Geist Mono for code blocks, crypto addresses, API keys
- **Scale:** 12 / 14 / 16 / 18 / 24 / 30 / 36 px

### Tone

- Clean, data-dense, developer-oriented
- Think Vercel dashboard × Stripe — functional, high information density, purposeful whitespace
- Every screen should prioritize readability of tabular data and financial figures

### Component Library

- `shadcn/ui` (Radix-based) — use components: Card, Table, Dialog, Sheet, Button, Input, Select, Badge, Tabs, Skeleton, Toast, Progress
- Icons: Lucide (standard with shadcn)

---

## 3. Navigation / Shell

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  Logo              Search          Notifications  User │  ← Top bar
├────────┬────────────────────────────────────────────────┤
│        │                                                │
│ Nav    │  Main Content Area                             │
│ Sidebar│                                                │
│        │                                                │
│        │                                                │
│        │                                                │
│        │                                                │
└────────┴────────────────────────────────────────────────┘
```

### Sidebar Nav Items

**Provider sees:**
```
Overview          → /dashboard
My APIs           → /dashboard/providers
  Endpoints       → /dashboard/providers/:id/endpoints
Analytics         → /dashboard/analytics
Settlements       → /dashboard/settlements
Settings          → /dashboard/settings
```

**Consumer sees:**
```
Overview          → /dashboard
Deposit           → /dashboard/deposit
Usage             → /dashboard/usage
API Keys          → /dashboard/api-keys
Settings          → /dashboard/settings
```

**Both roles:** If a user has both roles, show both nav groups separated by a divider with group labels ("Provider", "Consumer").

### Top Bar

- Left: Castellan logo + app name
- Center: global search (for providers: search endpoints; for consumers: search usage)
- Right: notification bell (icon, placeholder), user avatar dropdown (Settings, Sign out)

---

## 4. Screen Specifications

---

### 4.0 Auth — Login / Signup

**Route:** `/login`, `/signup`
**Role:** all (unauthenticated)

#### Layout
Centered card on a blank background. Logo above the card.

#### Login
- Email input
- Password input
- "Sign in" button
- Link to signup

#### Signup
- Email input
- Password input
- Role toggle: "I want to provide APIs" / "I want to use APIs" / "Both"
- "Create account" button
- On success: auto-create `accounts` row, redirect to dashboard

#### States
- **Loading:** Button shows spinner, inputs disabled
- **Error:** Inline error below form ("Invalid credentials", "Email already registered")
- **Success:** Redirect to `/dashboard`

---

### 4.1 Provider — Overview

**Route:** `/dashboard`
**Role:** provider
**Purpose:** At-a-glance earnings, request volume, recent activity

#### Layout
```
┌────────────────────────────────────────────────┐
│  Good morning, acme@co.  April 17 — Apr 24      │
├────────────┬────────────┬────────────┬──────────┤
│  Total     │  Requests  │  Active    │  Pending │
│  Earned    │  This Week │  Endpoints │  Settle. │
│  XLM 1,245 │  12,430    │  4 / 6     │  XLM 340 │
├────────────┴────────────┴────────────┴──────────┤
│  Earnings (Last 7 Days)                         │
│  ┌──────────────────────────────────────────┐   │
│  │  [Line chart: daily earnings across       │   │
│  │   each provider color-coded]              │   │
│  └──────────────────────────────────────────┘   │
├────────────────────────────────────────────────┤
│  Recent API Calls                              │
│  ┌─────┬──────────┬──────┬──────┬────────────┐ │
│  │ #   │ Endpoint │ Cost │ Lat  │ Consumer   │ │
│  ├─────┼──────────┼──────┼──────┼────────────┤ │
│  │ 1   │ /weather │ 0.01 │ 120ms│ user@a.com │ │
│  │ 2   │ /search  │ 0.02 │ 85ms │ user@b.com │ │
│  │ ... │          │      │      │            │ │
│  └─────┴──────────┴──────┴──────┴────────────┘ │
│  [View All →]                                   │
└────────────────────────────────────────────────┘
```

#### Data (from sqlc queries)
- **Stat cards:** `GetConsumerUsageSummary` scoped to provider, `AggregateProviderEarnings`, `ListUsageEventsByProvider`
- **Chart:** `AggregateProviderEarnings` grouped by day
- **Table:** `ListUsageEventsByProvider` (latest 5)

#### States
- **Empty:** "Register your first API to get started" → CTA button leads to Add Provider flow
- **Loading:** Skeleton cards + skeleton chart + skeleton table rows
- **Error:** Inline banner "Failed to load dashboard data. Retry."

#### Mock data
```json
{
  "total_earned": "1245.50",
  "requests_this_week": 12430,
  "active_endpoints": "4 / 6",
  "pending_settlement": "340.20",
  "recent_calls": [
    { "endpoint": "/weather", "cost": "0.01", "latency": "120ms", "consumer": "user@a.com", "time": "2 min ago" }
  ]
}
```

---

### 4.2 Provider — My APIs (List)

**Route:** `/dashboard/providers`
**Role:** provider
**Purpose:** See all registered API services, add new ones

#### Layout
```
┌────────────────────────────────────────────────┐
│  My APIs                          [+ Add API] │
├────────────────────────────────────────────────┤
│  ┌──────────┬────────────┬──────┬────────────┐ │
│  │ Name     │ Base URL   │ Endp │ Status     │ │
│  ├──────────┼────────────┼──────┼────────────┤ │
│  │ Weather  │ https://...│  3   │ ● Active   │ │
│  │ Search   │ https://...│  2   │ ● Active   │ │
│  │ Images   │ https://...│  1   │ ○ Inactive │ │
│  └──────────┴────────────┴──────┴────────────┘ │
└────────────────────────────────────────────────┘
```

#### Data
- `ListProvidersByOwner` — returns name, base_url, status
- Endpoint count is a secondary query or can be added as a subquery

#### Add API (Dialog)
- Name (text input)
- Base URL (text input, validated as URL)
- "Register" button
- On success: navigates to provider detail (endpoints page)

#### States
- **Empty:** Illustration + "You haven't registered any APIs yet. Add your first API to start monetizing."
- **Loading:** 3 skeleton rows

---

### 4.3 Provider — Endpoints

**Route:** `/dashboard/providers/:id/endpoints`
**Role:** provider
**Purpose:** Manage individual routes, set pricing, toggle status

#### Layout
```
┌──────────────────────────────────────────────────────┐
│  Weather API  ● Active     [Back to My APIs]         │
│  https://api.weather.com                              │
├──────────────────────────────────────────────────────┤
│  Endpoints                           [+ Add Endpoint]│
│  ┌──────┬────────┬──────────┬───────┬──────┬───────┐ │
│  │ Meth │ Route  │ Price    │ Rate  │ Stat │       │ │
│  ├──────┼────────┼──────────┼───────┼──────┼───────┤ │
│  │ GET  │/current│ 0.0001   │ 100/m │ ●    │ ... │ │
│  │      │        │ XLM      │       │      │     │ │
│  │ GET  │/forecast│ 0.0002  │ 60/m  │ ●    │ ... │ │
│  │ POST │/alerts │ 0.0005   │ 10/m  │ ○    │ ... │ │
│  └──────┴────────┴──────────┴───────┴──────┴───────┘ │
│                                                        │
│  Total: 3 endpoints  ·  2 active  ·  1 inactive        │
└────────────────────────────────────────────────────────┘
```

#### Data
- `ListEndpointsByProvider` — route, method, price_amount, rate_limit, status
- Provider base info from `GetProviderByID`

#### Add Endpoint (Dialog)
- HTTP Method (select: GET, POST, PUT, DELETE, ANY)
- Route path (text input, e.g. `/current`)
- Price per request (number input with XLM suffix)
- Rate limit (number input, "requests per minute", optional)
- Status toggle
- "Add" button

#### Row Actions (Dropdown: "...")
- Edit (opens dialog pre-filled)
- Toggle status (quick action)
- Delete (with confirmation)

#### States
- **Empty:** "No endpoints yet. Add your first route to start monetizing this API."
- **Loading:** Skeleton table

---

### 4.4 Provider — Analytics

**Route:** `/dashboard/analytics`
**Role:** provider
**Purpose:** Deep usage metrics across all APIs

#### Layout
```
┌──────────────────────────────────────────────────────────┐
│  Analytics                    [7d ▼] [All APIs ▼]        │
├──────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────┐  │
│  │  Requests Over Time                                │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │  [Area chart: requests/day, color by endpoint]│  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────┤
│  Revenue Breakdown                                       │
│  ┌──────────┬───────────┬──────────┬──────────────────┐  │
│  │ Endpoint │ Requests  │ Revenue  │ Avg Latency      │  │
│  ├──────────┼───────────┼──────────┼──────────────────┤  │
│  │/current  │ 8,234     │ 0.8234   │ 95ms             │  │
│  │/forecast │ 3,120     │ 0.6240   │ 120ms            │  │
│  │/alerts   │ 1,076     │ 0.5380   │ 210ms            │  │
│  └──────────┴───────────┴──────────┴──────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

#### Data
- `AggregateProviderEarnings` with time range filter
- `ListUsageEventsByProvider` grouped by endpoint

#### Filters
- Time range: 24h, 7d, 30d, custom
- API: All, or specific provider
- Date range picker

#### States
- **Empty (no data):** "No usage data for the selected period."
- **Loading:** Skeleton chart + skeleton table

---

### 4.5 Provider — Settlements

**Route:** `/dashboard/settlements`
**Role:** provider
**Purpose:** View payout history and pending earnings

#### Layout
```
┌──────────────────────────────────────────────────────────┐
│  Settlements                                             │
├──────────────────────────────────────────────────────────┤
│  ┌──────────────┬────────┬──────────┬──────────┬───────┐ │
│  │ Outstanding  │ Last   │ Next     │ Total    │       │ │
│  │ Payout       │ Payout │ Est.     │ All Time │       │ │
│  │ XLM 340.20   │ Apr 15 │ Apr 20   │ XLM 5,240│       │ │
│  └──────────────┴────────┴──────────┴──────────┴───────┘ │
├──────────────────────────────────────────────────────────┤
│  History                                                 │
│  ┌────────┬────────┬──────────┬──────────┬─────────────┐ │
│  │ Date   │ Amount │ Status   │ Tx Hash  │             │ │
│  ├────────┼────────┼──────────┼──────────┼─────────────┤ │
│  │Apr 15  │ 412.50 │ ✅ Sent  │ a3f2...  │             │ │
│  │Apr 10  │ 389.20 │ ✅ Sent  │ b7e1...  │             │ │
│  │Apr 05  │ 450.00 │ ⏳ Pending│ —        │             │ │
│  └────────┴────────┴──────────┴──────────┴─────────────┘ │
│  [Load More]                                              │
└──────────────────────────────────────────────────────────┘
```

#### Data
- `ListSettlementBatchesByProvider` — batches with amounts, status, tx_hash
- Earned but unsettled: from `AggregateProviderEarnings`

#### States
- **Empty:** "No settlements yet. Your earnings will appear here after consumers start using your APIs."
- **Loading:** Skeleton stat cards + skeleton table

---

### 4.6 Settings (Shared)

**Route:** `/dashboard/settings`
**Role:** both
**Purpose:** Manage API keys, payout address, profile

#### Layout (Tabs)

**Tab 1 — Profile**
```
Email: user@example.com
Role: Provider + Consumer
[Save]
```

**Tab 2 — Payout Address (provider)**
```
Stellar Payout Address:
[GABCDEF12345...                    ]
Funds from API usage are sent here.
[Save]
```

**Tab 3 — Deposit (consumer)**
```
Your Deposit Memo: user_abc123
This memo must be included when sending XLM to Castellan's deposit address.
```

**Tab 4 — API Keys (consumer)**
```
┌──────────┬────────────┬──────────┬──────────┬─────────┐
│ Label    │ Key Prefix │ Status   │ Created  │         │
├──────────┼────────────┼──────────┼──────────┼─────────┤
│ prod     │ fg_a1b2... │ ● Active │ Apr 01   │ ...     │
│ dev      │ fg_c3d4... │ ● Active │ Apr 10   │ ...     │
│ test-key │ fg_e5f6... │ ○ Revoked│ Mar 15   │ ...     │
└──────────┴────────────┴──────────┴──────────┴─────────┘
[+ Generate New Key]
```

Generate new key dialog:
- Label input
- Expires at (optional date picker)
- On create: show the raw key **once** in a modal with copy button + warning "This key will not be shown again"

#### Data
- `ListAPIKeysByUser` — label, status, created_at
- User info from `GetUserByID`
- Payout address from `users.payout_stellar_address`

---

### 4.7 Consumer — Overview

**Route:** `/dashboard`
**Role:** consumer
**Purpose:** Quick view of balance, recent spending, top providers

#### Layout
```
┌────────────────────────────────────────────────┐
│  Welcome back                              ⚡  │
│  Balance: XLM 1,250.50                         │
│  [Deposit]                    Spent this month │
│                                  XLM 430.20    │
├────────────┬────────────┬─────────────────────┤
│  Recent    │            │                      │
│  Usage     │            │  Top Providers       │
│  ┌────┬────┼──────┬─────┤  ┌─────────────────┐│
│  │API │Cost│ Date │     │  │ Weather  · 230.50││
│  ├────┼────┼──────┼─────┤  │ Search   · 180.20││
│  │/wea│0.01│ 2min │     │  │ Images   ·  19.50││
│  │... │    │      │     │  └─────────────────┘│
│  └────┴────┴──────┴─────┘                      │
└────────────────────────────────────────────────┘
```

#### Data
- `GetAccountByOwnerID` — balance
- `ListUsageEventsByConsumer` (latest 5)
- `GetConsumerUsageSummary` — grouped spend

#### States
- **Empty (no usage):** "Your balance is XLM 1,250. Start calling APIs to see usage here."
- **Empty (no deposits):** "Your balance is 0. Deposit funds to get started." → CTA to deposit screen
- **Loading:** Skeleton cards

---

### 4.8 Consumer — Deposit

**Route:** `/dashboard/deposit`
**Role:** consumer
**Purpose:** Fund account via Stellar

#### Layout
```
┌──────────────────────────────────────────────────────────┐
│  Deposit Funds                                           │
├──────────────────────────────────────────────────────────┤
│  ┌───────────────────────┐  ┌──────────────────────────┐ │
│  │                       │  │  Send XLM to:             │ │
│  │   [QR CODE]           │  │  GA4GH32...VWX            │ │
│  │                       │  │  [Copy Address]            │ │
│  │   Scan with any       │  │                            │ │
│  │   Stellar wallet      │  │  Memo: user_abc123         │ │
│  │                       │  │  [Copy Memo]               │ │
│  └───────────────────────┘  │                            │ │
│                              │  Minimum: 5 XLM           │ │
│                              │  Network: Stellar         │ │
│                              │                            │ │
│                              │  [Open in LOBSTR ▼]       │ │
│                              └──────────────────────────┘ │
├──────────────────────────────────────────────────────────┤
│  Recent Deposits                                         │
│  ┌────────┬──────────┬──────────────┬──────────────────┐ │
│  │ Amount │ Status   │ Tx Hash      │ Date             │ │
│  ├────────┼──────────┼──────────────┼──────────────────┤ │
│  │ 100 XLM│ ✅ Conf'd│ a3f2...     │ 2 min ago         │ │
│  │ 50 XLM │ ✅ Conf'd│ b7e1...     │ Yesterday         │ │
│  └────────┴──────────┴──────────────┴──────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

#### Data
- `ListDepositsByAccount` — amount, status, tx_hash, created_at
- QR generated from SEP-7 URI backend returns (`web+stellar:pay?destination=GABC...&memo=user_abc123&memo_type=text`)

#### States
- **Loading:** QR skeleton + address skeleton
- **Empty deposits:** "No deposits yet. Send XLM to the address above to fund your account."
- **Live status:** When a deposit is detected by the watcher, show a toast "Deposit detected! Crediting your account..." then update the table

---

### 4.9 Consumer — Usage History

**Route:** `/dashboard/usage`
**Role:** consumer
**Purpose:** Full request log

#### Layout
```
┌──────────────────────────────────────────────────────────┐
│  Usage History                    [7d ▼] [Export ▼]      │
├──────────────────────────────────────────────────────────┤
│  ┌──────────┬──────────┬───────┬──────────┬────────────┐ │
│  │ API      │ Endpoint │ Cost  │ Status   │ Date       │ │
│  ├──────────┼──────────┼───────┼──────────┼────────────┤ │
│  │ Weather  │ /current │ 0.0001│ 200 OK   │ Apr 17 2:03│ │
│  │ Search   │ /query   │ 0.0002│ 200 OK   │ Apr 17 2:01│ │
│  │ Images   │ /gen     │ 0.0005│ 402 PayR.│ Apr 17 1:58│ │
│  │ ...      │          │       │          │            │ │
│  └──────────┴──────────┴───────┴──────────┴────────────┘ │
│  Page 1 of 24  [← Prev] [1] [2] [3] ... [Next →]        │
└──────────────────────────────────────────────────────────┘
```

#### Data
- `ListUsageEventsByConsumer` — paginated with provider_name, endpoint_route, request_cost, status_code, created_at
- Total count for pagination

#### Filters
- Time range (date picker)
- Provider (multi-select)
- Status code filter

#### States
- **Empty:** "No API calls yet. Get an API key and start making requests."
- **Loading:** Skeleton table

---

### 4.10 Consumer — API Keys

**Same as Settings Tab 4 (4.6)** — shown as a top-level nav item for consumers.

---

### 4.11 402 Payment Required Page

**Route:** any proxy request with insufficient balance (returned as HTTP response, not a page)
**Role:** consumer
**Purpose:** Inform the developer their balance is too low

#### Response (JSON)
```json
{
  "error": "insufficient_balance",
  "balance": "0.50",
  "request_cost": "0.0001",
  "message": "Your prepaid balance is too low to process this request. Please deposit funds.",
  "deposit_url": "https://app.castellan.io/dashboard/deposit"
}
```

The dashboard should also show a persistent banner when balance drops below a threshold (e.g., < 1 XLM).

---

## 5. User Flows

### Flow A: Provider Onboarding

```
1. Sign up → set role = "provider"
2. Land on empty Overview
3. Click "Add API" → enter name + base_url
4. Land on Endpoints page (empty)
5. Click "Add Endpoint" → enter route + method + price + rate limit
6. See endpoint appear in table, active by default
7. Return to Overview → stat cards begin showing data as requests come in
8. Go to Settings → enter payout Stellar address
9. Settlements page populates after settlement worker runs
```

### Flow B: Consumer Depositing + Using

```
1. Sign up → set role = "consumer"
2. Land on empty Overview (balance = 0)
3. Click "Deposit" → see QR code + address + memo
4. Open Stellar wallet app → scan QR (dest + memo auto-filled)
5. Send 100 XLM
6. Wait ~5-10 seconds → deposit watcher detects → toast "Deposit confirmed!"
7. Overview now shows balance = 100
8. Copy API key from Settings → paste into curl command
9. Call API → usage appears in Usage History
10. Overview shows spending breakdown
```

### Flow C: Low Balance Warning

```
1. Consumer balance drops below 1 XLM
2. Dashboard shows persistent banner: "Balance low (0.50 XLM). [Deposit now]"
3. Calling an API with insufficient balance returns 402 JSON
4. Consumer clicks deposit → funds → resumes usage
```

---

## 6. API Integration Reference

Each screen maps to sqlc queries from `internal/repository/query/`:

| Screen | Query File | Queries Used |
|---|---|---|
| Login/Signup | `users.sql` | `GetUserByEmail`, `CreateUser` |
| Provider Overview | `usage_events.sql`, `settlement_entries.sql` | `AggregateProviderEarnings`, `ListUsageEventsByProvider` |
| My APIs | `providers.sql` | `ListProvidersByOwner` |
| Endpoints | `api_endpoints.sql`, `providers.sql` | `ListEndpointsByProvider`, `GetProviderByID` |
| Analytics | `usage_events.sql` | `AggregateProviderEarnings`, `ListUsageEventsByProvider` |
| Settlements | `settlement_batches.sql`, `settlement_entries.sql` | `ListSettlementBatchesByProvider`, `ListSettlementEntriesByProvider` |
| Settings — API Keys | `api_keys.sql` | `ListAPIKeysByUser`, `RevokeAPIKey`, `CreateAPIKey` |
| Settings — Payout | `users.sql` | `GetUserByID`, `UpdateUserPayoutAddress` |
| Consumer Overview | `accounts.sql`, `usage_events.sql` | `GetAccountByOwnerID`, `ListUsageEventsByConsumer`, `GetConsumerUsageSummary` |
| Deposit | `deposits.sql`, `users.sql` | `ListDepositsByAccount`, `SetUserDepositMemo` |
| Usage History | `usage_events.sql` | `ListUsageEventsByConsumer` |

## 7. Component Reference

### Reusable Components to Build

| Component | Used In | Props |
|---|---|---|
| `StatCard` | Overview (both), Settlements | `label`, `value`, `trend?`, `icon` |
| `DataTable` | Every list screen | `columns`, `data`, `loading`, `emptyMessage`, `onRowClick?` |
| `EmptyState` | All empty states | `title`, `description`, `action?` (CTA button) |
| `PageHeader` | Every page | `title`, `description?`, `actions?` (button group) |
| `BalanceBadge` | Top bar, Overview | `balance`, `currency`, `low?` |
| `QRDisplay` | Deposit | `septUri`, `address`, `memo` |
| `StatusBadge` | Tables | `status` (active/inactive/pending/confirmed/failed) |
| `CopyButton` | Deposit, API Keys | `value`, `label` |
| `DateRangePicker` | Analytics, Usage | `onChange`, `presets` |

### Shared Layout Patterns

- **Stat row:** 4 equal cards on desktop, 2×2 on tablet, scroll on mobile
- **Tables:** Sortable columns, paginated (20 per page), row hover highlight, sticky header
- **Dialogs:** Centered modal with backdrop blur, close on escape + click outside
- **Toasts:** Top-right, auto-dismiss, for deposit confirmations, key generation, errors
