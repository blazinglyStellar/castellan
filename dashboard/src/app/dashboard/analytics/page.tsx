"use client"

import { useState } from "react"
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import { AlertCircle, BarChart3, Download } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { DateRangePicker } from "@/components/DateRangePicker"
import { MOCK_ANALYTICS } from "@/lib/mock-data"
import { formatCurrency } from "@/lib/utils"

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [selectedApi, setSelectedApi] = useState("all")

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load analytics</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  const analytics = MOCK_ANALYTICS

  return (
    <div className="space-y-6">
      <PageHeader
        title="Analytics"
        description="Usage metrics across all your APIs."
        actions={
          <div className="flex items-center gap-2">
            <Select value={selectedApi} onValueChange={setSelectedApi}>
              <SelectTrigger className="w-36"><SelectValue placeholder="All APIs" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All APIs</SelectItem>
                <SelectItem value="weather">Weather API</SelectItem>
                <SelectItem value="geo">Geolocation</SelectItem>
                <SelectItem value="ai">AI Text</SelectItem>
              </SelectContent>
            </Select>
            <DateRangePicker />
            <Button variant="outline" size="sm" disabled>
              <Download className="h-4 w-4" /> Export
            </Button>
          </div>
        }
      />

      {isEmpty ? (
        <EmptyState title="No analytics data" description="Data will appear once your APIs receive traffic." icon={<BarChart3 className="h-12 w-12" />} />
      ) : (
        <>
          {loading ? (
            <Skeleton className="h-72 w-full" />
          ) : (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Requests Over Time</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-72">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={analytics.requestsOverTime}>
                      <defs>
                        {[
                          { id: "weather", color: "#3b82f6" },
                          { id: "geo", color: "#10b981" },
                          { id: "ai", color: "#f59e0b" },
                          { id: "email", color: "#8b5cf6" },
                        ].map(({ id, color }) => (
                          <linearGradient key={id} id={`grad-${id}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                            <stop offset="95%" stopColor={color} stopOpacity={0} />
                          </linearGradient>
                        ))}
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                      <YAxis className="text-xs text-muted-foreground" />
                      <Tooltip
                        contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                      />
                      <Area type="monotone" dataKey="weather" stackId="1" stroke="#3b82f6" fill="url(#grad-weather)" />
                      <Area type="monotone" dataKey="geo" stackId="1" stroke="#10b981" fill="url(#grad-geo)" />
                      <Area type="monotone" dataKey="ai" stackId="1" stroke="#f59e0b" fill="url(#grad-ai)" />
                      <Area type="monotone" dataKey="email" stackId="1" stroke="#8b5cf6" fill="url(#grad-email)" />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>
          )}

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Revenue Over Time</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={analytics.requestsOverTime.map((d) => ({ date: d.date, revenue: d.weather * 0.0001 + d.geo * 0.0002 + d.ai * 0.0005 }))}>
                      <defs>
                        <linearGradient id="revenueGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#22c55e" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                      <YAxis className="text-xs text-muted-foreground" tickFormatter={(v) => `${v.toFixed(1)}`} />
                      <Tooltip
                        contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                        formatter={(value: number) => [formatCurrency(value), "Revenue"]}
                      />
                      <Area type="monotone" dataKey="revenue" stroke="#22c55e" strokeWidth={2} fill="url(#revenueGrad)" />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Breakdown</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable
                  columns={[
                    { key: "endpoint", header: "Endpoint", cell: (r: typeof analytics.revenueBreakdown[0]) => <span className="font-mono text-xs">{r.endpoint}</span> },
                    { key: "requests", header: "Requests", cell: (r) => r.requests.toLocaleString() },
                    { key: "revenue", header: "Revenue", cell: (r) => formatCurrency(r.revenue) },
                    { key: "latency", header: "Avg Latency", cell: (r) => `${r.avgLatency}ms` },
                    { key: "rate", header: "Success", cell: (r) => <span className="text-success">{(Math.random() * 5 + 95).toFixed(1)}%</span> },
                  ]}
                  data={analytics.revenueBreakdown.map((r, i) => ({ ...r, id: `rev-${i}` }))}
                  loading={loading}
                />
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}
