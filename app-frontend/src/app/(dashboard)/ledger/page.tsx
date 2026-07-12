"use client"

import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  Wallet,
  DollarSign,
  Banknote,
  TrendingUp,
  BarChart3,
  BookOpen,
  Building2,
  Activity,
  ChevronDown,
  ChevronRight,
  ArrowUpDown,
} from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import {
  getBalance,
  getAccountEntries,
  getUsage,
  getEarnings,
  getSettlements,
} from "@/lib/api/endpoints"
import { useOffsetPagination } from "@/lib/use-offset-pagination"
import { useCursorPagination } from "@/lib/use-cursor-pagination"
import type {
  EntryResponse,
  UsageEvent,
  SettlementBatch,
  SettlementEntry,
} from "@/lib/api/types"
import { formatAmount, timeAgo, StatusBadge } from "@/lib/format"
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart"
import { UsageBarChart } from "@/components/analytics/usage-bar-chart"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/shared/empty-state"
import { ErrorState } from "@/components/shared/error-state"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const ENTRY_TYPES = [
  { value: "deposit", label: "Deposit" },
  { value: "reservation", label: "Reservation" },
  { value: "deduction", label: "Deduction" },
  { value: "refund", label: "Refund" },
  { value: "settlement", label: "Settlement" },
] as const

function defaultDateRange() {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 30)
  return {
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10),
  }
}

const iconBgMap: Record<string, string> = {
  wallet: "bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400",
  spend: "bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400",
  calls: "bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400",
  earnings: "bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400",
  unsettled: "bg-orange-100 text-orange-600 dark:bg-orange-950 dark:text-orange-400",
}

export default function UnifiedLedgerPage() {
  const { user, isLoading: isAccountLoading } = useAuth()
  const role = user?.role

  const [typeFilter, setTypeFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [startDate, setStartDate] = useState(defaultDateRange().start)
  const [endDate, setEndDate] = useState(defaultDateRange().end)

  const resolvedType = typeFilter || undefined
  const resolvedStatus = statusFilter || undefined

  const balanceQuery = useQuery({
    queryKey: ["balance"],
    queryFn: getBalance,
    enabled: role === "consumer",
  })

  const earningsQuery = useQuery({
    queryKey: ["earnings"],
    queryFn: () => getEarnings(),
  })

  const consumerUsageQuery = useQuery({
    queryKey: ["usage", "consumer-chart", startDate, endDate],
    queryFn: () =>
      getUsage({
        role: "consumer",
        start_date: `${startDate}T00:00:00Z`,
        end_date: `${endDate}T23:59:59Z`,
        limit: 500,
      }),
  })

  const providerUsageQuery = useQuery({
    queryKey: ["usage", "provider-chart", startDate, endDate],
    queryFn: () =>
      getUsage({
        role: "provider",
        start_date: `${startDate}T00:00:00Z`,
        end_date: `${endDate}T23:59:59Z`,
        limit: 500,
      }),
  })

  const {
    items: entries,
    isLoading: entriesLoading,
    total: entriesTotal,
    page: entriesPage,
    totalPages: entriesTotalPages,
    setPage: setEntriesPage,
    error: entriesError,
    refresh: entriesRefresh,
  } = useOffsetPagination<EntryResponse>({
    queryKey: ["account-entries", typeFilter],
    fetchFn: (p) =>
      getAccountEntries({ ...p, type: resolvedType }).then((r) => ({
        data: r.entries,
        total: r.total,
      })),
    initialPageSize: 20,
  })

  const {
    items: settlements,
    isLoading: settlementsLoading,
    isLoadingMore: settlementsLoadingMore,
    hasMore: settlementsHasMore,
    loadMore: settlementsLoadMore,
    refresh: settlementsRefresh,
    error: settlementsError,
  } = useCursorPagination<SettlementBatch>({
    queryKey: ["settlements", statusFilter ?? ""],
    fetchFn: (cp) => getSettlements({ ...cp, status: resolvedStatus }),
    limit: 20,
  })

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  const isConsumer = role === "consumer"
  const isProvider = role === "provider"

  const allDataLoading =
    (isConsumer &&
      (balanceQuery.isLoading ||
        consumerUsageQuery.isLoading ||
        entriesLoading)) ||
    (isProvider &&
      (earningsQuery.isLoading ||
        providerUsageQuery.isLoading ||
        settlementsLoading))

  const allDataError =
    (isConsumer &&
      (balanceQuery.isError ||
        consumerUsageQuery.isError ||
        entriesError)) ||
    (isProvider &&
      (earningsQuery.isError ||
        providerUsageQuery.isError ||
        settlementsError))

  if (allDataLoading) {
    return <LoadingSkeleton role={role} />
  }

  if (allDataError) {
    const errMsg =
      balanceQuery.error instanceof Error
        ? balanceQuery.error.message
        : earningsQuery.error instanceof Error
          ? earningsQuery.error.message
          : "Failed to load ledger data"
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          balanceQuery.refetch()
          earningsQuery.refetch()
          consumerUsageQuery.refetch()
          providerUsageQuery.refetch()
          entriesRefresh()
          settlementsRefresh()
        }}
      />
    )
  }

  const balance = balanceQuery.data
  const earnings = earningsQuery.data
  const consumerUsageEvents = consumerUsageQuery.data?.data ?? []
  const totalSpend = consumerUsageEvents.reduce(
    (s, e) => s + parseFloat(e.request_cost),
    0,
  )

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Ledger</h1>
        <p className="text-sm text-muted-foreground">
          {isConsumer && isProvider
            ? "Account activity, settlements, and financial overview."
            : isConsumer
              ? "View account ledger entries, balance history, and spending."
              : "View settlement payouts, earnings, and reconciliation."}
        </p>
      </div>

      <SummaryCards
        role={role}
        balance={balance}
        earnings={earnings}
        totalSpend={totalSpend}
        usageEvents={consumerUsageEvents}
      />

      <ChartsSection
        startDate={startDate}
        endDate={endDate}
        onStartDateChange={setStartDate}
        onEndDateChange={setEndDate}
        consumerUsageEvents={consumerUsageEvents}
        earningsSparkline={earnings?.sparkline ?? []}
        isProvider={isProvider}
      />

      {isConsumer && (
        <AccountActivitySection
          entries={entries}
          isLoading={entriesLoading}
          error={entriesError}
          typeFilter={typeFilter}
          onTypeFilterChange={setTypeFilter}
          page={entriesPage}
          totalPages={entriesTotalPages}
          total={entriesTotal}
          onPageChange={setEntriesPage}
          onRetry={() => entriesRefresh()}
        />
      )}

      {isProvider && (
        <SettlementsSection
          settlements={settlements}
          isLoading={settlementsLoading}
          isLoadingMore={settlementsLoadingMore}
          hasMore={settlementsHasMore}
          loadMore={settlementsLoadMore}
          error={settlementsError}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          onRetry={() => settlementsRefresh()}
        />
      )}
    </div>
  )
}

