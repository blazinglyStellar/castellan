"use client";

import { useAccount } from "@/lib/auth/account-context";
import { ApiKeysView } from "@/components/api-keys/api-keys-view";

export default function ProviderApiKeysPage() {
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
        <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
      </div>
      <ApiKeysView />
    </div>
  );
}
