"use client"

import { useState, useMemo, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import { Search, Inbox, Users, Activity, Copy, Check, Terminal } from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import { getDiscoverProviders, getPublicProviderEndpoints } from "@/lib/api/endpoints"
import type { Provider, Endpoint } from "@/lib/api/types"
import { StatusBadge } from "@/lib/format"
import { MethodBadge } from "@/components/usage/method-badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Sheet,
  SheetContent,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { EmptyState } from "@/components/shared/empty-state"
import { ErrorState } from "@/components/shared/error-state"

function formatCompact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

export default function DiscoverPage() {
  const { isLoading: isAccountLoading } = useAuth()
  const [search, setSearch] = useState("")
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null)

  const {
    data: discoverRes,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["discover"],
    queryFn: getDiscoverProviders,
  })

  const providers = useMemo(() => discoverRes?.data ?? [], [discoverRes])

  const filtered = useMemo(() => {
    if (!search) return providers
    const q = search.toLowerCase()
    return providers.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.base_url.toLowerCase().includes(q) ||
        (p.description || "").toLowerCase().includes(q),
    )
  }, [providers, search])

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  if (isLoading) {
    return <LoadingSkeleton />
  }

  if (isError) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load providers"}
        onRetry={() => refetch()}
      />
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Discover</h1>
        <p className="text-sm text-muted-foreground">
          Browse public API providers and their endpoints.
        </p>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-9"
          placeholder="Search providers, endpoints, or routes..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {filtered.length > 0 ? (
        <div className="space-y-2">
          <p className="text-sm text-muted-foreground">
            {filtered.length} {filtered.length === 1 ? "provider" : "providers"} found
          </p>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((provider) => (
              <ProviderCard
                key={provider.id}
                provider={provider}
                onSelect={setSelectedProvider}
              />
            ))}
          </div>
        </div>
      ) : (
        <EmptyState
          icon={Inbox}
          title={search ? "No providers match your search" : "No providers found"}
          description={search ? "Try a different search term." : "No public APIs are available yet. Check back later."}
        />
      )}

      <ProviderSheet
        provider={selectedProvider}
        onClose={() => setSelectedProvider(null)}
      />
    </div>
  )
}

function ProviderCard({
  provider,
  onSelect,
}: {
  provider: Provider
  onSelect: (p: Provider) => void
}) {
  return (
    <Card
      className="flex cursor-pointer flex-col transition-colors hover:border-primary/50"
      onClick={() => onSelect(provider)}
    >
      <CardHeader className="flex flex-row items-start justify-between gap-2 pb-2">
        <div className="min-w-0 flex-1 space-y-1">
          <CardTitle className="truncate text-base">{provider.name}</CardTitle>
          {provider.description && (
            <p className="line-clamp-2 text-xs text-muted-foreground">
              {provider.description}
            </p>
          )}
          <p className="truncate font-mono text-xs text-muted-foreground">
            {provider.base_url}
          </p>
        </div>
        <StatusBadge status={provider.status} />
      </CardHeader>
      {(provider.total_calls !== undefined ||
        provider.active_consumers !== undefined) && (
        <div className="flex gap-4 px-6 pb-4">
          {provider.total_calls !== undefined && (
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Activity className="size-3" />
              {formatCompact(provider.total_calls)} calls
            </div>
          )}
          {provider.active_consumers !== undefined && (
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Users className="size-3" />
              {provider.active_consumers} active{" "}
              {provider.active_consumers === 1 ? "consumer" : "consumers"}
            </div>
          )}
        </div>
      )}
    </Card>
  )
}

