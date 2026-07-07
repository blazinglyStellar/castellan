"use client";

import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import { getDashboardMe, getAccount } from "@/lib/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export default function SettingsPage() {
  const { isLoading: isAccountLoading } = useAccount();

  const {
    data: profile,
    isLoading: isProfileLoading,
    isError: isProfileError,
    error: profileError,
    refetch: refetchProfile,
  } = useQuery({
    queryKey: ["dashboard-me"],
    queryFn: getDashboardMe,
  });

  const {
    data: account,
    isLoading: isAccountDataLoading,
    isError: isAccountDataError,
    error: accountError,
    refetch: refetchAccount,
  } = useQuery({
    queryKey: ["account"],
    queryFn: getAccount,
  });

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const isLoading = isProfileLoading || isAccountDataLoading;
  const isError = isProfileError || isAccountDataError;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    const errMsg =
      profileError instanceof Error
        ? profileError.message
        : accountError instanceof Error
          ? accountError.message
          : "Failed to load settings";
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          refetchProfile();
          refetchAccount();
        }}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your account and deposit information.
        </p>
      </div>

      {profile && <ProfileCard profile={profile} />}

      {profile && <DepositCard profile={profile} />}

      {account && <AccountCard account={account} />}
    </div>
  );
}

function ProfileCard({
  profile,
}: {
  profile: { email: string; role: string };
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Profile</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Email</span>
          <span className="text-sm font-medium">{profile.email}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Role</span>
          <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-primary">
            {profile.role}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function DepositCard({
  profile,
}: {
  profile: { deposit_memo: string; payout_stellar_address?: string };
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Deposit Info</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="space-y-1">
          <span className="text-sm text-muted-foreground">
            Deposit Memo
          </span>
          <div className="rounded-md bg-muted p-2">
            <code className="break-all font-mono text-sm">
              {profile.deposit_memo}
            </code>
          </div>
        </div>
        {profile.payout_stellar_address && (
          <div className="space-y-1">
            <span className="text-sm text-muted-foreground">
              Payout Stellar Address
            </span>
            <div className="rounded-md bg-muted p-2">
              <code className="break-all font-mono text-sm">
                {profile.payout_stellar_address}
              </code>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function AccountCard({
  account,
}: {
  account: { balance: string; currency: string };
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Account Balance</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-3xl font-bold tracking-tight">
          {formatAmount(account.balance)}{" "}
          <span className="text-sm font-normal text-muted-foreground">
            {account.currency}
          </span>
        </p>
      </CardContent>
    </Card>
  );
}

// ── States ──

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-7 w-24" />
        <Skeleton className="mt-1 h-4 w-48" />
      </div>
      {Array.from({ length: 3 }).map((_, i) => (
        <Card key={i}>
          <CardHeader>
            <Skeleton className="h-4 w-20" />
          </CardHeader>
          <CardContent className="space-y-3">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-12">
        <p className="text-sm text-muted-foreground">{message}</p>
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Retry
        </Button>
      </CardContent>
    </Card>
  );
}

// ── Helpers ──

function formatAmount(amount: string): string {
  const num = parseFloat(amount);
  if (isNaN(num)) return "0.0000";
  return num.toFixed(4);
}
