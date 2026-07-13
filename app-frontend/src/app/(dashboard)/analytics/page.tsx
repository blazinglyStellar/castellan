"use client"

import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  TrendingUp,
  Clock,
  DollarSign,
  Activity,
  BarChart3,
  PieChart,
  AlertTriangle,
  Gauge,
} from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import { getEarnings, getUsage, ApiError } from "@/lib/api/endpoints"
import { formatAmount } from "@/lib/format"
import type { UsageEvent } from "@/lib/api/types"
import { EmptyState } from "@/components/shared/empty-state"
import { ErrorState } from "@/components/shared/error-state"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button, buttonVariants } from "@/components/ui/button"
import { EarningsChart } from "@/components/analytics/earnings-chart"
import { EarningsBreakdown } from "@/components/analytics/earnings-breakdown"
import { RevenueDistributionChart } from "@/components/analytics/revenue-distribution-chart"
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart"
import { UsageCostDonut } from "@/components/analytics/usage-cost-donut"
import { ErrorRateChart } from "@/components/analytics/error-rate-chart"
import { LatencyChart } from "@/components/analytics/latency-chart"
import { StatusDistribution } from "@/components/analytics/status-distribution"

function getDefaultRange() {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 30)
  return {
    startDate: start.toISOString().slice(0, 10),
    endDate: end.toISOString().slice(0, 10),
  }
}

const iconBgMap: Record<string, string> = {
  spend: "bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400",
  calls: "bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400",
  success: "bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400",
  latency: "bg-amber-100 text-amber-600 dark:bg-amber-950 dark:text-amber-400",
  earnings: "bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400",
  unsettled: "bg-orange-100 text-orange-600 dark:bg-orange-950 dark:text-orange-400",
}

