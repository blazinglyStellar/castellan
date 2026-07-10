"use client";

import { useState, useMemo } from "react";

import { useAccount } from "@/lib/auth/account-context";
import { FilterBar } from "@/components/usage/filter-bar";
import { UsageGroupedTable } from "@/components/usage/usage-grouped-table";
import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getUsage } from "@/lib/api/client";
import type { UsageEvent } from "@/lib/api/types";

export default function UsagePage() {
  const { isLoading: isAccountLoading } = useAccount();

  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [statusCode, setStatusCode] = useState(" ");
  const [selectedProvider, setSelectedProvider] = useState(" ");
  const [selectedEndpoint, setSelectedEndpoint] = useState(" ");

  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<UsageEvent>({
      queryKey: [
        "usage",
        "consumer",
        startDate ?? "",
        endDate ?? "",
        statusCode ?? "",
        selectedEndpoint ?? "",
        selectedProvider ?? "",
      ],
      fetchFn: (cp) => {
        const sc = statusCode?.trim();
        const sd = startDate ? `${startDate}T00:00:00Z` : undefined;
        const ed = endDate ? `${endDate}T23:59:59Z` : undefined;
        return getUsage({
          ...cp,
          role: "consumer",
          start_date: sd,
          end_date: ed,
          status_code: sc && sc !== " " ? parseInt(sc, 10) : undefined,
          provider_id:
            selectedProvider && selectedProvider !== " "
              ? selectedProvider
              : undefined,
          endpoint_id:
            selectedEndpoint && selectedEndpoint !== " "
              ? selectedEndpoint
              : undefined,
        });
      },
      limit: 50,
    });

  const providers = useMemo(() => {
    const map = new Map<string, string>();
    for (const ev of items) {
      if (!map.has(ev.provider_id)) {
        map.set(ev.provider_id, ev.provider_name);
      }
    }
    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [items]);

  const endpoints = useMemo(() => {
    const map = new Map<string, string>();
    const filtered =
      selectedProvider !== " "
        ? items.filter((ev) => ev.provider_id === selectedProvider)
        : items;
    for (const ev of filtered) {
      const key = ev.endpoint_id;
      const label = `${ev.method} ${ev.route}`;
      if (!map.has(key)) {
        map.set(key, label);
      }
    }
    return Array.from(map.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  }, [items, selectedProvider]);

  const filteredItems = items;

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Usage</h1>
        <p className="text-sm text-muted-foreground">
          View and filter your API usage events, grouped by provider and
          endpoint.
        </p>
      </div>

      <FilterBar
        providers={providers}
        endpoints={endpoints}
        selectedProvider={selectedProvider}
        selectedEndpoint={selectedEndpoint}
        onProviderChange={(v) => {
          setSelectedProvider(v);
          setSelectedEndpoint(" ");
        }}
        onEndpointChange={setSelectedEndpoint}
        startDate={startDate}
        endDate={endDate}
        onStartDateChange={setStartDate}
        onEndDateChange={setEndDate}
        statusCode={statusCode}
        onStatusCodeChange={setStatusCode}
      />

      <UsageGroupedTable
        items={filteredItems}
        isLoading={isLoading}
        isLoadingMore={isLoadingMore}
        hasMore={hasMore}
        loadMore={loadMore}
        refresh={refresh}
        error={error}
      />
    </div>
  );
}
