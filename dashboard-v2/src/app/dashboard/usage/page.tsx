"use client"

import { useState, useMemo } from "react"
import { Search, Activity, TrendingUp, AlertCircle, Banknote, ArrowUpDown } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
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
import { StatCard } from "@/components/StatCard"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { CurrencyDisplay } from "@/components/CurrencyDisplay"
import { MOCK_USAGE_EVENTS } from "@/lib/mock-data"
import { formatCompact, formatDateTime } from "@/lib/utils"
import type { ColumnDef } from "@tanstack/react-table"
import type { UsageEvent } from "@/lib/types"

const columns: ColumnDef<UsageEvent>[] = [
  {
    accessorKey: "date",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Date
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => formatDateTime(row.getValue("date")),
  },
  {
    accessorKey: "api",
    header: "API",
    cell: ({ row }) => <span className="text-xs">{row.getValue("api")}</span>,
  },
  {
    accessorKey: "endpoint",
    header: "Endpoint",
    cell: ({ row }) => (
      <span className="font-mono text-xs text-muted-foreground">{row.getValue("endpoint")}</span>
    ),
  },
  {
    accessorKey: "method",
    header: "Method",
    cell: ({ row }) => {
      const method: string = row.getValue("method")
      const colors: Record<string, string> = {
        GET: "bg-green/15 text-green border-green/20",
        POST: "bg-amber/15 text-amber border-amber/20",
        PUT: "bg-chart-1/15 text-chart-1 border-chart-1/20",
        DELETE: "bg-red/15 text-red border-red/20",
      }
      return (
        <span
          className={`inline-flex items-center rounded-md border px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase leading-none ${colors[method] || "bg-muted text-muted-foreground border-border"}`}
        >
          {method}
        </span>
      )
    },
  },
  {
    accessorKey: "cost",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Cost
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => <CurrencyDisplay amount={row.getValue("cost")} />,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Status
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => {
      const status: number = row.getValue("status")
      const color =
        status >= 200 && status < 300
          ? "text-green"
          : status >= 400 && status < 500
          ? "text-amber"
          : "text-red"
      return <span className={`font-mono text-xs font-medium ${color}`}>{status}</span>
    },
  },
]

const STATUS_GROUPS = [
  { value: "all", label: "All Status" },
  { value: "success", label: "Success (2xx)" },
  { value: "client_error", label: "Client Error (4xx)" },
  { value: "server_error", label: "Server Error (5xx)" },
] as const

function matchStatusGroup(status: number, group: string): boolean {
  if (group === "all") return true
  if (group === "success") return status >= 200 && status < 300
  if (group === "client_error") return status >= 400 && status < 500
  if (group === "server_error") return status >= 500 && status < 600
  return true
}

export default function UsagePage() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [apiFilter, setApiFilter] = useState("all")
  const [statusFilter, setStatusFilter] = useState("all")
  const [daysFilter, setDaysFilter] = useState(30)

  const uniqueAPIs = useMemo(
    () => ["all", ...new Set(MOCK_USAGE_EVENTS.map((e) => e.api))],
    []
  )

  const filtered = useMemo(() => {
    const cutoff = new Date()
    cutoff.setDate(cutoff.getDate() - daysFilter)
    return MOCK_USAGE_EVENTS.filter((e) => {
      if (new Date(e.date) < cutoff) return false
      if (apiFilter !== "all" && e.api !== apiFilter) return false
      if (!matchStatusGroup(e.status, statusFilter)) return false
      if (searchQuery) {
        const q = searchQuery.toLowerCase()
        if (
          !e.api.toLowerCase().includes(q) &&
          !e.endpoint.toLowerCase().includes(q)
        )
          return false
      }
      return true
    })
  }, [daysFilter, apiFilter, statusFilter, searchQuery])

  const summary = useMemo(() => {
    const totalRequests = filtered.length
    const totalSpent = filtered.reduce((sum, e) => sum + e.cost, 0)
    const avgCost = totalRequests > 0 ? totalSpent / totalRequests : 0
    return { totalRequests, totalSpent, avgCost }
  }, [filtered])

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load usage data</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Usage History"
        description="Track your API consumption and spending."
      />

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-3">
          <StatCard
            title="Total Requests"
            value={formatCompact(summary.totalRequests)}
            icon={Activity}
          />
          <StatCard
            title="Total Spent"
            value={<CurrencyDisplay amount={summary.totalSpent} className="text-2xl font-semibold tracking-tight" />}
            icon={Banknote}
          />
          <StatCard
            title="Avg Cost / Request"
            value={<CurrencyDisplay amount={summary.avgCost} className="text-2xl font-semibold tracking-tight" />}
            icon={TrendingUp}
          />
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by API or endpoint..."
            className="pl-9"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <Select value={apiFilter} onValueChange={(v) => v && setApiFilter(v)}>
          <SelectTrigger className="w-44">
            <SelectValue>
              {apiFilter === "all" ? "API: All" : `API: ${apiFilter}`}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {uniqueAPIs.map((api) => (
              <SelectItem key={api} value={api}>
                {api === "all" ? "All APIs" : api}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={statusFilter} onValueChange={(v) => v && setStatusFilter(v)}>
          <SelectTrigger className="w-44">
            <SelectValue>
              {statusFilter === "all" ? "Status: All" : `Status: ${STATUS_GROUPS.find(g => g.value === statusFilter)?.label ?? statusFilter}`}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {STATUS_GROUPS.map((g) => (
              <SelectItem key={g.value} value={g.value}>
                {g.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex gap-1">
          {[7, 30, 90].map((d) => (
            <Button
              key={d}
              variant={daysFilter === d ? "default" : "outline"}
              size="sm"
              onClick={() => setDaysFilter(d)}
            >
              {d}d
            </Button>
          ))}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Usage Log</CardTitle>
        </CardHeader>
        <CardContent>
          {filtered.length === 0 ? (
            <EmptyState
              icon={Search}
              title="No usage records match your filters"
              description="Try adjusting the search or filter criteria."
            />
          ) : (
            <DataTable columns={columns} data={filtered} loading={loading} />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
