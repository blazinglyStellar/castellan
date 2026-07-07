"use client";

import { useAccount } from "@/lib/auth/account-context";

export default function ProviderUsagePage() {
  const { isLoading } = useAccount();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <h2 className="text-lg font-medium text-foreground">Provider Usage</h2>
      <p className="mt-1 text-sm text-muted-foreground">Coming soon</p>
    </div>
  );
}
