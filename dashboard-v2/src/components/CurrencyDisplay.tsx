import { cn } from "@/lib/utils"
import { formatCurrency } from "@/lib/utils"
import { StellarLogo } from "@/components/StellarLogo"

interface CurrencyDisplayProps {
  amount: number
  className?: string
}

export function CurrencyDisplay({ amount, className }: CurrencyDisplayProps) {
  const formatted = formatCurrency(amount)
  return (
    <span className={cn("inline-flex items-center gap-1", className)}>
      <StellarLogo className="size-3.5 shrink-0" />
      <span>{formatted.replace("XLM ", "")}</span>
    </span>
  )
}
