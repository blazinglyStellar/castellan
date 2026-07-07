"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp, Clock, Inbox, RefreshCw, DollarSign } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getEarnings, getUsage } from "@/lib/api/client";
import type { UsageEvent } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { DateRangePicker } from "@/components/usage/date-range-picker";
import { EarningsChart } from "@/components/analytics/earnings-chart";
import { EarningsBreakdown } from "@/components/analytics/earnings-breakdown";
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart";
import { UsageCostBreakdown } from "@/components/analytics/usage-cost-breakdown";

function getDefaultRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 30);
  return {
    startDate: start.toISOString().slice(0, 10),
    endDate: end.toISOString().slice(0, 10),
  };
}

export default function AnalyticsPage() {
  const { user, isLoading: isAccountLoading } = useAccount();
  const defaults = getDefaultRange();
  const [startDate, setStartDate] = useState(defaults.startDate);
  const [endDate, setEndDate] = useState(defaults.endDate);
  const [role, setRole] = useState<"provider" | "consumer">("provider");

  const resolvedRole = role ?? user?.role ?? "consumer";

  const earningsQuery = useQuery({
    queryKey: ["earnings"],
    queryFn: getEarnings,
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

  const filteredSparkline = useMemo(() => {
    const data = earningsQuery.data?.sparkline;
    if (!data || data.length === 0) return [];
    const start = startDate ? new Date(startDate) : new Date(0);
    const end = endDate ? new Date(endDate) : new Date(864e12);
    return data.filter((d) => {
      const dt = new Date(d.date);
      return dt >= start && dt <= end;
    });
  }, [earningsQuery.data, startDate, endDate]);

  const filteredBreakdown = useMemo(() => {
    const data = earningsQuery.data?.by_endpoint;
    if (!data || data.length === 0) return [];
    if (!startDate && !endDate) return data;
    return data;
  }, [earningsQuery.data, startDate, endDate]);

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

  const hasProviderData =
    resolvedRole === "provider" &&
    earningsQuery.data &&
    earningsQuery.data.total_earnings !== "0";

  const hasConsumerData =
    resolvedRole === "consumer" &&
    usageQuery.data &&
    usageQuery.data.data.length > 0;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load analytics"}
        onRetry={refetch}
      />
    );
  }

  if (!hasProviderData && !hasConsumerData) {
    return (
      <div className="space-y-6">
        <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={role} onRoleToggle={setRole} />
        <EmptyState role={resolvedRole} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Header role={resolvedRole} startDate={startDate} endDate={endDate} onStartDateChange={setStartDate} onEndDateChange={setEndDate} roleToggle={role} onRoleToggle={setRole} />

      {resolvedRole === "provider" && earningsQuery.data && (
        <>
          <div className="grid gap-6 sm:grid-cols-2">
            <Card>
              <CardHeader className="flex flex-row items-center gap-2">
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
                <CardTitle className="text-sm font-medium">Total Earnings</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-3xl font-bold tracking-tight">
                  {formatAmount(earningsQuery.data.total_earnings)}{" "}
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
                  {formatAmount(earningsQuery.data.unsettled_earnings)}{" "}
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
              <EarningsChart data={filteredSparkline} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">Earnings Breakdown</CardTitle>
              <CardDescription>Revenue by endpoint</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <EarningsBreakdown data={filteredBreakdown} />
            </CardContent>
          </Card>
        </>
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

          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">Cost Breakdown</CardTitle>
              <CardDescription>Spending by endpoint</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <UsageCostBreakdown events={usageQuery.data.data} />
            </CardContent>
          </Card>
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
  const currency = events[0]?.currency ?? "XLM";
  return (
    <div className="grid gap-6 sm:grid-cols-1 md:grid-cols-2">
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <DollarSign className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Total Spend</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">
            {totalCost.toFixed(4)}{" "}
            <span className="text-sm font-normal text-muted-foreground">{currency}</span>
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <TrendingUp className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">Total Calls</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold tracking-tight">{events.length.toLocaleString()}</p>
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
      <Skeleton className="h-64 w-full rounded-lg" />
      <Skeleton className="h-48 w-full rounded-lg" />
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
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

function EmptyState({ role }: { role: string }) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
        <Inbox className="h-8 w-8 text-muted-foreground" />
        <div>
          <p className="text-sm font-medium text-foreground">
            No analytics data yet
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {role === "provider"
              ? "Earnings and usage data will appear once API calls are processed."
              : "Usage data will appear once you start making API calls."}
          </p>
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
