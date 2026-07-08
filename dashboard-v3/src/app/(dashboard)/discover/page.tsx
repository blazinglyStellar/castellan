"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Inbox } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getDiscoverProviders, getPublicProviderEndpoints } from "@/lib/api/client";
import type { Provider, Endpoint } from "@/lib/api/types";
import { StatusBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

export default function DiscoverPage() {
  const { isLoading: isAccountLoading } = useAccount();
  const [search, setSearch] = useState("");

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
        message={error instanceof Error ? error.message : "Failed to load providers"}
        onRetry={() => refetch()}
      />
    );
  }

  const providers = discoverRes?.data ?? [];
  const filtered = providers.filter(
    (p) =>
      !search ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.base_url.toLowerCase().includes(search.toLowerCase())
  );

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
          placeholder="Search providers by name or URL..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {filtered.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((provider) => (
            <ProviderCard key={provider.id} provider={provider} />
          ))}
        </div>
      ) : (
        <EmptyState
          icon={Inbox}
          title={search ? "No providers match your search" : "No providers found"}
          description={
            search
              ? "Try a different search term."
              : "No public providers are available yet."
          }
        />
      )}
    </div>
  );
}

function ProviderCard({ provider }: { provider: Provider }) {
  const [expanded, setExpanded] = useState(false);

  const { data: endpoints, isLoading, isError: isEndpointsError } = useQuery({
    queryKey: ["public-endpoints", provider.id],
    queryFn: () => getPublicProviderEndpoints(provider.id),
    enabled: expanded,
    staleTime: 60_000,
  });

  return (
    <Card className="flex flex-col">
      <CardHeader className="flex flex-row items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <CardTitle className="truncate text-base">{provider.name}</CardTitle>
          <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {provider.base_url}
          </p>
        </div>
        <StatusBadge status={provider.status} />
      </CardHeader>
      <CardContent className="flex-1">
        {!expanded ? (
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={() => setExpanded(true)}
          >
            View Endpoints
          </Button>
        ) : isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-8 w-full rounded" />
            ))}
          </div>
        ) : isEndpointsError ? (
          <p className="text-center text-xs text-destructive">
            Failed to load endpoints
          </p>
        ) : endpoints && endpoints.length > 0 ? (
          <div className="space-y-1.5">
            {endpoints.map((ep) => (
              <EndpointRow key={ep.id} endpoint={ep} />
            ))}
          </div>
        ) : (
          <p className="text-center text-xs text-muted-foreground">
            No public endpoints
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function EndpointRow({ endpoint }: { endpoint: Endpoint }) {
  return (
    <div className="flex items-center gap-2 rounded-md bg-muted/50 px-2.5 py-1.5">
      <MethodBadge method={endpoint.method} />
      <code className="min-w-0 flex-1 truncate font-mono text-xs">
        {endpoint.route}
      </code>
      <span className="shrink-0 font-mono text-xs text-muted-foreground">
        {endpoint.price_amount}{" "}
        <span className="text-[10px]">{endpoint.currency}</span>
      </span>
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
            <CardHeader className="flex flex-row items-start justify-between gap-2">
              <div className="flex-1 space-y-1">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-3 w-48" />
              </div>
              <Skeleton className="h-4 w-14 rounded" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-full rounded-md" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
