"use client"

import { use } from "react"
import { useQuery } from "@tanstack/react-query"
import { ArrowLeft } from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import { getAccountEntry } from "@/lib/api/endpoints"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ErrorState } from "@/components/shared/error-state"
import { Button } from "@/components/ui/button"

const ENTRY_TYPE_LABELS: Record<string, string> = {
  deposit: "Deposit",
  reservation: "Reservation",
  deduction: "Deduction",
  refund: "Refund",
  settlement: "Settlement",
}

export default function LedgerEntryPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const { isLoading: isAccountLoading } = useAuth()

  const {
    data: entry,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["account-entry", id],
    queryFn: () => getAccountEntry(id),
    enabled: !!id,
  })

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <a href="/ledger">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </a>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Entry Detail
          </h1>
          <p className="text-sm text-muted-foreground">
            View details of a ledger entry.
          </p>
        </div>
      </div>

      {isLoading ? (
        <LoadingSkeleton />
      ) : isError ? (
        <ErrorState
          message={
            error instanceof Error ? error.message : "Failed to load entry"
          }
          onRetry={() => refetch()}
        />
      ) : entry ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              {ENTRY_TYPE_LABELS[entry.entry_type] ?? entry.entry_type} Entry
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="divide-y">
              <Row label="Entry ID" value={entry.id} mono />
              <Row
                label="Type"
                value={ENTRY_TYPE_LABELS[entry.entry_type] ?? entry.entry_type}
              />
              <Row
                label="Amount"
                value={`${formatAmount(entry.amount)} ${entry.currency}`}
                mono
              />
              <Row
                label="Balance After"
                value={`${formatAmount(entry.balance_after)} ${entry.currency}`}
                mono
              />
              <Row label="Currency" value={entry.currency} />
              {entry.reference_type && (
                <Row label="Reference Type" value={entry.reference_type} />
              )}
              <Row label="Status" value={entry.status} />
              {entry.description && (
                <Row label="Description" value={entry.description} />
              )}
              <Row
                label="Created"
                value={new Date(entry.created_at).toLocaleString()}
              />
            </dl>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function Row({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between py-3">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd
        className={`text-sm font-medium ${
          mono ? "font-mono text-xs" : ""
        }`}
      >
        {value}
      </dd>
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-4 w-28" />
      </CardHeader>
      <CardContent>
        <div className="divide-y">
          {Array.from({ length: 7 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between py-3"
            >
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3 w-32" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function formatAmount(amount: string): string {
  const num = parseFloat(amount)
  if (isNaN(num)) return "0.0000"
  return num.toFixed(4)
}
