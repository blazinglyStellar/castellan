# Castellan — UI Requirements Document

**Project:** Usage-based API monetization gateway (Stellar settlement, prepaid balances, reverse proxy)
**Status:** Draft v2
**Target:** Complete UI redesign (ignore existing dashboard)

## Resolved Decisions (from Product Owner)

| # | Question | Decision |
|---|---|---|
| 1 | Marketing landing page? | Yes, public page with full content |
| 2 | Email/password in Settings? | No — OAuth only (Google + GitHub). No password fields anywhere |
| 3 | Permanent or regenerable deposit memo? | One permanent memo per user (MVP) |
| 4 | Consumer API discovery? | Yes — add a "Discover" or "Explore" page for consumers |
| 5 | Latency percentiles in Analytics? | No for now |
| 6 | Notifications pattern? | Modeless dialog accessible from top nav (bell icon) or sidebar |
| 7 | Wallet deep-link dropdown? | No — generic SEP-7 URI only |
| 8 | Style config? | CSS-variable-driven theming. Easy to swap colors and toggle dark/light |

---

## 1. Users & Roles

### Two primary roles (users can hold both):

| Role | Identity | Core Job |
|---|---|---|
| **Provider** | API developer, OSS maintainer, AI infra builder | Owns APIs, sets pricing, receives payouts, monitors usage |
| **Consumer** | API client, AI agent operator, developer | Calls APIs, prepays via Stellar, tracks consumption |

### Dual-role user
A single user can be both provider and consumer. The dashboard navigation must show both role groups with clear visual separation (group labels like "Provider" and "Consumer"), and the overview page must surface data relevant to both.

---

## 2. Design Principles

1. **Data-dense & developer-oriented** — Think Vercel x Stripe: high info density, purposeful whitespace, readable tabular + financial data.
2. **Role-aware navigation** — Sidebar adapts. Dual-role users see both sections.
3. **Trust-inspiring financial UI** — Balance, costs, earnings must be unambiguous. Semantic color (green = positive/credit, red = debit/error, amber = pending).
4. **Empty states as onboarding** — Every empty screen guides the user to the next action.
5. **Real-time feedback** — Deposit detection, balance changes, settlement status via toasts + live-updating components.
6. **Copy-friendly** — API keys, addresses, memos, tx hashes all need one-click copy with visual confirmation.

---

## 3. Design System

### Theme
- Default: Dark (`#0a0a0b` bg, `#18181b` surfaces, `#27272a` borders)
- Light theme: white/gray-50 bg, gray-900 text
- **All colors driven by CSS custom properties** in `globals.css` — swapping themes is a single class toggle (`.dark` / `.light` on `<html>`). No hardcoded color values in components.
- Smooth CSS variable transition (300ms ease on `--bg`, `--surface`, `--text-primary` etc.)

### Colors

| Token | Hex | Usage |
|---|---|---|
| `--bg` | `#0a0a0b` | Main background |
| `--surface` | `#18181b` | Cards, sheets, dropdowns |
| `--surface-hover` | `#1f1f23` | Hover on cards/rows |
| `--border` | `#27272a` | Borders, dividers |
| `--text-primary` | `#fafafa` | Primary text |
| `--text-secondary` | `#a1a1aa` | Metadata, muted text |
| `--accent` | `#3b82f6` | Primary actions, links, active |
| `--green` | `#22c55e` | Positive, success, confirmed |
| `--red` | `#ef4444` | Insufficient balance, errors, revoked |
| `--amber` | `#f59e0b` | Pending, warnings, draft |

### Typography
- **UI:** Inter (sans-serif, developer-friendly)
- **Monospace:** JetBrains Mono for addresses, API keys, tx hashes, code
- **Scale:** 12 / 14 / 16 / 18 / 24 / 30 / 36 px

### Component Library
- **shadcn/ui** (Radix primitives) — Card, Table, Dialog, Sheet, Button, Input, Select, Badge, Tabs, Skeleton, Toast, Progress, DropdownMenu, Separator
- **Icons:** Lucide
- **Charts:** Recharts or Tremor for line/area charts
- **QR:** `qrcode.react` for deposit codes

### Reusable Components

