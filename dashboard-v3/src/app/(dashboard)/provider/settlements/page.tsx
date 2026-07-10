"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Landmark,
  Clock,
  CheckCircle2,
  Copy,
  BarChart3,
  ChevronDown,
  ChevronRight,
  ExternalLink,
} from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import {
  getDashboardMe,
  getEarnings,
  getSettlements,
  getSettlementSummary,
  getSettlementThreshold,
} from "@/lib/api/client";
import { useCursorPagination } from "@/lib/use-cursor-pagination";
import { STELLAR_EXPLORER_URL } from "@/lib/stellar";
import { DateRangePicker } from "@/components/usage/date-range-picker";
import type { SettlementBatch, SettlementEntry } from "@/lib/api/types";
import { formatAmount, formatShortDateTime, StatusBadge } from "@/lib/format";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import {
  Bar,
  BarChart,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { useToast } from "@/hooks/use-toast";

export default function ProviderSettlementsPage() {
  const { isLoading: isAccountLoading } = useAccount();
  const { toast } = useToast();

  const [statusFilter, setStatusFilter] = useState(" ");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [chartMonthStart, setChartMonthStart] = useState("");
  const [chartMonthEnd, setChartMonthEnd] = useState("");

  const resolvedStatus = statusFilter !== " " ? statusFilter : undefined;

  const meQuery = useQuery({
    queryKey: ["me"],
    queryFn: getDashboardMe,
  });

  const earningsQuery = useQuery({
    queryKey: ["earnings"],
    queryFn: () => getEarnings(),
  });

  const summaryQuery = useQuery({
    queryKey: ["settlement-summary"],
    queryFn: getSettlementSummary,
  });

  const thresholdQuery = useQuery({
    queryKey: ["settlement-threshold"],
    queryFn: getSettlementThreshold,
  });

  const {
    items: batches,
    isLoading: batchesLoading,
    isLoadingMore,
    hasMore,
    loadMore,
    refresh,
    error: batchesError,
  } = useCursorPagination<SettlementBatch>({
    queryKey: ["settlements", statusFilter ?? ""],
    fetchFn: (cp) => getSettlements({ ...cp, status: resolvedStatus }),
    limit: 20,
  });

  const earnings = earningsQuery.data;
  const summary = summaryQuery.data;
  const threshold = thresholdQuery.data;
  const payoutAddress = meQuery.data?.payout_stellar_address;

  const unsettled = earnings ? parseFloat(earnings.unsettled_earnings) : 0;
  const minThreshold = threshold ? parseFloat(threshold.min_threshold) : 0;
  const totalSettled = summary ? parseFloat(summary.total_settled) : 0;
  const thresholdProgress =
    minThreshold > 0 ? Math.min((unsettled / minThreshold) * 100, 100) : 0;

  const filteredBatches = useMemo(() => {
    if (!startDate && !endDate) return batches;
    const start = startDate ? new Date(startDate) : null;
    const end = endDate ? new Date(endDate + "T23:59:59Z") : null;
    return batches.filter((b) => {
      const d = new Date(b.created_at);
      if (start && d < start) return false;
      if (end && d > end) return false;
      return true;
    });
  }, [batches, startDate, endDate]);

  const chartData = useMemo(() => {
    if (!summary?.monthly_history) return [];
    const start = chartMonthStart ? new Date(chartMonthStart) : null;
    const end = chartMonthEnd ? new Date(chartMonthEnd + "T23:59:59Z") : null;
    return [...summary.monthly_history]
      .filter((m) => {
        const d = new Date(m.month);
        if (start && d < start) return false;
        if (end && d > end) return false;
        return true;
      })
      .reverse()
      .map((m) => ({
        month: new Date(m.month + "-02").toLocaleDateString("en-US", {
          month: "short",
          year: "2-digit",
        }),
        amount: parseFloat(m.amount),
      }));
  }, [summary, chartMonthStart, chartMonthEnd]);

  const handleCopyWallet = () => {
    if (payoutAddress) {
      navigator.clipboard.writeText(payoutAddress);
      toast({ title: "Copied", description: "Wallet address copied to clipboard." });
    }
  };

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isDataLoading =
    meQuery.isLoading ||
    earningsQuery.isLoading ||
    summaryQuery.isLoading ||
    thresholdQuery.isLoading ||
    batchesLoading;

  const isDataError =
    meQuery.isError ||
    earningsQuery.isError ||
    summaryQuery.isError ||
    thresholdQuery.isError ||
    batchesError;

  if (isDataError) {
    const errMsg =
      meQuery.error instanceof Error
        ? meQuery.error.message
        : earningsQuery.error instanceof Error
          ? earningsQuery.error.message
          : summaryQuery.error instanceof Error
            ? summaryQuery.error.message
            : thresholdQuery.error instanceof Error
              ? thresholdQuery.error.message
              : batchesError instanceof Error
                ? batchesError.message
                : "Failed to load settlement data";
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          meQuery.refetch();
          earningsQuery.refetch();
          summaryQuery.refetch();
          thresholdQuery.refetch();
          refresh();
        }}
      />
    );
  }

  if (isDataLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Settlements</h1>
          <p className="text-sm text-muted-foreground">
            Track payout batches and earnings.
          </p>
        </div>
        {payoutAddress && (
          <div className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
            <span className="text-muted-foreground">Payout Wallet:</span>
            <span className="font-mono text-xs">
              {payoutAddress.slice(0, 6)}...{payoutAddress.slice(-5)}
            </span>
            <button
              onClick={handleCopyWallet}
              className="text-muted-foreground hover:text-foreground transition-colors"
              title="Copy wallet address"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>

      <SummaryCards
        totalSettled={totalSettled}
        unsettled={unsettled}
        currency={earnings?.currency ?? "XLM"}
      />

      <ThresholdSection
        unsettled={unsettled}
        minThreshold={minThreshold}
        progress={thresholdProgress}
        currency={earnings?.currency ?? "XLM"}
      />

      {chartData.length > 0 && (
        <SettlementChart
          data={chartData}
          monthStart={chartMonthStart}
          monthEnd={chartMonthEnd}
          onMonthStartChange={setChartMonthStart}
          onMonthEndChange={setChartMonthEnd}
        />
      )}

      <div className="flex items-center justify-between gap-4">
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
        <DateRangePicker
          startDate={startDate}
          endDate={endDate}
          onStartDateChange={setStartDate}
          onEndDateChange={setEndDate}
        />
      </div>

      <SettlementTable
        batches={filteredBatches}
        isLoading={batchesLoading}
        isLoadingMore={isLoadingMore}
        hasMore={hasMore}
        loadMore={loadMore}
        error={batchesError}
        onRetry={() => refresh()}
      />
    </div>
  );
}

function SummaryCards({
  totalSettled,
  unsettled,
  currency,
}: {
  totalSettled: number;
  unsettled: number;
  currency: string;
}) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Landmark className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Total Settled</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(totalSettled.toFixed(7))}{" "}
            <span className="text-sm font-normal text-muted-foreground">
              {currency}
            </span>
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">
            Unsettled Earnings
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(unsettled.toFixed(7))}{" "}
            <span className="text-sm font-normal text-muted-foreground">
              {currency}
            </span>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function ThresholdSection({
  unsettled,
  minThreshold,
  progress,
  currency,
}: {
  unsettled: number;
  minThreshold: number;
  progress: number;
  currency: string;
}) {
  if (minThreshold <= 0) return null;

  const remaining = Math.max(minThreshold - unsettled, 0);
  const isReady = unsettled >= minThreshold;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <BarChart3 className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Progress Toward Next Payout
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="h-2.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className={`h-full rounded-full transition-all ${
              isReady
                ? "bg-emerald-500"
                : "bg-primary"
            }`}
            style={{ width: `${Math.max(progress, 2)}%` }}
          />
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="font-mono text-xs">
            {formatAmount(unsettled.toFixed(7))} /{" "}
            {formatAmount(minThreshold.toFixed(7))}{" "}
            <span className="text-muted-foreground">{currency}</span>
          </span>
          {isReady ? (
            <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Ready for next settlement cycle
            </span>
          ) : unsettled > 0 ? (
            <span className="text-muted-foreground">
              {formatAmount(remaining.toFixed(7))}{" "}
              {currency} more until next payout
            </span>
          ) : (
            <span className="text-muted-foreground">
              No unsettled earnings yet
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

const chartConfig = {
  amount: {
    label: "Settled",
    color: "hsl(var(--primary))",
  },
} satisfies Record<string, { label: string; color: string }>;

function SettlementChart({
  data,
  monthStart,
  monthEnd,
  onMonthStartChange,
  onMonthEndChange,
}: {
  data: { month: string; amount: number }[];
  monthStart: string;
  monthEnd: string;
  onMonthStartChange: (v: string) => void;
  onMonthEndChange: (v: string) => void;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <BarChart3 className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">
            Settlement History
          </CardTitle>
        </div>
        <DateRangePicker
          startDate={monthStart}
          endDate={monthEnd}
          onStartDateChange={onMonthStartChange}
          onEndDateChange={onMonthEndChange}
        />
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="h-64 w-full">
          <BarChart data={data} accessibilityLayer>
            <CartesianGrid strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="month"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              className="text-muted-foreground"
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              className="text-muted-foreground"
              tickFormatter={(v: number) => `${v}`}
            />
            <ChartTooltip
              cursor={{ fill: "hsl(var(--muted))", opacity: 0.3 }}
              content={
                <ChartTooltipContent
                  formatter={(value) => [
                    `${formatAmount(Number(value).toFixed(7))} XLM`,
                  ]}
                />
              }
            />
            <Bar
              dataKey="amount"
              fill="var(--color-amount)"
              radius={[4, 4, 0, 0]}
              maxBarSize={48}
            />
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

function SettlementTable({
  batches,
  isLoading,
  isLoadingMore,
  hasMore,
  loadMore,
  error,
  onRetry,
}: {
  batches: SettlementBatch[];
  isLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  loadMore: () => void;
  error: Error | null;
  onRetry: () => void;
}) {
  if (isLoading) {
    return <TableSkeleton />;
  }

  if (error) {
    return <ErrorState message={error.message} onRetry={onRetry} />;
  }

  if (batches.length === 0) {
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
        <CardTitle className="text-sm font-medium">
          Settlement Batches
        </CardTitle>
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
              <TableHead>TX Hash</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Completed</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {batches.map((batch) => (
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

  const stellarExplorerUrl = batch.tx_hash
    ? `${STELLAR_EXPLORER_URL}/${batch.tx_hash}`
    : null;

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
        <TableCell className="max-w-[120px]">
          {batch.tx_hash ? (
            <a
              href={stellarExplorerUrl!}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              {batch.tx_hash.slice(0, 8)}...
              <ExternalLink className="h-3 w-3" />
            </a>
          ) : (
            <span className="text-xs text-muted-foreground">&mdash;</span>
          )}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {formatShortDateTime(batch.created_at)}
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {batch.completed_at ? formatShortDateTime(batch.completed_at) : "\u2014"}
        </TableCell>
      </TableRow>
      {expanded && (
        <TableRow>
          <TableCell colSpan={8} className="bg-muted/30 p-0">
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
  const { toast } = useToast();

  const handleCopyWallet = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(entry.wallet_address);
    toast({ title: "Copied", description: "Wallet address copied to clipboard." });
  };

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
      <TableCell className="max-w-[160px]">
        <div className="flex items-center gap-1">
          <span className="truncate font-mono text-xs text-muted-foreground">
            {entry.wallet_address}
          </span>
          <button
            onClick={handleCopyWallet}
            className="shrink-0 text-muted-foreground hover:text-foreground transition-colors"
            title="Copy wallet address"
          >
            <Copy className="h-3 w-3" />
          </button>
        </div>
      </TableCell>
      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
        {formatShortDateTime(entry.created_at)}
      </TableCell>
    </TableRow>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <Skeleton className="h-7 w-28" />
          <Skeleton className="mt-1 h-4 w-52" />
        </div>
        <Skeleton className="h-9 w-44 rounded-lg" />
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center gap-2">
              <Skeleton className="h-4 w-4 rounded-full" />
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-32" />
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full" />
          <Skeleton className="h-4 w-40" />
        </CardHeader>
        <CardContent className="space-y-3">
          <Skeleton className="h-2.5 w-full rounded-full" />
          <Skeleton className="h-4 w-64" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full" />
          <Skeleton className="h-4 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full rounded-lg" />
        </CardContent>
      </Card>
      <TableSkeleton />
    </div>
  );
}

function TableSkeleton() {
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
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
