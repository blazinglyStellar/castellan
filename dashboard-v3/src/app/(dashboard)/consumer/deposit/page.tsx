"use client";

import { useAccount } from "@/lib/auth/account-context";
import { DepositIntentCard } from "@/components/deposit/deposit-intent-card";
import { DepositHistoryTable } from "@/components/deposit/deposit-history-table";

export default function DepositPage() {
  const { isLoading } = useAccount();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

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