| Component | Used In |
|---|---|
| `StatCard` | Overview (both), Settlements |
| `DataTable` | Every list screen |
| `DataTableSkeleton` | Loading state for tables |
| `EmptyState` | All empty states (illustration + CTA) |
| `PageHeader` | Every page (title + description + actions) |
| `BalanceBadge` | Top bar, Overview, low-balance banner |
| `QRDisplay` | Deposit page |
| `StatusBadge` | Tables (status visualization) |
| `StatusDot` | Compact status indicator |
| `CopyButton` | Deposit address, API keys, tx hashes |
| `DateRangePicker` | Analytics, Usage history |
| `RoleToggle` | Signup and settings |
| `GlobalSearch` | Top bar search (scoped by role) |
| `NotificationBell` | Top bar (placeholder) |
| `LowBalanceBanner` | Persistent alert when balance < 1 XLM |
| `KeyDisplay` | One-time reveal modal for new API keys |
| `ConfirmDialog` | Destructive action confirmations |

---

## 4. Navigation / Shell Layout

```
+------------------------------------------------------------------+
| Logo + Name    Global Search         Notifs   User [Balance]     |  Top bar
+--------+---------------------------------------------------------+
|        |                                                          |
| Nav    |  Main Content Area                                       |
| Sidebar|                                                          |
|        |                                                          |
+--------+---------------------------------------------------------+
```

### Sidebar (Role-Aware)

**Provider section (if user has provider role):**
```
Overview          -> /dashboard
My APIs           -> /dashboard/providers
Analytics         -> /dashboard/analytics
Settlements       -> /dashboard/settlements
```

**Consumer section (if user has consumer role):**
```
Overview          -> /dashboard
Discover          -> /dashboard/discover
Deposit           -> /dashboard/deposit
Usage             -> /dashboard/usage
API Keys          -> /dashboard/api-keys
```

**Shared (always visible):**
```
Settings          -> /dashboard/settings
API Docs          -> /docs
```

Dual-role users see both groups separated by a divider with labels. Active route highlighted. Collapsed on smaller screens.

### Top Bar
- **Left:** Castellan logo + wordmark
- **Center:** Global search (scoped by role)
- **Right:** Balance badge (consumer only), notification bell (modeless dialog — opens a slide-over panel from the right showing notification history), user dropdown (Settings, Sign out)

---

## 5. Page Specifications

---

### 5.0 Marketing Landing Page

**Route:** `/`
**Audience:** Unauthenticated visitors, developers discovering Castellan
**Purpose:** Explain the product, drive signups

#### Sections
1. **Hero** — "Usage-based API monetization infrastructure" + tagline + CTA ("Start Building")
2. **How It Works** — 3-step explainer (provider flow: Wrap API > Set Pricing > Get Paid; consumer flow: Get Key > Deposit > Use APIs)
3. **For Providers** — Per-request billing, no Stripe overhead, Stellar settlement, instant payout
4. **For Consumers** — Pay for what you use, no subscriptions, transparent billing
5. **Architecture Diagram** — Client > Castellan Gateway > Your API
6. **Code Example** — curl snippet showing Bearer auth
7. **Open Source** — "Apache 2.0. Self-host or use our cloud."
8. **Footer** — Links to docs, GitHub, status

**Auth state:** Redirect to `/dashboard` if already logged in.

---

### 5.1 Authentication — Login

**Route:** `/login`
**Audience:** Returning users
**Purpose:** Sign in via OAuth

#### Layout
Centered card on blank background with logo. No sidebar.

#### Features
- **OAuth buttons:**
  - "Continue with Google" (Google-branded button)
  - "Continue with GitHub" (GitHub-branded button)
- Divider line with "or"
- Link to signup
- Small note: "No password needed. We use OAuth."

#### States
- **Loading:** Selected OAuth button shows spinner, other button disabled
- **Error:** Inline error ("OAuth login failed. Try again.")
- **Success:** OAuth callback redirect, then redirect to `/dashboard`

---

### 5.2 Authentication — Sign Up

**Route:** `/signup`
**Audience:** New users
**Purpose:** Create account with role selection via OAuth

#### Features
- **OAuth buttons:**
  - "Continue with Google" (Google-branded)
  - "Continue with GitHub" (GitHub-branded)
