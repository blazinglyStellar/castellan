"use client";

/* eslint-disable @next/next/no-img-element */

import { useQuery } from "@tanstack/react-query";
import { Wallet } from "lucide-react";

import { ErrorState } from "@/components/ui/error-state";

import { getDepositIntent } from "@/lib/api/client";
import type { IntentResponse } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";

export function DepositIntentCard() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["deposit-intent"],
    queryFn: getDepositIntent,
  });

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Wallet className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Deposit Instructions
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <IntentSkeleton />
        ) : isError ? (
          <ErrorState
            message={
              error instanceof Error
                ? error.message
                : "Failed to load deposit instructions"
            }
            onRetry={() => refetch()}
          />
        ) : data ? (
          <IntentDetails data={data} />
        ) : null}
      </CardContent>
    </Card>
  );
}

function IntentDetails({
  data,
}: {
  data: IntentResponse;
}) {
  return (
    <div className="flex flex-col items-center gap-6 sm:flex-row sm:items-start">
      <div className="flex-shrink-0">
        {data.qr_code ? (
          <img
            src={data.qr_code}
            alt="Deposit QR code"
            className="h-40 w-40 rounded-lg border"
          />
        ) : (
          <div className="flex h-40 w-40 items-center justify-center rounded-lg border bg-muted">
            <span className="text-xs text-muted-foreground">No QR</span>
          </div>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-3 text-sm">
        <div>
          <span className="text-muted-foreground">Destination</span>
          <p className="mt-0.5 font-mono text-xs break-all">{data.destination}</p>
        </div>

        <div>
          <span className="text-muted-foreground">Memo</span>
          <p className="mt-0.5 font-mono text-xs break-all">{data.memo}</p>
        </div>

        <div>
          <span className="text-muted-foreground">Minimum Amount</span>
          <p className="mt-0.5 font-mono text-xs">
            {data.minimum_amount} {data.asset}
          </p>
        </div>

        <div className="pt-1">
          <Button variant="outline" size="sm" asChild>
            <a
              href={data.sep7_uri}
              target="_blank"
              rel="noopener noreferrer"
            >
              Open in Stellar Wallet
            </a>
          </Button>
        </div>
      </div>
    </div>
  );
}

function IntentSkeleton() {
  return (
    <div className="flex flex-col items-center gap-6 sm:flex-row sm:items-start">
      <Skeleton className="h-40 w-40 rounded-lg" />
      <div className="flex flex-1 flex-col gap-3">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-4 w-24" />
      </div>
    </div>
  );
}


