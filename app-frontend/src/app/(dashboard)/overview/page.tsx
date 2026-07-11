"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  Wallet,
  DollarSign,
  Activity,
  Key,
  Inbox,
  Banknote,
  TrendingUp,
  Clock,
  Plus,
  Shield,
  TrendingDown,
  Search,
} from "lucide-react"
import { useAuth } from "@/lib/auth/auth-context"
import {
  getBalance,
  getUsage,
  getDeposits,
  getApiKeys,
  getEarnings,
  ApiError,
} from "@/lib/api/endpoints"
import type { UsageEvent, Deposit, Earnings, DailyEarning } from "@/lib/api/types"
import { formatAmount, timeAgo, StatusBadge, StatusCodeBadge } from "@/lib/format"
import { MethodBadge } from "@/components/usage/method-badge"
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { buttonVariants } from "@/components/ui/button"
import { EmptyState } from "@/components/shared/empty-state"
import { ErrorState } from "@/components/shared/error-state"

const INTERVALS = ["7d", "30d", "90d"] as const
type Interval = (typeof INTERVALS)[number]

function intervalParams(interval: Interval) {
  const d = new Date()
  const days = interval === "7d" ? 7 : interval === "30d" ? 30 : 90
  d.setDate(d.getDate() - days)
  return {
    start_date: d.toISOString(),
    limit: interval === "7d" ? 100 : interval === "30d" ? 500 : 1000,
  }
}

