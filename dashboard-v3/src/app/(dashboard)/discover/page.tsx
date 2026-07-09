"use client";

import { useState, useMemo, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Inbox, Users, Activity, Copy, Check } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getDiscoverProviders, getPublicProviderEndpoints } from "@/lib/api/client";
import type { Provider, Endpoint } from "@/lib/api/types";
import { StatusBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

function formatCompact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString();
}

export default function DiscoverPage() {
  const { isLoading: isAccountLoading } = useAccount();
  const [search, setSearch] = useState("");
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(
    null,
  );

  const {
    data: discoverRes,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["discover"],
    queryFn: getDiscoverProviders,
  });

  const providers = useMemo(
    () => discoverRes?.data ?? [],
    [discoverRes],
  );

  const filtered = useMemo(() => {
    if (!search) return providers;
    const q = search.toLowerCase();
    return providers.filter((p) => {
      if (
        p.name.toLowerCase().includes(q) ||
        p.base_url.toLowerCase().includes(q) ||
        (p.description || "").toLowerCase().includes(q)
      ) {
        return true;
      }
      return false;
    });
  }, [providers, search]);

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Failed to load providers"
        }
        onRetry={() => refetch()}
      />
    );
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
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
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
            {filtered.length}{" "}
            {filtered.length === 1 ? "provider" : "providers"} found
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
          title={
            search
              ? "No providers match your search"
              : "No providers found"
          }
          description={
            search
              ? "Try a different search term."
              : "No public APIs are available yet. Check back later."
          }
        />
      )}

      <ProviderSheet
        provider={selectedProvider}
        onClose={() => setSelectedProvider(null)}
      />
    </div>
  );
}

function ProviderCard({
  provider,
  onSelect,
}: {
  provider: Provider;
  onSelect: (p: Provider) => void;
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
              <Activity className="h-3 w-3" />
              {formatCompact(provider.total_calls)} calls
            </div>
          )}
          {provider.active_consumers !== undefined && (
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <Users className="h-3 w-3" />
              {provider.active_consumers} active{" "}
              {provider.active_consumers === 1 ? "consumer" : "consumers"}
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

function ProviderSheet({
  provider,
  onClose,
}: {
  provider: Provider | null;
  onClose: () => void;
}) {
  const open = provider !== null;

  const {
    data: endpoints,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["public-endpoints", provider?.id],
    queryFn: () => getPublicProviderEndpoints(provider!.id),
    enabled: open,
    staleTime: 60_000,
  });

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-full max-w-lg overflow-y-auto sm:max-w-lg">
        {provider && (
          <>
            <SheetHeader className="flex flex-row items-start justify-between gap-2 pr-8">
              <div className="space-y-1">
                <SheetTitle>{provider.name}</SheetTitle>
                <StatusBadge status={provider.status} />
              </div>
            </SheetHeader>

            <div className="mt-4 space-y-4">
              {provider.description && (
                <p className="text-sm text-muted-foreground">
                  {provider.description}
                </p>
              )}

              <p className="truncate font-mono text-xs text-muted-foreground">
                {provider.base_url}
              </p>

              {(provider.total_calls !== undefined ||
                provider.active_consumers !== undefined) && (
                <div className="flex gap-4">
                  {provider.total_calls !== undefined && (
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Activity className="h-3 w-3" />
                      {formatCompact(provider.total_calls)} calls
                    </div>
                  )}
                  {provider.active_consumers !== undefined && (
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Users className="h-3 w-3" />
                      {provider.active_consumers} active{" "}
                      {provider.active_consumers === 1
                        ? "consumer"
                        : "consumers"}
                    </div>
                  )}
                </div>
              )}

              <div>
                <h3 className="mb-2 text-sm font-medium">Endpoints</h3>
                {isLoading ? (
                  <div className="space-y-2">
                    {Array.from({ length: 3 }).map((_, i) => (
                      <Skeleton key={i} className="h-16 w-full rounded" />
                    ))}
                  </div>
                ) : isError ? (
                  <p className="text-center text-xs text-destructive">
                    Failed to load endpoints
                  </p>
                ) : endpoints && endpoints.length > 0 ? (
                  <div className="space-y-2">
                    {endpoints.map((ep) => (
                      <EndpointDetailRow key={ep.id} endpoint={ep} baseUrl={provider.base_url} />
                    ))}
                  </div>
                ) : (
                  <p className="text-center text-xs text-muted-foreground">
                    No public endpoints available
                  </p>
                )}
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

function EndpointDetailRow({
  endpoint,
  baseUrl,
}: {
  endpoint: Endpoint;
  baseUrl: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    const fullUrl = `${baseUrl.replace(/\/+$/, "")}/${endpoint.route.replace(/^\//, "")}`;
    navigator.clipboard.writeText(fullUrl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [baseUrl, endpoint.route]);

  return (
    <div className="rounded-md border bg-muted/30 p-3">
      <div className="flex items-center gap-2">
        <MethodBadge method={endpoint.method} />
        <code className="min-w-0 flex-1 truncate font-mono text-sm">
          {endpoint.route}
        </code>
        <span className="shrink-0 font-mono text-xs text-muted-foreground">
          {endpoint.price_amount}{" "}
          <span className="text-[10px]">{endpoint.currency}</span>
        </span>
        {endpoint.rate_limit && (
          <span className="hidden shrink-0 font-mono text-[11px] text-muted-foreground sm:inline">
            {endpoint.rate_limit}/min
          </span>
        )}
        <button
          onClick={handleCopy}
          className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          title={copied ? "Copied!" : "Copy URL"}
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-green-500" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </button>
      </div>
      {endpoint.description && (
        <p className="mt-1.5 text-xs text-muted-foreground">
          {endpoint.description}
        </p>
      )}
    </div>
  );
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
  );
}