export default function AnalyticsPage() {
  const { user, isLoading: isAccountLoading } = useAuth()
  const defaults = getDefaultRange()
  const [startDate, setStartDate] = useState(defaults.startDate)
  const [endDate, setEndDate] = useState(defaults.endDate)
  const [role, setRole] = useState<"provider" | "consumer" | null>(null)

  const resolvedRole = role ?? user?.role ?? "consumer"

  const earningsQuery = useQuery({
    queryKey: ["earnings", startDate, endDate],
    queryFn: () =>
      getEarnings({
        start_date: startDate ? `${startDate}T00:00:00Z` : undefined,
        end_date: endDate ? `${endDate}T23:59:59Z` : undefined,
      }),
    enabled: resolvedRole === "provider",
  })

  const usageQuery = useQuery({
    queryKey: ["usage", "analytics", startDate, endDate],
    queryFn: () =>
      getUsage({
        role: resolvedRole,
        start_date: startDate ? `${startDate}T00:00:00Z` : undefined,
        end_date: endDate ? `${endDate}T23:59:59Z` : undefined,
        limit: 1000,
      }),
    enabled: resolvedRole === "consumer",
  })

  const breakdownData = useMemo(() => {
    const data = earningsQuery.data?.by_provider
    if (!data || data.length === 0) return []
    return data.map((d) => ({ name: d.name, total: d.total }))
  }, [earningsQuery.data])

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  const isLoading =
    (resolvedRole === "provider" && earningsQuery.isLoading) ||
    (resolvedRole === "consumer" && usageQuery.isLoading)

  const isError =
    (resolvedRole === "provider" && earningsQuery.isError) ||
    (resolvedRole === "consumer" && usageQuery.isError)

  const error =
    earningsQuery.error || usageQuery.error

  const refetch =
    resolvedRole === "provider"
      ? () => earningsQuery.refetch()
      : () => usageQuery.refetch()

  const isProviderForbidden =
    resolvedRole === "provider" &&
    earningsQuery.isError &&
    earningsQuery.error instanceof ApiError &&
    earningsQuery.error.status === 403

  const hasProviderData =
    resolvedRole === "provider" &&
    earningsQuery.data

  const hasConsumerData =
    resolvedRole === "consumer" &&
    usageQuery.data &&
    usageQuery.data.data.length > 0

  if (isLoading) {
    return <LoadingSkeleton />
  }

  if (isError && !isProviderForbidden) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load analytics"}
        onRetry={refetch}
      />
    )
  }

  if (!hasProviderData && !hasConsumerData && !isProviderForbidden) {
    return (
      <div className="space-y-6">
        <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={resolvedRole} onRoleToggle={setRole} />
        <EmptyState
          title="No analytics data yet"
          description={
            resolvedRole === "provider"
              ? "Earnings and usage data will appear once API calls are processed."
              : "Usage data will appear once you start making API calls."
          }
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={resolvedRole} onRoleToggle={setRole} />

      {resolvedRole === "provider" && (
        <div className="relative">
          {isProviderForbidden && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-sm">
              <Card className="w-80 shadow-lg">
                <CardContent className="flex flex-col items-center gap-4 pt-6 text-center">
                  <p className="text-sm text-muted-foreground">
                    Provider analytics are only available for provider accounts.
                  </p>
                  <a
                    href="/settings"
                    className={buttonVariants({ variant: "default", size: "sm" })}
                  >
                    Set up provider account
                  </a>
                </CardContent>
              </Card>
            </div>
          )}
          <div className={isProviderForbidden ? "pointer-events-none select-none space-y-6" : "space-y-6"}>
            <div className="grid gap-6 sm:grid-cols-2">
              <StatCard
                icon={<TrendingUp className="h-5 w-5" />}
                iconClass={iconBgMap.earnings}
                title="Total Earnings"
              >
                {formatAmount(earningsQuery.data?.total_earnings ?? "0")}{" "}
                <span className="text-sm font-normal text-muted-foreground">XLM</span>
              </StatCard>
              <StatCard
                icon={<Clock className="h-5 w-5" />}
                iconClass={iconBgMap.unsettled}
                title="Unsettled"
              >
                {formatAmount(earningsQuery.data?.unsettled_earnings ?? "0")}{" "}
                <span className="text-sm font-normal text-muted-foreground">XLM</span>
              </StatCard>
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Earnings Over Time</CardTitle>
                <CardDescription>Daily earnings for the selected period</CardDescription>
              </CardHeader>
              <CardContent>
                {earningsQuery.data ? (
                  <EarningsChart data={earningsQuery.data?.sparkline ?? []} />
                ) : (
                  <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
                    No earnings data yet.
                  </div>
                )}
              </CardContent>
            </Card>

            <div className="grid gap-6 md:grid-cols-2">
              <Card>
                <CardHeader className="flex flex-row items-center gap-2">
                  <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
                    <BarChart3 className="h-4 w-4" />
                  </div>
                  <CardTitle className="text-sm font-medium">Earnings Breakdown</CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                  {earningsQuery.data ? (
                    <EarningsBreakdown data={breakdownData} />
                  ) : (
                    <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
                      No earnings data yet.
                    </div>
                  )}
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="flex flex-row items-center gap-2">
                  <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400">
                    <PieChart className="h-4 w-4" />
                  </div>
                  <CardTitle className="text-sm font-medium">Revenue Distribution</CardTitle>
                </CardHeader>
                <CardContent>
                  {earningsQuery.data ? (
                    <RevenueDistributionChart data={earningsQuery.data.by_endpoint} />
                  ) : (
                    <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
                      No data yet.
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      )}

      {resolvedRole === "consumer" && usageQuery.data && (
        <>
          <SummaryCards events={usageQuery.data.data} />

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
                <BarChart3 className="h-4 w-4" />
              </div>
              <div>
                <CardTitle className="text-sm font-medium">Usage Over Time</CardTitle>
                <CardDescription>Daily API call costs for the selected period</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <UsageVolumeChart events={usageQuery.data.data} />
            </CardContent>
          </Card>

          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader className="flex flex-row items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-600 dark:bg-amber-950 dark:text-amber-400">
                  <AlertTriangle className="h-4 w-4" />
                </div>
                <div>
                  <CardTitle className="text-sm font-medium">Error Rate Over Time</CardTitle>
                  <CardDescription>Daily non-2xx rate by endpoint</CardDescription>
                </div>
              </CardHeader>
              <CardContent>
                <ErrorRateChart events={usageQuery.data.data} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400">
                  <Gauge className="h-4 w-4" />
                </div>
                <div>
                  <CardTitle className="text-sm font-medium">Latency Over Time</CardTitle>
                  <CardDescription>Daily average response time by endpoint</CardDescription>
                </div>
              </CardHeader>
              <CardContent>
                <LatencyChart events={usageQuery.data.data} />
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader className="flex flex-row items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400">
                  <PieChart className="h-4 w-4" />
                </div>
                <div>
                  <CardTitle className="text-sm font-medium">Status Code Distribution</CardTitle>
                  <CardDescription>Response status breakdown by endpoint</CardDescription>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <StatusDistribution events={usageQuery.data.data} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
                  <DollarSign className="h-4 w-4" />
                </div>
                <div>
                  <CardTitle className="text-sm font-medium">Cost Breakdown</CardTitle>
                  <CardDescription>Spending by endpoint</CardDescription>
                </div>
              </CardHeader>
              <CardContent>
                <UsageCostDonut events={usageQuery.data.data} />
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}

function Header({
  role,
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  roleToggle,
  onRoleToggle,
}: {
  role: "provider" | "consumer"
  startDate: string
  endDate: string
  onStartDateChange: (d: string) => void
  onEndDateChange: (d: string) => void
  roleToggle: "provider" | "consumer"
  onRoleToggle: (r: "provider" | "consumer") => void
}) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Analytics</h1>
        <p className="text-sm text-muted-foreground">
          {role === "provider"
            ? "Revenue and earnings insights."
            : "Usage and spending insights."}
        </p>
      </div>
      <div className="flex items-end gap-4">
        <div className="flex gap-0">
          <Button
            variant={roleToggle === "consumer" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleToggle("consumer")}
            className="rounded-r-none"
          >
            Consumer
          </Button>
          <Button
            variant={roleToggle === "provider" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleToggle("provider")}
            className="rounded-l-none"
          >
            Provider
          </Button>
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
      </div>
    </div>
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
        <p className="text-3xl font-bold tracking-tight">{children}</p>
      </CardContent>
    </Card>
  )
}

function SummaryCards({ events }: { events: UsageEvent[] }) {
  const totalCost = events.reduce((s, e) => s + parseFloat(e.request_cost), 0)
  const totalCalls = events.length
  const successCalls = events.filter(
    (e) => e.status_code != null && e.status_code < 400,
  ).length
  const successRate = totalCalls > 0 ? (successCalls / totalCalls) * 100 : 0
  const latencyValues = events
    .filter((e) => e.latency_ms != null)
    .map((e) => e.latency_ms!)
  const avgLatency =
    latencyValues.length > 0
      ? latencyValues.reduce((s, v) => s + v, 0) / latencyValues.length
      : 0
  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        icon={<DollarSign className="h-5 w-5" />}
        iconClass={iconBgMap.spend}
        title="Total Spend"
      >
        {totalCost.toFixed(4)}{" "}
        <span className="text-sm font-normal text-muted-foreground">XLM</span>
      </StatCard>
      <StatCard
        icon={<TrendingUp className="h-5 w-5" />}
        iconClass={iconBgMap.calls}
        title="Total Calls"
      >
        {totalCalls.toLocaleString()}
      </StatCard>
      <StatCard
        icon={<Activity className="h-5 w-5" />}
        iconClass={iconBgMap.success}
        title="Success Rate"
      >
        {successRate.toFixed(1)}%
        <p className="mt-1 text-xs font-normal text-muted-foreground">
          {successCalls} / {totalCalls} calls
        </p>
      </StatCard>
      <StatCard
        icon={<Clock className="h-5 w-5" />}
        iconClass={iconBgMap.latency}
        title="Avg Latency"
      >
        {latencyValues.length > 0 ? `${Math.round(avgLatency)}ms` : "\u2014"}
        <p className="mt-1 text-xs font-normal text-muted-foreground">
          avg response time
        </p>
      </StatCard>
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <Skeleton className="h-7 w-24" />
          <Skeleton className="mt-1 h-4 w-64" />
        </div>
        <Skeleton className="h-8 w-72" />
      </div>
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center gap-3 pb-2">
              <Skeleton className="h-10 w-10 rounded-xl" />
              <Skeleton className="h-4 w-28" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-32" />
            </CardContent>
          </Card>
        ))}
      </div>
      <Skeleton className="h-64 w-full rounded-lg" />
      <div className="grid gap-6 md:grid-cols-2">
        <Skeleton className="h-64 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        <Skeleton className="h-48 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    </div>
  )
}
