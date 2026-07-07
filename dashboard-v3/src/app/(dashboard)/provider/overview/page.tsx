"use client";

import { useQuery } from "@tanstack/react-query";
import {
  TrendingUp,
  Clock,
  Inbox,
  RefreshCw,
  Activity,
} from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getEarnings, getUsage } from "@/lib/api/client";
import type { UsageEvent } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

export default function ProviderOverviewPage() {
  const { isLoading: isAccountLoading } = useAccount();

  const {
    data: earnings,
    isLoading: isEarningsLoading,
    isError: isEarningsError,
    error: earningsError,
    refetch: refetchEarnings,
  } = useQuery({
    queryKey: ["earnings"],
    queryFn: getEarnings,
  });

  const {
    data: recentCalls,
    isLoading: isUsageLoading,
    isError: isUsageError,
    error: usageError,
    refetch: refetchUsage,
  } = useQuery({
    queryKey: ["usage", "recent"],
    queryFn: () => getUsage({ role: "provider", limit: 5 }),
  });

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isLoading = isEarningsLoading || isUsageLoading;
  const isError = isEarningsError || isUsageError;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    const errMsg =
      earningsError instanceof Error
        ? earningsError.message
        : usageError instanceof Error
          ? usageError.message
          : "Failed to load overview data";
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          refetchEarnings();
          refetchUsage();
        }}
      />
    );
  }

  const hasData =
    earnings &&
    earnings.total_earnings !== "0" &&
    recentCalls &&
    recentCalls.data.length > 0;
  const hasEarnings = earnings && earnings.total_earnings !== "0";

  if (!hasData && !hasEarnings) {
    return <EmptyState />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          Your earnings and API usage at a glance.
        </p>
      </div>

      {earnings && (
        <SummaryCards earnings={earnings} />
      )}

      {earnings && earnings.sparkline.length > 0 && (
        <SparklineSection data={earnings.sparkline} />
      )}

      {recentCalls && recentCalls.data.length > 0 ? (
        <RecentCallsTable calls={recentCalls.data} />
      ) : (
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            <CardTitle className="text-sm font-medium">Recent Calls</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
            <Inbox className="h-6 w-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">No recent calls</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

// ── Sub-components ──

function SummaryCards({
  earnings,
}: {
  earnings: { total_earnings: string; unsettled_earnings: string; currency?: string };
}) {
  const currency = "currency" in earnings ? (earnings as any).currency : "XLM";
  return (
    <div className="grid gap-6 sm:grid-cols-2">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <TrendingUp className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">
            Total Earnings
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(earnings.total_earnings)}{" "}
            <span className="text-sm font-normal text-muted-foreground">
              {currency}
            </span>
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Unsettled</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(earnings.unsettled_earnings)}{" "}
            <span className="text-sm font-normal text-muted-foreground">
              {currency}
            </span>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function SparklineSection({
  data,
}: {
  data: { date: string; amount: string }[];
}) {
  const amounts = data.map((d) => parseFloat(d.amount));
  const maxAmount = Math.max(...amounts, 1);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <TrendingUp className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Earnings (7 days)
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-end gap-1" style={{ height: 80 }}>
          {data.map((d) => {
            const pct = (parseFloat(d.amount) / maxAmount) * 100;
            return (
              <div
                key={d.date}
                className="flex flex-1 flex-col items-center gap-1"
              >
                <div
                  className="w-full rounded-sm bg-primary/60 transition-all hover:bg-primary/80"
                  style={{ height: `${Math.max(pct, 4)}%` }}
                />
                <span className="text-[10px] text-muted-foreground">
                  {formatDateLabel(d.date)}
                </span>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

function RecentCallsTable({ calls }: { calls: UsageEvent[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Activity className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">Recent Calls</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Method</TableHead>
              <TableHead>Route</TableHead>
              <TableHead>Cost</TableHead>
              <TableHead className="text-right">Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {calls.map((call) => (
              <TableRow key={call.id}>
                <TableCell>
                  <MethodBadge method={call.method} />
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {call.route}
                </TableCell>
                <TableCell>
                  {formatAmount(call.request_cost)}{" "}
                  <span className="text-xs text-muted-foreground">
                    {call.currency}
                  </span>
                </TableCell>
                <TableCell className="text-right text-xs text-muted-foreground">
                  {timeAgo(call.timestamp)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

// ── States ──

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-7 w-24" />
        <Skeleton className="mt-1 h-4 w-64" />
      </div>
      <div className="grid gap-6 sm:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-28" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-32" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-20" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-32" />
          </CardContent>
        </Card>
      </div>
      <Skeleton className="h-40 w-full rounded-lg" />
      <Skeleton className="h-48 w-full rounded-lg" />
    </div>
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
    <div className="flex flex-col items-center gap-4 py-20">
      <p className="text-sm text-muted-foreground">{message}</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        <RefreshCw className="mr-2 h-3 w-3" />
        Retry
      </Button>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center gap-4 py-20 text-center">
      <Inbox className="h-8 w-8 text-muted-foreground" />
      <div>
        <p className="text-sm font-medium text-foreground">
          No usage data yet
        </p>
        <p className="mt-1 text-sm text-muted-foreground">
          Get started by publishing an API.
        </p>
      </div>
    </div>
  );
}

// ── Helpers ──

function formatAmount(amount: string): string {
  const num = parseFloat(amount);
  if (isNaN(num)) return "0.0000";
  return num.toFixed(4);
}

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { weekday: "short" });
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

function MethodBadge({ method }: { method: string }) {
  const colorMap: Record<string, string> = {
    GET: "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950",
    POST: "text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-950",
    PUT: "text-orange-600 bg-orange-100 dark:text-orange-400 dark:bg-orange-950",
    PATCH:
      "text-purple-600 bg-purple-100 dark:text-purple-400 dark:bg-purple-950",
    DELETE: "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950",
  };

  return (
    <span
      className={cn(
        "inline-block rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase",
        colorMap[method.toUpperCase()] ||
          "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-800"
      )}
    >
      {method}
    </span>
  );
}
