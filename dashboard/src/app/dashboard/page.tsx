"use client"

import { useState } from "react"
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import { DollarSign, Activity, Cable, Wallet, TrendingUp, AlertCircle, ArrowUpRight } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
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

const mockUsageTrend = [
  { date: "Mon", requests: 1250 },
  { date: "Tue", requests: 2340 },
  { date: "Wed", requests: 1890 },
  { date: "Thu", requests: 3100 },
  { date: "Fri", requests: 2780 },
  { date: "Sat", requests: 1420 },
  { date: "Sun", requests: 980 },
]

export default function DashboardPage() {
  const [role] = useState<"provider" | "consumer" | "both">("both")
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)

  const providerOverview = MOCK_PROVIDER_OVERVIEW
  const consumerOverview = MOCK_CONSUMER_OVERVIEW
  const isProvider = role === "provider" || role === "both"
  const isConsumer = role === "consumer" || role === "both"

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

  return (
    <div className="space-y-8">
      {isConsumer && consumerOverview.isLowBalance && (
        <LowBalanceBanner balance={consumerOverview.balance} />
      )}

      {isProvider && (
        <section>
          <PageHeader
            title="Good morning"
            description="Here's what's happening with your APIs today."
          />
          {isEmpty ? (
            <EmptyState
              title="No activity yet"
              description="Register your first API to get started."
              action={<Button>Add API</Button>}
            />
          ) : (
            <div className="space-y-6">
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <StatCard
                  title="Total Earned"
                  value={formatCurrency(providerOverview.totalEarned)}
                  icon={<DollarSign className="h-4 w-4" />}
                  trend="up"
                  loading={loading}
                  subtitle="+12.4% this week"
                />
                <StatCard
                  title="Requests This Week"
                  value={providerOverview.requestsThisWeek.toLocaleString()}
                  icon={<Activity className="h-4 w-4" />}
                  loading={loading}
                  subtitle="+8.2% vs last week"
                />
                <StatCard
                  title="Active Endpoints"
                  value={`${providerOverview.activeEndpoints} / ${providerOverview.activeEndpoints + 2}`}
                  icon={<Cable className="h-4 w-4" />}
                  loading={loading}
                />
                <StatCard
                  title="Pending Settlement"
                  value={formatCurrency(providerOverview.pendingSettlement)}
                  icon={<Wallet className="h-4 w-4" />}
                  loading={loading}
                />
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
                        <AreaChart data={providerOverview.earningsByDay}>
                          <defs>
                            <linearGradient id="earningsGrad" x1="0" y1="0" x2="0" y2="1">
                              <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                              <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                            </linearGradient>
                          </defs>
                          <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                          <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                          <YAxis className="text-xs text-muted-foreground" tickFormatter={(v) => `${v} XLM`} />
                          <Tooltip
                            contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                            formatter={(value: number) => [formatCurrency(value), "Earnings"]}
                          />
                          <Area type="monotone" dataKey="amount" stroke="hsl(var(--primary))" strokeWidth={2} fill="url(#earningsGrad)" />
                        </AreaChart>
                      </ResponsiveContainer>
                    </div>
                  </CardContent>
                </Card>
              )}

              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">Recent API Calls</CardTitle>
                    <Button variant="ghost" size="sm" className="text-xs text-muted-foreground">View All</Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <DataTable
                    columns={[
                      { key: "time", header: "Time", cell: (r: RecentCall) => <span className="text-muted-foreground text-xs">{timeAgo(r.time)}</span> },
                      { key: "endpoint", header: "Endpoint", cell: (r: RecentCall) => <span className="font-mono text-xs">{r.endpoint}</span> },
                      { key: "cost", header: "Cost", cell: (r: RecentCall) => <span className="font-mono text-xs">{formatCurrency(r.cost)}</span> },
                      { key: "latency", header: "Latency", cell: (r: RecentCall) => formatLatency(r.latency) },
                      { key: "consumer", header: "Consumer", cell: (r: RecentCall) => r.consumer },
                      { key: "status", header: "Status", cell: (r: RecentCall) => <StatusBadge status={r.status === 200 ? "completed" : r.status === 402 ? "failed" : "completed"} /> },
                    ]}
                    data={mockRecentCalls}
                    loading={loading}
                  />
                </CardContent>
              </Card>
            </div>
          )}
        </section>
      )}

      {isProvider && isConsumer && (
        <Separator className="my-2" />
      )}

      {isConsumer && (
        <section>
          <PageHeader
            title="Welcome back"
            actions={
              <div className="flex items-center gap-3">
                <BalanceBadge balance={consumerOverview.balance} />
                <Button size="sm" variant="default">
                  <ArrowUpRight className="mr-1.5 h-3.5 w-3.5" /> Deposit
                </Button>
              </div>
            }
          />
          <p className="-mt-4 mb-6 text-sm text-muted-foreground">
            Spent this month: <span className="font-medium text-foreground">{formatCurrency(consumerOverview.spentThisMonth)}</span>
          </p>

          {isEmpty ? (
            <EmptyState
              title="No usage yet"
              description={parseFloat(consumerOverview.balance) === 0 ? "Deposit funds to get started." : "Start calling APIs to see usage."}
              action={<Button>Deposit Now</Button>}
            />
          ) : (
            <div className="space-y-6">
              <div className="grid gap-6 lg:grid-cols-2">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Recent Usage</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <DataTable
                      columns={[
                        { key: "endpoint", header: "API", cell: (r: RecentCall) => <span className="font-mono text-xs">{r.endpoint}</span> },
                        { key: "cost", header: "Cost", cell: (r: RecentCall) => <span className="font-mono">{formatCurrency(r.cost)}</span> },
                        { key: "time", header: "Time", cell: (r: RecentCall) => <span className="text-muted-foreground text-xs">{timeAgo(r.time)}</span> },
                      ]}
                      data={mockRecentCalls.slice(0, 3)}
                      loading={loading}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Top Providers by Spend</CardTitle>
                  </CardHeader>
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

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Usage Trend</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={mockUsageTrend}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                        <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                        <YAxis className="text-xs text-muted-foreground" />
                        <Tooltip
                          contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                        />
                        <Line type="monotone" dataKey="requests" stroke="hsl(var(--primary))" strokeWidth={2} dot={{ fill: "hsl(var(--primary))", r: 3 }} />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </section>
      )}
    </div>
  )
}
