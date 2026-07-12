"use client"

import { useState, useEffect, useRef } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { User, Wallet, Banknote, Pencil, Check, X, AlertCircle, Loader, CheckCircle2, XCircle } from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import { getDashboardMe, getAccount, checkPayoutAddress, updatePayoutAddress } from "@/lib/api/endpoints"
import type { DashboardMeResponse, AccountResponse } from "@/lib/api/types"
import { formatAmount } from "@/lib/format"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ErrorState } from "@/components/shared/error-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

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

      {profile && <PayoutCard profile={profile} onSaved={refetchProfile} />}

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
  profile: Pick<DashboardMeResponse, "deposit_memo">
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
          <span className="text-sm text-muted-foreground">Deposit Memo</span>
          <div className="rounded-md bg-muted p-2">
            <code className="break-all font-mono text-sm">
              {profile.deposit_memo}
            </code>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function PayoutCard({
  profile,
  onSaved,
}: {
  profile: Pick<DashboardMeResponse, "payout_stellar_address">;
  onSaved: () => void;
}) {
  const [editing, setEditing] = useState(!profile.payout_stellar_address)
  const [address, setAddress] = useState(profile.payout_stellar_address ?? "")
  const [validationStatus, setValidationStatus] = useState<"idle" | "checking" | "valid" | "invalid">("idle")
  const [validationMessage, setValidationMessage] = useState("")
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const queryClient = useQueryClient()

  const saveMutation = useMutation({
    mutationFn: updatePayoutAddress,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dashboard-me"] })
      onSaved()
      setEditing(false)
    },
  })

  useEffect(() => {
    if (!address.trim()) {
      setValidationStatus("idle")
      setValidationMessage("")
      return
    }

    if (!address.startsWith("G") || address.length !== 56) {
      setValidationStatus("invalid")
      setValidationMessage("Address must start with G and be exactly 56 characters")
      return
    }

    setValidationStatus("checking")
    setValidationMessage("")

    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(async () => {
      try {
        const result = await checkPayoutAddress(address.trim())
        if (typeof result.valid !== "boolean") {
          setValidationStatus("invalid")
          setValidationMessage("Server returned an unexpected response. Try rebuilding the API server.")
          return
        }
        setValidationStatus(result.valid ? "valid" : "invalid")
        setValidationMessage(result.message ?? "")
      } catch {
        setValidationStatus("invalid")
        setValidationMessage("Failed to verify address. Check your connection and try again.")
      }
    }, 500)

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [address])

  const isValidated = validationStatus === "valid"

  function ValidationIndicator() {
    if (validationStatus === "idle") return null
    if (validationStatus === "checking") {
      return <Loader className="h-4 w-4 animate-spin text-muted-foreground" />
    }
    if (validationStatus === "valid") {
      return <CheckCircle2 className="h-4 w-4 text-green-500" />
    }
    return <XCircle className="h-4 w-4 text-red-500" />
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-600 dark:bg-amber-950 dark:text-amber-400">
          <Wallet className="h-4 w-4" />
        </div>
        <CardTitle className="text-sm font-medium">Payout Address</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {editing ? (
          <>
            <div className="space-y-1">
              <span className="text-sm text-muted-foreground">
                Stellar Wallet Address
              </span>
              <div className="relative">
                <Input
                  value={address}
                  onChange={(e) => setAddress(e.target.value)}
                  placeholder="G..."
                  className="font-mono text-sm pr-10"
                />
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  <ValidationIndicator />
                </div>
              </div>
              {validationStatus === "invalid" && validationMessage && (
                <p className="text-xs text-red-500">{validationMessage}</p>
              )}
              <p className="text-xs text-muted-foreground">
                We will verify the address exists on the Stellar network before enabling the Save button.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={() => saveMutation.mutate(address.trim())}
                disabled={!isValidated || saveMutation.isPending}
              >
                {saveMutation.isPending ? (
                  <>
                    <Loader className="mr-1 h-3 w-3 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Check className="mr-1 h-3 w-3" />
                    Save
                  </>
                )}
              </Button>
              {profile.payout_stellar_address && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setAddress(profile.payout_stellar_address ?? "")
                    setEditing(false)
                    setValidationStatus("idle")
                    setValidationMessage("")
                  }}
                  disabled={saveMutation.isPending}
                >
                  <X className="mr-1 h-3 w-3" />
                  Cancel
                </Button>
              )}
            </div>
            {saveMutation.isError && (
              <div className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-950 dark:text-red-400">
                <AlertCircle className="h-3.5 w-3.5 flex-shrink-0" />
                <span>
                  {saveMutation.error instanceof Error
                    ? saveMutation.error.message
                    : "Failed to save payout address"}
                </span>
              </div>
            )}
          </>
        ) : (
          <>
            <div className="space-y-1">
              <span className="text-sm text-muted-foreground">
                Stellar Wallet Address
              </span>
              <div className="flex items-center gap-2">
                <div className="flex-1 rounded-md bg-muted p-2">
                  <code className="break-all font-mono text-sm">
                    {profile.payout_stellar_address}
                  </code>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditing(true)}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              Settlement payments will be sent to this address.
            </p>
          </>
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
