"use client";

import { useQuery } from "@tanstack/react-query";
import {
  Wallet,
  Copy,
  Check,
  ExternalLink,
  Info,
  HelpCircle,
} from "lucide-react";

import { getDepositIntent } from "@/lib/api/endpoints";
import type { IntentResponse } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button, buttonVariants } from "@/components/ui/button";
import { useCopyToClipboard } from "@/lib/use-copy-to-clipboard";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-3 py-6 text-center">
      <p className="text-sm text-red-500">{message}</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        Retry
      </Button>
    </div>
  );
}

export function DepositIntentCard() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["deposit-intent"],
    queryFn: getDepositIntent,
  });

  return (
    <Card className="flex flex-1 flex-col">
      <CardHeader className="flex flex-row items-center gap-2">
        <Wallet className="h-4 w-4 text-muted-foreground" />
        <CardTitle className="text-sm font-medium">
          Deposit Instructions
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1">
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

function IntentDetails({ data }: { data: IntentResponse }) {
  const { copied, copy } = useCopyToClipboard();

  return (
    <TooltipProvider delay={0}>
      <div className="flex flex-col gap-6 sm:flex-row sm:items-start">
        <div className="flex flex-shrink-0 flex-col items-center gap-2">
          {data.qr_code ? (
            <img
              src={data.qr_code}
              alt="Deposit QR code"
              className="h-[260px] w-[260px] rounded-lg border"
            />
          ) : (
            <div className="flex h-[260px] w-[260px] items-center justify-center rounded-lg border bg-muted">
              <span className="text-xs text-muted-foreground">No QR</span>
            </div>
          )}
          <p className="text-center text-[10px] text-muted-foreground">
            Scan to auto-fill the address
          </p>
        </div>

        <div className="flex flex-1 flex-col gap-3 text-sm">
          <div>
            <span className="text-muted-foreground">Destination</span>
            <div className="mt-0.5 flex items-center gap-1.5">
              <p className="flex-1 truncate font-mono text-xs">{data.destination}</p>
              <button
                type="button"
                onClick={() => copy(data.destination, "address")}
                className="flex-shrink-0 rounded p-1 text-muted-foreground hover:bg-muted"
                title="Copy address"
              >
                {copied === "address" ? (
                  <Check className="h-3.5 w-3.5 text-green-500" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </button>
            </div>
          </div>

          <div>
            <span className="text-muted-foreground">Memo (required)</span>
            <div className="mt-0.5 flex items-center gap-1.5">
              <p className="flex-1 font-mono text-xs">{data.memo}</p>
              <button
                type="button"
                onClick={() => copy(data.memo, "memo")}
                className="flex-shrink-0 rounded p-1 text-muted-foreground hover:bg-muted"
                title="Copy memo"
              >
                {copied === "memo" ? (
                  <Check className="h-3.5 w-3.5 text-green-500" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </button>
              <Tooltip>
                <TooltipTrigger>
                  <button
                    type="button"
                    className="flex-shrink-0 rounded p-1 text-muted-foreground hover:bg-muted"
                  >
                    <Info className="h-3.5 w-3.5" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-[220px] text-xs">
                  This memo ensures your deposit is credited to the right
                  account. Always include it when sending.
                </TooltipContent>
              </Tooltip>
            </div>
          </div>

          <div>
            <span className="text-muted-foreground">Minimum Amount</span>
            <p className="mt-0.5 font-mono text-xs">
              {data.minimum_amount} {data.asset}
            </p>
          </div>

          <div className="flex flex-wrap gap-2 pt-1">
            <a
              href={data.sep7_uri}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "default", size: "sm" })}
            >
              <Wallet className="mr-1.5 h-3.5 w-3.5" />
              Open in Stellar Wallet
            </a>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const text = `Send ${data.minimum_amount} ${data.asset} to ${data.destination} with memo ${data.memo}`;
                copy(text, "all");
              }}
            >
              {copied === "all" ? (
                <>
                  <Check className="mr-1.5 h-3.5 w-3.5 text-green-500" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="mr-1.5 h-3.5 w-3.5" />
                  Copy All
                </>
              )}
            </Button>
          </div>

          <Tooltip>
            <TooltipTrigger>
              <button
                type="button"
                className="mt-1 flex w-fit items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
              >
                <HelpCircle className="h-3 w-3" />
                Need help?
              </button>
            </TooltipTrigger>
            <TooltipContent
              side="bottom"
              align="start"
              className="w-[300px] max-w-[calc(100vw-2rem)] p-4"
              sideOffset={8}
            >
              <div className="space-y-3 text-xs">
                <div>
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    SEP-7 URI
                  </span>
                  <div className="mt-1 flex items-center gap-1.5">
                    <p className="flex-1 truncate font-mono text-[11px]">
                      {data.sep7_uri}
                    </p>
                    <button
                      type="button"
                      onClick={(e) => {
                        e.preventDefault();
                        copy(data.sep7_uri, "sep7");
                      }}
                      className="flex-shrink-0 rounded p-1 text-muted-foreground hover:bg-muted"
                    >
                      {copied === "sep7" ? (
                        <Check className="h-3 w-3 text-green-500" />
                      ) : (
                        <Copy className="h-3 w-3" />
                      )}
                    </button>
                  </div>
                </div>

                <div className="space-y-2">
                  <p className="flex items-start gap-1.5">
                    <Info className="mt-0.5 h-3 w-3 flex-shrink-0" />
                    <span>
                      <strong>What is a memo?</strong> A unique identifier that
                      tells us which account to credit. Your deposit won&apos;t
                      be processed without the correct memo.
                    </span>
                  </p>
                  <p className="flex items-start gap-1.5">
                    <ExternalLink className="mt-0.5 h-3 w-3 flex-shrink-0" />
                    <span>
                      <strong>Need XLM?</strong> You can buy XLM on exchanges
                      like Kraken, Coinbase, or StellarX, then withdraw to the
                      destination address above.
                    </span>
                  </p>
                  <p className="flex items-start gap-1.5 text-amber-600 dark:text-amber-400">
                    <Info className="mt-0.5 h-3 w-3 flex-shrink-0" />
                    <span>
                      <strong>Important:</strong> Only send XLM on the Stellar
                      network. Sending from other networks (Ethereum, BSC, etc.)
                      will result in permanent loss of funds.
                    </span>
                  </p>
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
    </TooltipProvider>
  );
}

function IntentSkeleton() {
  return (
    <div className="flex flex-col items-center gap-6 sm:flex-row sm:items-start">
      <Skeleton className="h-[260px] w-[260px] rounded-lg" />
      <div className="flex flex-1 flex-col gap-3">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-4 w-24" />
      </div>
    </div>
  );
}
