"use client";

import Image from "next/image";
import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import {
  ExternalLink,
  Search,
  Inbox,
  CheckCircle2,
  Clock,
  XCircle,
} from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { DataTable } from "@/components/ui/data-table";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { getDeposits } from "@/lib/api/client";
import type { Deposit } from "@/lib/api/types";
import { formatShortDateTime, formatAmount, StatusBadge } from "@/lib/format";
import { STELLAR_EXPLORER_URL } from "@/lib/stellar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const STATUS_OPTIONS = [
  { value: "all", label: "All Status" },
  { value: "pending", label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "failed", label: "Failed" },
] as const;

const STATUS_ICONS: Record<string, React.ReactNode> = {
  confirmed: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
  pending: <Clock className="h-3.5 w-3.5 text-yellow-500" />,
  failed: <XCircle className="h-3.5 w-3.5 text-red-500" />,
};

export function DepositHistoryTable() {
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");

  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<Deposit>({
      queryKey: ["deposits"],
      fetchFn: (cp) => getDeposits(cp),
      limit: 50,
    });

  const filtered = useMemo(() => {
    let result = items;
    if (statusFilter !== "all") {
      result = result.filter((d) => d.status === statusFilter);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      result = result.filter(
        (d) =>
          d.tx_hash.toLowerCase().includes(q) ||
          d.from_address.toLowerCase().includes(q) ||
          d.id.toLowerCase().includes(q),
      );
    }
    return result;
  }, [items, statusFilter, searchQuery]);

  const columns: ColumnDef<Deposit>[] = useMemo(
    () => [
      {
        id: "statusIcon",
        header: "",
        cell: ({ row }) => (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>{STATUS_ICONS[row.original.status] || null}</span>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs capitalize">
                {row.original.status}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ),
        enableSorting: false,
      },
      {
        accessorKey: "amount",
        header: "Amount",
        cell: ({ row }) => (
          <span className="whitespace-nowrap font-mono text-xs">
            {formatAmount(row.original.amount)}{" "}
            <span className="text-muted-foreground">
              {row.original.currency || (
                <Image
                  src="/stellar-xlm-logo.svg"
                  alt=""
                  width={12}
                  height={10}
                  className="inline-block align-middle"
                />
              )}
            </span>
          </span>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        id: "from_address",
        header: "From",
        cell: ({ row }) => (
          <span
            className="block max-w-[120px] truncate font-mono text-xs text-muted-foreground"
            title={row.original.from_address}
          >
            {row.original.from_address}
          </span>
        ),
        enableSorting: false,
      },
      {
        id: "explorer",
        header: "",
        cell: ({ row }) => (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            asChild
          >
            <a
              href={`${STELLAR_EXPLORER_URL}/${row.original.tx_hash}`}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
            >
              View
              <ExternalLink className="ml-1 h-3 w-3" />
            </a>
          </Button>
        ),
        enableSorting: false,
      },
      {
        accessorKey: "created_at",
        header: "Created",
        cell: ({ row }) => (
          <span className="whitespace-nowrap text-xs text-muted-foreground">
            {formatShortDateTime(row.original.created_at)}
          </span>
        ),
      },
    ],
    [],
  );

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <ErrorState message={error.message} onRetry={() => refresh()} />
    );
  }

  if (items.length === 0) {
    return (
      <EmptyState
        title="No deposits yet"
        description={
          <>
            Use the deposit instructions above to send{" "}
            <Image
              src="/stellar-xlm-logo.svg"
              alt=""
              width={12}
              height={10}
              className="inline-block align-middle"
            />{" "}
            to your account. Your deposit history will appear here once
            confirmed by the Stellar network.
          </>
        }
        action={
          <Button variant="outline" size="sm" onClick={() => refresh()}>
            Refresh
          </Button>
        }
      />
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <CardTitle className="text-sm font-medium">
          Deposit History
          {items.length > 0 && (
            <span className="ml-2 text-xs font-normal text-muted-foreground">
              ({filtered.length} of {items.length})
            </span>
          )}
        </CardTitle>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search tx hash..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-8 w-44 pl-8 text-xs"
            />
          </div>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="h-8 w-32 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((opt) => (
                <SelectItem
                  key={opt.value}
                  value={opt.value}
                  className="text-xs"
                >
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {filtered.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-10 text-center">
            <Inbox className="h-6 w-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              No deposits match your filter.
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setStatusFilter("all");
                setSearchQuery("");
              }}
            >
              Clear filters
            </Button>
          </div>
        ) : (
          <>
            <DataTable columns={columns} data={filtered} />
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
          </>
        )}
      </CardContent>
    </Card>
  );
}

function LoadingSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <Skeleton className="h-4 w-28" />
        <div className="flex gap-2">
          <Skeleton className="h-8 w-44 rounded-md" />
          <Skeleton className="h-8 w-32 rounded-md" />
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-0">
          <div className="flex items-center gap-4 border-b px-4 py-3">
            <Skeleton className="h-3 w-8" />
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-3 flex-1" />
            <Skeleton className="h-3 w-14" />
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-4 border-b px-4 py-3"
            >
              <Skeleton className="h-4 w-4 rounded-full" />
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-16 rounded" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-6 w-14 rounded-md" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
