"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getSettlements } from "@/lib/api/client";
import type { SettlementBatch, SettlementEntry } from "@/lib/api/types";
import { timeAgo, StatusBadge } from "@/lib/format";
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
  const [statusFilter, setStatusFilter] = useState(" ");

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
    return (
      <EmptyState
        title="No settlements yet"
        description="Settlement batches will appear here once payouts are processed."
      />
    );
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




