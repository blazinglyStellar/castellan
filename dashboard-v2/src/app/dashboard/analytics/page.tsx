"use client"

import * as React from "react"
import { useState, useEffect, useMemo } from "react"
import { AreaChart, Area, CartesianGrid, XAxis, YAxis } from "recharts"
import { BarChart3, Activity, CheckCircle, AlertTriangle, Clock, Banknote } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Skeleton } from "@/components/ui/skeleton"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import { StatCard } from "@/components/StatCard"
import { PageHeader } from "@/components/PageHeader"
import { EmptyState } from "@/components/EmptyState"
import { formatCompact, formatLatency } from "@/lib/utils"
import { StellarLogo } from "@/components/StellarLogo"
import type { AnalyticsSummary, TopEndpoint, ErrorBreakdown } from "@/lib/types"

type Interval = "24h" | "7d" | "30d"

const intervals: Interval[] = ["24h", "7d", "30d"]

function generateRequestVolumeData(interval: Interval) {
  const now = Date.now()
  if (interval === "24h") {
    return Array.from({ length: 24 }, (_, i) => {
      const requests = 800 + Math.floor(Math.random() * 600)
      const failed = Math.floor(requests * (0.02 + Math.random() * 0.05))
      return {
        date: new Date(now - (23 - i) * 60 * 60 * 1000).toLocaleTimeString("en-US", { hour: "2-digit" }),
        requests,
        successful: requests - failed,
        failed,
      }
    })
  }
  if (interval === "30d") {
    return Array.from({ length: 30 }, (_, i) => {
      const requests = 6000 + Math.floor(Math.random() * 4000)
      const failed = Math.floor(requests * (0.02 + Math.random() * 0.05))
      return {
        date: new Date(now - (29 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
        requests,
        successful: requests - failed,
        failed,
      }
    })
  }
  return Array.from({ length: 7 }, (_, i) => {
    const requests = 28000 + Math.floor(Math.random() * 12000)
    const failed = Math.floor(requests * (0.02 + Math.random() * 0.05))
    return {
      date: new Date(now - (6 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      requests,
      successful: requests - failed,
      failed,
    }
  })
}

function generateLatencyData(interval: Interval) {
  const now = Date.now()
  if (interval === "24h") {
    return Array.from({ length: 24 }, (_, i) => ({
      date: new Date(now - (23 - i) * 60 * 60 * 1000).toLocaleTimeString("en-US", { hour: "2-digit" }),
      p50: 20 + Math.floor(Math.random() * 30),
      p95: 60 + Math.floor(Math.random() * 80),
      p99: 120 + Math.floor(Math.random() * 150),
    }))
  }
  if (interval === "30d") {
    return Array.from({ length: 30 }, (_, i) => ({
      date: new Date(now - (29 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      p50: 25 + Math.floor(Math.random() * 25),
      p95: 70 + Math.floor(Math.random() * 60),
      p99: 140 + Math.floor(Math.random() * 100),
    }))
  }
  return Array.from({ length: 7 }, (_, i) => ({
    date: new Date(now - (6 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
    p50: 22 + Math.floor(Math.random() * 28),
    p95: 65 + Math.floor(Math.random() * 55),
    p99: 130 + Math.floor(Math.random() * 120),
  }))
}

function generateSummary(interval: Interval): AnalyticsSummary {
  if (interval === "24h") {
    return {
      totalRequests: 8200,
      successfulRequests: 7950,
      failedRequests: 250,
      avgLatency: 30,
      totalEarnings: 285.5,
      errorRate: 3.0,
    }
  }
  if (interval === "30d") {
    return {
      totalRequests: 1_102_000,
      successfulRequests: 1_061_000,
      failedRequests: 41_000,
      avgLatency: 34,
      totalEarnings: 36_520.0,
      errorRate: 3.7,
    }
  }
  return {
    totalRequests: 258_000,
    successfulRequests: 248_500,
    failedRequests: 9_500,
    avgLatency: 32,
    totalEarnings: 8540.27,
    errorRate: 3.7,
  }
}

function generateTopEndpoints(interval: Interval): TopEndpoint[] {
  const mult = interval === "24h" ? 0.15 : interval === "30d" ? 4.3 : 1
  return [
    { id: "1", endpoint: "/v1/payments", method: "GET", requests: Math.round(89200 * mult), avgLatency: 28, errorRate: 1.2 },
    { id: "2", endpoint: "/v1/transactions", method: "POST", requests: Math.round(65400 * mult), avgLatency: 45, errorRate: 2.8 },
    { id: "3", endpoint: "/v1/balances", method: "GET", requests: Math.round(43100 * mult), avgLatency: 18, errorRate: 0.5 },
    { id: "4", endpoint: "/v1/webhooks", method: "DELETE", requests: Math.round(21800 * mult), avgLatency: 52, errorRate: 4.1 },
    { id: "5", endpoint: "/v1/accounts", method: "PUT", requests: Math.round(12500 * mult), avgLatency: 35, errorRate: 1.9 },
  ]
}

function generateErrorBreakdown(interval: Interval): ErrorBreakdown[] {
  const mult = interval === "24h" ? 0.15 : interval === "30d" ? 4.3 : 1
  return [
    { statusCode: 200, count: Math.round(248500 * mult), label: "200 OK" },
    { statusCode: 400, count: Math.round(5200 * mult), label: "400 Bad Request" },
    { statusCode: 429, count: Math.round(2800 * mult), label: "429 Rate Limited" },
    { statusCode: 500, count: Math.round(1500 * mult), label: "500 Server Error" },
  ]
}

const requestChartConfig = {
  successful: {
    label: "Successful",
    color: "var(--chart-2)",
  },
  failed: {
    label: "Failed",
    color: "var(--chart-3)",
  },
} satisfies ChartConfig

const latencyChartConfig = {
  p50: {
    label: "p50",
    color: "var(--chart-1)",
  },
  p95: {
    label: "p95",
    color: "var(--chart-4)",
  },
  p99: {
    label: "p99",
    color: "var(--chart-5)",
  },
} satisfies ChartConfig

const methodColors: Record<string, string> = {
  GET: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  POST: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  DELETE: "bg-red-500/10 text-red-600 dark:text-red-400",
  PUT: "bg-purple-500/10 text-purple-600 dark:text-purple-400",
}

function MethodBadge({ method }: { method: string }) {
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${methodColors[method] || "bg-muted text-muted-foreground"}`}
    >
      {method}
    </span>
  )
}

function AnalyticsKpiXlm({ amount }: { amount: number }) {
  return (
    <span className="inline-flex items-center gap-1 text-2xl font-semibold tracking-tight">
      <StellarLogo className="size-4" />
      {amount.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
    </span>
  )
}

function KpiMs({ ms }: { ms: number }) {
  return (
    <span className="text-2xl font-semibold tracking-tight">
      {ms}
      <span className="text-xs text-muted-foreground font-normal ms-1">ms</span>
    </span>
  )
}

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [selectedInterval, setSelectedInterval] = useState<Interval>("7d")

  useEffect(() => {
    const t = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(t)
  }, [])

  const summary = useMemo(() => generateSummary(selectedInterval), [selectedInterval])
  const requestVolumeData = useMemo(() => generateRequestVolumeData(selectedInterval), [selectedInterval])
  const latencyData = useMemo(() => generateLatencyData(selectedInterval), [selectedInterval])
  const topEndpoints = useMemo(() => generateTopEndpoints(selectedInterval), [selectedInterval])
  const errorBreakdown = useMemo(() => generateErrorBreakdown(selectedInterval), [selectedInterval])
  const errorTotal = useMemo(() => errorBreakdown.reduce((s, e) => s + e.count, 0), [errorBreakdown])

  if (error) {
    return (
      <div className="space-y-6">
      <PageHeader
        title="Analytics"
        description="Performance metrics for your APIs."
        actions={
          <div className="flex items-center gap-1">
            {intervals.map((i) => (
              <Button
                key={i}
                variant={selectedInterval === i ? "default" : "outline"}
                size="sm"
                className="h-7 text-xs"
                onClick={() => setSelectedInterval(i)}
              >
                {i}
              </Button>
            ))}
          </div>
        }
      />
        <EmptyState
          icon={BarChart3}
          title="Failed to load analytics"
          description="Something went wrong. Please try again."
          action={<Button onClick={() => setError(false)}>Retry</Button>}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Analytics"
        description="Performance metrics for your APIs."
        actions={
          <div className="flex items-center gap-1">
            {intervals.map((i) => (
              <Button
                key={i}
                variant={selectedInterval === i ? "default" : "outline"}
                size="sm"
                className="h-7 text-xs"
                onClick={() => setSelectedInterval(i)}
              >
                {i}
              </Button>
            ))}
          </div>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <StatCard
          title="Total Requests"
          value={loading ? "" : formatCompact(summary.totalRequests)}
          icon={Activity}
          loading={loading}
        />
        <StatCard
          title="Successful"
          value={loading ? "" : `${formatCompact(summary.successfulRequests)} (${(100 - summary.errorRate).toFixed(1)}%)`}
          icon={CheckCircle}
          loading={loading}
        />
        <StatCard
          title="Error Rate"
          value={loading ? "" : `${summary.errorRate.toFixed(1)}%`}
          icon={AlertTriangle}
          trend={summary.errorRate > 5 ? "down" : undefined}
          loading={loading}
        />
        <StatCard
          title="Avg Latency"
          value={loading ? "" : <KpiMs ms={summary.avgLatency} />}
          icon={Clock}
          loading={loading}
        />
        <StatCard
          title="Total Earnings"
          value={loading ? "" : <AnalyticsKpiXlm amount={summary.totalEarnings} />}
          icon={Banknote}
          loading={loading}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Request Volume</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="h-[250px] animate-pulse rounded-lg bg-muted" />
            ) : (
              <ChartContainer config={requestChartConfig} className="aspect-auto h-[250px] w-full">
                <AreaChart data={requestVolumeData}>
                  <defs>
                    <linearGradient id="fillSuccessful" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-successful)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-successful)" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="fillFailed" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-failed)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-failed)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} />
                  <XAxis
                    dataKey="date"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    tick={{ fontSize: 11 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    tick={{ fontSize: 11 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                  <Area
                    dataKey="successful"
                    type="monotone"
                    fill="url(#fillSuccessful)"
                    stroke="var(--color-successful)"
                    stackId="1"
                  />
                  <Area
                    dataKey="failed"
                    type="monotone"
                    fill="url(#fillFailed)"
                    stroke="var(--color-failed)"
                    stackId="1"
                  />
                </AreaChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Latency</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="h-[250px] animate-pulse rounded-lg bg-muted" />
            ) : (
              <ChartContainer config={latencyChartConfig} className="aspect-auto h-[250px] w-full">
                <AreaChart data={latencyData}>
                  <defs>
                    <linearGradient id="fillP50" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-p50)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-p50)" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="fillP95" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-p95)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-p95)" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="fillP99" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--color-p99)" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="var(--color-p99)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} />
                  <XAxis
                    dataKey="date"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    tick={{ fontSize: 11 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    tick={{ fontSize: 11 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                  <Area dataKey="p50" type="monotone" fill="url(#fillP50)" stroke="var(--color-p50)" />
                  <Area dataKey="p95" type="monotone" fill="url(#fillP95)" stroke="var(--color-p95)" />
                  <Area dataKey="p99" type="monotone" fill="url(#fillP99)" stroke="var(--color-p99)" />
                </AreaChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Top Endpoints</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <div className="h-[200px] animate-pulse rounded-lg bg-muted" />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Method</TableHead>
                    <TableHead>Endpoint</TableHead>
                    <TableHead className="text-right">Requests</TableHead>
                    <TableHead className="text-right">Latency</TableHead>
                    <TableHead className="text-right">Error %</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {topEndpoints.map((ep) => (
                    <TableRow key={ep.id}>
                      <TableCell><MethodBadge method={ep.method} /></TableCell>
                      <TableCell className="font-mono text-xs">{ep.endpoint}</TableCell>
                      <TableCell className="text-right">{formatCompact(ep.requests)}</TableCell>
                      <TableCell className="text-right">{formatLatency(ep.avgLatency)}</TableCell>
                      <TableCell className="text-right">{ep.errorRate.toFixed(1)}%</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Status Code Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="h-[200px] animate-pulse rounded-lg bg-muted" />
            ) : (
              <div className="space-y-4">
                {errorBreakdown.map((item) => {
                  const maxCount = Math.max(...errorBreakdown.map((e) => e.count))
                  const barWidth = (item.count / maxCount) * 100
                  const pct = ((item.count / errorTotal) * 100).toFixed(1)
                  const statusColor =
                    item.statusCode >= 500
                      ? "bg-red-500"
                      : item.statusCode >= 400
                        ? "bg-amber-500"
                        : "bg-green-500"
                  return (
                    <div key={item.statusCode} className="space-y-1.5">
                      <div className="flex items-center justify-between text-sm">
                        <span className="font-medium">{item.label}</span>
                        <span className="text-muted-foreground">{formatCompact(item.count)} <span className="text-xs text-muted-foreground/60">({pct}%)</span></span>
                      </div>
                      <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full ${statusColor} transition-all`}
                          style={{ width: `${barWidth}%` }}
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
