"use client";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getDeposits } from "@/lib/api/client";
import type { Deposit } from "@/lib/api/types";
import { timeAgo, StatusBadge } from "@/lib/format";
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

export function DepositHistoryTable() {
  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<Deposit>({
      queryKey: ["deposits"],
      fetchFn: (cp) => getDeposits(cp),
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
        title="No deposits yet"
        description="Use the deposit instructions above to send funds to your account."
      />
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <CardTitle className="text-sm font-medium">Deposit History</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Amount</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>From</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Confirmed</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((deposit) => (
              <TableRow key={deposit.id}>
                <TableCell className="whitespace-nowrap font-mono text-xs">
                  {deposit.amount}{" "}
                  <span className="text-muted-foreground">
                    {deposit.currency}
                  </span>
                </TableCell>
                <TableCell>
                  <StatusBadge status={deposit.status} />
                </TableCell>
                <TableCell className="max-w-[160px] truncate font-mono text-xs text-muted-foreground">
                  {deposit.from_address}
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {timeAgo(deposit.created_at)}
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {deposit.confirmed_at ? timeAgo(deposit.confirmed_at) : "\u2014"}
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
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}