- **Role selection** — Three styled cards below OAuth buttons:
  - "I provide APIs" (provider)
  - "I use APIs" (consumer)
  - "Both" (provider + consumer)
  - User selects role first, then clicks OAuth button. Role is passed to OAuth callback.
- Link to login

#### On success
- OAuth provider returns, Castellan creates `users` + `accounts` rows with selected role
- Auto-generate initial API key (shown once in modal)
- Redirect to `/dashboard`

#### States
- **Loading:** Selected button shows spinner
- **Error:** Inline errors ("OAuth failed", "Email already registered")
- **No role selected:** OAuth buttons disabled until role is chosen

---

### 5.3 Provider — Overview

**Route:** `/dashboard`
**Audience:** Providers (dashboard home)
**Purpose:** At-a-glance earnings, volume, recent activity

#### Layout
```
+------------------------------------------------------------------+
| Good morning, user@co            Period: Apr 17 - Apr 24          |
+-----------+----------+----------+--------------------------------+
|Total Earn | Requests | Active   | Pending Settlement              |
| XLM 1,245 | This Week| Endpoints| XLM 340                         |
|           | 12,430   | 4 / 6    |                                 |
+-----------+----------+----------+--------------------------------+
| Earnings (Last 7 Days)                                             |
| [Area chart: daily earnings, color-coded by API]                   |
+------------------------------------------------------------------+
| Recent API Calls                      [View All]                   |
| Time  | Endpoint | Cost  | Lat | Consumer   | Status              |
|-------+----------+-------+-----+------------+------               |
| 2min  | /weather | 0.01  |120ms| user@a.co  | 200                 |
| 5min  | /search  | 0.02  | 85ms| user@b.co  | 200                 |
+--------+---------+-------+-----+------------+--------------------+
```

#### States
- **Empty (no APIs):** "Register your first API to get started" + CTA
- **Empty (no usage):** "Share your API key with consumers"
- **Loading:** Skeleton cards + skeleton chart + skeleton rows
- **Error:** Banner "Failed to load. [Retry]"

#### Data
- `AggregateProviderEarnings`, `ListUsageEventsByProvider`, `ListProvidersByOwner`

---

### 5.4 Consumer — Overview

**Route:** `/dashboard`
**Audience:** Consumers (dashboard home)
**Purpose:** Balance, recent spending, top providers

#### Layout
```
+------------------------------------------------------------------+
| Welcome back                         Balance: XLM 1,250           |
| [Deposit]           Spent this month: XLM 430.20                  |
+--------------------------+----------------------------------------+
| Recent Usage             | Top Providers by Spend                 |
| API       | Cost | Time  | Weather API         XLM 230.50         |
|-----------+------+-------| Search API          XLM 180.20         |
| /weather  | 0.01 | 2min  | Image Gen           XLM  19.50         |
| /query    | 0.02 | 5min  |                                        |
+-----------+------+-------+----------------------------------------+
| Usage Trend (Last 7 Days)                                          |
| [Area chart: requests/day]                                         |
+------------------------------------------------------------------+
```

#### States
- **Empty (no deposits):** Balance = 0. "Deposit funds to get started." CTA
- **Empty (deposits but no usage):** "Start calling APIs to see usage"
- **Loading:** Skeleton cards + skeleton chart
- **Low balance (< 1 XLM):** Persistent amber banner on all pages

#### Dual-role view
Both sections stacked vertically with headers.

#### Data
- `GetAccountByOwnerID`, `ListUsageEventsByConsumer`, `GetConsumerUsageSummary`

---

### 5.5 Provider — My APIs

**Route:** `/dashboard/providers`
**Audience:** Providers
**Purpose:** List, add, manage API services

#### Features
- Table: Name | Base URL | Endpoints count | Status | Actions
- **Add API dialog:** Name, Base URL, optional description
- **Row actions:** Edit, View Endpoints, Delete (with confirmation)
- Click row to navigate to endpoints

#### States
- **Empty:** "You haven't registered any APIs yet." + [Add API]
- **Loading:** 3 skeleton rows

#### Data
- `ListProvidersByOwner`

---

### 5.6 Provider — Endpoints

**Route:** `/dashboard/providers/[id]/endpoints`
**Audience:** Providers
**Purpose:** Manage routes, pricing, status per API

