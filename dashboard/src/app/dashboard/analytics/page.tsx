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
import { AlertCircle, BarChart3 } from "lucide-react"
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
                      <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                      <XAxis dataKey="date" className="text-xs text-muted-foreground" />
                      <YAxis className="text-xs text-muted-foreground" />
                      <Tooltip
                        contentStyle={{ background: "hsl(var(--popover))", border: "1px solid hsl(var(--border))", borderRadius: "8px" }}
                      />
                      <Area type="monotone" dataKey="weather" stackId="1" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.3} />
                      <Area type="monotone" dataKey="geo" stackId="1" stroke="#10b981" fill="#10b981" fillOpacity={0.3} />
                      <Area type="monotone" dataKey="ai" stackId="1" stroke="#f59e0b" fill="#f59e0b" fillOpacity={0.3} />
                      <Area type="monotone" dataKey="email" stackId="1" stroke="#8b5cf6" fill="#8b5cf6" fillOpacity={0.3} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>
          )}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Revenue Breakdown</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable
                columns={[
                  { key: "endpoint", header: "Endpoint", cell: (r: typeof analytics.revenueBreakdown[0]) => <span className="font-mono text-xs">{r.endpoint}</span> },
                  { key: "requests", header: "Requests", cell: (r) => r.requests.toLocaleString() },
                  { key: "revenue", header: "Revenue", cell: (r) => formatCurrency(r.revenue) },
                  { key: "latency", header: "Avg Latency", cell: (r) => `${r.avgLatency}ms` },
                ]}
                data={analytics.revenueBreakdown.map((r, i) => ({ ...r, id: `rev-${i}` }))}
                loading={loading}
              />
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
