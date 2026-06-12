"use client"

import { useState } from "react"
import Link from "next/link"
import { Plus, AlertCircle, Cable } from "lucide-react"
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
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { MOCK_PROVIDERS } from "@/lib/mock-data"
import type { Provider } from "@/lib/mock-data"

export default function ProvidersPage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [name, setName] = useState("")
  const [baseUrl, setBaseUrl] = useState("")

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault()
    setAddOpen(false)
    setName("")
    setBaseUrl("")
  }

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load APIs</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="My APIs"
        actions={
          <Dialog open={addOpen} onOpenChange={setAddOpen}>
            <DialogTrigger asChild>
              <Button size="sm"><Plus className="h-4 w-4" />Add API</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Register New API</DialogTitle>
                <DialogDescription>Add a new API provider to start monetizing.</DialogDescription>
              </DialogHeader>
              <form onSubmit={handleAdd} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name">API Name</Label>
                  <Input id="name" placeholder="My API" value={name} onChange={(e) => setName(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="baseUrl">Base URL</Label>
                  <Input id="baseUrl" placeholder="https://api.example.com" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} required />
                </div>
                <DialogFooter>
                  <Button type="submit">Register</Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      />
      {isEmpty ? (
        <EmptyState
          title="You haven't registered any APIs yet."
          description="Add your first API to start tracking usage and earnings."
          icon={<Cable className="h-12 w-12" />}
          action={
            <Button onClick={() => setAddOpen(true)}><Plus className="h-4 w-4" />Add API</Button>
          }
        />
      ) : (
        <div className="rounded-lg border">
          <DataTable
            columns={[
              { key: "name", header: "Name", cell: (p: Provider) => (
                <Link href={`/dashboard/providers/${p.id}/endpoints`} className="font-medium hover:text-primary">
                  {p.name}
                </Link>
              )},
              { key: "baseUrl", header: "Base URL", cell: (p: Provider) => <span className="font-mono text-xs text-muted-foreground">{p.baseUrl}</span> },
              { key: "endpoints", header: "Endpoints", cell: (p: Provider) => p.endpoints.length },
              { key: "status", header: "Status", cell: (p: Provider) => <StatusBadge status={p.status} /> },
            ]}
            data={MOCK_PROVIDERS}
            loading={loading}
          />
        </div>
      )}
    </div>
  )
}