// ── Summary Cards ──

function SummaryCards({
  role,
  balance,
  earnings,
  totalSpend,
  usageEvents,
}: {
  role?: string
  balance?: { balance: string; available_balance: string; currency: string }
  earnings?: { total_earnings: string; unsettled_earnings: string; currency: string }
  totalSpend: number
  usageEvents: UsageEvent[]
}) {
  const isConsumer = role === "consumer"
  const isProvider = role === "provider"

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {isConsumer && balance && (
        <BalanceCard
          balance={balance.balance}
          available={balance.available_balance}
        />
      )}
      {isConsumer && (
        <StatCard
          icon={<DollarSign className="h-5 w-5" />}
          iconClass={iconBgMap.spend}
          title="Spent This Period"
        >
          {usageEvents.length > 0 ? (
            <>{formatAmount(String(totalSpend))} <span className="text-sm font-normal text-muted-foreground">XLM</span></>
          ) : "\u2014"}
        </StatCard>
      )}
      {isConsumer && (
        <StatCard
          icon={<Activity className="h-5 w-5" />}
          iconClass={iconBgMap.calls}
          title="API Calls This Period"
        >
          {usageEvents.length > 0
            ? usageEvents.length.toLocaleString()
            : "\u2014"}
        </StatCard>
      )}
      {isProvider && earnings && (
        <StatCard
          icon={<TrendingUp className="h-5 w-5" />}
          iconClass={iconBgMap.earnings}
          title="Total Earnings"
        >
          {formatAmount(earnings.total_earnings)} <span className="text-sm font-normal text-muted-foreground">XLM</span>
        </StatCard>
      )}
      {isProvider && earnings && (
        <StatCard
          icon={<Banknote className="h-5 w-5" />}
          iconClass={iconBgMap.unsettled}
          title="Unsettled Earnings"
        >
          {formatAmount(earnings.unsettled_earnings)} <span className="text-sm font-normal text-muted-foreground">XLM</span>
        </StatCard>
      )}
    </div>
  )
}

