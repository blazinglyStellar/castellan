"use client"

import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Trash2, Power, PowerOff, List, Server, AlertTriangle } from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import {
  getProviders,
  createProvider,
  deleteProvider,
  updateProviderStatus,
} from "@/lib/api/endpoints"
import type { Provider, CreateProviderRequest } from "@/lib/api/types"
import { timeAgo, StatusBadge } from "@/lib/format"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/shared/empty-state"
import { ErrorState } from "@/components/shared/error-state"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export default function ProvidersPage() {
  const { user, isLoading: isAccountLoading } = useAuth()
  const [dialogOpen, setDialogOpen] = useState(false)

  const {
    data: providers,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["providers"],
    queryFn: getProviders,
  })

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  if (isLoading) {
    return <LoadingSkeleton />
  }

  if (isError) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Failed to load providers"
        }
        onRetry={() => refetch()}
      />
    )
  }

  const hasProviders = providers && providers.length > 0

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Providers</h1>
          <p className="text-sm text-muted-foreground">
            Manage your registered API providers.
          </p>
        </div>
        <CreateProviderDialog open={dialogOpen} onOpenChange={setDialogOpen} />
      </div>

      {!user?.payout_stellar_address && (
        <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
          <AlertTriangle className="h-4 w-4 flex-shrink-0" />
          <span>
            You haven&apos;t set a payout address yet.{" "}
            <a href="/settings" className="font-medium underline underline-offset-2 hover:text-amber-900 dark:hover:text-amber-100">
              Go to Settings
            </a>{" "}
            to add your Stellar wallet address so you can receive settlement payments.
          </span>
        </div>
      )}

      {hasProviders ? (
        <ProvidersTable providers={providers} />
      ) : (
        <EmptyState
          title="No providers yet"
          description="Add your first provider to start publishing APIs."
          action={
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              Create Your First Provider
            </Button>
          }
        />
      )}
    </div>
  )
}

function ProvidersTable({ providers }: { providers: Provider[] }) {
  const queryClient = useQueryClient()

  const deleteMutation = useMutation({
    mutationFn: deleteProvider,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["providers"] }),
  })

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      updateProviderStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["providers"] }),
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
          <Server className="h-4 w-4" />
        </div>
        <CardTitle className="text-sm font-medium">
          Registered Providers
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full border-separate border-spacing-y-1 px-6 pb-2 text-left">
            <thead>
              <tr className="text-[11px] font-bold uppercase tracking-widest text-muted-foreground">
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2">Base URL</th>
                <th className="px-4 py-2">Endpoints</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Created</th>
                <th className="px-4 py-2 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="text-sm">
              {providers.map((provider) => (
                <tr
                  key={provider.id}
                  className="rounded-lg bg-muted/30 transition-colors hover:bg-muted/60"
                >
                  <td className="px-4 py-3 font-medium">
                    {provider.name}
                  </td>
                  <td className="max-w-[240px] truncate px-4 py-3 font-mono text-xs text-muted-foreground">
                    {provider.base_url}
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">
                    {provider.endpoint_count ?? "\u2014"}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={provider.status} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-muted-foreground">
                    {timeAgo(provider.created_at)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <a href={`/providers/${provider.id}/endpoints`}>
                        <Button
                          variant="ghost"
                          size="icon"
                          title="View Endpoints"
                        >
                          <List className="h-4 w-4 text-muted-foreground" />
                        </Button>
                      </a>
                      <Button
                        variant="ghost"
                        size="icon"
                        title={
                          provider.status === "active"
                            ? "Deactivate"
                            : "Activate"
                        }
                        onClick={() =>
                          statusMutation.mutate({
                            id: provider.id,
                            status:
                              provider.status === "active"
                                ? "inactive"
                                : "active",
                          })
                        }
                        disabled={statusMutation.isPending}
                      >
                        {provider.status === "active" ? (
                          <PowerOff className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <Power className="h-4 w-4 text-muted-foreground" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Delete"
                        onClick={() => deleteMutation.mutate(provider.id)}
                        disabled={deleteMutation.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

function CreateProviderDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [name, setName] = useState("")
  const [baseUrl, setBaseUrl] = useState("")

  const mutation = useMutation({
    mutationFn: (data: CreateProviderRequest) => createProvider(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["providers"] })
      onOpenChange(false)
      setName("")
      setBaseUrl("")
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !baseUrl.trim()) return
    mutation.mutate({ name: name.trim(), base_url: baseUrl.trim() })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Add Provider
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Provider</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My API Service"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="baseUrl">Base URL</Label>
            <Input
              id="baseUrl"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="https://api.example.com"
              type="url"
              required
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={mutation.isPending}>
              {mutation.isPending ? "Adding..." : "Add Provider"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ── States ──

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-7 w-24" />
          <Skeleton className="mt-1 h-4 w-56" />
        </div>
        <Skeleton className="h-9 w-32 rounded-md" />
      </div>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Skeleton className="h-4 w-32" />
        </CardHeader>
        <CardContent className="p-0">
          <div className="space-y-0">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-4 border-t px-4 py-3"
              >
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-3 w-44" />
                <Skeleton className="h-4 w-16 rounded" />
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-8 w-16" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
