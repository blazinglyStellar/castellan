"use client"

import { useState } from "react"
import { AlertCircle, Landmark, Plus, RefreshCw } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { QRDisplay } from "@/components/QRDisplay"
import { Toaster } from "@/components/ui/toaster"
import { useToast } from "@/components/ui/use-toast"
import { MOCK_DEPOSITS } from "@/lib/mock-data"
import { formatCurrency, truncateHash } from "@/lib/utils"
import type { Deposit } from "@/lib/mock-data"

export default function DepositPage() {
  const [loading, setLoading] = useState(false)
  const [isEmpty, setIsEmpty] = useState(false)
  const [error, setError] = useState(false)
  const [polling, setPolling] = useState(false)
  const { toast } = useToast()

  const handleSimulateDeposit = () => {
    setPolling(true)
    setTimeout(() => {
      setPolling(false)
      toast({
        title: "Deposit detected",
        description: "10,000 XLM received. Your balance has been updated.",
        variant: "success",
      })
    }, 2000)
  }

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load deposit info</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  const sep7Uri = "web+stellar:pay?destination=GBRFXKQH45J6X7K8L9M0N1P2Q3R4S5T6U7V8W9X0Y&memo=castellan-deposit-memo-001&memo_type=MEMO_TEXT"
  const stellarAddress = "GBRFXKQH45J6X7K8L9M0N1P2Q3R4S5T6U7V8W9X0Y"
  const depositMemo = "castellan-deposit-memo-001"

  return (
    <div className="space-y-6">
      <PageHeader
        title="Deposit Funds"
        description="Send XLM to fund your Castellan wallet."
        actions={
          <div className="flex items-center gap-2">
            {polling && (
              <Badge variant="secondary" className="gap-1.5">
                <RefreshCw className="h-3 w-3 animate-spin" /> Detecting...
              </Badge>
            )}
            <Button variant="outline" size="sm" onClick={handleSimulateDeposit} disabled={polling}>
              <Plus className="h-4 w-4" /> Simulate Deposit
            </Button>
          </div>
        }
      />

      {isEmpty ? (
        <EmptyState
          title="No deposits yet"
          description="Send XLM to your deposit address to get started."
          icon={<Landmark className="h-12 w-12" />}
          action={<Button>Learn How</Button>}
        />
      ) : (
        <>
          <div className="grid gap-6 lg:grid-cols-2">
            {loading ? (
              <Skeleton className="h-72 w-full" />
            ) : (
              <QRDisplay value={sep7Uri} label="Scan with any SEP-7 wallet" />
            )}
            {loading ? (
              <Skeleton className="h-72 w-full" />
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Deposit Details</CardTitle>
                  <CardDescription>Send XLM to the address below with the memo text.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-muted-foreground">Stellar Address</p>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 break-all rounded-lg bg-muted p-3 text-sm font-mono">
                        {stellarAddress}
                      </code>
                      <CopyButton text={stellarAddress} />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-muted-foreground">Deposit Memo (required)</p>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 rounded-lg bg-muted px-3 py-2 text-sm font-mono">
                        {depositMemo}
                      </code>
                      <CopyButton text={depositMemo} />
                    </div>
                  </div>
                  <div className="rounded-lg bg-warning/10 border border-warning/30 p-3">
                    <p className="text-xs text-warning">
                      Minimum deposit: 5 XLM. Funds are credited within 30 seconds of network confirmation.
                    </p>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Scan the QR code with any Stellar wallet that supports SEP-7.
                  </p>
                </CardContent>
              </Card>
            )}
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Recent Deposits</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable
                columns={[
                  { key: "amount", header: "Amount", cell: (d: Deposit) => <span className="font-mono">{formatCurrency(d.amount)}</span> },
                  { key: "status", header: "Status", cell: (d: Deposit) => <StatusBadge status={d.status} /> },
                  { key: "txHash", header: "Tx Hash", cell: (d: Deposit) => (
                    <div className="flex items-center gap-1">
                      <span className="font-mono text-xs text-muted-foreground">{truncateHash(d.txHash)}</span>
                      <CopyButton text={d.txHash} />
                    </div>
                  )},
                  { key: "date", header: "Date", cell: (d: Deposit) => d.date },
                ]}
                data={MOCK_DEPOSITS}
                loading={loading}
              />
            </CardContent>
          </Card>
        </>
      )}
      <Toaster />
    </div>
  )
}
