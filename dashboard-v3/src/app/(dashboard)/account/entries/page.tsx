"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { useAccount } from "@/lib/auth/account-context";
import { getAccountEntries } from "@/lib/api/client";
import { useOffsetPagination } from "@/lib/use-offset-pagination";
import type { EntryResponse } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
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

const ENTRY_TYPES = [
  { value: "deposit", label: "Deposit" },
  { value: "reservation", label: "Reservation" },
  { value: "deduction", label: "Deduction" },
  { value: "refund", label: "Refund" },
  { value: "settlement", label: "Settlement" },
] as const;

export default function AccountEntriesPage() {
  const { isLoading: isAccountLoading } = useAccount();

  const [typeFilter, setTypeFilter] = useState(" ");

  const resolvedType = typeFilter !== " " ? typeFilter : undefined;

  const {
    items,
    isLoading,
    total,
    page,
    totalPages,
    setPage,
    error,
    refresh,
  } = useOffsetPagination<EntryResponse>({
    queryKey: ["account-entries", typeFilter],
    fetchFn: (p) =>
      getAccountEntries({ ...p, type: resolvedType }).then((r) => ({
        data: r.entries,
        total: r.total,
      })),
    initialPageSize: 20,
  });

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
        <h1 className="text-2xl font-semibold tracking-tight">Ledger</h1>
        <p className="text-sm text-muted-foreground">
          View account ledger entries and balance history.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Type:</span>
        <Select value={typeFilter} onValueChange={setTypeFilter}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value=" ">All Types</SelectItem>
            {ENTRY_TYPES.map((t) => (
              <SelectItem key={t.value} value={t.value}>
                {t.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <EntriesTable
        items={items}
        isLoading={isLoading}
        error={error}
        onRetry={() => refresh()}
      />

      {!isLoading && items.length > 0 && (
        <PaginationBar
          page={page}
          totalPages={totalPages}
          total={total}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}

function EntriesTable({
  items,
  isLoading,
  error,
  onRetry,
}: {
  items: EntryResponse[];
  isLoading: boolean;
  error: Error | null;
  onRetry: () => void;
}) {
  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return <ErrorState message={error.message} onRetry={onRetry} />;
  }

  if (items.length === 0) {
    return (
      <EmptyState
        title="No entries yet"
        description="Ledger entries will appear here once account activity begins."
      />
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <CardTitle className="text-sm font-medium">Entries</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Type</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Balance After</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((entry) => (
              <TableRow key={entry.id} className="cursor-pointer">
                <TableCell>
                  <EntryTypeBadge type={entry.entry_type} />
                </TableCell>
                <TableCell className="whitespace-nowrap font-mono text-xs">
                  {formatAmount(entry.amount)}{" "}
                  <span className="text-muted-foreground">
                    {entry.currency}
                  </span>
                </TableCell>
                <TableCell className="whitespace-nowrap font-mono text-xs text-muted-foreground">
                  {formatAmount(entry.balance_after)}{" "}
                  <span className="text-muted-foreground">
                    {entry.currency}
                  </span>
                </TableCell>
                <TableCell className="max-w-[200px] truncate text-xs text-muted-foreground">
                  {entry.description || "\u2014"}
                </TableCell>
                <TableCell>
                  <StatusBadge status={entry.status} />
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {timeAgo(entry.created_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function PaginationBar({
  page,
  totalPages,
  total,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (p: number) => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <p className="text-sm text-muted-foreground">
        {total} total entries
      </p>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Prev
        </Button>
        <span className="text-sm text-muted-foreground">
          Page {page} of {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

function EntryTypeBadge({ type }: { type: string }) {
  const entry = ENTRY_TYPES.find((t) => t.value === type);
  const label = entry?.label ?? type;

  let color: string;
  switch (type) {
    case "deposit":
      color =
        "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950";
      break;
    case "deduction":
      color = "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950";
      break;
    case "reservation":
      color =
        "text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-950";
      break;
    case "refund":
      color =
        "text-purple-600 bg-purple-100 dark:text-purple-400 dark:bg-purple-950";
      break;
    case "settlement":
      color =
        "text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-950";
      break;
    default:
      color =
        "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950";
  }

  return (
    <span
      className={`inline-block rounded px-1.5 py-0.5 font-mono text-[11px] capitalize ${color}`}
    >
      {label}
    </span>
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
        <Skeleton className="h-4 w-20" />
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-t px-4 py-3"
            >
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-4 w-14 rounded" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function formatAmount(amount: string): string {
  const num = parseFloat(amount);
  if (isNaN(num)) return "0.0000";
  return num.toFixed(4);
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
