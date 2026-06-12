"use client"

import { useState } from "react"
import { useParams } from "next/navigation"
import Link from "next/link"
import { ArrowLeft, Plus, MoreHorizontal, AlertCircle, Cable } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
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
import { MOCK_PROVIDERS } from "@/lib/mock-data"
import type { Endpoint } from "@/lib/mock-data"

export default function EndpointsPage() {
  const params = useParams()
  const provider = MOCK_PROVIDERS.find((p) => p.id === params.id)
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editingEndpoint, setEditingEndpoint] = useState<Endpoint | null>(null)

  const [method, setMethod] = useState("GET")
  const [route, setRoute] = useState("")
  const [price, setPrice] = useState("")
  const [rateLimit, setRateLimit] = useState("")
  const [status, setStatus] = useState("active")

  const resetForm = () => {
    setMethod("GET"); setRoute(""); setPrice(""); setRateLimit(""); setStatus("active")
  }

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault()
    setAddOpen(false)
    resetForm()
  }

  const handleEdit = (endpoint: Endpoint) => {
    setEditingEndpoint(endpoint)
    setMethod(endpoint.method)
    setRoute(endpoint.route)
    setPrice(endpoint.price)
    setRateLimit(String(endpoint.rateLimit))
    setStatus(endpoint.status)
    setEditOpen(true)
  }

  const handleEditSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setEditOpen(false)
    setEditingEndpoint(null)
    resetForm()
  }

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load endpoints</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  if (!provider) {
    return <EmptyState title="Provider not found" description="This API provider does not exist." />
  }

  const endpoints = provider.endpoints
  const endpointsData = endpoints ?? []
  const activeCount = endpointsData.filter((e) => e.status === "active").length
  const inactiveCount = endpointsData.filter((e) => e.status === "inactive").length

  return (
    <div className="space-y-6">
      <Link href="/dashboard/providers" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="h-4 w-4" /> Back to My APIs
      </Link>
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold">{provider.name}</h1>
        <StatusBadge status={provider.status} />
      </div>
      <p className="font-mono text-sm text-muted-foreground -mt-3">{provider.baseUrl}</p>
      <PageHeader
        title="Endpoints"
        actions={
          <Dialog open={addOpen} onOpenChange={setAddOpen}>
            <DialogTrigger asChild>
              <Button size="sm"><Plus className="h-4 w-4" />Add Endpoint</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add Endpoint</DialogTitle>
                <DialogDescription>Register a new endpoint for this API.</DialogDescription>
              </DialogHeader>
              <form onSubmit={handleAdd} className="space-y-4">
                <div className="space-y-2">
                  <Label>HTTP Method</Label>
                  <Select value={method} onValueChange={setMethod}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {["GET", "POST", "PUT", "DELETE", "ANY"].map((m) => (
                        <SelectItem key={m} value={m}>{m}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="route">Route</Label>
                  <Input id="route" placeholder="/my-route" value={route} onChange={(e) => setRoute(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="price">Price (XLM)</Label>
                  <Input id="price" type="number" step="0.01" placeholder="0.50" value={price} onChange={(e) => setPrice(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="rateLimit">Rate Limit (req/s)</Label>
                  <Input id="rateLimit" type="number" placeholder="1000" value={rateLimit} onChange={(e) => setRateLimit(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label>Status</Label>
                  <Select value={status} onValueChange={setStatus}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="active">Active</SelectItem>
                      <SelectItem value="inactive">Inactive</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <DialogFooter>
                  <Button type="submit">Add Endpoint</Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      />
      {isEmpty ? (
        <EmptyState
          title="No endpoints yet"
          description="Add your first endpoint to start monetizing."
          icon={<Cable className="h-12 w-12" />}
          action={<Button onClick={() => setAddOpen(true)}><Plus className="h-4 w-4" />Add Endpoint</Button>}
        />
      ) : (
        <div className="rounded-lg border">
          <DataTable
            columns={[
              { key: "method", header: "Method", cell: (e: Endpoint) => (
                <Badge variant={e.method === "GET" ? "default" : e.method === "POST" ? "success" : e.method === "DELETE" ? "destructive" : "secondary"}>
                  {e.method}
                </Badge>
              )},
              { key: "route", header: "Route", cell: (e: Endpoint) => <span className="font-mono text-xs">{e.route}</span> },
              { key: "price", header: "Price", cell: (e: Endpoint) => `${e.price} XLM` },
              { key: "rateLimit", header: "Rate Limit", cell: (e: Endpoint) => `${e.rateLimit}/s` },
              { key: "status", header: "Status", cell: (e: Endpoint) => <StatusBadge status={e.status} /> },
              { key: "actions", header: "", cell: (e: Endpoint) => (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8"><MoreHorizontal className="h-4 w-4" /></Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => handleEdit(e)}>Edit</DropdownMenuItem>
                    <DropdownMenuItem>Toggle Status</DropdownMenuItem>
                    <DropdownMenuItem className="text-destructive">Delete</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )},
            ]}
            data={endpointsData}
            loading={loading}
          />
        </div>
      )}
      <p className="text-sm text-muted-foreground">
        Total: {endpointsData.length} endpoints &middot; {activeCount} active &middot; {inactiveCount} inactive
      </p>

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Endpoint</DialogTitle>
            <DialogDescription>Update endpoint configuration.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleEditSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label>HTTP Method</Label>
              <Select value={method} onValueChange={setMethod}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {["GET", "POST", "PUT", "DELETE", "ANY"].map((m) => (
                    <SelectItem key={m} value={m}>{m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-route">Route</Label>
              <Input id="edit-route" value={route} onChange={(e) => setRoute(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-price">Price (XLM)</Label>
              <Input id="edit-price" type="number" step="0.01" value={price} onChange={(e) => setPrice(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-rate">Rate Limit (req/s)</Label>
              <Input id="edit-rate" type="number" value={rateLimit} onChange={(e) => setRateLimit(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label>Status</Label>
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="inactive">Inactive</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="submit">Save Changes</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
