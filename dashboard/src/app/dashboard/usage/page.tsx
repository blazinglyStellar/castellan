"use client"

import { useState } from "react"
import { AlertCircle, Activity, Download } from "lucide-react"
import { Button } from "@/components/ui/button"
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
import { StatusBadge } from "@/components/StatusBadge"
import { DateRangePicker } from "@/components/DateRangePicker"
import { MOCK_USAGE_EVENTS } from "@/lib/mock-data"
import { formatCurrency } from "@/lib/utils"
import type { UsageEvent } from "@/lib/mock-data"

const ITEMS_PER_PAGE = 5

export default function UsagePage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [statusFilter, setStatusFilter] = useState("all")
  const [currentPage, setCurrentPage] = useState(1)

  const filtered = MOCK_USAGE_EVENTS.filter((e) => {
    if (statusFilter === "200") return e.status === 200
    if (statusFilter === "402") return e.status === 402
    return true
  })

  const totalPages = Math.ceil(filtered.length / ITEMS_PER_PAGE)
  const paginated = filtered.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE)

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
        actions={
          <div className="flex items-center gap-2">
            <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setCurrentPage(1) }}>
              <SelectTrigger className="w-36"><SelectValue placeholder="Status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="200">200 OK</SelectItem>
                <SelectItem value="402">402 Pay Required</SelectItem>
              </SelectContent>
            </Select>
            <DateRangePicker />
            <Button variant="outline" size="sm" disabled>
              <Download className="h-4 w-4" />Export
            </Button>
          </div>
        }
      />
      {isEmpty ? (
        <EmptyState title="No usage data" description="Usage events will appear as you make API calls." icon={<Activity className="h-12 w-12" />} />
      ) : (
        <div className="rounded-lg border">
          <DataTable
            columns={[
              { key: "api", header: "API", cell: (e: UsageEvent) => e.api },
              { key: "endpoint", header: "Endpoint", cell: (e: UsageEvent) => <span className="font-mono text-xs">{e.endpoint}</span> },
              { key: "cost", header: "Cost", cell: (e: UsageEvent) => <span className="font-mono">{formatCurrency(e.cost)}</span> },
              { key: "status", header: "Status", cell: (e: UsageEvent) => (
                <StatusBadge status={e.status === 200 ? "completed" : "failed"} />
              )},
              { key: "date", header: "Date", cell: (e: UsageEvent) => new Date(e.date).toLocaleString() },
            ]}
            data={paginated}
            loading={loading}
          />
        </div>
      )}
      {!isEmpty && totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={currentPage === 1} onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}>
            Prev
          </Button>
          {Array.from({ length: totalPages }, (_, i) => (
            <Button
              key={i + 1}
              variant={currentPage === i + 1 ? "default" : "outline"}
              size="sm"
              className="min-w-9"
              onClick={() => setCurrentPage(i + 1)}
            >
              {i + 1}
            </Button>
          ))}
          <Button variant="outline" size="sm" disabled={currentPage === totalPages} onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}>
            Next
          </Button>
        </div>
      )}
    </div>
  )
}
