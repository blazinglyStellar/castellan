"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Inbox, RefreshCw } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getSettlements } from "@/lib/api/client";
import type { SettlementBatch, SettlementEntry } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function ProviderSettlementsPage() {
  const { isLoading: isAccountLoading } = useAccount();
  const [statusFilter, setStatusFilter] = useState(" ");

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const resolvedStatus =
    statusFilter !== " " ? statusFilter : undefined;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settlements</h1>
        <p className="text-sm text-muted-foreground">
          View settlement batch payouts and reconciliation.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Status:</span>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value=" ">All Statuses</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <SettlementTable statusFilter={resolvedStatus} />
    </div>
  );
}

function SettlementTable({ statusFilter }: { statusFilter?: string }) {
  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<SettlementBatch>({
      queryKey: ["settlements", statusFilter ?? ""],
      fetchFn: (cp) => getSettlements({ ...cp, status: statusFilter }),
      limit: 20,
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
        <CardTitle className="text-sm font-medium">Settlement Batches</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8" />
              <TableHead>Batch ID</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Entries</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Completed</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((batch) => (
              <SettlementBatchRow key={batch.id} batch={batch} />
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

function SettlementBatchRow({ batch }: { batch: SettlementBatch }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <TableRow
        className="cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <TableCell className="w-8">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </TableCell>
        <TableCell className="font-mono text-xs">{batch.id}</TableCell>
        <TableCell className="whitespace-nowrap font-mono text-xs">
          {batch.total_amount}{" "}
          <span className="text-muted-foreground">{batch.currency}</span>
        </TableCell>
        <TableCell>
          <StatusBadge status={batch.status} />
        </TableCell>
        <TableCell className="text-xs text-muted-foreground">
          {batch.entry_count}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {timeAgo(batch.created_at)}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {batch.completed_at ? timeAgo(batch.completed_at) : "\u2014"}
        </TableCell>
      </TableRow>
      {expanded && (
        <TableRow>
          <TableCell colSpan={7} className="bg-muted/30 p-0">
            <div className="px-6 py-4">
              {batch.entries.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No entries in this batch.
                </p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Entry ID</TableHead>
                      <TableHead>Provider</TableHead>
                      <TableHead>Amount</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Wallet</TableHead>
                      <TableHead>Created</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {batch.entries.map((entry) => (
                      <SettlementEntryRow key={entry.id} entry={entry} />
                    ))}
                  </TableBody>
                </Table>
              )}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function SettlementEntryRow({ entry }: { entry: SettlementEntry }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs">{entry.id}</TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">
        {entry.provider_id}
      </TableCell>
      <TableCell className="whitespace-nowrap font-mono text-xs">
        {entry.amount}{" "}
        <span className="text-muted-foreground">{entry.currency}</span>
      </TableCell>
      <TableCell>
        <StatusBadge status={entry.status} />
      </TableCell>
      <TableCell className="max-w-[160px] truncate font-mono text-xs text-muted-foreground">
        {entry.wallet_address}
      </TableCell>
      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
        {timeAgo(entry.created_at)}
      </TableCell>
    </TableRow>
  );
}

function StatusBadge({ status }: { status: string }) {
  let color: string;
  switch (status) {
    case "completed":
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
        <Skeleton className="h-4 w-32" />
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-t px-4 py-3"
            >
              <Skeleton className="h-3 w-4" />
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 w-8" />
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
            No settlements yet
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Settlement batches will appear here once payouts are processed.
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