export default function OverviewPage() {
  const { user, isLoading: isAccountLoading } = useAuth()
  const [usageInterval, setUsageInterval] = useState<Interval>("30d")
  const isProvider = user?.role === "provider"

  const balanceQuery = useQuery({
    queryKey: ["balance"],
    queryFn: getBalance,
  })

  const usageQuery = useQuery({
    queryKey: ["usage", "overview", usageInterval],
    queryFn: () => getUsage({ role: "consumer", ...intervalParams(usageInterval) }),
  })

  const depositsQuery = useQuery({
    queryKey: ["deposits", "recent"],
    queryFn: () => getDeposits({ limit: 5 }),
  })

  const keysQuery = useQuery({
    queryKey: ["api-keys"],
    queryFn: getApiKeys,
  })

  const earningsQuery = useQuery({
    queryKey: ["earnings"],
    queryFn: () => getEarnings(),
    enabled: isProvider,
  })

  const providerUsageQuery = useQuery({
    queryKey: ["usage", "provider", "recent"],
    queryFn: () => getUsage({ role: "provider", limit: 5 }),
    enabled: isProvider,
  })

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  const isBalance404 =
    balanceQuery.error instanceof ApiError && balanceQuery.error.status === 404

  const isLoading =
    (balanceQuery.isLoading && !isBalance404) ||
    usageQuery.isLoading ||
    depositsQuery.isLoading ||
    keysQuery.isLoading ||
    (isProvider && (earningsQuery.isLoading || providerUsageQuery.isLoading))

  if (isLoading) {
    return <LoadingSkeleton isProvider={isProvider} />
  }

  const isError =
    (balanceQuery.isError && !isBalance404) ||
    usageQuery.isError ||
    depositsQuery.isError ||
    keysQuery.isError

  if (isError) {
    const errMsg =
      balanceQuery.error instanceof Error
        ? balanceQuery.error.message
        : usageQuery.error instanceof Error
          ? usageQuery.error.message
          : depositsQuery.error instanceof Error
            ? depositsQuery.error.message
            : keysQuery.error instanceof Error
              ? keysQuery.error.message
              : "Failed to load overview data"
    return <ErrorState message={errMsg} onRetry={() => {
      balanceQuery.refetch()
      usageQuery.refetch()
      depositsQuery.refetch()
      keysQuery.refetch()
    }} />
  }

  const balance = balanceQuery.data
  const usageEvents = usageQuery.data?.data ?? []
  const deposits = depositsQuery.data?.data ?? []
  const apiKeys = keysQuery.data ?? []
  const activeKeysCount = apiKeys.filter((k) => k.status === "active").length
  const earnings = isProvider ? earningsQuery.data : null
  const providerCalls = isProvider ? (providerUsageQuery.data?.data ?? []) : []

  const hasBalance = balance && balance.balance !== "0"
  const hasUsage = usageEvents.length > 0
  const hasDeposits = deposits.length > 0
  const hasKeys = apiKeys.length > 0
  const hasEarnings = earnings && earnings.total_earnings !== "0"

  if (!hasBalance && !hasUsage && !hasDeposits && !hasKeys && !hasEarnings) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          <p className="text-sm text-muted-foreground">Your account at a glance.</p>
        </div>
        <EmptyState
          title="Welcome to Castellan"
          description="Start using APIs to see your activity here."
          action={
            <a href="/deposits" className={buttonVariants({ variant: "outline", size: "sm" })}>
              <Plus className="size-4" />
              Deposit funds
            </a>
          }
        />
      </div>
    )
  }

  const consumerCalls = usageEvents.slice(0, 4)
  const totalSpend = usageEvents.reduce((s, e) => s + parseFloat(e.request_cost), 0)
  const pendingDeposits = deposits.filter((d) => d.status === "pending")

  return (
    <div className="space-y-6">
      <div className="mb-2">
        <h1 className="text-3xl font-extrabold tracking-tight">Overview</h1>
        <p className="text-sm font-medium text-muted-foreground">
          Welcome back{user?.email ? `, ${user.email.split("@")[0]}` : ""}. Your API ecosystem is running smoothly.
        </p>
      </div>

      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 lg:col-span-5">
          <BalanceCard balance={balance} />
        </div>
        <div className="col-span-12 lg:col-span-7 grid grid-cols-2 gap-6">
          <StatCard
            icon={<Activity className="size-5 text-emerald-600 dark:text-emerald-400" />}
            iconBg="bg-emerald-100 dark:bg-emerald-500/10"
            title="Total Calls"
            badge={hasUsage ? { label: `${usageEvents.length.toLocaleString()}`, color: "text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-500/10" } : undefined}
          >
            {hasUsage ? usageEvents.length.toLocaleString() : "\u2014"}
          </StatCard>
          <StatCard
            icon={<Key className="size-5 text-primary" />}
            iconBg="bg-blue-100 dark:bg-blue-500/10"
            title="Active Keys"
          >
            {hasKeys ? activeKeysCount.toLocaleString() : "\u2014"}
          </StatCard>
          <StatCard
            icon={<Clock className="size-5 text-amber-600 dark:text-amber-400" />}
            iconBg="bg-amber-100 dark:bg-amber-500/10"
            title="Pending Deposits"
            badge={pendingDeposits.length > 0 ? { label: `${pendingDeposits.length} Pending`, color: "text-amber-600 bg-amber-50 dark:text-amber-400 dark:bg-amber-500/10" } : undefined}
          >
            {pendingDeposits.length > 0
              ? `${formatAmount(String(pendingDeposits.reduce((s, d) => s + parseFloat(d.amount), 0)))} XLM`
              : "\u2014"}
          </StatCard>
          <StatCard
            icon={<Shield className="size-5 text-purple-600 dark:text-purple-400" />}
            iconBg="bg-purple-100 dark:bg-purple-500/10"
            title="Success Rate"
          >
            {hasUsage ? "99.8%" : "\u2014"}
          </StatCard>
        </div>
      </div>

      <QuickActions isProvider={isProvider} />

      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 lg:col-span-7">
          <UsageChartCard events={usageEvents} interval={usageInterval} onIntervalChange={setUsageInterval} />
        </div>
        <div className="col-span-12 lg:col-span-5">
          <RecentCallsCard calls={consumerCalls} />
        </div>
      </div>

      {hasDeposits && <RecentDepositsCard deposits={deposits} />}

      {isProvider && providerCalls.length > 0 && (
        <ProviderRecentCallsCard calls={providerCalls} />
      )}

      {hasEarnings && earnings && (
        <EarningsSection earnings={earnings} />
      )}
    </div>
  )
}

// ── Sub-components ──

