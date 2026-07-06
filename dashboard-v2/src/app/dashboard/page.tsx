"use client"

import * as React from "react"
import { useState, useEffect } from "react"
import { AreaChart, Area, CartesianGrid, XAxis, YAxis } from "recharts"
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table"
import { ArrowUpDown, Banknote, Activity, Globe, Clock, Plus, ChevronDown, ChevronLeft, ChevronRight, MoreHorizontal } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import { StatCard } from "@/components/StatCard"
import { PageHeader } from "@/components/PageHeader"

import { StatusBadge } from "@/components/StatusBadge"
import { BalanceBadge } from "@/components/BalanceBadge"
import { EmptyState } from "@/components/EmptyState"
import { formatCurrency, formatLatency } from "@/lib/utils"
import { StellarLogo } from "@/components/StellarLogo"
import {
  MOCK_PROVIDER_OVERVIEW,
  MOCK_CONSUMER_OVERVIEW,
  MOCK_RECENT_CALLS,
  MOCK_CONSUMER_USAGE,
  MOCK_USAGE_TREND,
} from "@/lib/mock-data"
import type { RecentCall, ConsumerUsage, Role } from "@/lib/types"

function XlmValue({ amount }: { amount: number }) {
  return (
    <span className="inline-flex items-center gap-1 text-2xl font-semibold tracking-tight">
      <StellarLogo className="size-4" />
      {amount.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}
    </span>
  )
}

const recentCallColumns: ColumnDef<RecentCall>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected()
            ? true
            : table.getIsSomePageRowsSelected()
              ? ("indeterminate" as unknown as boolean)
              : false
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "time",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Time
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => {
      const t = new Date(Date.now() - row.original.time * 60 * 1000)
      return (
        <span className="text-muted-foreground text-xs whitespace-nowrap">
          {t.toLocaleDateString("en-US", { month: "short", day: "numeric" })}
          {" "}
          {t.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" })}
        </span>
      )
    },
  },
  {
    accessorKey: "endpoint",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Endpoint
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs">{row.original.endpoint}</span>
    ),
  },
  {
    accessorKey: "cost",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Cost
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs">{formatCurrency(row.original.cost)}</span>
    ),
  },
  {
    accessorKey: "latency",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Latency
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="text-muted-foreground">{formatLatency(row.original.latency)}</span>
    ),
  },
  {
    accessorKey: "consumer",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Consumer
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="text-muted-foreground">{row.original.consumer}</span>
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
  },
  {
    id: "actions",
    enableHiding: false,
    cell: ({ row }) => {
      const call = row.original
      return (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" className="h-8 w-8 p-0" />}>
            <span className="sr-only">Open menu</span>
            <MoreHorizontal />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>Actions</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => navigator.clipboard.writeText(call.endpoint)}>
              Copy endpoint path
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem>View details</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )
    },
  },
]

const consumerUsageColumns: ColumnDef<ConsumerUsage>[] = [
  {
    accessorKey: "api",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        API
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => <span className="text-xs">{row.original.api}</span>,
  },
  {
    accessorKey: "endpoint",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Endpoint
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs text-muted-foreground">
        {row.original.endpoint}
      </span>
    ),
  },
  {
    accessorKey: "cost",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Cost
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs">{formatCurrency(row.original.cost)}</span>
    ),
  },
  {
    accessorKey: "time",
    header: ({ column }) => (
      <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
        Time
        <ArrowUpDown />
      </Button>
    ),
    cell: ({ row }) => {
      const t = new Date(Date.now() - row.original.time * 60 * 1000)
      return (
        <span className="text-muted-foreground text-xs whitespace-nowrap">
          {t.toLocaleDateString("en-US", { month: "short", day: "numeric" })}
          {" "}
          {t.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" })}
        </span>
      )
    },
  },
]

type Interval = "24h" | "7d" | "30d"

const intervals: Interval[] = ["24h", "7d", "30d"]

