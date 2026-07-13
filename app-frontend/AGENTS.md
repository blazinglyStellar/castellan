<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

## Work State

### Completed
- Phase 1 foundation: auth context, query provider, root layout, proxy middleware, login page, dashboard shell (sidebar + top bar), 14 routes scaffolded
- Phase 2 real pages: overview, usage, analytics, deposits, api-keys, ledger, settings — all with TanStack Query data fetching
- Shared API layer: `src/lib/api/types.ts` (all types), `src/lib/api/endpoints.ts` (all endpoints), format utilities, cursor/offset pagination hooks
- Feature components: usage (method-badge, filter-bar, grouped-table), analytics (9 chart components), deposit (intent card, balance card, projection card, history table), api-keys (CRUD view)

### Phase 3 complete
All 5 remaining pages built: ledger detail, providers CRUD, endpoint config, settlement batches (with expandable rows, threshold, chart), discover browser

### Phase 4 (design polish) — in progress
- Overview: bento-grid rewrite with colored stat cards, BalanceCard with spend threshold bar, usage chart with interval toggle, RecentCalls/CallsDeposits tables
- Badge system: all status badges now use `rounded-full` pill shape with colored dot indicator (green/amber/red per status)
- StatusCodeBadge: pill+dot pattern keyed by HTTP status prefix
- Analytics: colored icon backgrounds on stat cards, section headers, consistent card styling
- Ledger: colored icon backgrounds on summary cards, section headers, section icons
- Settings: colored icon backgrounds on Profile/Deposit/Balance cards, updated skeletons
- Providers: colored icon in table header
- Deposits: colored icon on balance card header, replaced hardcoded button classes with `buttonVariants()`
- Fixed `asChild` violation in analytics provider-forbidden overlay

### Next
- Run `npm run build` before any commit
- Use `src/lib/auth/auth-context` (useAuth), NOT account-context
- Import API functions from `src/lib/api/endpoints`, not client.ts
- Do NOT use `asChild` on base-ui components
- Do NOT use `next/link` or `next/image` — use `<a>` / `<img>`
- Use `buttonVariants()` for `<a>` tags styled as buttons (import from `@/components/ui/button`)
- Design tokens: primary `#4361ee`, secondary `#f72585`, tertiary `#4cc9f0`
- Colored icon bg: `flex h-10 w-10 items-center justify-center rounded-xl bg-{color}-100 text-{color}-600`