#### Features
- Provider header: name, status, base URL, [Back] button
- Table: Method | Route | Price | Rate limit | Status | Actions
- **Add Endpoint dialog:** Method (select), Route (text), Price (number + XLM), Rate limit (optional), Status toggle
- **Row actions:** Edit, Toggle Status, Delete
- **Draft rows** visually distinct (grey text, amber badge)
- **Summary line:** "3 active / 1 draft / 1 inactive"
- Bulk import placeholder for future OpenAPI spec upload

#### States
- **Empty:** "No endpoints yet. Add your first route."
- **Loading:** Skeleton table

#### Data
- `ListEndpointsByProvider`, `GetProviderByID`

---

### 5.7 Provider — Analytics

**Route:** `/dashboard/analytics`
**Audience:** Providers
**Purpose:** Deep usage metrics across all APIs

#### Features
- **Filters:** Time range (24h/7d/30d/90d/custom), API filter
- **Chart 1:** Requests Over Time (area chart, color by endpoint)
- **Chart 2:** Revenue Over Time (area chart, color by endpoint)
- **Breakdown table:** Endpoint | Requests | Revenue | Avg Latency | Success Rate
- Sortable columns
- Export (CSV — future)

#### States
- **Empty:** "No usage data for the selected period."
- **Loading:** Skeleton chart + skeleton table

#### Data
- `AggregateProviderEarnings` (by day), `ListUsageEventsByProvider` (grouped)

---

### 5.8 Provider — Settlements

**Route:** `/dashboard/settlements`
**Audience:** Providers
**Purpose:** Payout history and pending earnings

#### Features
- **Stat cards:** Outstanding payout | Last payout | Next est. payout | Total all time
- **History table:** Date | Amount | Status (badge) | Tx Hash (truncated + copy) | Actions
- Paginated or "Load More"
- Status: Sent (green), Pending (amber), Failed (red)

#### States
- **Empty:** "No settlements yet."
- **Loading:** Skeleton cards + skeleton table

#### Data
- `ListSettlementBatchesByProvider`, `AggregateProviderEarnings`

---

### 5.9 Consumer — Deposit

**Route:** `/dashboard/deposit`
**Audience:** Consumers
**Purpose:** Fund account via Stellar

#### Features
- **QR code** (SEP-7 URI: `web+stellar:pay?destination=...&memo=...&memo_type=text`)
- **Address display** + copy button
- **Memo display** + copy button
- Note: "Scan with any Stellar wallet that supports SEP-7"
- Minimum deposit notice (5 XLM)
- **Recent deposits table:** Amount | Status | Tx Hash | Date
- **Live deposit detection:** Poll every 5s, toast on detection, row flips from pending to confirmed

#### States
- **Loading:** QR skeleton + address skeletons
- **Empty:** "No deposits yet. Send XLM to the address above."
- **Live status:** Polling spinner, toast notification

#### Data
- SEP-7 URI from backend endpoint
- `ListDepositsByAccount`

---

### 5.10 Consumer — Usage History

**Route:** `/dashboard/usage`
**Audience:** Consumers
**Purpose:** Full request log

#### Features
- **Filters:** Time range (presets + custom date picker), Provider (multi-select), Status code filter
- **Search:** Top bar global search scoped to usage
- **Sortable columns**
- **Pagination** (20/page)
- **Export** (CSV — future)
- Status color: 2xx green, 4xx amber, 5xx red. 402 = "Payment Required" badge

#### States
- **Empty:** "No API calls yet. Get an API key." + link to API Keys
- **Loading:** Skeleton rows

#### Data
- `ListUsageEventsByConsumer` (paginated, filterable)

---

### 5.11 Consumer — API Keys

**Route:** `/dashboard/api-keys`
**Audience:** Consumers
**Purpose:** Manage API keys for authentication

#### Features
- Table: Label | Key Prefix | Status (badge) | Created | Actions
- **Generate new key dialog:** Label input, optional expiry
- **Key reveal modal:** Full key shown once + copy button + "This key will not be shown again" warning
- **Revoke:** Confirm dialog, then update status
- **Delete:** Confirm dialog (revoked/expired only)
- Status: Active (green), Revoked (red), Expired (grey)

