"use client"

import { useState, useMemo, useEffect } from "react"
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table"
import { Wallet, Calendar, Clock, TrendingUp, AlertCircle, Banknote, ArrowUpDown } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { CurrencyDisplay } from "@/components/CurrencyDisplay"
import { PageHeader } from "@/components/PageHeader"
import { StatCard } from "@/components/StatCard"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { EmptyState } from "@/components/EmptyState"
import { MOCK_SETTLEMENTS } from "@/lib/mock-data"
import { formatDate, truncateHash } from "@/lib/utils"
import type { Settlement } from "@/lib/types"

type SettlementStatus = Settlement["status"] | "all"

const STATUS_OPTIONS: SettlementStatus[] = ["all", "completed", "pending", "failed"]

export default function SettlementsPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [sorting, setSorting] = useState<SortingState>([])
  const [statusFilter, setStatusFilter] = useState<SettlementStatus>("all")

  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(timer)
  }, [])

  const settlements = useMemo(() => MOCK_SETTLEMENTS, [])

  const filteredSettlements = useMemo(() => {
    if (statusFilter === "all") return settlements
    return settlements.filter((s) => s.status === statusFilter)
  }, [settlements, statusFilter])

  const summary = useMemo(() => {
    const completed = settlements.filter((s) => s.status === "completed")
    const pending = settlements.filter((s) => s.status === "pending")
    const totalAllTime = completed.reduce((s, c) => s + c.amount, 0)
    const outstanding = pending.reduce((s, p) => s + p.amount, 0)
    const lastPayout = completed.length > 0 ? completed[0].amount : 0
    const lastPayoutDate = completed.length > 0 ? completed[0].date : null
    const last3 = completed.slice(0, 3)
    const nextEst = last3.length > 0
      ? Math.round((last3.reduce((s, c) => s + c.amount, 0) / last3.length) * 100) / 100
      : 0
    return { totalAllTime, outstanding, lastPayout, lastPayoutDate, nextEst }
  }, [settlements])

  const columns: ColumnDef<Settlement>[] = [
    {
      accessorKey: "date",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Date
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => (
        <span className="whitespace-nowrap text-sm">{formatDate(row.original.date)}</span>
      ),
    },
    {
      accessorKey: "amount",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Amount
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => (
        <span className="font-mono text-sm"><CurrencyDisplay amount={row.original.amount} /></span>
      ),
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Status
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "txHash",
      header: "Tx Hash",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <span className="font-mono text-xs text-muted-foreground">
            {truncateHash(row.original.txHash)}
          </span>
          <CopyButton value={row.original.txHash} />
        </div>
      ),
    },
  ]

  const table = useReactTable({
    data: filteredSettlements,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  if (error) {
    return (
      <div className="space-y-6">
        <PageHeader title="Settlements" description="View your payout history and pending settlements." />
        <div className="flex flex-col items-center gap-4 py-20">
          <AlertCircle className="size-12 text-destructive" />
          <h2 className="text-xl font-semibold">Failed to load settlements</h2>
          <Button onClick={() => setError(false)}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settlements"
        description="View your payout history and pending settlements."
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Outstanding Payout"
          value={loading ? null : <CurrencyDisplay amount={summary.outstanding} className="text-2xl font-semibold tracking-tight" />}
          subtitle="Pending"
          icon={Wallet}
          loading={loading}
        />
        <StatCard
          title="Last Payout"
          value={loading ? null : <CurrencyDisplay amount={summary.lastPayout} className="text-2xl font-semibold tracking-tight" />}
          subtitle={summary.lastPayoutDate ? formatDate(summary.lastPayoutDate) : "No payouts yet"}
          icon={Calendar}
          loading={loading}
        />
        <StatCard
          title="Next Est."
          value={loading ? null : <CurrencyDisplay amount={summary.nextEst} className="text-2xl font-semibold tracking-tight" />}
          subtitle="Est. next payout"
          icon={Clock}
          loading={loading}
        />
        <StatCard
          title="Total All Time"
          value={loading ? null : <CurrencyDisplay amount={summary.totalAllTime} className="text-2xl font-semibold tracking-tight" />}
          icon={TrendingUp}
          loading={loading}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Settlement History</CardTitle>
          <CardAction>
            <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="outline" size="sm" />}>
              {statusFilter === "all" ? "All Status" : statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)}
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {STATUS_OPTIONS.map((s) => (
                <DropdownMenuItem key={s} onClick={() => setStatusFilter(s)}>
                  {statusFilter === s && "✓ "}
                  {s === "all" ? "All Status" : s.charAt(0).toUpperCase() + s.slice(1)}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          </CardAction>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-10 w-full" />
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : filteredSettlements.length === 0 ? (
            statusFilter !== "all" ? (
              <EmptyState
                icon={Banknote}
                title="No matching settlements"
                description={`No settlements with status "${statusFilter}". Try a different filter.`}
              />
            ) : (
              <EmptyState
                icon={Banknote}
                title="No settlements yet"
                description="They will appear after your first payout cycle."
              />
            )
          ) : (
            <div className="rounded-xl border">
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow key={headerGroup.id}>
                      {headerGroup.headers.map((header) => (
                        <TableHead key={header.id}>
                          {header.isPlaceholder
                            ? null
                            : flexRender(header.column.columnDef.header, header.getContext())}
                        </TableHead>
                      ))}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.map((row) => (
                    <TableRow key={row.id}>
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id}>
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
