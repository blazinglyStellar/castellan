"use client";

import Image from "next/image";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Wallet, DollarSign, Activity, Key, Inbox, Banknote, TrendingUp, Clock } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getBalance, getUsage, getDeposits, getApiKeys, getEarnings } from "@/lib/api/client";
import type { UsageEvent, Deposit, Earnings, DailyEarning } from "@/lib/api/types";
import { formatAmount, timeAgo, StatusBadge, StatusCodeBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
import { UsageVolumeChart } from "@/components/analytics/usage-volume-chart";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardFooter,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const LOGO = (
  <Image
    src="/stellar-xlm-logo.svg"
    alt="XLM"
    width={14}
    height={12}
    className="inline-block align-middle"
  />
);

const INTERVALS = ["7d", "30d", "90d"] as const;
type Interval = (typeof INTERVALS)[number];

function intervalParams(interval: Interval) {
  const d = new Date();
  const days = interval === "7d" ? 7 : interval === "30d" ? 30 : 90;
  d.setDate(d.getDate() - days);
  return {
    start_date: d.toISOString(),
    limit: interval === "7d" ? 100 : interval === "30d" ? 500 : 1000,
  };
}

export default function OverviewPage() {
  const { user, isLoading: isAccountLoading } = useAccount();
  const [usageInterval, setUsageInterval] = useState<Interval>("30d");
  const isProvider = user?.role === "provider";

  const balanceQuery = useQuery({
    queryKey: ["balance"],
    queryFn: getBalance,
  });

  const usageQuery = useQuery({
    queryKey: ["usage", "overview", usageInterval],
    queryFn: () =>
      getUsage({ role: "consumer", ...intervalParams(usageInterval) }),
  });

  const depositsQuery = useQuery({
    queryKey: ["deposits", "recent"],
    queryFn: () => getDeposits({ limit: 5 }),
  });

  const keysQuery = useQuery({
    queryKey: ["api-keys"],
    queryFn: getApiKeys,
  });

  const earningsQuery = useQuery({
    queryKey: ["earnings"],
    queryFn: () => getEarnings(),
    enabled: isProvider,
  });

  const providerUsageQuery = useQuery({
    queryKey: ["usage", "provider", "recent"],
    queryFn: () => getUsage({ role: "provider", limit: 5 }),
    enabled: isProvider,
  });

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isLoading =
    balanceQuery.isLoading ||
    usageQuery.isLoading ||
    depositsQuery.isLoading ||
    keysQuery.isLoading ||
    (isProvider && (earningsQuery.isLoading || providerUsageQuery.isLoading));

  if (isLoading) {
    return <LoadingSkeleton isProvider={isProvider} />;
  }

  const isError =
    balanceQuery.isError ||
    usageQuery.isError ||
    depositsQuery.isError ||
    keysQuery.isError;

  if (isError) {
    const errMsg =
      balanceQuery.error instanceof Error
        ? balanceQuery.error.message
        : usageQuery.error instanceof Error
          ? usageQuery.error.message
          : depositsQuery.error instanceof Error
            ? depositsQuery.error.message
            : keysQuery.error instanceof Error
              ? keysQuery.error.message
              : "Failed to load overview data";
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          balanceQuery.refetch();
          usageQuery.refetch();
          depositsQuery.refetch();
          keysQuery.refetch();
        }}
      />
    );
  }

  const balance = balanceQuery.data;
  const usageEvents = usageQuery.data?.data ?? [];
  const deposits = depositsQuery.data?.data ?? [];
  const apiKeys = keysQuery.data ?? [];
  const activeKeysCount = apiKeys.filter((k) => k.status === "active").length;
  const earnings = isProvider ? earningsQuery.data : null;
  const providerCalls = isProvider ? (providerUsageQuery.data?.data ?? []) : [];

  const hasBalance = balance && balance.balance !== "0";
  const hasUsage = usageEvents.length > 0;
  const hasDeposits = deposits.length > 0;
  const hasKeys = apiKeys.length > 0;
  const hasEarnings = earnings && earnings.total_earnings !== "0";

  if (!hasBalance && !hasUsage && !hasDeposits && !hasKeys && !hasEarnings) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          <p className="text-sm text-muted-foreground">
            Your account at a glance.
          </p>
        </div>
        <EmptyState
          title="Welcome to Castellan"
          description="Start using APIs to see your activity here."
          action={
            <Button variant="outline" size="sm" asChild>
              <a href="/deposit">Deposit funds</a>
            </Button>
          }
        />
      </div>
    );
  }

  const consumerCalls = usageEvents.slice(0, 5);
  const totalSpend = usageEvents.reduce(
    (s, e) => s + parseFloat(e.request_cost),
    0,
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          {isProvider
            ? "Your balance, usage, and earnings at a glance."
            : "Your account balance and usage at a glance."}
        </p>
      </div>

      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {balance && (
          <BalanceCard
            balance={balance.balance}
            available={balance.available_balance}
          />
        )}
        <StatCard
          icon={<DollarSign className="h-4 w-4 text-muted-foreground" />}
          title="Total Spend"
        >
          {hasUsage ? (
            <>
              {formatAmount(String(totalSpend))} {LOGO}
            </>
          ) : (
            "\u2014"
          )}
        </StatCard>
        <StatCard
          icon={<Activity className="h-4 w-4 text-muted-foreground" />}
          title="Total Calls"
        >
          {hasUsage ? usageEvents.length.toLocaleString() : "\u2014"}
        </StatCard>
        <StatCard
          icon={<Key className="h-4 w-4 text-muted-foreground" />}
          title="Active Keys"
        >
          {hasKeys ? activeKeysCount.toLocaleString() : "\u2014"}
        </StatCard>
      </div>

      {hasEarnings && earnings && (
        <EarningsCards earnings={earnings} />
      )}

      <UsageChartCard
        events={usageEvents}
        interval={usageInterval}
        onIntervalChange={setUsageInterval}
      />

      <RecentCallsCard calls={consumerCalls} />

      {hasDeposits && (
        <RecentDepositsCard deposits={deposits} />
      )}

      {isProvider && providerCalls.length > 0 && (
        <ProviderRecentCallsCard calls={providerCalls} />
      )}

      {hasEarnings && earnings && earnings.sparkline.length > 0 && (
        <SparklineSection data={earnings.sparkline} />
      )}

      <QuickActions isProvider={isProvider} />
    </div>
  );
}

