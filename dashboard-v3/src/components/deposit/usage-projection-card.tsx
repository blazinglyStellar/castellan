"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp } from "lucide-react";

import { getUsage, getBalance, ApiError } from "@/lib/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatAmount } from "@/lib/format";

export function UsageProjectionCard() {
  const now = useMemo(() => new Date(), []);
  const thirtyDaysAgo = useMemo(() => {
    const d = new Date(now);
    d.setDate(d.getDate() - 30);
    return d.toISOString();
  }, [now]);

  const balanceQuery = useQuery({
    queryKey: ["deposit-page-balance"],
    queryFn: getBalance,
  });

  const usageQuery = useQuery({
    queryKey: ["deposits", "projection", thirtyDaysAgo],
    queryFn: () =>
      getUsage({
        role: "consumer",
        start_date: thirtyDaysAgo,
        limit: 500,
      }),
  });

  const projection = useMemo(() => {
    if (!usageQuery.data?.data.length || !balanceQuery.data) return null;

    const events = usageQuery.data.data;
    const days = Math.max(
      1,
      Math.ceil(
        (now.getTime() - new Date(thirtyDaysAgo).getTime())
          / (1000 * 60 * 60 * 24),
      ),
    );
    const totalSpent = events.reduce(
      (s, e) => s + parseFloat(e.request_cost),
      0,
    );
    const dailyBurn = totalSpent / days;
    const available =
      parseFloat(balanceQuery.data.available_balance ?? balanceQuery.data.balance ?? "0");

    if (dailyBurn <= 0) return null;
    const daysRemaining = Math.floor(available / dailyBurn);

    return { dailyBurn, available, daysRemaining };
  }, [usageQuery.data, balanceQuery.data, now, thirtyDaysAgo]);

  const isBalance404 =
    balanceQuery.error instanceof ApiError && balanceQuery.error.status === 404;

  const isLoading = usageQuery.isLoading || (balanceQuery.isLoading && !isBalance404);
  const isError = usageQuery.isError || balanceQuery.isError;

  if (isLoading) return <ProjectionSkeleton />;
  if (isError || !projection) return null;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <TrendingUp className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Usage Projection
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-xs text-muted-foreground">
          You spend{" "}
          <strong className="text-foreground">
            ~{formatAmount(String(projection.dailyBurn))} XLM
          </strong>{" "}
          per day on average. Your available balance will last approximately
        </p>
        <p className="text-2xl font-bold tracking-tight">
          {projection.daysRemaining >= 365
            ? `~${Math.floor(projection.daysRemaining / 365)} year${Math.floor(projection.daysRemaining / 365) > 1 ? "s" : ""}`
            : projection.daysRemaining >= 30
              ? `~${Math.floor(projection.daysRemaining / 30)} months`
              : `${projection.daysRemaining} days`}
        </p>


      </CardContent>
    </Card>
  );
}

function ProjectionSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Skeleton className="h-4 w-4 rounded" />
        <Skeleton className="h-4 w-28" />
      </CardHeader>
      <CardContent className="space-y-4">
        <Skeleton className="h-3 w-64" />
        <Skeleton className="h-7 w-24" />
      </CardContent>
    </Card>
  );
}
