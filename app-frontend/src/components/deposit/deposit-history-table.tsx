"use client";

import { useEffect, useState, useMemo } from "react";
import {
  ExternalLink,
  Search,
  Inbox,
  CheckCircle2,
  Clock,
  XCircle,
  ChevronUp,
  ChevronDown,
  Copy,
  Check,
} from "lucide-react";

import { ErrorState } from "@/components/shared/error-state";
import { EmptyState } from "@/components/shared/empty-state";

import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { copyToClipboard } from "@/lib/clipboard";
import { getDeposits } from "@/lib/api/endpoints";
import type { Deposit } from "@/lib/api/types";
import { formatShortDateTime, formatAmount, StatusBadge } from "@/lib/format";
import { getStellarExplorerUrl } from "@/lib/stellar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button, buttonVariants } from "@/components/ui/button";
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

const stellarExplorerUrl = getStellarExplorerUrl()

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

type SortKey = "status" | "created_at";

export function DepositHistoryTable() {
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [rawQuery, setRawQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [sortKey, setSortKey] = useState<SortKey | null>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const [copiedTxHash, setCopiedTxHash] = useState<string | null>(null);

  function handleCopyTxHash(e: React.MouseEvent, hash: string) {
    e.stopPropagation();
    copyToClipboard(hash);
    setCopiedTxHash(hash);
    setTimeout(() => setCopiedTxHash(null), 2000);
  }

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(rawQuery), 300);
    return () => clearTimeout(t);
  }, [rawQuery]);

  const { items, isLoading, isLoadingMore, hasMore, loadMore, refresh, error } =
    useCursorPagination<Deposit>({
      queryKey: ["deposits"],
      fetchFn: (cp) => getDeposits(cp),
      limit: 50,
      refetchInterval: 15_000,
    });

  const filtered = useMemo(() => {
    let result = items;
    if (statusFilter !== "all") {
      result = result.filter((d) => d.status === statusFilter);
    }
    if (debouncedQuery.trim()) {
      const q = debouncedQuery.trim().toLowerCase();
      result = result.filter(
        (d) =>
          d.tx_hash.toLowerCase().includes(q) ||
          d.from_address.toLowerCase().includes(q) ||
          d.id.toLowerCase().includes(q),
      );
    }
    return result;
  }, [items, statusFilter, debouncedQuery]);

  const sorted = useMemo(() => {
    if (!sortKey) return filtered;
    return [...filtered].sort((a, b) => {
      const valA = a[sortKey];
      const valB = b[sortKey];
      const cmp = String(valA ?? "").localeCompare(String(valB ?? ""));
      return sortDir === "asc" ? cmp : -cmp;
    });
  }, [filtered, sortKey, sortDir]);

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  }

  function SortHeader({
    sortable,
    children,
  }: {
    sortable?: SortKey;
    children: React.ReactNode;
  }) {
    if (!sortable) {
      return <>{children}</>;
    }
    const active = sortKey === sortable;
    return (
      <button
        className="flex items-center gap-1"
        onClick={() => toggleSort(sortable)}
      >
        {children}
        {active && sortDir === "asc" && (
          <ChevronUp className="h-3 w-3" />
        )}
        {active && sortDir === "desc" && (
          <ChevronDown className="h-3 w-3" />
        )}
      </button>
    );
  }

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
        description="Use the deposit instructions above to send XLM to your account. Your deposit history will appear here once confirmed by the Stellar network."
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
              value={rawQuery}
              onChange={(e) => setRawQuery(e.target.value)}
              className="h-8 w-44 pl-8 text-xs"
            />
          </div>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "")}>
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
        {sorted.length === 0 ? (
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
                setRawQuery("");
                setDebouncedQuery("");
              }}
            >
              Clear filters
            </Button>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full border-separate border-spacing-y-1 px-6 pb-2 text-left">
                <thead>
                  <tr className="text-[11px] font-bold uppercase tracking-widest text-muted-foreground">
                    <th className="px-4 py-2 w-8" />
                    <th className="px-4 py-2">Amount</th>
                    <th className="px-4 py-2">
                      <SortHeader sortable="status">Status</SortHeader>
                    </th>
                    <th className="px-4 py-2">From</th>
                    <th className="px-4 py-2">Tx Hash</th>
                    <th className="px-4 py-2" />
                    <th className="px-4 py-2">
                      <SortHeader sortable="created_at">Created</SortHeader>
                    </th>
                  </tr>
                </thead>
                <tbody className="text-sm">
                  {sorted.map((d) => (
                    <tr
                      key={d.id}
                      className="rounded-lg bg-muted/30 transition-colors hover:bg-muted/60"
                    >
                      <td className="px-4 py-3">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger>
                              <span className="flex items-center justify-center">
                                {STATUS_ICONS[d.status] || null}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs capitalize">
                              {d.status}
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap font-mono text-xs">
                        {formatAmount(d.amount)}{" "}
                        <span className="text-muted-foreground">
                          {d.currency || "XLM"}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={d.status} />
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className="block max-w-[120px] truncate font-mono text-xs text-muted-foreground"
                          title={d.from_address}
                        >
                          {d.from_address}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <a
                            href={`${stellarExplorerUrl}/${d.tx_hash}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-mono text-xs text-muted-foreground hover:text-primary hover:underline"
                            title={d.tx_hash}
                          >
                            {d.tx_hash.slice(0, 8)}...
                          </a>
                          <button
                            onClick={(e) => handleCopyTxHash(e, d.tx_hash)}
                            className="shrink-0 text-muted-foreground transition-colors hover:text-foreground"
                            title="Copy transaction hash"
                          >
                            {copiedTxHash === d.tx_hash ? (
                              <Check className="size-3 text-green-500" />
                            ) : (
                              <Copy className="size-3" />
                            )}
                          </button>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <a
                          href={`${stellarExplorerUrl}/${d.tx_hash}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          onClick={(e) => e.stopPropagation()}
                          className={buttonVariants({ variant: "ghost", size: "xs" })}
                        >
                          View
                          <ExternalLink className="ml-1 h-3 w-3" />
                        </a>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground">
                        {formatShortDateTime(d.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
        <div className="overflow-x-auto px-6 pb-2">
          <div className="space-y-1">
            <div className="flex items-center gap-4 px-4 py-2">
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-12" />
              <Skeleton className="h-3 flex-1" />
              <Skeleton className="h-3 w-14" />
              <Skeleton className="h-6 w-14" />
              <Skeleton className="h-3 w-20" />
            </div>
            {Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-4 rounded-lg bg-muted/30 px-4 py-3"
              >
                <Skeleton className="h-4 w-4 rounded-full" />
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-4 w-16 rounded" />
                <Skeleton className="h-3 flex-1" />
                <Skeleton className="h-3 w-14" />
                <Skeleton className="h-6 w-14 rounded-md" />
                <Skeleton className="h-3 w-20" />
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