// ── Sub-components ──

function BalanceCard({
  balance,
  available,
}: {
  balance: string;
  available: string;
}) {
  const b = parseFloat(balance);
  const a = parseFloat(available);
  const reserved = b - a;
  const pct = b > 0 ? Math.min((a / b) * 100, 100) : 0;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Wallet className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">Balance</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-3xl font-bold tracking-tight">
          {formatAmount(balance)} {LOGO}
        </p>
        <div className="mt-4 space-y-1.5">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Available</span>
            <span className="font-medium">
              {formatAmount(available)} {LOGO}
            </span>
          </div>
          {reserved > 0 && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Reserved</span>
              <span className="font-medium">
                {formatAmount(reserved.toFixed(7))} {LOGO}
              </span>
            </div>
          )}
          <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function StatCard({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        {icon}
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-3xl font-bold tracking-tight">{children}</p>
      </CardContent>
    </Card>
  );
}

function EarningsCards({ earnings }: { earnings: Earnings }) {
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
              {earnings.currency}
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
              {earnings.currency}
            </span>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function UsageChartCard({
  events,
  interval,
  onIntervalChange,
}: {
  events: UsageEvent[];
  interval: Interval;
  onIntervalChange: (i: Interval) => void;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <div className="flex flex-row items-center gap-2">
          <TrendingUp className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">
            Usage Over Time
          </CardTitle>
        </div>
        <div className="flex gap-0">
          {INTERVALS.map((i) => (
            <Button
              key={i}
              variant={interval === i ? "default" : "outline"}
              size="sm"
              onClick={() => onIntervalChange(i)}
              className={
                i === "7d"
                  ? "rounded-r-none"
                  : i === "90d"
                    ? "rounded-l-none"
                    : "rounded-none"
              }
            >
              {i}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        <UsageVolumeChart events={events} />
      </CardContent>
    </Card>
  );
}

function RecentCallsCard({ calls }: { calls: UsageEvent[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Activity className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">Recent API Calls</CardTitle>
      </CardHeader>
      {calls.length > 0 ? (
        <>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Method</TableHead>
                  <TableHead>Route</TableHead>
                  <TableHead>Cost</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="pr-6 text-right">Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {calls.map((call) => (
                  <TableRow key={call.id}>
                    <TableCell className="pl-6">
                      <MethodBadge method={call.method} />
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate font-mono text-xs">
                      {call.route}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {formatAmount(call.request_cost)} {LOGO}
                    </TableCell>
                    <TableCell>
                      <StatusCodeBadge code={call.status_code} />
                    </TableCell>
                    <TableCell className="pr-6 text-right text-xs text-muted-foreground">
                      {timeAgo(call.timestamp)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
          <CardFooter>
            <Button variant="ghost" size="sm" asChild>
              <a href="/usage">View all usage</a>
            </Button>
          </CardFooter>
        </>
      ) : (
        <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
          <Inbox className="h-6 w-6 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No recent API calls</p>
        </CardContent>
      )}
    </Card>
  );
}

function RecentDepositsCard({ deposits }: { deposits: Deposit[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Banknote className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">Recent Deposits</CardTitle>
      </CardHeader>
      {deposits.length > 0 ? (
        <>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Amount</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead className="pr-6">Transaction</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deposits.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell className="pl-6 font-medium whitespace-nowrap">
                      {formatAmount(d.amount)} {LOGO}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={d.status} />
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {timeAgo(d.created_at)}
                    </TableCell>
                    <TableCell className="pr-6 font-mono text-xs text-muted-foreground">
                      {d.tx_hash.length > 12
                        ? `${d.tx_hash.slice(0, 8)}\u2026`
                        : d.tx_hash}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
          <CardFooter>
            <Button variant="ghost" size="sm" asChild>
              <a href="/deposit">View all deposits</a>
            </Button>
          </CardFooter>
        </>
      ) : (
        <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
          <Inbox className="h-6 w-6 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No deposits yet</p>
        </CardContent>
      )}
    </Card>
  );
}

function ProviderRecentCallsCard({ calls }: { calls: UsageEvent[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Activity className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Recent Calls to Your APIs
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pl-6">Method</TableHead>
              <TableHead>Route</TableHead>
              <TableHead>Cost</TableHead>
              <TableHead className="pr-6 text-right">Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {calls.map((call) => (
              <TableRow key={call.id}>
                <TableCell className="pl-6">
                  <MethodBadge method={call.method} />
                </TableCell>
                <TableCell className="max-w-[200px] truncate font-mono text-xs">
                  {call.route}
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  {formatAmount(call.request_cost)}{" "}
                  <span className="text-xs text-muted-foreground">
                    {call.currency}
                  </span>
                </TableCell>
                <TableCell className="pr-6 text-right text-xs text-muted-foreground">
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

function SparklineSection({ data }: { data: DailyEarning[] }) {
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

function QuickActions({ isProvider }: { isProvider: boolean }) {
  return (
    <div className="flex flex-wrap gap-2">
      <Button variant="default" size="sm" asChild>
        <a href="/deposit">Deposit Funds</a>
      </Button>
      <Button variant="outline" size="sm" asChild>
        <a href="/api-keys">Manage API Keys</a>
      </Button>
      <Button variant="outline" size="sm" asChild>
        <a href="/usage">View Usage</a>
      </Button>
      {isProvider && (
        <Button variant="outline" size="sm" asChild>
          <a href="/provider/providers">Manage Providers</a>
        </Button>
      )}
    </div>
  );
}

// ── Loading skeleton ──

function LoadingSkeleton({ isProvider }: { isProvider: boolean }) {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-7 w-24" />
        <Skeleton className="mt-1 h-4 w-64" />
      </div>
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Skeleton className="h-4 w-4 rounded-full" />
            <Skeleton className="h-4 w-20" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-32" />
            <Skeleton className="mt-2 h-3 w-48" />
          </CardContent>
        </Card>
        {[1, 2, 3].map((i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center gap-2">
              <Skeleton className="h-4 w-4 rounded-full" />
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-28" />
            </CardContent>
          </Card>
        ))}
      </div>
      {isProvider && (
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
      )}
      <Skeleton className="h-72 w-full rounded-lg" />
      <Skeleton className="h-48 w-full rounded-lg" />
      <Skeleton className="h-48 w-full rounded-lg" />
    </div>
  );
}

// ── Helpers ──

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { weekday: "short" });
}
