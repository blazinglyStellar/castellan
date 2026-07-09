"use client";

import { useState, useMemo } from "react";
import Image from "next/image";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, Clock, DollarSign, Activity } from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useAccount } from "@/lib/auth/account-context";
import { getEarnings, getUsage, ApiError } from "@/lib/api/client";
import { formatAmount } from "@/lib/format";
import type { UsageEvent } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { DateRangePicker } from "@/components/usage/date-range-picker";
import { EarningsChart } from "@/components/analytics/earnings-chart";
import { EarningsBreakdown } from "@/components/analytics/earnings-breakdown";
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart";
import { UsageCostDonut } from "@/components/analytics/usage-cost-donut";
import { ErrorRateChart } from "@/components/analytics/error-rate-chart";
import { LatencyChart } from "@/components/analytics/latency-chart";
import { StatusDistribution } from "@/components/analytics/status-distribution";

function getDefaultRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 30);
  return {
    startDate: start.toISOString().slice(0, 10),
    endDate: end.toISOString().slice(0, 10),
  };
}

const LOGO = (
  <Image
    src="/stellar-xlm-logo.svg"
    alt="XLM"
    width={14}
    height={12}
    className="inline-block align-middle"
  />
);

export default function AnalyticsPage() {
  const { user, isLoading: isAccountLoading } = useAccount();
  const defaults = getDefaultRange();
  const [startDate, setStartDate] = useState(defaults.startDate);
  const [endDate, setEndDate] = useState(defaults.endDate);
  const [role, setRole] = useState<"provider" | "consumer" | null>(null);

  const resolvedRole = role ?? user?.role ?? "consumer";

  const earningsQuery = useQuery({
    queryKey: ["earnings", startDate, endDate],
    queryFn: () =>
      getEarnings({
        start_date: startDate ? `${startDate}T00:00:00Z` : undefined,
        end_date: endDate ? `${endDate}T23:59:59Z` : undefined,
      }),
    enabled: resolvedRole === "provider",
  });

  const usageQuery = useQuery({
    queryKey: ["usage", "analytics", startDate, endDate],
    queryFn: () =>
      getUsage({
        role: resolvedRole,
        start_date: startDate ? `${startDate}T00:00:00Z` : undefined,
        end_date: endDate ? `${endDate}T23:59:59Z` : undefined,
        limit: 1000,
      }),
    enabled: resolvedRole === "consumer",
  });

  const breakdownData = useMemo(() => {
    const data = earningsQuery.data?.by_provider;
    if (!data || data.length === 0) return [];
    return data.map((d) => ({ name: d.name, total: d.total }));
  }, [earningsQuery.data]);

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isLoading =
    (resolvedRole === "provider" && earningsQuery.isLoading) ||
    (resolvedRole === "consumer" && usageQuery.isLoading);

  const isError =
    (resolvedRole === "provider" && earningsQuery.isError) ||
    (resolvedRole === "consumer" && usageQuery.isError);

  const error =
    earningsQuery.error || usageQuery.error;

  const refetch =
    resolvedRole === "provider"
      ? () => earningsQuery.refetch()
      : () => usageQuery.refetch();

  const isProviderForbidden =
    resolvedRole === "provider" &&
    earningsQuery.isError &&
    earningsQuery.error instanceof ApiError &&
    earningsQuery.error.status === 403;

  const hasProviderData =
    resolvedRole === "provider" &&
    earningsQuery.data;

  const hasConsumerData =
    resolvedRole === "consumer" &&
    usageQuery.data &&
    usageQuery.data.data.length > 0;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError && !isProviderForbidden) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load analytics"}
        onRetry={refetch}
      />
    );
  }

  if (!hasProviderData && !hasConsumerData && !isProviderForbidden) {
    return (
      <div className="space-y-6">
        <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={resolvedRole} onRoleToggle={setRole} />
        <EmptyState
          title="No analytics data yet"
          description={
            resolvedRole === "provider"
              ? "Earnings and usage data will appear once API calls are processed."
              : "Usage data will appear once you start making API calls."
          }
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={resolvedRole} onRoleToggle={setRole} />

      {resolvedRole === "provider" && (
        <div className="relative">
          {isProviderForbidden && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-sm">
              <Card className="w-80 shadow-lg">
                <CardContent className="flex flex-col items-center gap-4 pt-6 text-center">
                  <p className="text-sm text-muted-foreground">
                    Provider analytics are only available for provider accounts.
                  </p>
                  <Button asChild>
                    <a href="/settings">Set up provider account</a>
                  </Button>
                </CardContent>
              </Card>
            </div>
          )}
          <div className={isProviderForbidden ? "pointer-events-none select-none space-y-6" : "space-y-6"}>
            <div className="grid gap-6 sm:grid-cols-2">
              <Card>
                <CardHeader className="flex flex-row items-center gap-2">
                  <TrendingUp className="h-4 w-4 text-muted-foreground" />
                  <CardTitle className="text-sm font-medium">Total Earnings</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-3xl font-bold tracking-tight">
                    {formatAmount(earningsQuery.data?.total_earnings ?? "0")}{" "}
                    <span className="text-sm font-normal text-muted-foreground">XLM</span>
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
                    {formatAmount(earningsQuery.data?.unsettled_earnings ?? "0")}{" "}
                    <span className="text-sm font-normal text-muted-foreground">XLM</span>
                  </p>
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Earnings Over Time</CardTitle>
                <CardDescription>Daily earnings for the selected period</CardDescription>
              </CardHeader>
              <CardContent>
                {earningsQuery.data ? (
                  <EarningsChart data={earningsQuery.data?.sparkline ?? []} />
                ) : (
                  <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
                    No earnings data yet.
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Earnings Breakdown</CardTitle>
                <CardDescription>Revenue by provider</CardDescription>
              </CardHeader>
              <CardContent className="p-0">
                {earningsQuery.data ? (
                  <EarningsBreakdown data={breakdownData} />
                ) : (
                  <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
                    No earnings data yet.
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {resolvedRole === "consumer" && usageQuery.data && (
        <>
          <SummaryCards events={usageQuery.data.data} />

          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">Usage Over Time</CardTitle>
              <CardDescription>Daily API call costs for the selected period</CardDescription>
            </CardHeader>
            <CardContent>
              <UsageVolumeChart events={usageQuery.data.data} />
            </CardContent>
          </Card>

          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Error Rate Over Time</CardTitle>
                <CardDescription>Daily non-2xx rate by endpoint</CardDescription>
              </CardHeader>
              <CardContent>
                <ErrorRateChart events={usageQuery.data.data} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Latency Over Time</CardTitle>
                <CardDescription>Daily average response time by endpoint</CardDescription>
              </CardHeader>
              <CardContent>
                <LatencyChart events={usageQuery.data.data} />
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Status Code Distribution</CardTitle>
                <CardDescription>Response status breakdown by endpoint</CardDescription>
              </CardHeader>
              <CardContent className="p-0">
                <StatusDistribution events={usageQuery.data.data} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Cost Breakdown</CardTitle>
                <CardDescription>Spending by endpoint</CardDescription>
              </CardHeader>
              <CardContent>
                <UsageCostDonut events={usageQuery.data.data} />
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}

function Header({
  role,
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  roleToggle,
  onRoleToggle,
}: {
  role: "provider" | "consumer";
  startDate: string;
  endDate: string;
  onStartDateChange: (d: string) => void;
  onEndDateChange: (d: string) => void;
  roleToggle: "provider" | "consumer";
  onRoleToggle: (r: "provider" | "consumer") => void;
}) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Analytics</h1>
        <p className="text-sm text-muted-foreground">
          {role === "provider"
            ? "Revenue and earnings insights."
            : "Usage and spending insights."}
        </p>
      </div>
      <div className="flex items-end gap-4">
        <div className="flex gap-0">
          <Button
            variant={roleToggle === "consumer" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleToggle("consumer")}
            className="rounded-r-none"
          >
            Consumer
          </Button>
          <Button
            variant={roleToggle === "provider" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleToggle("provider")}
            className="rounded-l-none"
          >
            Provider
          </Button>
        </div>
        <DateRangePicker
          startDate={startDate}
          endDate={endDate}
          onStartDateChange={onStartDateChange}
          onEndDateChange={onEndDateChange}
        />
      </div>
    </div>
  );
}

function SummaryCards({ events }: { events: UsageEvent[] }) {
  const totalCost = events.reduce((s, e) => s + parseFloat(e.request_cost), 0);
  const totalCalls = events.length;
  const successCalls = events.filter(
    (e) => e.status_code != null && e.status_code < 400,
  ).length;
  const successRate = totalCalls > 0 ? (successCalls / totalCalls) * 100 : 0;
  const latencyValues = events
    .filter((e) => e.latency_ms != null)
    .map((e) => e.latency_ms!);
  const avgLatency =
    latencyValues.length > 0
      ? latencyValues.reduce((s, v) => s + v, 0) / latencyValues.length
      : 0;
  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <DollarSign className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Total Spend</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {totalCost.toFixed(4)} {LOGO}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <TrendingUp className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Total Calls</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">{totalCalls.toLocaleString()}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Activity className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {successRate.toFixed(1)}%
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {successCalls} / {totalCalls} calls
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {latencyValues.length > 0 ? `${Math.round(avgLatency)}ms` : "\u2014"}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            avg response time
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <Skeleton className="h-7 w-24" />
          <Skeleton className="mt-1 h-4 w-64" />
        </div>
        <Skeleton className="h-8 w-72" />
      </div>
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
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
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-28" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-24" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-24" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-20" />
          </CardContent>
        </Card>
      </div>
      <Skeleton className="h-64 w-full rounded-lg" />
      <div className="grid gap-6 md:grid-cols-2">
        <Skeleton className="h-64 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        <Skeleton className="h-48 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    </div>
  );
}




