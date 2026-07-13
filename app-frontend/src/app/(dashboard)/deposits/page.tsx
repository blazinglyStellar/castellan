"use client"

import { DepositIntentCard } from "@/components/deposit/deposit-intent-card"
import { DepositBalanceCard } from "@/components/deposit/deposit-balance-card"
import { UsageProjectionCard } from "@/components/deposit/usage-projection-card"
import { DepositHistoryTable } from "@/components/deposit/deposit-history-table"

export default function DepositsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Deposit</h1>
        <p className="text-sm text-muted-foreground">
          Add funds to your account to pay for API usage.
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        <div className="md:col-span-1 flex">
          <DepositBalanceCard />
        </div>
        <div className="md:col-span-2 flex">
          <DepositIntentCard />
        </div>
      </div>

      <UsageProjectionCard />
      <DepositHistoryTable />
    </div>
  )
}