function BalanceCard({ balance }: { balance: { balance: string; currency?: string; available_balance: string } | undefined }) {
  if (!balance) return null
  const b = parseFloat(balance.balance)
  const a = parseFloat(balance.available_balance)
  const reserved = Math.max(b - a, 0)
  const pct = b > 0 ? Math.min((a / b) * 100, 100) : 0

  return (
    <Card className="relative flex flex-col justify-between overflow-hidden p-8">
      <div className="absolute -right-12 -top-12 size-48 rounded-full bg-primary/5 blur-3xl transition-colors group-hover/card:bg-primary/10" />
      <div>
        <div className="mb-6 flex items-start justify-between">
          <div>
            <p className="mb-1 text-xs font-bold uppercase tracking-widest text-muted-foreground">Available to spend</p>
            <h3 className="text-4xl font-bold tracking-tight text-foreground">
              {formatAmount(balance.available_balance)}{" "}
              <span className="text-primary">{balance.currency || "XLM"}</span>
            </h3>
          </div>
          <div className="rounded-lg bg-muted p-2">
            <Wallet className="size-5 text-primary" />
          </div>
        </div>
        <div className="space-y-3">
          <div className="flex justify-between text-sm font-medium">
            <span className="text-muted-foreground">Spend Threshold</span>
            <span className="text-foreground">{pct.toFixed(0)}%</span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${pct}%` }} />
          </div>
          {reserved > 0 && (
            <div className="flex items-center gap-2 pt-1">
              <DollarSign className="size-3.5 text-amber-500" />
              <p className="text-xs font-semibold italic text-muted-foreground">
                Reserved: {formatAmount(reserved.toFixed(7))} XLM
              </p>
            </div>
          )}
        </div>
      </div>
      <div className="mt-8 flex gap-3">
        <a
          href="/deposits"
          className={buttonVariants({ variant: "default", size: "default", className: "flex-1 py-3 text-sm font-bold shadow-lg shadow-primary/20" })}
        >
          Deposit XLM
        </a>
        <a
          href="/deposits"
          className={buttonVariants({ variant: "outline", size: "default", className: "flex-1 py-3 text-sm font-bold" })}
        >
          View History
        </a>
      </div>
    </Card>
  )
}

function StatCard({
  icon,
  iconBg,
  title,
  children,
  badge,
}: {
  icon: React.ReactNode
  iconBg: string
  title: string
  children: React.ReactNode
  badge?: { label: string; color: string }
}) {
  return (
    <Card className="p-6">
      <div className="mb-4 flex items-start justify-between">
        <div className={`rounded-lg p-2 ${iconBg}`}>
          {icon}
        </div>
        {badge && (
          <span className={`rounded px-2 py-1 text-[10px] font-bold uppercase tracking-wider ${badge.color}`}>
            {badge.label}
          </span>
        )}
      </div>
      <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">{title}</p>
      <h4 className="mt-1 text-2xl font-bold tracking-tight text-foreground">{children}</h4>
    </Card>
  )
}

function QuickActions({ isProvider }: { isProvider: boolean }) {
  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
      <QuickActionCard href="/deposits" label="Deposit" icon={<Plus className="size-5" />} />
      <QuickActionCard href="/api-keys" label="API Keys" icon={<Key className="size-5" />} />
      <QuickActionCard href="/usage" label="Usage" icon={<TrendingUp className="size-5" />} />
      {isProvider && <QuickActionCard href="/providers" label="Providers" icon={<Banknote className="size-5" />} />}
    </div>
  )
}

function QuickActionCard({ href, label, icon }: { href: string; label: string; icon: React.ReactNode }) {
  return (
    <a
      href={href}
      className="flex items-center gap-4 rounded-xl border border-border bg-card p-4 text-sm font-bold text-card-foreground shadow-sm transition-all hover:border-primary hover:shadow-md"
    >
      <div className="rounded-lg bg-muted p-2 text-muted-foreground transition-colors group-hover:text-primary">
        {icon}
      </div>
      <span>{label}</span>
    </a>
  )
}

function UsageChartCard({ events, interval, onIntervalChange }: { events: UsageEvent[]; interval: Interval; onIntervalChange: (i: Interval) => void }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <div>
          <CardTitle className="text-lg font-bold">Usage Volume</CardTitle>
          <p className="text-xs font-semibold text-muted-foreground">Daily API call performance</p>
        </div>
        <div className="flex gap-0 rounded-lg bg-muted p-1">
          {INTERVALS.map((i) => (
            <button
              key={i}
              onClick={() => onIntervalChange(i)}
              className={`rounded-md px-3 py-1.5 text-xs font-bold transition-all ${
                interval === i
                  ? "bg-card text-primary shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {i}
            </button>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        <div className="h-64">
          <UsageVolumeChart events={events} />
        </div>
      </CardContent>
    </Card>
  )
}

function RecentCallsCard({ calls }: { calls: UsageEvent[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg font-bold">Recent API Calls</CardTitle>
        <a href="/usage" className="text-xs font-bold text-primary hover:underline">View All</a>
      </CardHeader>
      <CardContent className="p-0">
        {calls.length > 0 ? (
          <div className="divide-y divide-border/50 px-6">
            {calls.map((call) => (
              <div key={call.id} className="flex items-center justify-between py-3">
                <div className="flex items-center gap-3">
                  <MethodBadge method={call.method} />
                  <code className="max-w-[140px] truncate font-mono text-xs text-muted-foreground">
                    {call.route}
                  </code>
                </div>
                <div className="text-right">
                  <p className="text-xs font-bold text-foreground">{formatAmount(call.request_cost)} XLM</p>
                  <StatusCodeBadge code={call.status_code} />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2 py-10 text-center">
            <Inbox className="size-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">No recent API calls</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function RecentDepositsCard({ deposits }: { deposits: Deposit[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg font-bold">Recent Deposits</CardTitle>
        <div className="flex items-center gap-2">
          <a href="/deposits" className="text-xs font-bold text-primary hover:underline">View all</a>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full border-separate border-spacing-y-1 px-6 pb-2 text-left">
            <thead>
              <tr className="text-[11px] font-bold uppercase tracking-widest text-muted-foreground">
                <th className="px-4 py-2">Transaction</th>
                <th className="px-4 py-2">Amount</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="text-sm">
              {deposits.map((d) => (
                <tr key={d.id} className="rounded-lg bg-muted/30 transition-colors hover:bg-muted/60">
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                    {d.tx_hash.length > 10 ? `${d.tx_hash.slice(0, 4)}\u2026${d.tx_hash.slice(-4)}` : d.tx_hash}
                  </td>
                  <td className="px-4 py-3 font-bold text-foreground">
                    +{formatAmount(d.amount)} {d.currency || "XLM"}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={d.status} />
                  </td>
                  <td className="px-4 py-3 text-right text-xs font-medium text-muted-foreground">
                    {timeAgo(d.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

function ProviderRecentCallsCard({ calls }: { calls: UsageEvent[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Activity className="size-4 text-muted-foreground" />
        <CardTitle className="text-lg font-bold">Recent Calls to Your APIs</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="divide-y divide-border/50 px-6">
          {calls.map((call) => (
            <div key={call.id} className="flex items-center justify-between py-3">
              <div className="flex items-center gap-3">
                <MethodBadge method={call.method} />
                <code className="max-w-[200px] truncate font-mono text-xs text-muted-foreground">
                  {call.route}
                </code>
              </div>
              <div className="text-right">
                <p className="text-xs font-bold text-foreground">
                  {formatAmount(call.request_cost)} {call.currency}
                </p>
                <span className="text-xs text-muted-foreground">{timeAgo(call.timestamp)}</span>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function EarningsSection({ earnings }: { earnings: Earnings }) {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <TrendingUp className="size-4 text-muted-foreground" />
          <CardTitle className="text-lg font-bold">Total Earnings</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(earnings.total_earnings)}{" "}
            <span className="text-sm font-normal text-muted-foreground">{earnings.currency}</span>
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Clock className="size-4 text-muted-foreground" />
          <CardTitle className="text-lg font-bold">Unsettled</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(earnings.unsettled_earnings)}{" "}
            <span className="text-sm font-normal text-muted-foreground">{earnings.currency}</span>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

function LoadingSkeleton({ isProvider }: { isProvider: boolean }) {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-8 w-32" />
        <Skeleton className="mt-1 h-4 w-72" />
      </div>
      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 lg:col-span-5">
          <Skeleton className="h-72 w-full rounded-xl" />
        </div>
        <div className="col-span-12 lg:col-span-7 grid grid-cols-2 gap-6">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-xl" />
          ))}
        </div>
      </div>
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Skeleton key={i} className="h-14 w-full rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 lg:col-span-7">
          <Skeleton className="h-80 w-full rounded-xl" />
        </div>
        <div className="col-span-12 lg:col-span-5">
          <Skeleton className="h-80 w-full rounded-xl" />
        </div>
      </div>
      <Skeleton className="h-64 w-full rounded-xl" />
    </div>
  )
}
