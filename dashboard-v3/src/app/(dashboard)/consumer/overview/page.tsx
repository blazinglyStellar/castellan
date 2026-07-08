"use client";

import { useQuery } from "@tanstack/react-query";
import { Wallet } from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useAccount } from "@/lib/auth/account-context";
import { getBalance } from "@/lib/api/client";
import { formatAmount } from "@/lib/format";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";

export default function ConsumerOverviewPage() {
  const { isLoading: isAccountLoading } = useAccount();

  const {
    data: balance,
    isLoading: isBalanceLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["balance"],
    queryFn: getBalance,
  });

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isLoading = isBalanceLoading;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-sm text-muted-foreground">
          Your account balance and usage at a glance.
        </p>
      </div>

      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2">
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
        </div>
      ) : isError ? (
        <ErrorState
          message={error instanceof Error ? error.message : "Failed to load balance"}
          onRetry={() => refetch()}
        />
      ) : balance && balance.balance !== "0" ? (
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Wallet className="h-4 w-4 text-muted-foreground" />
              <CardTitle className="text-sm font-medium">Balance</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-3xl font-bold tracking-tight">
                {formatAmount(balance.balance)}{" "}
                <span className="text-sm font-normal text-muted-foreground">
                  {balance.currency}
                </span>
              </p>
              <div className="mt-4 space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Available</span>
                  <span className="font-medium">
                    {formatAmount(balance.available_balance)} {balance.currency}
                  </span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{
                      width: `${availablePercentage(
                        balance.balance,
                        balance.available_balance
                      )}%`,
                    }}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      ) : (
        <EmptyState
          title="No balance yet"
          description="Make a deposit to get started."
          action={
            <Button variant="outline" size="sm" asChild>
              <a href="/consumer/deposit">Deposit funds</a>
            </Button>
          }
        />
      )}
    </div>
  );
}



function availablePercentage(total: string, available: string): number {
  const t = parseFloat(total);
  const a = parseFloat(available);
  if (t <= 0) return 0;
  return Math.min((a / t) * 100, 100);
}