function BalanceCard({
  balance,
  available,
}: {
  balance: string
  available: string
}) {
  const b = parseFloat(balance)
  const a = parseFloat(available)
  const reserved = b - a
  const pct = b > 0 ? Math.min((a / b) * 100, 100) : 0

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
          <Wallet className="h-5 w-5" />
        </div>
        <CardTitle className="text-sm font-medium">Balance</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-bold tracking-tight">
          {formatAmount(balance)} <span className="text-sm font-normal text-muted-foreground">XLM</span>
        </p>
        <div className="mt-3 space-y-1.5">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Available</span>
            <span className="font-medium">
              {formatAmount(available)} <span className="text-muted-foreground">XLM</span>
            </span>
          </div>
          {reserved > 0 && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Reserved</span>
              <span className="font-medium">
                {formatAmount(reserved.toFixed(7))} <span className="text-muted-foreground">XLM</span>
              </span>
            </div>
          )}
          <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function StatCard({
  icon,
  iconClass,
  title,
  children,
}: {
  icon: React.ReactNode
  iconClass: string
  title: string
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3 pb-2">
        <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${iconClass}`}>
          {icon}
        </div>
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-bold tracking-tight">{children}</p>
      </CardContent>
    </Card>
  )
}

function XlmLogo({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 236.36 200"
      className={className ?? "inline-block h-3 w-auto"}
      fill="currentColor"
    >
      <path d="M203,26.16l-28.46,14.5-137.43,70a82.49,82.49,0,0,1-.7-10.69A81.87,81.87,0,0,1,158.2,28.6l16.29-8.3,2.43-1.24A100,100,0,0,0,18.18,100q0,3.82.29,7.61a18.19,18.19,0,0,1-9.88,17.58L0,129.57V150l25.29-12.89,0,0,8.19-4.18,8.07-4.11v0L186.43,55l16.28-8.29,33.65-17.15V9.14Z" />
      <path d="M236.36,50,49.78,145,33.5,153.31,0,170.38v20.41l33.27-16.95,28.46-14.5L199.3,89.24A83.45,83.45,0,0,1,200,100,81.87,81.87,0,0,1,78.09,171.36l-1,.53-17.66,9A100,100,0,0,0,218.18,100c0-2.57-.1-5.14-.29-7.68a18.2,18.2,0,0,1,9.87-17.58l8.6-4.38Z" />
    </svg>
  )
}

// ── Charts Section ──

function ChartsSection({
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  consumerUsageEvents,
  earningsSparkline,
  isProvider,
}: {
  startDate: string
  endDate: string
  onStartDateChange: (d: string) => void
  onEndDateChange: (d: string) => void
  consumerUsageEvents: UsageEvent[]
  earningsSparkline: { date: string; amount: string }[]
  isProvider?: boolean
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <div className="flex flex-row items-center gap-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
            <BarChart3 className="h-4 w-4" />
          </div>
          <CardTitle className="text-sm font-medium">
            Spending & Earnings
          </CardTitle>
        </div>
        <div className="flex items-end gap-3">
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">From</span>
            <input
              type="date"
              value={startDate}
              onChange={(e) => onStartDateChange(e.target.value)}
              className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
            />
          </div>
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">To</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => onEndDateChange(e.target.value)}
              className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
            />
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid gap-6 lg:grid-cols-2">
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">
              Spending
            </p>
            <UsageVolumeChart events={consumerUsageEvents} />
          </div>
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">
              Earnings
            </p>
            <UsageBarChart data={earningsSparkline} isProvider={isProvider} />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// ── Sortable Table Head ──

function SortableHeadComponent({
  label,
  sortKey: sk,
  currentSortKey,
  onToggle,
  className,
}: {
  label: string
  sortKey: string
  currentSortKey: string
  onToggle: (key: string) => void
  className?: string
}) {
  const isActive = currentSortKey === sk
  return (
    <TableHead className={className}>
      <Button
        variant="ghost"
        size="sm"
        className="-ml-3 h-8 gap-1 text-xs font-medium hover:bg-transparent"
        onClick={() => onToggle(sk)}
      >
        {label}
        <ArrowUpDown
          className={`h-3 w-3 ${isActive ? "opacity-100" : "opacity-30"}`}
        />
      </Button>
    </TableHead>
  )
}

// ── Account Activity Section (Consumer) ──

function AccountActivitySection({
  entries,
  isLoading,
  error,
  typeFilter,
  onTypeFilterChange,
  page,
  totalPages,
  total,
  onPageChange,
  onRetry,
}: {
  entries: EntryResponse[]
  isLoading: boolean
  error: Error | null
  typeFilter: string
  onTypeFilterChange: (v: string) => void
  page: number
  totalPages: number
  total: number
  onPageChange: (p: number) => void
  onRetry: () => void
}) {
  const [sortKey, setSortKey] = useState<string>("created_at")
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc")
  const [activityStatusFilter, setActivityStatusFilter] = useState("")
  const [activityStartDate, setActivityStartDate] = useState(
    defaultDateRange().start,
  )
  const [activityEndDate, setActivityEndDate] = useState(
    defaultDateRange().end,
  )

  function toggleSort(key: string) {
    if (sortKey === key) {
      setSortDir(sortDir === "asc" ? "desc" : "asc")
    } else {
      setSortKey(key)
      setSortDir(key === "amount" || key === "balance_after" ? "desc" : "asc")
    }
  }

  const sortedEntries = useMemo(() => {
    const sorted = [...entries]
    sorted.sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case "entry_type":
          cmp = a.entry_type.localeCompare(b.entry_type)
          break
        case "amount":
          cmp = parseFloat(a.amount) - parseFloat(b.amount)
          break
        case "balance_after":
          cmp = parseFloat(a.balance_after) - parseFloat(b.balance_after)
          break
        case "status":
          cmp = a.status.localeCompare(b.status)
          break
        case "created_at":
          cmp = a.created_at.localeCompare(b.created_at)
          break
      }
      return sortDir === "asc" ? cmp : -cmp
    })
    return sorted
  }, [entries, sortKey, sortDir])

  const filteredByStatus = useMemo(() => {
    if (!activityStatusFilter) return sortedEntries
    return sortedEntries.filter((e) => e.status === activityStatusFilter)
  }, [sortedEntries, activityStatusFilter])

  const filteredByDate = useMemo(() => {
    const start = new Date(activityStartDate)
    const end = new Date(activityEndDate + "T23:59:59Z")
    return filteredByStatus.filter((e) => {
      const d = new Date(e.created_at)
      return d >= start && d <= end
    })
  }, [filteredByStatus, activityStartDate, activityEndDate])

  const entriesToShow = filteredByDate

  function formatShortDate(iso: string): string {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return iso
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400">
          <BookOpen className="h-4 w-4" />
        </div>
        <h2 className="text-sm font-medium">Account Activity</h2>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Type:</span>
        <Select value={typeFilter} onValueChange={(v) => onTypeFilterChange(v ?? "")}>
          <SelectTrigger className="w-40 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Types</SelectItem>
            {ENTRY_TYPES.map((t) => (
              <SelectItem key={t.value} value={t.value}>
                {t.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-sm text-muted-foreground">Status:</span>
        <Select
          value={activityStatusFilter}
          onValueChange={(v) => setActivityStatusFilter(v ?? "")}
        >
          <SelectTrigger className="w-40 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All Statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Statuses</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
            <SelectItem value="revoked">Revoked</SelectItem>
          </SelectContent>
        </Select>
        <div className="ml-auto flex items-end gap-3">
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">From</span>
            <input
              type="date"
              value={activityStartDate}
              onChange={(e) => setActivityStartDate(e.target.value)}
              className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
            />
          </div>
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">To</span>
            <input
              type="date"
              value={activityEndDate}
              onChange={(e) => setActivityEndDate(e.target.value)}
              className="h-8 rounded-lg border border-input bg-transparent px-2.5 text-sm transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
            />
          </div>
        </div>
      </div>

      {isLoading ? (
        <ActivityTableSkeleton />
      ) : error ? (
        <ErrorState message={error.message} onRetry={onRetry} />
      ) : entriesToShow.length === 0 ? (
        <EmptyState
          title="No entries yet"
          description="Ledger entries will appear here once account activity begins."
        />
      ) : (
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <CardTitle className="text-sm font-medium">Entries</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <SortableHeadComponent label="Type" sortKey="entry_type" currentSortKey={sortKey} onToggle={toggleSort} />
                  <SortableHeadComponent label="Amount" sortKey="amount" currentSortKey={sortKey} onToggle={toggleSort} />
                  <SortableHeadComponent label="Balance After" sortKey="balance_after" currentSortKey={sortKey} onToggle={toggleSort} />
                  <TableHead className="text-xs font-medium text-muted-foreground">
                    Description
                  </TableHead>
                  <SortableHeadComponent label="Status" sortKey="status" currentSortKey={sortKey} onToggle={toggleSort} />
                  <SortableHeadComponent label="Date" sortKey="created_at" currentSortKey={sortKey} onToggle={toggleSort} />
                </TableRow>
              </TableHeader>
              <TableBody>
                {entriesToShow.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>
                      <EntryTypeBadge type={entry.entry_type} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap font-mono text-xs">
                      {formatAmount(entry.amount)}{" "}
                      <span className="text-muted-foreground">
                        <XlmLogo className="inline-block h-2.5 w-auto" />
                      </span>
                    </TableCell>
                    <TableCell className="whitespace-nowrap font-mono text-xs text-muted-foreground">
                      {formatAmount(entry.balance_after)}{" "}
                      <span className="text-muted-foreground">
                        <XlmLogo className="inline-block h-2.5 w-auto" />
                      </span>
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate text-xs text-muted-foreground">
                      {entry.description || "\u2014"}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={entry.status} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {formatShortDate(entry.created_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
          {total > 20 && (
            <div className="flex items-center justify-between border-t px-6 py-3">
              <p className="text-sm text-muted-foreground">
                {total} total entries
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => onPageChange(page - 1)}
                >
                  Prev
                </Button>
                <span className="text-sm text-muted-foreground">
                  Page {page} of {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => onPageChange(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      )}
    </div>
  )
}

function EntryTypeBadge({ type }: { type: string }) {
  const entry = ENTRY_TYPES.find((t) => t.value === type)
  const label = entry?.label ?? type

  const colors: Record<string, string> = {
    deposit: "text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-950",
    deduction: "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-950",
    reservation: "text-blue-700 bg-blue-50 dark:text-blue-300 dark:bg-blue-950",
    refund: "text-purple-700 bg-purple-50 dark:text-purple-300 dark:bg-purple-950",
    settlement: "text-yellow-700 bg-yellow-50 dark:text-yellow-300 dark:bg-yellow-950",
  }

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold capitalize ${colors[type] || "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950"}`}
    >
      <span className={`size-1.5 rounded-full ${type === "deposit" ? "bg-green-500" : type === "deduction" ? "bg-red-500" : type === "reservation" ? "bg-blue-500" : type === "refund" ? "bg-purple-500" : type === "settlement" ? "bg-yellow-500" : "bg-gray-400"}`} />
      {label}
    </span>
  )
}

