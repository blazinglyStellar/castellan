import { cn } from "@/lib/utils"
import { Wallet } from "lucide-react"

interface BalanceBadgeProps {
  balance: string
  className?: string
}

export function BalanceBadge({ balance, className }: BalanceBadgeProps) {
  const num = parseFloat(balance)
  const isLow = num < 100
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm font-medium",
        isLow ? "border-warning/30 bg-warning/15 text-warning" : "border-success/30 bg-success/15 text-success",
        className
      )}
    >
      <Wallet className="h-4 w-4" />
      {balance} XLM
    </span>
  )
}
