"use client"

import { useState } from "react"
import { QRCodeSVG } from "qrcode.react"
import {
  Landmark,
  Copy,
  RefreshCw,
  AlertCircle,
  ArrowUpRight,
} from "lucide-react"
import { Card, CardAction, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { CurrencyDisplay } from "@/components/CurrencyDisplay"
import { MOCK_DEPOSITS } from "@/lib/mock-data"
import { formatDate, truncateHash } from "@/lib/utils"
import { toast } from "sonner"
import type { ColumnDef } from "@tanstack/react-table"
import type { Deposit } from "@/lib/types"

const stellarAddress = "GBRFXKQH45J6X7K8L9M0N1P2Q3R4S5T6U7V8W9X0Y"
const depositMemo = "castellan-deposit-memo-001"
const sep7Uri = `web+stellar:pay?destination=${stellarAddress}&memo=${depositMemo}&memo_type=MEMO_TEXT`

const columns: ColumnDef<Deposit>[] = [
  {
    accessorKey: "amount",
    header: "Amount",
    cell: ({ row }) => <CurrencyDisplay amount={row.getValue("amount")} />,
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.getValue("status")} />,
  },
  {
    accessorKey: "txHash",
    header: "Tx Hash",
    cell: ({ row }) => {
      const hash: string = row.getValue("txHash")
      return (
        <div className="flex items-center gap-1">
          <span className="font-mono text-xs text-muted-foreground">{truncateHash(hash)}</span>
          <CopyButton value={hash} />
        </div>
      )
    },
  },
  {
    accessorKey: "date",
    header: "Date",
    cell: ({ row }) => formatDate(row.getValue("date")),
  },
]

export default function DepositPage() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [polling, setPolling] = useState(false)

  const handleSimulate = () => {
    setPolling(true)
    setTimeout(() => {
      setPolling(false)
      toast.success("Deposit detected", {
        description: "10,000 XLM received. Your balance has been updated.",
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

  return (
    <div className="space-y-6">
      <PageHeader
        title="Deposit Funds"
        description="Send XLM to fund your Castellan wallet."
      />

      {loading ? (
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-80" />
          <Skeleton className="h-80" />
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-2">
          <Card>
            <CardContent className="flex flex-col items-center gap-4 p-6">
              <p className="text-sm font-medium text-muted-foreground">
                Scan with any SEP-7 wallet
              </p>
              <div className="flex items-center justify-center rounded-lg border bg-white p-4">
                <QRCodeSVG value={sep7Uri} size={192} />
              </div>
              <code className="max-w-full break-all rounded bg-muted px-3 py-2 text-xs text-muted-foreground">
                {stellarAddress}
              </code>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Deposit Details</CardTitle>
              <CardDescription>
                Send XLM to the address below with the memo text.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <p className="text-sm font-medium text-muted-foreground">Stellar Address</p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 break-all rounded-lg bg-muted p-3 font-mono text-sm">
                    {stellarAddress}
                  </code>
                  <CopyButton value={stellarAddress} />
                </div>
              </div>
              <div className="space-y-2">
                <p className="text-sm font-medium text-muted-foreground">Deposit Memo (required)</p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded-lg bg-muted px-3 py-2 font-mono text-sm">
                    {depositMemo}
                  </code>
                  <CopyButton value={depositMemo} />
                </div>
              </div>
              <div className="rounded-lg border border-warning/30 bg-warning/10 p-3">
                <p className="text-xs text-warning">
                  Minimum deposit: 5 XLM. Funds are credited within 30 seconds of network confirmation.
                </p>
              </div>
              <p className="text-xs text-muted-foreground">
                Scan the QR code with any Stellar wallet that supports SEP-7.
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Recent Deposits</CardTitle>
          <CardAction>
            <div className="flex items-center gap-2">
              {polling && (
                <Badge variant="secondary" className="gap-1.5">
                  <RefreshCw className="h-3 w-3 animate-spin" /> Detecting...
                </Badge>
              )}
              <Button variant="outline" size="sm" onClick={handleSimulate} disabled={polling}>
                <ArrowUpRight className="mr-1 h-3.5 w-3.5" /> Simulate Deposit
              </Button>
            </div>
          </CardAction>
        </CardHeader>
        <CardContent>
          <DataTable columns={columns} data={MOCK_DEPOSITS} loading={loading} />
        </CardContent>
      </Card>
    </div>
  )
}
