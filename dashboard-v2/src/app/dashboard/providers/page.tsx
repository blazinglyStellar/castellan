"use client"

import * as React from "react"
import { useState } from "react"
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
import {
  ArrowUpDown,
  Globe,
  Plus,
  Search,
  MoreHorizontal,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { StatCard } from "@/components/StatCard"
import { PageHeader } from "@/components/PageHeader"
import { StatusBadge } from "@/components/StatusBadge"
import { EmptyState } from "@/components/EmptyState"
import { formatCompact, formatCurrency, formatLatency, formatDate } from "@/lib/utils"
import { StellarLogo } from "@/components/StellarLogo"
import { MOCK_PROVIDER_APIS } from "@/lib/mock-data"
import type { ProviderAPI, APIStatus } from "@/lib/types"

function KpiXlm({ amount }: { amount: number }) {
  return (
    <span className="inline-flex items-center gap-1 text-2xl font-semibold tracking-tight">
      <StellarLogo className="size-4" />
      {amount.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}
    </span>
  )
}

function KpiMs({ ms }: { ms: number }) {
  if (ms === 0) return <span className="text-2xl font-semibold tracking-tight">-</span>
  const display = ms >= 1000 ? (ms / 1000).toFixed(1) : String(ms)
  return (
    <span className="text-2xl font-semibold tracking-tight">
      {display}
      <span className="text-xs text-muted-foreground font-normal"> ms</span>
    </span>
  )
}

const ALL_STATUSES: APIStatus[] = ["active", "inactive", "rate_limited"]

function APIActions({ api }: { api: ProviderAPI }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" className="h-8 w-8 p-0" />}>
        <span className="sr-only">Open menu</span>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Actions</DropdownMenuLabel>
        <DropdownMenuItem>View Details</DropdownMenuItem>
        <DropdownMenuItem>Edit API</DropdownMenuItem>
        <DropdownMenuSeparator />
        {api.status === "active" ? (
          <DropdownMenuItem>Disable</DropdownMenuItem>
        ) : (
          <DropdownMenuItem>Enable</DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem data-variant="destructive">Delete</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export default function MyAPIsPage() {
  const [loading, setLoading] = useState(true)
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [rowSelection, setRowSelection] = useState({})
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<APIStatus | "all">("all")

  const apis = React.useMemo(() => MOCK_PROVIDER_APIS, [])

  const filteredApis = React.useMemo(() => {
    return apis.filter((api) => {
      const matchesSearch =
        !search ||
        api.name.toLowerCase().includes(search.toLowerCase()) ||
        api.endpoint.toLowerCase().includes(search.toLowerCase()) ||
        api.description.toLowerCase().includes(search.toLowerCase())
      const matchesStatus = statusFilter === "all" || api.status === statusFilter
      return matchesSearch && matchesStatus
    })
  }, [apis, search, statusFilter])

  React.useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(timer)
  }, [])

  const summary = React.useMemo(() => {
    return {
      total: apis.length,
      totalRequests: apis.reduce((s, a) => s + a.totalRequests, 0),
      totalEarnings: apis.reduce((s, a) => s + a.totalEarnings, 0),
      avgLatency: apis.length > 0 ? Math.round(apis.reduce((s, a) => s + a.avgLatency, 0) / apis.length) : 0,
    }
  }, [apis])

  const columns: ColumnDef<ProviderAPI>[] = [
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
      accessorKey: "name",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          API Name
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => {
        const api = row.original
        return (
          <div className="flex items-center gap-2.5">
            <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
              <Globe className="size-4 text-primary" />
            </div>
            <div>
              <div className="font-medium">{api.name}</div>
              <div className="text-xs text-muted-foreground">{api.endpoint}</div>
            </div>
          </div>
        )
      },
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
      accessorKey: "totalRequests",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Requests
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => (
        <span className="font-mono text-sm">{formatCompact(row.original.totalRequests)}</span>
      ),
    },
    {
      accessorKey: "totalEarnings",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Earnings
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => <span className="font-mono text-sm">{formatCurrency(row.original.totalEarnings)}</span>,
    },
    {
      accessorKey: "avgLatency",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Latency
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => <span className="font-mono text-sm">{formatLatency(row.original.avgLatency)}</span>,
    },
    {
      accessorKey: "errorRate",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Errors
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => <span className="font-mono text-sm">{row.original.errorRate}%</span>,
    },
    {
      accessorKey: "createdAt",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Created
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => (
        <span className="whitespace-nowrap text-sm text-muted-foreground">
          {formatDate(row.original.createdAt)}
        </span>
      ),
    },
    {
      id: "actions",
      cell: ({ row }) => <APIActions api={row.original} />,
    },
  ]

  const table = useReactTable({
    data: filteredApis,
    columns,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    initialState: {
      pagination: { pageSize: 10 },
    },
  })

  return (
    <div className="space-y-6">
      <PageHeader
        title="My APIs"
        description="Manage your registered API endpoints"
        actions={
          <Dialog>
            <DialogTrigger render={<Button><Plus className="size-4" /> Register API</Button>} />
              <DialogContent className="sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>Register New API</DialogTitle>
                  <DialogDescription>Add a new API endpoint to your provider profile.</DialogDescription>
                </DialogHeader>
                <div className="flex flex-col gap-5 px-1 py-4">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">API Name</label>
                  <Input placeholder="e.g. Weather API" />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Endpoint</label>
                  <Input placeholder="e.g. /weather" />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Description</label>
                  <Input placeholder="Brief description of your API" />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Price per request</label>
                  <div className="relative">
                    <Input placeholder="0.00" className="pr-12" />
                    <span className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"><StellarLogo className="size-3.5" /></span>
                  </div>
                </div>
              </div>
              <DialogFooter showCloseButton>
                <Button>Register</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Total APIs" value={loading ? null : String(summary.total)} icon={Globe} loading={loading} />
        <StatCard
          title="Total Requests"
          value={loading ? null : formatCompact(summary.totalRequests)}
          icon={Globe}
          loading={loading}
        />
        <StatCard
          title="Total Earnings"
          value={loading ? null : <KpiXlm amount={summary.totalEarnings} />}
          icon={Globe}
          loading={loading}
        />
        <StatCard
          title="Avg Latency"
          value={loading ? null : <KpiMs ms={summary.avgLatency} />}
          icon={Globe}
          loading={loading}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search APIs..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-56 pl-8"
            />
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="outline" size="sm" />}>
              {statusFilter === "all" ? "All Status" : statusFilter.replace("_", " ")}
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setStatusFilter("all")}>
                {statusFilter === "all" && "✓ "}All Status
              </DropdownMenuItem>
              {ALL_STATUSES.map((s) => (
                <DropdownMenuItem key={s} onClick={() => setStatusFilter(s)}>
                  {statusFilter === s && "✓ "}
                  {s.replace("_", " ").charAt(0).toUpperCase() + s.replace("_", " ").slice(1)}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {loading ? (
        <div className="space-y-3">
          <Skeleton className="h-10 w-full" />
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : filteredApis.length === 0 ? (
        <EmptyState
          icon={Globe}
          title="No APIs found"
          description={search || statusFilter !== "all" ? "Try a different search or filter." : "Register your first API to get started."}
          action={
            !search && statusFilter === "all" ? (
              <Dialog>
                <DialogTrigger render={<Button><Plus className="size-4" /> Register API</Button>} />
                <DialogContent className="sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>Register New API</DialogTitle>
                    <DialogDescription>Add a new API endpoint to your provider profile.</DialogDescription>
                  </DialogHeader>
                  <div className="flex flex-col gap-5 px-1 py-4">
                    <div className="space-y-1.5">
                      <label className="text-sm font-medium">API Name</label>
                      <Input placeholder="e.g. Weather API" />
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-sm font-medium">Endpoint</label>
                      <Input placeholder="e.g. /weather" />
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-sm font-medium">Description</label>
                      <Input placeholder="Brief description of your API" />
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-sm font-medium">Price per request</label>
                      <div className="relative">
                        <Input placeholder="0.00" className="pr-12" />
                        <span className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"><StellarLogo className="size-3.5" /></span>
                      </div>
                    </div>
                  </div>
                  <DialogFooter showCloseButton>
                    <Button>Register</Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            ) : undefined
          }
        />
      ) : (
        <div className="space-y-4">
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
                  <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
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

          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {table.getFilteredSelectedRowModel().rows.length} of {table.getFilteredRowModel().rows.length} selected
            </span>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => table.previousPage()}
                disabled={!table.getCanPreviousPage()}
              >
                Prev
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
              </span>
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
    </div>
  )
}