#### States
- **Empty:** "No API keys yet. Generate one." + [New Key]
- **Loading:** Skeleton table

#### Data
- `ListAPIKeysByUser`, `CreateAPIKey`, `RevokeAPIKey`

---

### 5.15 Settings (Shared)

**Route:** `/dashboard/settings`
**Audience:** Both
**Purpose:** Account management

#### Tabs

**Tab 1 — Profile**
- Email (display, sourced from OAuth)
- Avatar (display, sourced from OAuth)
- Role badges: Provider + Consumer (or single role)
- "Connected with Google/GitHub" badge (which OAuth provider linked)
- Theme toggle (dark/light)
- No email/password change fields

**Tab 2 — Payout Address (provider only)**
- Stellar payout address input + save
- Address format validation
- Help text

**Tab 3 — Deposit Info (consumer only)**
- Deposit memo (display + copy)
- Castellan deposit address (display + copy)
- Minimum deposit notice

**Tab 4 — API Keys (consumer only)**
- Same as 5.11 (or redirect)

**Tab 5 — Notifications (future)**
- Placeholder for email/opt-in notification toggles

#### Data
- `GetUserByID`, `UpdateUserPayoutAddress`, `ListAPIKeysByUser`

---

### 5.13 Consumer — Discover APIs

**Route:** `/dashboard/discover`
**Audience:** Consumers
**Purpose:** Browse available providers and their endpoints to discover APIs to call

#### Layout
```
+------------------------------------------------------------------+
| Discover APIs                   [Search...]                       |
+------------------------------------------------------------------+
| +---------------------------------------------------------------+ |
| | Weather API                                   3 endpoints     | |
| | https://api.weather.com                         Active        | |
| | /current  - GET  - 0.0001 XLM  - 100/min                     | |
| | /forecast - GET  - 0.0002 XLM  - 60/min                      | |
| | /alerts   - POST - 0.0005 XLM  - 10/min                      | |
| | [Copy curl] [View Docs]                                       | |
| +---------------------------------------------------------------+ |
| +---------------------------------------------------------------+ |
| | Search API                                    2 endpoints     | |
| | https://api.search.com                          Active        | |
| | /query - GET - 0.0002 XLM - 200/min                           | |
| | [Copy curl] [View Docs]                                       | |
| +---------------------------------------------------------------+ |
+------------------------------------------------------------------+
```