// ── Settlements Section (Provider) ──

function SettlementsSection({
  settlements,
  isLoading,
  isLoadingMore,
  hasMore,
  loadMore,
  error,
  statusFilter,
  onStatusFilterChange,
  onRetry,
}: {
  settlements: SettlementBatch[]
  isLoading: boolean
  isLoadingMore: boolean
  hasMore: boolean
  loadMore: () => void
  error: Error | null
  statusFilter: string
  onStatusFilterChange: (v: string) => void
  onRetry: () => void
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-orange-100 text-orange-600 dark:bg-orange-950 dark:text-orange-400">
          <Building2 className="h-4 w-4" />
        </div>
        <h2 className="text-sm font-medium">Settlements</h2>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Status:</span>
        <Select value={statusFilter} onValueChange={(v) => onStatusFilterChange(v ?? "")}>
          <SelectTrigger className="w-40 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All Statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Statuses</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <SettlementSkeleton />
      ) : error ? (
        <ErrorState message={error.message} onRetry={onRetry} />
      ) : settlements.length === 0 ? (
        <EmptyState
          title="No settlements yet"
          description="Settlement batches will appear here once payouts are processed."
        />
      ) : (
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <CardTitle className="text-sm font-medium">
              Settlement Batches
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" />
                  <TableHead>Batch ID</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Entries</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Completed</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {settlements.map((batch) => (
                  <SettlementBatchRow key={batch.id} batch={batch} />
                ))}
              </TableBody>
            </Table>

            {hasMore && (
              <div className="flex justify-center border-t py-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={loadMore}
                  disabled={isLoadingMore}
                >
                  {isLoadingMore ? "Loading..." : "Load More"}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function SettlementBatchRow({ batch }: { batch: SettlementBatch }) {
  const [expanded, setExpanded] = useState(false)

  const providerGroups = useMemo(() => {
    const groups = new Map<string, SettlementEntry[]>()
    for (const entry of batch.entries) {
      const key = entry.provider_name || entry.provider_id
      const list = groups.get(key) ?? []
      list.push(entry)
      groups.set(key, list)
    }
    return Array.from(groups.entries()).map(([key, entries]) => ({
      key,
      name: entries[0].provider_name || key,
      entries,
      total: entries.reduce((s, e) => s + parseFloat(e.amount), 0),
      count: entries.length,
      status: entries[0].status,
    }))
  }, [batch.entries])

  return (
    <>
      <TableRow
        className="cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <TableCell className="w-8">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </TableCell>
        <TableCell className="font-mono text-xs">{batch.id}</TableCell>
        <TableCell className="whitespace-nowrap font-mono text-xs">
          {batch.total_amount}{" "}
          <span className="text-muted-foreground">{batch.currency}</span>
        </TableCell>
        <TableCell>
          <StatusBadge status={batch.status} />
        </TableCell>
        <TableCell className="text-xs text-muted-foreground">
          {batch.entry_count}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {timeAgo(batch.created_at)}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {batch.completed_at ? timeAgo(batch.completed_at) : "\u2014"}
        </TableCell>
      </TableRow>
      {expanded && (
        <TableRow>
          <TableCell colSpan={7} className="bg-muted/30 p-0">
            <div className="px-6 py-4">
              {providerGroups.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No entries in this batch.
                </p>
              ) : (
                <Table>
                    <TableHeader>
                    <TableRow>
                      <TableHead>Provider</TableHead>
                      <TableHead>Total Amount</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {providerGroups.map((group) => (
                      <ProviderGroup
                        key={group.key}
                        group={group}
                      />
                    ))}
                  </TableBody>
                </Table>
              )}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  )
}

function ProviderGroup({
  group,
}: {
  group: { key: string; name: string; entries: SettlementEntry[]; total: number; count: number; status: string }
}) {
  return (
    <TableRow>
      <TableCell className="text-sm font-medium">{group.name}</TableCell>
      <TableCell className="whitespace-nowrap font-mono text-xs">
        {formatAmount(group.total.toFixed(7))}{" "}
        <span className="text-muted-foreground">{group.entries[0].currency}</span>
      </TableCell>
      <TableCell>
        <StatusBadge status={group.status} />
      </TableCell>
    </TableRow>
  )
}

// ── Loading Skeletons ──

function LoadingSkeleton({ role }: { role?: string }) {
  const isConsumer = role === "consumer"
  const isProvider = role === "provider"

  const cards = isConsumer ? 3 : isProvider ? 3 : 0

  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-7 w-24" />
        <Skeleton className="mt-1 h-4 w-64" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: cards }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center gap-3 pb-2">
              <Skeleton className="h-10 w-10 rounded-xl" />
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-7 w-28" />
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-40" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full rounded-lg" />
        </CardContent>
      </Card>
      {isConsumer && (
        <Card>
          <CardHeader>
            <Skeleton className="h-4 w-20" />
          </CardHeader>
          <CardContent className="p-0">
            <div className="space-y-0">
              {Array.from({ length: 6 }).map((_, i) => (
                <div
                  key={i}
                  className="flex items-center gap-4 border-t px-4 py-3"
                >
                  <Skeleton className="h-4 w-16 rounded" />
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-3 flex-1" />
                  <Skeleton className="h-4 w-14 rounded" />
                  <Skeleton className="h-3 w-16" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
      {isProvider && (
        <Card>
          <CardHeader>
            <Skeleton className="h-4 w-32" />
          </CardHeader>
          <CardContent className="p-0">
            <div className="space-y-0">
              {Array.from({ length: 5 }).map((_, i) => (
                <div
                  key={i}
                  className="flex items-center gap-4 border-t px-4 py-3"
                >
                  <Skeleton className="h-3 w-4" />
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-4 w-16 rounded" />
                  <Skeleton className="h-3 w-8" />
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-3 w-16" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function ActivityTableSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Skeleton className="h-4 w-20" />
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-t px-4 py-3"
            >
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-4 w-14 rounded" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function SettlementSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Skeleton className="h-4 w-32" />
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-t px-4 py-3"
            >
              <Skeleton className="h-3 w-4" />
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
