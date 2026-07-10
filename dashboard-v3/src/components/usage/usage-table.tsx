"use client";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getUsage } from "@/lib/api/client";
import type { UsageEvent } from "@/lib/api/types";
import { timeAgo, StatusCodeBadge } from "@/lib/format";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { MethodBadge } from "@/components/usage/method-badge";

interface UsageTableProps {
  role: "provider" | "consumer";
  startDate?: string;
  endDate?: string;
  statusCode?: string;
  endpointId?: string;
}

export function UsageTable({
  role,
  startDate,
  endDate,
  statusCode,
  endpointId,
}: UsageTableProps) {
  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<UsageEvent>({
      queryKey: [
        "usage",
        role,
        startDate ?? "",
        endDate ?? "",
        statusCode ?? "",
        endpointId ?? "",
      ],
      fetchFn: (cp) => {
        const sc = statusCode?.trim();
        return getUsage({
          ...cp,
          role,
          start_date: startDate || undefined,
          end_date: endDate || undefined,
          status_code: sc && sc !== " " ? parseInt(sc, 10) : undefined,
          endpoint_id: endpointId || undefined,
        });
      },
      limit: 50,
    });

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <ErrorState
        message={error.message}
        onRetry={() => refresh()}
      />
    );
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
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <CardTitle className="text-sm font-medium">Usage Events</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Method</TableHead>
              <TableHead>Route</TableHead>
              <TableHead>Cost</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Latency</TableHead>
              <TableHead className="text-right">Request ID</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((event) => (
              <TableRow key={event.id}>
                <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                  {timeAgo(event.timestamp)}
                </TableCell>
                <TableCell>
                  <MethodBadge method={event.method} />
                </TableCell>
                <TableCell className="max-w-[200px] truncate font-mono text-xs">
                  {event.route}
                </TableCell>
                <TableCell className="whitespace-nowrap font-mono text-xs">
                  {event.request_cost}{" "}
                  <span className="text-muted-foreground">
                    {event.currency}
                  </span>
                </TableCell>
                <TableCell>
                  <StatusCodeBadge code={event.status_code} />
  
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {event.latency_ms != null ? `${event.latency_ms}ms` : "\u2014"}
                </TableCell>
                <TableCell className="max-w-[100px] truncate text-right font-mono text-xs text-muted-foreground">
                  {event.request_id}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

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
  );
}



function LoadingSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Skeleton className="h-4 w-28" />
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-t px-4 py-3"
            >
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-4 w-12 rounded" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-8 rounded" />
              <Skeleton className="h-3 w-12" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}