#### Features
- **Filter/search:** Text search across provider names, endpoint routes, descriptions
- **Provider cards** each showing: provider name, base URL, status badge, endpoint count
- **Expandable endpoints** within each card: method badge, route path, price per request, rate limit
- **Quick actions per provider:** "Copy curl example" (pre-filled with a consumer's active API key), "View Docs" (link to external docs if configured)
- Provider list is all active providers with at least one active endpoint

#### States
- **Empty:** "No APIs available yet. Check back later for new providers."
- **Loading:** Skeleton provider cards
- **No search results:** "No providers match your search."

#### Data
- `ListActiveProviders` (all providers with active status)
- `ListActiveEndpointsByProvider` (only active endpoints)

---

### 5.14 API Docs

**Route:** `/docs`
**Audience:** Both
**Purpose:** Developer API reference

#### Features
- Scalar/OpenAPI reference viewer
- Sections: Auth, Proxy Requests, Wallet, Providers, Endpoints
- Code examples: curl, JS, Python, Go

---

### 5.16 Low Balance Banner (Global)

Not a page — a persistent consumer dashboard element.

- When `account.balance < 1 XLM`
- Amber background with icon: "Balance low (XLM 0.50). [Deposit now]"
- Link to `/dashboard/deposit`
- Dismissible per session

---

## 6. User Flows

### Flow A: Provider Onboarding
1. Sign up > select "I provide APIs"
2. Empty overview > "Add API" dialog > name + base_url
3. Endpoints page (empty) > "Add Endpoint" > route, method, price, rate
4. Endpoint active > gateway starts billing
5. Settings > set Stellar payout address
6. Settlements populate after batch worker runs

### Flow B: Consumer Onboarding
1. Sign up > select "I use APIs"
2. Empty overview (balance = 0)
3. Deposit page > QR + address + memo
4. Scan QR with Stellar wallet > send XLM
5. ~5-10s later: toast "Deposit confirmed!" > balance updates
6. Generate API key > use in curl
7. Usage appears in history > overview shows spending

### Flow C: Low Balance > Deposit
1. Balance drops below 1 XLM
2. Amber banner persists across all pages
3. Calling APIs returns 402 Payment Required
4. Click banner or navigate to Deposit > send funds
5. Detected > toast + balance updates > banner disappears

### Flow D: CLI Import > Dashboard Review
1. `castellan import --provider-id <id> ./openapi.yaml`
2. Endpoints imported as status=draft
3. Dashboard shows Draft badge > set prices > toggle Active
4. Gateway starts routing

---

## 7. Role-Aware Routing Summary

| Route | Provider | Consumer | Dual-Role |
|---|---|---|---|
| `/dashboard` | Provider overview | Consumer overview | Both stacked |
| `/dashboard/providers` | API list | Locked | API list |
| `/dashboard/providers/[id]/endpoints` | Endpoint manager | Locked | Endpoint manager |
| `/dashboard/analytics` | Analytics | Locked | Analytics |
| `/dashboard/settlements` | Settlements | Locked | Settlements |
| `/dashboard/discover` | Locked | Discover APIs | Discover APIs |
| `/dashboard/deposit` | Locked | Deposit | Deposit |
| `/dashboard/usage` | Locked | Usage history | Usage history |
| `/dashboard/api-keys` | Locked | API key manager | API key manager |
| `/dashboard/settings` | Full settings | Full settings | Full settings |
| `/docs` | API reference | API reference | API reference |

Locked = sidebar hidden, route redirects to `/dashboard`.

---

## 8. Page Inventory & Priority

| Priority | Page | Route | Effort | Value |
|---|---|---|---|---|
| P0 | Login | `/login` | Low | Auth gate |
| P0 | Sign Up | `/signup` | Low | User acquisition |
| P0 | Dashboard Shell | `/dashboard/layout` | Medium | Navigation foundation |
| P0 | Provider Overview | `/dashboard` | Medium | Primary provider home |
| P0 | Consumer Overview | `/dashboard` | Medium | Primary consumer home |
| P0 | My APIs | `/dashboard/providers` | Medium | API management |
| P0 | Endpoints | `/dashboard/providers/[id]/endpoints` | High | Core pricing config |
| P0 | Settings | `/dashboard/settings` | Medium | Account management |
| P0 | API Keys | `/dashboard/api-keys` | Medium | Auth credentials |
| P1 | Discover APIs | `/dashboard/discover` | Medium | Consumer API discovery |
| P1 | Deposit | `/dashboard/deposit` | High | Funding flow |
| P1 | Usage History | `/dashboard/usage` | Medium | Consumption visibility |
| P1 | Analytics | `/dashboard/analytics` | High | Deep insights |
| P1 | Settlements | `/dashboard/settlements` | Medium | Payout transparency |
| P2 | Marketing Landing | `/` | Low | Public presence |
| P2 | API Docs | `/docs` | Low | Developer reference |

---

## 9. Key UX Requirements

1. **Every data table** must have: sortable columns, pagination (20/page), hover highlight, sticky header, loading skeleton, empty state
2. **Every financial figure** must show: formatted number, currency suffix (XLM), consistent decimals
3. **Every action** must show: loading state on trigger button, success toast, error toast on failure
4. **Every destructive action** (delete, revoke) must have: confirmation dialog
5. **Every copyable value** (key, address, memo, hash) must have: inline copy button with "Copied!" tooltip
6. **Navigation must be role-aware** — hide inaccessible links, redirect unauthorized routes
7. **Dual-role users** must have clear visual separation between Provider and Consumer nav groups
8. **Low-balance detection** must be global across all dashboard pages
9. **Deposit flow** must feel real-time: polling, toast, live badge update
10. **API keys** shown exactly once with prominent warning and no retrieval after dismissal

---

## 10. Open Questions (Remaining Unknowns)

1. Should the landing page include a live demo/sandbox, or just marketing content?
2. Rate limit display — show as human-readable ("100 req/min") or technical (burst/refill config)?
3. Should the Discover page show price-per-request prominently, or bury it in expandable details?
4. Any plans for team/multi-user accounts, or is every account single-user?
