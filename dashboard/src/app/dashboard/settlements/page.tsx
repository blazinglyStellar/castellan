"use client"

import { useState } from "react"
import { Banknote, AlertCircle, DollarSign, Calendar, Clock, TrendingUp } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { PageHeader } from "@/components/PageHeader"
import { StatCard } from "@/components/StatCard"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { MOCK_SETTLEMENTS } from "@/lib/mock-data"
import { formatCurrency, truncateHash } from "@/lib/utils"
import type { Settlement } from "@/lib/mock-data"

export default function SettlementsPage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load settlements</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  const settlements = MOCK_SETTLEMENTS
  const outstanding = "3420.00"
  const lastPayout = "4520.00"
  const nextEst = "3100.00"
  const totalAllTime = "19680.00"

  return (
    <div className="space-y-6">
      <PageHeader title="Settlements" />
      {isEmpty ? (
        <EmptyState title="No settlements yet" description="Settlements will appear after your first payout cycle." icon={<Banknote className="h-12 w-12" />} />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard title="Outstanding Payout" value={formatCurrency(outstanding)} icon={<DollarSign className="h-4 w-4" />} loading={loading} />
            <StatCard title="Last Payout" value={formatCurrency(lastPayout)} icon={<Calendar className="h-4 w-4" />} loading={loading} />
            <StatCard title="Next Est." value={formatCurrency(nextEst)} icon={<Clock className="h-4 w-4" />} loading={loading} />
            <StatCard title="Total All Time" value={formatCurrency(totalAllTime)} icon={<TrendingUp className="h-4 w-4" />} loading={loading} />
          </div>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Settlement History</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable
                columns={[
                  { key: "date", header: "Date", cell: (s: Settlement) => s.date },
                  { key: "amount", header: "Amount", cell: (s: Settlement) => <span className="font-mono">{formatCurrency(s.amount)}</span> },
                  { key: "status", header: "Status", cell: (s: Settlement) => <StatusBadge status={s.status} /> },
                  { key: "txHash", header: "Tx Hash", cell: (s: Settlement) => (
                    <div className="flex items-center gap-1">
                      <span className="font-mono text-xs text-muted-foreground">{truncateHash(s.txHash)}</span>
                      <CopyButton text={s.txHash} />
                    </div>
                  )},
                ]}
                data={settlements}
                loading={loading}
              />
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
