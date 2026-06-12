"use client"

import { useState } from "react"
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import { DollarSign, Activity, Cable, Wallet, TrendingUp, AlertCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/PageHeader"
import { StatCard } from "@/components/StatCard"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { BalanceBadge } from "@/components/BalanceBadge"
import { StatusBadge } from "@/components/StatusBadge"
import { LowBalanceBanner } from "@/components/LowBalanceBanner"
import { MOCK_PROVIDER_OVERVIEW, MOCK_CONSUMER_OVERVIEW } from "@/lib/mock-data"
import { formatCurrency, formatLatency, timeAgo } from "@/lib/utils"
import type { RecentCall } from "@/lib/mock-data"

const mockRecentCalls: RecentCall[] = [
  { id: "c1", endpoint: "/current", cost: "0.50", latency: 45, consumer: "Acme Corp", time: new Date().toISOString(), status: 200 },
  { id: "c2", endpoint: "/forecast", cost: "1.00", latency: 120, consumer: "Beta Inc", time: new Date(Date.now() - 60000).toISOString(), status: 200 },
  { id: "c3", endpoint: "/lookup", cost: "1.00", latency: 30, consumer: "Gamma LLC", time: new Date(Date.now() - 120000).toISOString(), status: 402 },
  { id: "c4", endpoint: "/analyze", cost: "5.00", latency: 850, consumer: "Delta Co", time: new Date(Date.now() - 180000).toISOString(), status: 200 },
  { id: "c5", endpoint: "/current", cost: "0.50", latency: 52, consumer: "Acme Corp", time: new Date(Date.now() - 240000).toISOString(), status: 200 },
]

export default function DashboardPage() {
  const [role] = useState<"provider" | "consumer" | "both">("provider")
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load dashboard</h2>
        <p className="text-sm text-muted-foreground">Something went wrong. Please try again.</p>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  if (role === "provider") {
    const overview = MOCK_PROVIDER_OVERVIEW
    return (
      <div className="space-y-6">
        <PageHeader
          title="Overview"
          description="Good morning! Here's what's happening with your APIs today."
        />
        {isEmpty ? (
          <EmptyState title="No activity yet" description="Start by adding an API and registering your first endpoint." />
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard title="Total Earned" value={formatCurrency(overview.totalEarned)} icon={<DollarSign className="h-4 w-4" />} trend="up" loading={loading} />
              <StatCard title="Requests This Week" value={overview.requestsThisWeek.toLocaleString()} icon={<Activity className="h-4 w-4" />} loading={loading} />
              <StatCard title="Active Endpoints" value={String(overview.activeEndpoints)} icon={<Cable className="h-4 w-4" />} loading={loading} />
              <StatCard title="Pending Settlement" value={formatCurrency(overview.pendingSettlement)} icon={<Wallet className="h-4 w-4" />} loading={loading} />
            </div>
            {loading ? (
              <Skeleton className="h-72 w-full" />
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Earnings This Week</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={overview.earningsByDay}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                        <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                        <YAxis className="text-xs text-muted-foreground" tickFormatter={(v) => `${v}`} />
                        <Tooltip
                          contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                          formatter={(value: number) => [formatCurrency(value), "Earnings"]}
                        />
                        <Line type="monotone" dataKey="amount" stroke="hsl(var(--primary))" strokeWidth={2} dot={{ fill: "hsl(var(--primary))" }} />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>
            )}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Recent Calls</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable
                  columns={[
                    { key: "endpoint", header: "Endpoint", cell: (r: RecentCall) => <span className="font-mono text-xs">{r.endpoint}</span> },
                    { key: "cost", header: "Cost", cell: (r: RecentCall) => formatCurrency(r.cost) },
                    { key: "latency", header: "Latency", cell: (r: RecentCall) => formatLatency(r.latency) },
                    { key: "consumer", header: "Consumer", cell: (r: RecentCall) => r.consumer },
                    { key: "time", header: "Time", cell: (r: RecentCall) => <span className="text-muted-foreground">{timeAgo(r.time)}</span> },
                  ]}
                  data={mockRecentCalls}
                  loading={loading}
                />
              </CardContent>
            </Card>
          </>
        )}
      </div>
    )
  }

  const consumerOverview = MOCK_CONSUMER_OVERVIEW
  return (
    <div className="space-y-6">
      <PageHeader title="Overview">
        <BalanceBadge balance={consumerOverview.balance} />
      </PageHeader>
      {consumerOverview.isLowBalance && <LowBalanceBanner balance={consumerOverview.balance} />}
      {isEmpty ? (
        <EmptyState title="No usage yet" description="Start by generating an API key and making your first call." />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <StatCard title="Balance" value={formatCurrency(consumerOverview.balance)} icon={<Wallet className="h-4 w-4" />} loading={loading} />
            <StatCard title="Spent This Month" value={formatCurrency(consumerOverview.spentThisMonth)} icon={<TrendingUp className="h-4 w-4" />} loading={loading} />
          </div>
          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader><CardTitle className="text-base">Recent Usage</CardTitle></CardHeader>
              <CardContent>
                <DataTable
                  columns={[
                    { key: "endpoint", header: "Endpoint", cell: (r: RecentCall) => <span className="font-mono text-xs">{r.endpoint}</span> },
                    { key: "cost", header: "Cost", cell: (r: RecentCall) => formatCurrency(r.cost) },
                    { key: "status", header: "Status", cell: (r: RecentCall) => <StatusBadge status={r.status === 200 ? "completed" : "failed"} /> },
                  ]}
                  data={mockRecentCalls.slice(0, 3)}
                  loading={loading}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader><CardTitle className="text-base">Top Providers</CardTitle></CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {consumerOverview.topProviders.map((p) => (
                    <div key={p.name} className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium">{p.name}</p>
                        <p className="text-xs text-muted-foreground">{p.calls.toLocaleString()} calls</p>
                      </div>
                      <span className="text-sm font-mono">{formatCurrency(p.spent)}</span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}
