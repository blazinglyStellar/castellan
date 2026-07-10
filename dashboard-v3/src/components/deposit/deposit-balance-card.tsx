"use client";

import Image from "next/image";
import { useQuery } from "@tanstack/react-query";
import {
  Wallet,
  Lock,
  TrendingUp,
  Banknote,
  Clock,
  CheckCircle2,
  AlertCircle,
} from "lucide-react";

import { getBalance, getDeposits, ApiError } from "@/lib/api/client";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/error-state";
import { formatAmount, timeAgo } from "@/lib/format";

const XLM_LOGO = (
  <Image
    src="/stellar-xlm-logo.svg"
    alt="XLM"
    width={14}
    height={12}
    className="inline-block align-middle"
  />
);

export function DepositBalanceCard() {
  const balanceQuery = useQuery({
    queryKey: ["deposit-page-balance"],
    queryFn: getBalance,
    refetchInterval: 15_000,
  });

  const statsQuery = useQuery({
    queryKey: ["deposits", "stats"],
    queryFn: () => getDeposits({ limit: 200 }),
  });

  const isBalance404 =
    balanceQuery.error instanceof ApiError && balanceQuery.error.status === 404;

  if ((balanceQuery.isLoading && !isBalance404) || statsQuery.isLoading) {
    return <BalanceSkeleton />;
  }

  if (balanceQuery.isError && !isBalance404) {
    return (
      <ErrorState
        message={
          balanceQuery.error instanceof Error
            ? balanceQuery.error.message
            : "Failed to load balance"
        }
        onRetry={() => balanceQuery.refetch()}
      />
    );
  }

  const balance = !isBalance404 && balanceQuery.data
    ? parseFloat(balanceQuery.data.balance)
    : 0;
  const available = !isBalance404 && balanceQuery.data
    ? parseFloat(balanceQuery.data.available_balance)
    : 0;
  const reserved = balance - available;
  const pct = balance > 0 ? (available / balance) * 100 : 0;
  const hasLowBalance = balance > 0 && pct < 20;

  const deposits = statsQuery.data?.data ?? [];
  const totalDeposited = deposits.reduce(
    (sum, d) => sum + parseFloat(d.amount),
    0,
  );
  const pending = deposits.filter((d) => d.status === "pending").length;
  const confirmed = deposits.filter((d) => d.status === "confirmed").length;
  const failed = deposits.filter((d) => d.status === "failed").length;
  const lastDeposit = deposits
    .slice()
    .sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    )[0];

  return (
    <Card className={`flex flex-1 flex-col ${hasLowBalance ? "border-amber-400" : ""}`}>
      <CardHeader className="flex flex-row items-center gap-2 pb-3">
        <Wallet className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">Your Balance</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col justify-between space-y-4">
        <div>
          <p className="text-3xl font-bold tracking-tight">
            {formatAmount(String(available))}{" "}
            <span className="inline-flex items-center text-base font-normal text-muted-foreground">
              {XLM_LOGO}
            </span>
          </p>
          <p className="text-xs text-muted-foreground">available to spend</p>
        </div>

        <div className="space-y-1.5">
          <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              className={`h-full rounded-full transition-all ${
                hasLowBalance ? "bg-amber-500" : "bg-primary"
              }`}
              style={{ width: `${Math.max(pct, 4)}%` }}
            />
          </div>

          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <span className="h-1.5 w-1.5 rounded-full bg-primary" />
              {formatAmount(String(balance))} {XLM_LOGO} total
            </span>
            {reserved > 0 && (
              <span className="flex items-center gap-1">
                <Lock className="h-3 w-3" />
                {formatAmount(String(reserved))} {XLM_LOGO} reserved
              </span>
            )}
          </div>
        </div>

        {hasLowBalance && (
          <div className="flex items-center gap-2 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-950 dark:text-amber-400">
            <TrendingUp className="h-3.5 w-3.5 flex-shrink-0" />
            <span>
              Your available balance is running low. Consider depositing more
              funds.
            </span>
          </div>
        )}

        <div className="space-y-2 border-t pt-3">
          <KpiRow
            icon={<Banknote className="h-4 w-4 text-muted-foreground" />}
            label="Total Deposited"
            value={<>{formatAmount(String(totalDeposited))} {XLM_LOGO}</>}
          />
          <KpiRow
            icon={
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            }
            label="Confirmed"
            value={String(confirmed)}
          />
          <KpiRow
            icon={<Clock className="h-4 w-4 text-yellow-500" />}
            label="Pending"
            value={String(pending)}
          />
          <KpiRow
            icon={
              lastDeposit ? (
                <Banknote className="h-4 w-4 text-muted-foreground" />
              ) : (
                <AlertCircle className="h-4 w-4 text-destructive" />
              )
            }
            label="Last Deposit"
            value={lastDeposit ? timeAgo(lastDeposit.created_at) : "—"}
          />
          {failed > 0 && (
            <p className="text-xs text-red-600 dark:text-red-400">
              {failed} deposit{failed > 1 ? "s" : ""} failed.
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function KpiRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="flex items-center gap-2 text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="flex items-center gap-1 font-medium">{value}</span>
    </div>
  );
}

function BalanceSkeleton() {
  return (
    <Card className="flex flex-1 flex-col">
      <CardHeader className="flex flex-row items-center gap-2 pb-3">
        <Skeleton className="h-4 w-4 rounded-full" />
        <Skeleton className="h-4 w-24" />
      </CardHeader>
      <CardContent className="flex flex-1 flex-col justify-between space-y-4">
        <div>
          <Skeleton className="h-8 w-36" />
          <Skeleton className="mt-1 h-3 w-24" />
        </div>
        <div className="space-y-1.5">
          <Skeleton className="h-2 w-full rounded-full" />
          <div className="flex items-center justify-between">
            <Skeleton className="h-3 w-28" />
            <Skeleton className="h-3 w-28" />
          </div>
        </div>
        <div className="space-y-2 border-t pt-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Skeleton className="h-4 w-4 rounded" />
                <Skeleton className="h-3 w-24" />
              </div>
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
