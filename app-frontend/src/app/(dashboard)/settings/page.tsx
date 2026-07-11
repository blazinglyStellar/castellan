"use client"

import { useQuery } from "@tanstack/react-query"

import { User, Wallet, Banknote } from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import { getDashboardMe, getAccount } from "@/lib/api/endpoints"
import type { DashboardMeResponse, AccountResponse } from "@/lib/api/types"
import { formatAmount } from "@/lib/format"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ErrorState } from "@/components/shared/error-state"

export default function SettingsPage() {
  const { isLoading: isAccountLoading } = useAuth()

  const {
    data: profile,
    isLoading: isProfileLoading,
    isError: isProfileError,
    error: profileError,
    refetch: refetchProfile,
  } = useQuery({
    queryKey: ["dashboard-me"],
    queryFn: getDashboardMe,
  })

  const {
    data: account,
    isLoading: isAccountDataLoading,
    isError: isAccountDataError,
    error: accountError,
    refetch: refetchAccount,
  } = useQuery({
    queryKey: ["account"],
    queryFn: getAccount,
  })

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    )
  }

  const isLoading = isProfileLoading || isAccountDataLoading
  const isError = isProfileError || isAccountDataError

  if (isLoading) {
    return <LoadingSkeleton />
  }

  if (isError) {
    const errMsg =
      profileError instanceof Error
        ? profileError.message
        : accountError instanceof Error
          ? accountError.message
          : "Failed to load settings"
    return (
      <ErrorState
        message={errMsg}
        onRetry={() => {
          refetchProfile()
          refetchAccount()
        }}
      />
    )
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
  )
}

function ProfileCard({
  profile,
}: {
  profile: Pick<DashboardMeResponse, "email" | "role">
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400">
          <User className="h-4 w-4" />
        </div>
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
  )
}

function DepositCard({
  profile,
}: {
  profile: Pick<DashboardMeResponse, "deposit_memo" | "payout_stellar_address">
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400">
          <Wallet className="h-4 w-4" />
        </div>
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
  )
}

function AccountCard({
  account,
}: {
  account: Pick<AccountResponse, "balance" | "currency">
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400">
          <Banknote className="h-4 w-4" />
        </div>
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
  )
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
          <CardHeader className="flex flex-row items-center gap-3">
            <Skeleton className="h-7 w-7 rounded-lg" />
            <Skeleton className="h-4 w-20" />
          </CardHeader>
          <CardContent className="space-y-3">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
