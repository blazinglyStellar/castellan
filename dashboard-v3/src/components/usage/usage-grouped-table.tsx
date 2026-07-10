"use client";

import { useState, useMemo, useEffect } from "react";
import Image from "next/image";
import {
  ChevronDown,
  ChevronRight,
  ChevronDownSquare,
} from "lucide-react";

import type { UsageEvent } from "@/lib/api/types";
import { formatBytes, StatusCodeBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { UsageDetailPanel } from "@/components/usage/usage-detail-panel";

const LOGO = (
  <Image
    src="/stellar-xlm-logo.svg"
    alt="XLM"
    width={14}
    height={12}
    className="inline-block align-middle"
  />
);

function shortDatetime(timestamp: string): string {
  const d = new Date(timestamp);
  if (isNaN(d.getTime())) return "\u2014";
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface GroupedTableProps {
  items: UsageEvent[];
  isLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  loadMore: () => void;
  refresh: () => void;
  error?: Error | null;
}

interface EndpointGroup {
  method: string;
  route: string;
  endpointId: string;
  events: UsageEvent[];
  totalCost: number;
}

interface ProviderGroup {
  providerName: string;
  providerId: string;
  endpoints: EndpointGroup[];
  totalCost: number;
  totalCalls: number;
}

function groupEvents(events: UsageEvent[]): ProviderGroup[] {
  const providerMap = new Map<string, ProviderGroup>();

  for (const ev of events) {
    if (!providerMap.has(ev.provider_id)) {
      providerMap.set(ev.provider_id, {
        providerName: ev.provider_name,
        providerId: ev.provider_id,
        endpoints: [],
        totalCost: 0,
        totalCalls: 0,
      });
    }
    const pg = providerMap.get(ev.provider_id)!;
    pg.totalCalls++;
    pg.totalCost += parseFloat(ev.request_cost);

    const endpointKey = `${ev.method} ${ev.route}`;
    let eg = pg.endpoints.find((e) => e.endpointId === ev.endpoint_id);
    if (!eg) {
      eg = {
        method: ev.method,
        route: ev.route,
        endpointId: ev.endpoint_id,
        events: [],
        totalCost: 0,
      };
      pg.endpoints.push(eg);
    }
    eg.events.push(ev);
    eg.totalCost += parseFloat(ev.request_cost);
  }

  return Array.from(providerMap.values());
}

export function UsageGroupedTable({
  items,
  isLoading,
  isLoadingMore,
  hasMore,
  loadMore,
  refresh,
  error,
}: GroupedTableProps) {
  const [selectedEvent, setSelectedEvent] = useState<UsageEvent | null>(null);
  const [expandedProviders, setExpandedProviders] = useState<
    Record<string, boolean>
  >({});
  const [expandedEndpoints, setExpandedEndpoints] = useState<
    Record<string, boolean>
  >({});

  const groups = useMemo(() => groupEvents(items), [items]);

  const toggleProvider = (id: string) =>
    setExpandedProviders((prev) => ({ ...prev, [id]: !prev[id] }));

  const toggleEndpoint = (key: string) =>
    setExpandedEndpoints((prev) => ({ ...prev, [key]: !prev[key] }));

  // Auto-expand first provider on first load — endpoints stay collapsed
  useEffect(() => {
    if (groups.length > 0 && Object.keys(expandedProviders).length === 0) {
      setExpandedProviders({ [groups[0].providerId]: true });
    }
  }, [groups, expandedProviders]);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <ErrorState message={error.message} onRetry={() => refresh()} />
    );
  }

  // DEBUG: inspect runtime shape — remove after fixing
  if (typeof window !== "undefined") {
    console.log("[debug] first event:", items[0]);
  }

  if (items.length === 0) {
    return (
      <EmptyState
        title="No usage events yet"
        description="Usage events will appear here once API calls are made."
      />
    );
  }

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <CardTitle className="text-sm font-medium">Usage Events</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {groups.map((pg) => {
            const provExpanded = expandedProviders[pg.providerId] ?? true;
            return (
              <div key={pg.providerId}>
                <button
                  type="button"
                  onClick={() => toggleProvider(pg.providerId)}
                  className="flex w-full items-center gap-2 border-b bg-muted/30 px-6 py-3 text-left text-sm font-medium hover:bg-muted/50 transition-colors"
                >
                  {provExpanded ? (
                    <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                  )}
                  <span title={pg.providerName || pg.providerId}>
                    {pg.providerName || pg.providerId}
                  </span>
                  <span className="ml-auto text-xs text-muted-foreground">
                    {pg.totalCalls} call{pg.totalCalls !== 1 ? "s" : ""},{" "}
                    {pg.totalCost.toFixed(4)} {LOGO}
                  </span>
                </button>

                {provExpanded &&
                  pg.endpoints.map((eg) => {
                    const endpointKey = `${pg.providerId}|${eg.endpointId}`;
                    const epExpanded =
                      expandedEndpoints[endpointKey] ?? false;
                    return (
                      <div key={endpointKey}>
                        <button
                          type="button"
                          onClick={() => toggleEndpoint(endpointKey)}
                          className="flex w-full items-center gap-2 border-b px-10 py-2 text-left text-xs text-muted-foreground hover:bg-muted/20 transition-colors"
                        >
                          {epExpanded ? (
                            <ChevronDownSquare className="h-3 w-3 shrink-0" />
                          ) : (
                            <ChevronRight className="h-3 w-3 shrink-0" />
                          )}
                          <MethodBadge method={eg.method} />
                          <span className="font-mono">{eg.route}</span>
                          <span className="ml-auto">
                            {eg.events.length} call
                            {eg.events.length !== 1 ? "s" : ""},{" "}
                            {eg.totalCost.toFixed(4)} {LOGO}
                          </span>
                        </button>

                        {epExpanded && (
                          <div>
                            <div className="grid w-full grid-cols-[1.5fr_2.5fr_0.75fr_1.5fr_1fr_1fr] gap-6 border-b px-14 py-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground/50">
                              <span>Time</span>
                              <span>Request ID</span>
                              <span className="text-center">Status</span>
                              <span className="text-right">Cost</span>
                              <span className="text-right">Latency</span>
                              <span className="text-right">Response</span>
                            </div>
                            {eg.events.map((ev) => (
                              <button
                                key={ev.id}
                                type="button"
                                onClick={() => setSelectedEvent(ev)}
                                className="grid w-full grid-cols-[1.5fr_2.5fr_0.75fr_1.5fr_1fr_1fr] gap-6 border-b px-14 py-2.5 text-left text-xs hover:bg-muted/10 transition-colors"
                              >
                                <span className="truncate text-muted-foreground">
                                  {shortDatetime(ev.timestamp)}
                                </span>
                                <span className="truncate font-mono text-muted-foreground">
                                  {ev.request_id}
                                </span>
                                <span className="flex justify-center">
                                  <StatusCodeBadge code={ev.status_code} />
                                </span>
                                <span className="truncate text-right font-mono tabular-nums">
                                  {ev.request_cost} {LOGO}
                                </span>
                                <span className="text-right font-mono tabular-nums text-muted-foreground">
                                  {ev.latency_ms != null
                                    ? `${ev.latency_ms}ms`
                                    : "\u2014"}
                                </span>
                                <span className="text-right font-mono tabular-nums text-muted-foreground">
                                  {formatBytes(ev.response_size)}
                                </span>
                              </button>
                            ))}
                          </div>
                        )}
                      </div>
                    );
                  })}
              </div>
            );
          })}

          {hasMore && (
            <div className="flex justify-center border-t py-4">
              <Button
                variant="outline"
                size="sm"
                onClick={loadMore}
                disabled={isLoadingMore}
              >
                {isLoadingMore ? "Loading..." : "Load More"}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <UsageDetailPanel
        event={selectedEvent}
        onClose={() => setSelectedEvent(null)}
      />
    </>
  );
}

function LoadingSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Skeleton className="h-4 w-28" />
      </CardHeader>
      <CardContent className="p-0">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i}>
            <div className="flex items-center gap-2 border-b bg-muted/30 px-6 py-3">
              <Skeleton className="h-4 w-4" />
              <Skeleton className="h-4 w-36" />
              <Skeleton className="ml-auto h-3 w-32" />
            </div>
            <div className="space-y-0">
              {Array.from({ length: 2 }).map((_, j) => (
                <div
                  key={j}
                  className="flex items-center gap-2 border-b px-10 py-2"
                >
                  <Skeleton className="h-3 w-3" />
                  <Skeleton className="h-4 w-10 rounded" />
                  <Skeleton className="h-3 w-40" />
                  <Skeleton className="ml-auto h-3 w-24" />
                </div>
              ))}
              {Array.from({ length: 3 }).map((_, j) => (
                <div
                  key={j}
                  className="flex items-center gap-4 border-b px-14 py-2.5"
                >
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-4 w-8 rounded" />
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-3 w-12" />
                  <Skeleton className="h-3 w-12" />
                  <Skeleton className="h-3 flex-1" />
                </div>
              ))}
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
