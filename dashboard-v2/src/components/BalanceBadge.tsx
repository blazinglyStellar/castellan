"use client"

import { Wallet } from "lucide-react"
import { Skeleton } from "@/components/ui/skeleton"
import { StellarLogo } from "@/components/StellarLogo"
import { cn } from "@/lib/utils"

interface BalanceBadgeProps {
  balance: number
  loading?: boolean
}

export function BalanceBadge({ balance, loading }: BalanceBadgeProps) {
  if (loading) {
    return <Skeleton className="h-7 w-28 rounded-full" />
  }

  const isLow = balance < 1

  return (
    <div
      className={cn(
        "flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors",
        isLow
          ? "border-amber/30 bg-amber/10 text-amber"
          : "border-green/30 bg-green/10 text-green",
      )}
    >
      <Wallet className="size-3.5" />
      <StellarLogo className="size-3" />
      <span className="font-mono">
        {balance.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}
      </span>
    </div>
  )
}
