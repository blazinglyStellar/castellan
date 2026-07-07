"use client";

import { Inbox, RefreshCw } from "lucide-react";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getDeposits } from "@/lib/api/client";
import type { Deposit } from "@/lib/api/types";
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
    return <EmptyState />;
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
                  <DepositStatusBadge status={deposit.status} />
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

function DepositStatusBadge({ status }: { status: string }) {
  let color: string;
  switch (status) {
    case "confirmed":
      color =
        "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950";
      break;
    case "pending":
      color =
        "text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-950";
      break;
    case "failed":
      color = "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950";
      break;
    default:
      color =
        "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950";
  }

  return (
    <span
      className={`inline-block rounded px-1.5 py-0.5 font-mono text-[11px] capitalize ${color}`}
    >
      {status}
    </span>
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

function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-12">
        <p className="text-sm text-muted-foreground">{message}</p>
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Retry
        </Button>
      </CardContent>
    </Card>
  );
}

function EmptyState() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
        <Inbox className="h-8 w-8 text-muted-foreground" />
        <div>
          <p className="text-sm font-medium text-foreground">
            No deposits yet
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Use the deposit instructions above to send funds to your account.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

function timeAgo(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}
