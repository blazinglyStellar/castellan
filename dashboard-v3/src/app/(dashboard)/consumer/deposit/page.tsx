"use client";

import { DepositIntentCard } from "@/components/deposit/deposit-intent-card";
import { DepositHistoryTable } from "@/components/deposit/deposit-history-table";

export default function DepositPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Deposit</h1>
        <p className="text-sm text-muted-foreground">
          Add funds to your account.
        </p>
      </div>

      <DepositIntentCard />
      <DepositHistoryTable />
    </div>
  );
}
