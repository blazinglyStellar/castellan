"use client"

import { useState, useMemo } from "react"
import {
  Key,
  Plus,
  Search,
  Trash2,
  AlertCircle,
  ArrowUpDown,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { PageHeader } from "@/components/PageHeader"
import { StatCard } from "@/components/StatCard"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { MOCK_API_KEYS } from "@/lib/mock-data"
import { formatDate } from "@/lib/utils"
import type { ColumnDef } from "@tanstack/react-table"
import type { ApiKey } from "@/lib/types"

export default function ApiKeysPage() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [keys, setKeys] = useState<ApiKey[]>(MOCK_API_KEYS)
  const [searchQuery, setSearchQuery] = useState("")
  const [genDialogOpen, setGenDialogOpen] = useState(false)
  const [label, setLabel] = useState("")
  const [revealDialogOpen, setRevealDialogOpen] = useState(false)
  const [newKeyValue, setNewKeyValue] = useState("")
  const [confirmRevoke, setConfirmRevoke] = useState<ApiKey | null>(null)

  const filtered = useMemo(() => {
    if (!searchQuery) return keys
    const q = searchQuery.toLowerCase()
    return keys.filter((k) => k.label.toLowerCase().includes(q) || k.prefix.toLowerCase().includes(q))
  }, [keys, searchQuery])

  const summary = useMemo(() => {
    const total = keys.length
    const active = keys.filter((k) => k.status === "active").length
    const revoked = keys.filter((k) => k.status === "revoked").length
    return { total, active, revoked }
  }, [keys])

  const handleGenerate = (e: React.FormEvent) => {
    e.preventDefault()
    const slug = label.toLowerCase().replace(/\s+/g, "_")
    const random = Math.random().toString(36).slice(2, 10)
    const raw = `cg_${slug}_${random}`
    const prefix = raw.slice(0, 12) + "••••"
    const newKey: ApiKey = {
      id: `ak-${Date.now()}`,
      label,
      prefix,
      status: "active",
      createdAt: new Date().toISOString(),
    }
    setKeys((prev) => [newKey, ...prev])
    setNewKeyValue(raw)
    setGenDialogOpen(false)
    setLabel("")
    setRevealDialogOpen(true)
  }

  const handleRevoke = () => {
    if (!confirmRevoke) return
    setKeys((prev) => prev.map((k) => (k.id === confirmRevoke.id ? { ...k, status: "revoked" } : k)))
    setConfirmRevoke(null)
  }

  const columns: ColumnDef<ApiKey>[] = useMemo(() => [
    {
      accessorKey: "label",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Label
          <ArrowUpDown />
        </Button>
      ),
    },
    {
      accessorKey: "prefix",
      header: "Key",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <span className="font-mono text-xs text-muted-foreground">{row.getValue("prefix")}</span>
          <CopyButton value={row.original.prefix} />
        </div>
      ),
    },
    {
      accessorKey: "expiresAt",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Expires
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => {
        const val: string | undefined = row.getValue("expiresAt")
        return val ? <span>{formatDate(val)}</span> : <span className="text-muted-foreground">—</span>
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
      cell: ({ row }) => <StatusBadge status={row.getValue("status")} />,
    },
    {
      accessorKey: "createdAt",
      header: ({ column }) => (
        <Button variant="ghost" onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}>
          Created
          <ArrowUpDown />
        </Button>
      ),
      cell: ({ row }) => formatDate(row.getValue("createdAt")),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const key = row.original
        return (
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="h-8 w-8"><Trash2 className="size-4" /></Button>} />
            <DropdownMenuContent align="end">
              {key.status === "active" ? (
                <DropdownMenuItem className="text-destructive" onClick={() => setConfirmRevoke(key)}>
                  Revoke
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem disabled>Already revoked</DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )
      },
    },
  ], [])

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load API keys</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="API Keys"
        description="Manage keys for programmatic access."
        actions={
          <Dialog open={genDialogOpen} onOpenChange={setGenDialogOpen}>
            <DialogTrigger render={<Button size="sm"><Plus className="size-4" /> Generate New Key</Button>} />
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Generate API Key</DialogTitle>
                <DialogDescription>Create a new API key for programmatic access.</DialogDescription>
              </DialogHeader>
              <form onSubmit={handleGenerate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="label">Key Label</Label>
                  <Input id="label" placeholder="e.g. Production" value={label} onChange={(e) => setLabel(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="expires">Expires (optional)</Label>
                  <Input id="expires" type="date" />
                </div>
                <DialogFooter showCloseButton>
                  <Button type="submit">Generate</Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      />

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard title="Total Keys" value={String(summary.total)} icon={Key} />
        <StatCard title="Active Keys" value={String(summary.active)} icon={Key} />
        <StatCard title="Revoked Keys" value={String(summary.revoked)} icon={Key} />
      </div>

      {keys.length === 0 ? (
        <EmptyState
          icon={Key}
          title="No API keys yet"
          description="Generate your first API key to start making requests."
          action={
            <Dialog open={genDialogOpen} onOpenChange={setGenDialogOpen}>
              <DialogTrigger render={<Button size="sm"><Plus className="size-4" /> Generate Key</Button>} />
            </Dialog>
          }
        />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">All API Keys</CardTitle>
            <CardAction>
              <div className="relative max-w-xs">
                <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search keys..."
                  className="h-9 pl-9"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            </CardAction>
          </CardHeader>
          <CardContent>
            <DataTable columns={columns} data={filtered} loading={loading} />
          </CardContent>
        </Card>
      )}

      <Dialog open={revealDialogOpen} onOpenChange={setRevealDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>API Key Generated</DialogTitle>
            <DialogDescription>Copy this key now. You won&apos;t be able to see it again.</DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 rounded-lg bg-muted p-4">
            <code className="flex-1 break-all font-mono text-sm">{newKeyValue}</code>
            <CopyButton value={newKeyValue} />
          </div>
          <p className="text-xs text-destructive">
            Make sure to copy your API key now. You won&apos;t be able to see it again!
          </p>
          <DialogFooter showCloseButton>
            <Button onClick={() => setRevealDialogOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!confirmRevoke} onOpenChange={(o) => !o && setConfirmRevoke(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Revoke API Key</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke <span className="font-medium text-foreground">{confirmRevoke?.label}</span>?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter showCloseButton>
            <Button variant="outline" onClick={() => setConfirmRevoke(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleRevoke}>Revoke</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