function ProviderSheet({
  provider,
  onClose,
}: {
  provider: Provider | null
  onClose: () => void
}) {
  const open = provider !== null
  const [copiedBaseUrl, setCopiedBaseUrl] = useState(false)

  const {
    data: endpoints,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["public-endpoints", provider?.id],
    queryFn: () => getPublicProviderEndpoints(provider!.id),
    enabled: open,
    staleTime: 60_000,
  })

  const handleCopyBaseUrl = useCallback(() => {
    if (!provider) return
    navigator.clipboard.writeText(provider.base_url).then(() => {
      setCopiedBaseUrl(true)
      setTimeout(() => setCopiedBaseUrl(false), 2000)
    })
  }, [provider])

  return (
    <Sheet open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <SheetContent className="w-full max-w-lg overflow-y-auto p-0 sm:max-w-lg">
        {provider && (
          <div className="flex h-full flex-col">
            <div className="border-b px-6 py-5">
              <div className="space-y-1.5">
                <SheetTitle className="text-lg">{provider.name}</SheetTitle>
                <StatusBadge status={provider.status} />
              </div>
              {provider.description && (
                <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                  {provider.description}
                </p>
              )}
            </div>

            <div className="border-b px-6 py-4">
              <div className="flex items-center gap-2 rounded-lg border bg-muted/20 px-3 py-2.5">
                <code className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
                  {provider.base_url}
                </code>
                <button
                  onClick={handleCopyBaseUrl}
                  className="shrink-0 rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                >
                  {copiedBaseUrl ? (
                    <Check className="size-3.5 text-green-500" />
                  ) : (
                    <Copy className="size-3.5" />
                  )}
                </button>
              </div>

              {(provider.total_calls !== undefined ||
                provider.active_consumers !== undefined) && (
                <div className="mt-4 flex gap-4">
                  {provider.total_calls !== undefined && (
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100 text-blue-600">
                        <Activity className="size-4" />
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">Total calls</p>
                        <p className="font-mono text-sm font-semibold tabular-nums text-foreground">
                          {formatCompact(provider.total_calls)}
                        </p>
                      </div>
                    </div>
                  )}
                  {provider.active_consumers !== undefined && (
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600">
                        <Users className="size-4" />
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">
                          Active {provider.active_consumers === 1 ? "consumer" : "consumers"}
                        </p>
                        <p className="font-mono text-sm font-semibold tabular-nums text-foreground">
                          {provider.active_consumers}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-4">
              <div className="mb-3 flex items-center gap-2">
                <h3 className="text-sm font-medium text-foreground">Endpoints</h3>
                {endpoints && endpoints.length > 0 && (
                  <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] tabular-nums text-muted-foreground">
                    {endpoints.length}
                  </span>
                )}
              </div>
              {isLoading ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-[76px] w-full rounded-lg" />
                  ))}
                </div>
              ) : isError ? (
                <div className="flex flex-col items-center gap-2 py-10 text-center">
                  <p className="text-xs text-destructive">Failed to load endpoints</p>
                </div>
              ) : endpoints && endpoints.length > 0 ? (
                <div className="space-y-2">
                  {endpoints.map((ep) => (
                    <EndpointDetailRow
                      key={ep.id}
                      endpoint={ep}
                      baseUrl={provider.base_url}
                    />
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center gap-2 py-10 text-center">
                  <p className="text-xs text-muted-foreground">
                    No public endpoints available
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}

function EndpointDetailRow({
  endpoint,
  baseUrl,
}: {
  endpoint: Endpoint
  baseUrl: string
}) {
  const [copiedAction, setCopiedAction] = useState<"url" | "curl" | null>(null)

  const apiBaseUrl =
    process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080"

  const handleCopyUrl = useCallback(() => {
    const fullUrl = `${baseUrl.replace(/\/+$/, "")}/${endpoint.route.replace(/^\//, "")}`
    navigator.clipboard.writeText(fullUrl).then(() => {
      setCopiedAction("url")
      setTimeout(() => setCopiedAction(null), 2000)
    })
  }, [baseUrl, endpoint.route])

  const handleCopyCurl = useCallback(() => {
    const gatewayUrl = `${apiBaseUrl.replace(/\/+$/, "")}/api/gateway/${endpoint.provider_id}/${endpoint.route.replace(/^\//, "")}`
    const method = endpoint.method.toUpperCase()
    const authHeader = `-H "Authorization: Bearer <API-KEY>"`
    const parts = [`curl -X ${method} "${gatewayUrl}"`, authHeader]
    if (method === "POST" || method === "PUT" || method === "PATCH") {
      parts.push(`-H "Content-Type: application/json"`, `-d '{}'`)
    }
    navigator.clipboard.writeText(parts.join(" \\\n  ")).then(() => {
      setCopiedAction("curl")
      setTimeout(() => setCopiedAction(null), 2000)
    })
  }, [apiBaseUrl, endpoint.provider_id, endpoint.route, endpoint.method])

  return (
    <div className="rounded-lg border bg-card p-4 transition-colors hover:bg-muted/20">
      <div className="flex items-center gap-3">
        <MethodBadge method={endpoint.method} />
        <code className="min-w-0 flex-1 truncate font-mono text-sm font-medium">
          {endpoint.route}
        </code>
        <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
          {endpoint.price_amount}{" "}
          <span className="text-[10px] uppercase tracking-wider">{endpoint.currency}</span>
        </span>
        {endpoint.rate_limit && (
          <span className="hidden shrink-0 rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] tabular-nums text-muted-foreground sm:inline-flex">
            {endpoint.rate_limit}/min
          </span>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger className="shrink-0 rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">
            {copiedAction ? (
              <Check className="size-3.5 text-green-500" />
            ) : (
              <Copy className="size-3.5" />
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleCopyUrl} className="cursor-pointer gap-2">
              <Copy className="size-3.5" />
              Copy URL
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleCopyCurl} className="cursor-pointer gap-2">
              <Terminal className="size-3.5" />
              Copy cURL
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      {endpoint.description && (
        <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
          {endpoint.description}
        </p>
      )}
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-7 w-24" />
        <Skeleton className="mt-1 h-4 w-56" />
      </div>
      <Skeleton className="h-9 w-full rounded-md" />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-start justify-between gap-2 pb-2">
              <div className="flex-1 space-y-1">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-3 w-48" />
              </div>
              <Skeleton className="h-4 w-14 rounded" />
            </CardHeader>
            <div className="flex gap-4 px-6 pb-4">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 w-24" />
            </div>
          </Card>
        ))}
      </div>
    </div>
  )
}
