"use client"

import { useState, useMemo } from "react"
import { ArrowUpDown } from "lucide-react"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

type SortKey = "route" | "cost" | "volume"

interface CostRow {
  route: string
  cost: number
  calls: number
}

interface UsageCostBreakdownProps {
  events: { request_cost: string; route: string }[]
}

function SortableHead({
  label,
  sortKey,
  currentKey,
  direction,
  onToggle,
}: {
  label: string
  sortKey: SortKey
  currentKey: SortKey
  direction: "asc" | "desc"
  onToggle: (k: SortKey) => void
}) {
  const active = currentKey === sortKey
  return (
    <TableHead>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => onToggle(sortKey)}
        className={cn(
          "-ml-3 h-8 gap-1 text-xs font-medium",
          active ? "text-foreground" : "text-muted-foreground",
        )}
      >
        {label}
        <ArrowUpDown className="h-3 w-3" />
      </Button>
    </TableHead>
  )
}

export function UsageCostBreakdown({ events }: UsageCostBreakdownProps) {
  const [sortKey, setSortKey] = useState<SortKey>("cost")
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc")

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"))
    } else {
      setSortKey(key)
      setSortDir(key === "route" ? "asc" : "desc")
    }
  }

  const byRoute = useMemo(() => {
    const map = new Map<string, { cost: number; calls: number }>()
    for (const ev of events) {
      const existing = map.get(ev.route) || { cost: 0, calls: 0 }
      existing.cost += parseFloat(ev.request_cost)
      existing.calls += 1
      map.set(ev.route, existing)
    }
    return map
  }, [events])

  const rawRows = useMemo(() => {
    const rows: CostRow[] = []
    for (const [route, v] of byRoute) {
      rows.push({ route, cost: v.cost, calls: v.calls })
    }
    return rows
  }, [byRoute])

  const totalCost = useMemo(
    () => rawRows.reduce((s, r) => s + r.cost, 0),
    [rawRows],
  )

  const rows = useMemo(() => {
    if (rawRows.length === 0) return []
    const sorted = [...rawRows]
    sorted.sort((a, b) => {
      let cmp: number
      if (sortKey === "route") {
        cmp = a.route.localeCompare(b.route)
      } else if (sortKey === "cost") {
        cmp = a.cost - b.cost
      } else {
        cmp = a.calls - b.calls
      }
      return sortDir === "asc" ? cmp : -cmp
    })
    return sorted
  }, [rawRows, sortKey, sortDir])

  if (events.length === 0) {
    return (
      <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
        No cost data for the selected period
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <SortableHead
            label="Endpoint"
            sortKey="route"
            currentKey={sortKey}
            direction={sortDir}
            onToggle={toggleSort}
          />
          <SortableHead
            label="Total Cost"
            sortKey="cost"
            currentKey={sortKey}
            direction={sortDir}
            onToggle={toggleSort}
          />
          <SortableHead
            label="Volume"
            sortKey="volume"
            currentKey={sortKey}
            direction={sortDir}
            onToggle={toggleSort}
          />
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => {
          const share = totalCost > 0 ? (row.cost / totalCost) * 100 : 0
          return (
            <TableRow key={row.route}>
              <TableCell className="font-mono text-xs">{row.route}</TableCell>
              <TableCell className="font-medium">
                {row.cost.toFixed(4)} XLM
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <div className="h-2 w-20 rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{ width: `${share}%` }}
                    />
                  </div>
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {row.calls} / {share.toFixed(1)}%
                  </span>
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
