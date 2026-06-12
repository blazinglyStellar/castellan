"use client"

import { useState } from "react"
import { AlertCircle, Key, Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { MOCK_API_KEYS } from "@/lib/mock-data"
import type { ApiKey } from "@/lib/mock-data"

export default function ApiKeysPage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [keys, setKeys] = useState(MOCK_API_KEYS)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const [label, setLabel] = useState("")
  const [expires, setExpires] = useState("")
  const [showKeyModal, setShowKeyModal] = useState(false)
  const [newKey, setNewKey] = useState("")
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null)

  const handleGenerate = (e: React.FormEvent) => {
    e.preventDefault()
    const raw = `fg_${label.toLowerCase().replace(/\s+/g, "_")}_${Math.random().toString(36).slice(2, 14)}`
    setNewKey(raw)
    setKeyDialogOpen(false)
    setShowKeyModal(true)
    setLabel("")
    setExpires("")
    setKeys((prev) => [
      { id: `ak-${Date.now()}`, label, prefix: raw.slice(0, 12) + "••••", status: "active", createdAt: new Date().toISOString().slice(0, 10) },
      ...prev,
    ])
  }

  const handleRevoke = (id: string) => {
    setKeys((prev) => prev.map((k) => (k.id === id ? { ...k, status: "revoked" as const } : k)))
    setConfirmRevoke(null)
  }

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
        actions={
          <Dialog open={keyDialogOpen} onOpenChange={setKeyDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm"><Plus className="h-4 w-4" />Generate New Key</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Generate API Key</DialogTitle>
                <DialogDescription>Create a new API key for programmatic access.</DialogDescription>
              </DialogHeader>
              <form onSubmit={handleGenerate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="gen-label">Label</Label>
                  <Input id="gen-label" placeholder="e.g. Production" value={label} onChange={(e) => setLabel(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="gen-expires">Expires at (optional)</Label>
                  <Input id="gen-expires" type="date" value={expires} onChange={(e) => setExpires(e.target.value)} />
                </div>
                <DialogFooter>
                  <Button type="submit">Generate</Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      />
      {isEmpty ? (
        <EmptyState
          title="No API keys yet"
          description="Generate your first API key to start making requests."
          icon={<Key className="h-12 w-12" />}
          action={<Button onClick={() => setKeyDialogOpen(true)}><Plus className="h-4 w-4" />Generate Key</Button>}
        />
      ) : (
        <div className="rounded-lg border">
          <DataTable
            columns={[
              { key: "label", header: "Label", cell: (k: ApiKey) => k.label },
              { key: "key", header: "Key", cell: (k: ApiKey) => (
                <div className="flex items-center gap-1">
                  <span className="font-mono text-xs text-muted-foreground">{k.prefix}</span>
                  <CopyButton text={k.prefix + k.id} />
                </div>
              )},
              { key: "status", header: "Status", cell: (k: ApiKey) => <StatusBadge status={k.status} /> },
              { key: "created", header: "Created", cell: (k: ApiKey) => k.createdAt },
              { key: "actions", header: "", cell: (k: ApiKey) => (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8"><Trash2 className="h-4 w-4" /></Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {k.status === "active" ? (
                      <DropdownMenuItem className="text-destructive" onClick={() => setConfirmRevoke(k.id)}>
                        Revoke
                      </DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem disabled>Already revoked</DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )},
            ]}
            data={keys}
            loading={loading}
          />
        </div>
      )}

      <Dialog open={showKeyModal} onOpenChange={setShowKeyModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key Generated</DialogTitle>
            <DialogDescription>Copy this key now. You won&apos;t be able to see it again.</DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 rounded-lg bg-muted p-4">
            <code className="flex-1 break-all text-sm font-mono">{newKey}</code>
            <CopyButton text={newKey} />
          </div>
          <p className="text-xs text-destructive">Make sure to copy your API key now. You won&apos;t be able to see it again!</p>
          <DialogFooter>
            <Button onClick={() => setShowKeyModal(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!confirmRevoke} onOpenChange={(o) => !o && setConfirmRevoke(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke API Key</DialogTitle>
            <DialogDescription>Are you sure you want to revoke this key? This action cannot be undone.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmRevoke(null)}>Cancel</Button>
            <Button variant="destructive" onClick={() => confirmRevoke && handleRevoke(confirmRevoke)}>Revoke</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