function generateEarningsData(interval: Interval): { date: string; amount: number }[] {
  const now = Date.now()
  if (interval === "24h") {
    return Array.from({ length: 12 }, (_, i) => ({
      date: new Date(now - (11 - i) * 2 * 60 * 60 * 1000).toLocaleTimeString("en-US", { hour: "2-digit" }),
      amount: +(10 + Math.random() * 30).toFixed(2),
    }))
  }
  if (interval === "30d") {
    return Array.from({ length: 30 }, (_, i) => ({
      date: new Date(now - (29 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      amount: +(120 + Math.random() * 120).toFixed(2),
    }))
  }
  return MOCK_PROVIDER_OVERVIEW.earningsByDay
}

function generateUsageTrendData(interval: Interval): { date: string; requests: number }[] {
  const now = Date.now()
  if (interval === "24h") {
    return Array.from({ length: 12 }, (_, i) => ({
      date: new Date(now - (11 - i) * 2 * 60 * 60 * 1000).toLocaleTimeString("en-US", { hour: "2-digit" }),
      requests: 200 + Math.floor(Math.random() * 600),
    }))
  }
  if (interval === "30d") {
    return Array.from({ length: 30 }, (_, i) => ({
      date: new Date(now - (29 - i) * 24 * 60 * 60 * 1000).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      requests: 300 + Math.floor(Math.random() * 500),
    }))
  }
  return MOCK_USAGE_TREND
}

const earningsChartConfig = {
  amount: {
    label: "Earnings",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig

const usageTrendChartConfig = {
  requests: {
    label: "Requests",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig

function ProviderView() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [earningsInterval, setEarningsInterval] = useState<Interval>("7d")
  const overview = MOCK_PROVIDER_OVERVIEW
  const earningsData = generateEarningsData(earningsInterval)
  const calls = MOCK_RECENT_CALLS
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [rowSelection, setRowSelection] = useState({})

  const table = useReactTable({
    data: calls,
    columns: recentCallColumns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    state: { sorting, columnFilters, columnVisibility, rowSelection },
  })

  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(timer)
  }, [])

  if (error) {
    return (
      <div className="flex flex-col items-center gap-3 py-16">
        <p className="text-sm text-red">Failed to load provider overview.</p>
        <Button variant="outline" size="sm" onClick={() => setError(false)}>
          Retry
        </Button>
      </div>
    )
  }

  const hasNoApis = !loading && overview.totalEndpoints === 0
  const hasNoUsage = !loading && !hasNoApis && calls.length === 0

  if (hasNoApis) {
    return (
      <div className="space-y-6">
        <PageHeader title="Provider Overview" description="Register your first API to get started" />
        <EmptyState
          icon={Globe}
          title="No APIs registered"
          description="Register your first API to start monetizing."
          action={<Button size="sm"><Plus className="size-4" />Add API</Button>}
        />
      </div>
    )
  }

  if (hasNoUsage) {
    return (
      <div className="space-y-6">
        <PageHeader title="Provider Overview" description="Share your API key with consumers" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard title="Total Earned" value={<XlmValue amount={overview.totalEarned} />} icon={Banknote} loading={loading} />
          <StatCard title="Requests This Week" value={overview.requestsThisWeek.toLocaleString()} icon={Activity} loading={loading} />
          <StatCard title="Active Endpoints" value={`${overview.activeEndpoints} / ${overview.totalEndpoints}`} icon={Globe} loading={loading} />
          <StatCard title="Pending Settlement" value={<XlmValue amount={overview.pendingSettlement} />} icon={Clock} loading={loading} />
        </div>
        <EmptyState
          icon={Activity}
          title="No usage data yet"
          description="Share your API key with consumers to start seeing usage."
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Good morning, developer@castellan.io"
        description="Here&apos;s what happened with your APIs."
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Total Earned" value={<XlmValue amount={overview.totalEarned} />} icon={Banknote} loading={loading} />
        <StatCard title="Requests This Week" value={overview.requestsThisWeek.toLocaleString()} icon={Activity} loading={loading} />
        <StatCard title="Active Endpoints" value={`${overview.activeEndpoints} / ${overview.totalEndpoints}`} icon={Globe} loading={loading} />
        <StatCard title="Pending Settlement" value={<XlmValue amount={overview.pendingSettlement} />} icon={Clock} loading={loading} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Earnings</CardTitle>
          <CardAction>
            <div className="flex items-center gap-1">
              {intervals.map((i) => (
                <Button
                  key={i}
                  variant={earningsInterval === i ? "default" : "outline"}
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setEarningsInterval(i)}
                >
                  {i}
                </Button>
              ))}
            </div>
          </CardAction>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="h-[250px] animate-pulse rounded-lg bg-muted" />
          ) : (
            <ChartContainer config={earningsChartConfig} className="aspect-auto h-[250px] w-full">
              <AreaChart data={earningsData}>
                <defs>
                  <linearGradient id="fillAmount" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-amount)" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="var(--color-amount)" stopOpacity={0} />
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
                  type="monotone"
                  dataKey="amount"
                  stroke="var(--color-amount)"
                  fill="url(#fillAmount)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Recent API Calls</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {loading ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  {recentCallColumns.filter((c) => c.id !== "select" && c.id !== "actions").map((col, i) => (
                    <TableHead key={i}>
                      {typeof col.header === "string" ? col.header : ""}
                    </TableHead>
                  ))}
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="size-4 rounded-sm" /></TableCell>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton className="h-4 w-full max-w-[100px]" />
                      </TableCell>
                    ))}
                    <TableCell><Skeleton className="size-8 rounded-md" /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : calls.length === 0 ? (
            <EmptyState title="No data" description="No recent API calls found." />
          ) : (
            <div className="w-full">
              <div className="flex items-center px-4 py-4">
                <Input
                  placeholder="Filter endpoints..."
                  value={(table.getColumn("endpoint")?.getFilterValue() as string) ?? ""}
                  onChange={(event) =>
                    table.getColumn("endpoint")?.setFilterValue(event.target.value)
                  }
                  className="max-w-sm"
                />
                <DropdownMenu>
                  <DropdownMenuTrigger render={<Button variant="outline" className="ml-auto" />}>
                    Columns <ChevronDown />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                  {table
                    .getAllColumns()
                    .filter((column) => column.getCanHide())
                    .map((column) => (
                      <DropdownMenuCheckboxItem
                        key={column.id}
                        className="capitalize"
                        checked={column.getIsVisible()}
                        onCheckedChange={(value) => column.toggleVisibility(!!value)}
                      >
                        {column.id}
                      </DropdownMenuCheckboxItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <div className="overflow-hidden rounded-md border mx-4 mb-4">
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
                    {table.getRowModel().rows.length ? (
                      table.getRowModel().rows.map((row) => (
                        <TableRow
                          key={row.id}
                          data-state={row.getIsSelected() && "selected"}
                        >
                          {row.getVisibleCells().map((cell) => (
                            <TableCell key={cell.id}>
                              {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={recentCallColumns.length} className="h-24 text-center">
                          No results.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
              <div className="flex items-center justify-end px-4 pb-4">
                <div className="flex-1 text-sm text-muted-foreground">
                  {table.getFilteredSelectedRowModel().rows.length} of{" "}
                  {table.getFilteredRowModel().rows.length} row(s) selected.
                </div>
                <div className="space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => table.previousPage()}
                    disabled={!table.getCanPreviousPage()}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => table.nextPage()}
                    disabled={!table.getCanNextPage()}
                  >
                    Next
                  </Button>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function ConsumerView() {
  const [loading, setLoading] = useState(true)
  const [sorting, setSorting] = useState<SortingState>([])
  const [usageInterval, setUsageInterval] = useState<Interval>("7d")
  const overview = MOCK_CONSUMER_OVERVIEW
  const usage = MOCK_CONSUMER_USAGE
  const usageTrendData = generateUsageTrendData(usageInterval)

  const table = useReactTable({
    data: usage,
    columns: consumerUsageColumns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  })

  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(timer)
  }, [])

  const hasNoDeposits = !loading && overview.balance === 0
  const hasNoUsage = !loading && overview.balance > 0 && overview.spentThisMonth === 0

  if (hasNoDeposits) {
    return (
      <div className="space-y-6">
        <PageHeader title="Consumer Overview" description="Deposit funds to get started" />
        <EmptyState
          icon={Banknote}
          title="No funds deposited"
          description="Deposit XLM to start using APIs."
          action={<Button size="sm">Deposit</Button>}
        />
      </div>
    )
  }

  if (hasNoUsage) {
    return (
      <div className="space-y-6">
        <PageHeader title="Consumer Overview" description="Start calling APIs to see usage" />
        <div className="flex items-center gap-4 rounded-lg border p-4">
          <BalanceBadge balance={overview.balance} loading={loading} />
          <span className="text-sm text-muted-foreground">
            Spent this month: <span className="font-mono font-medium text-foreground">{formatCurrency(overview.spentThisMonth)}</span>
          </span>
        </div>
        <EmptyState
          icon={Activity}
          title="No usage yet"
          description="Get an API key and start making requests."
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Welcome back" />

      <div className="flex flex-wrap items-center gap-4 rounded-lg border p-4">
        <BalanceBadge balance={overview.balance} loading={loading} />
        <span className="text-sm text-muted-foreground">
          Spent this month: <span className="font-mono font-medium text-foreground">{formatCurrency(overview.spentThisMonth)}</span>
        </span>
        <Button size="sm" className="ml-auto">Deposit</Button>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Recent Usage</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    {consumerUsageColumns.map((col, i) => (
                      <TableHead key={i}>
                        {typeof col.header === "string" ? col.header : ""}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {Array.from({ length: 3 }).map((_, i) => (
                    <TableRow key={i}>
                      {consumerUsageColumns.map((_, j) => (
                        <TableCell key={j}>
                          <Skeleton className="h-4 w-full max-w-[100px]" />
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div>
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
                    {table.getRowModel().rows.length ? (
                      table.getRowModel().rows.map((row) => (
                        <TableRow key={row.id}>
                          {row.getVisibleCells().map((cell) => (
                            <TableCell key={cell.id}>
                              {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={consumerUsageColumns.length} className="h-24 text-center">
                          No results.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
                {table.getPageCount() > 1 && (
                  <div className="flex items-center justify-end gap-2 border-t px-4 py-3">
                    <div className="flex-1 text-sm text-muted-foreground">
                      Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => table.previousPage()}
                        disabled={!table.getCanPreviousPage()}
                      >
                        <ChevronLeft className="size-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => table.nextPage()}
                        disabled={!table.getCanNextPage()}
                      >
                        <ChevronRight className="size-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Top Providers by Spend</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                {Array.from({ length: 3 }).map((_, i) => (
                  <div key={i} className="h-8 animate-pulse rounded bg-muted" />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {overview.topProviders.map((p) => (
                  <div key={p.name} className="flex items-center justify-between">
                    <div>
                      <p className="text-sm">{p.name}</p>
                      <p className="text-xs text-muted-foreground">{p.calls.toLocaleString()} calls</p>
                    </div>
                    <span className="font-mono text-sm font-medium">{formatCurrency(p.spent)}</span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Usage Trend</CardTitle>
          <CardAction>
            <div className="flex items-center gap-1">
              {intervals.map((i) => (
                <Button
                  key={i}
                  variant={usageInterval === i ? "default" : "outline"}
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setUsageInterval(i)}
                >
                  {i}
                </Button>
              ))}
            </div>
          </CardAction>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="h-[250px] animate-pulse rounded-lg bg-muted" />
          ) : (
            <ChartContainer config={usageTrendChartConfig} className="aspect-auto h-[250px] w-full">
              <AreaChart data={usageTrendData}>
                <defs>
                  <linearGradient id="fillRequests" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-requests)" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="var(--color-requests)" stopOpacity={0} />
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
                  type="monotone"
                  dataKey="requests"
                  stroke="var(--color-requests)"
                  fill="url(#fillRequests)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function DashboardOverview() {
  const [role, setRole] = useState<Role>("both")

  return (
    <div className="space-y-8">
      <div className="flex items-center gap-1.5 border-b pb-4">
        <Button
          variant={role === "provider" ? "default" : "ghost"}
          size="sm"
          onClick={() => setRole("provider")}
        >
          Provider
        </Button>
        <Button
          variant={role === "consumer" ? "default" : "ghost"}
          size="sm"
          onClick={() => setRole("consumer")}
        >
          Consumer
        </Button>
        <Button
          variant={role === "both" ? "default" : "ghost"}
          size="sm"
          onClick={() => setRole("both")}
        >
          Both
        </Button>
      </div>

      {(role === "provider" || role === "both") && (
        <section>
          {role === "both" && (
            <div className="mb-4 flex items-center gap-2">
              <div className="h-1.5 w-1.5 rounded-full bg-primary" />
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Provider</span>
            </div>
          )}
          <ProviderView />
        </section>
      )}

      {(role === "consumer" || role === "both") && role === "both" && <Separator className="my-2" />}

      {(role === "consumer" || role === "both") && (
        <section>
          {role === "both" && (
            <div className="mb-4 flex items-center gap-2">
              <div className="h-1.5 w-1.5 rounded-full bg-primary" />
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Consumer</span>
            </div>
          )}
          <ConsumerView />
        </section>
      )}
    </div>
  )
}
