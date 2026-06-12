import { AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import Link from "next/link"

interface LowBalanceBannerProps {
  balance: string
}

export function LowBalanceBanner({ balance }: LowBalanceBannerProps) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-warning/30 bg-warning/10 p-4">
      <AlertTriangle className="h-5 w-5 shrink-0 text-warning" />
      <div className="flex-1 text-sm">
        <span className="font-medium text-warning">Low balance:</span>{" "}
        <span className="text-muted-foreground">You have {balance} XLM. Add funds to avoid service interruption.</span>
      </div>
      <Button variant="outline" size="sm" asChild>
        <Link href="/dashboard/deposit">Deposit</Link>
      </Button>
    </div>
  )
}
