import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

type StatusType = "active" | "inactive" | "pending" | "completed" | "failed" | "revoked" | "rate_limited" | number

interface StatusBadgeProps {
  status: StatusType
}

const statusStyles: Record<string, string> = {
  active: "border-green/30 bg-green/10 text-green",
  completed: "border-green/30 bg-green/10 text-green",
  inactive: "border-border bg-muted text-muted-foreground",
  rate_limited: "border-amber/30 bg-amber/10 text-amber",
  pending: "border-amber/30 bg-amber/10 text-amber",
  failed: "border-red/30 bg-red/10 text-red",
  revoked: "border-red/30 bg-red/10 text-red",
  draft: "border-amber/30 bg-amber/10 text-amber",
}

function getStatusLabel(status: StatusType): string {
  if (typeof status === "number") {
    if (status === 200) return "200 OK"
    if (status === 402) return "402 Payment Required"
    return String(status)
  }
  return status.charAt(0).toUpperCase() + status.slice(1)
}

function getStatusStyle(status: StatusType): string {
  if (typeof status === "number") {
    if (status >= 200 && status < 300) return "border-green/30 bg-green/10 text-green"
    if (status >= 400 && status < 500) return "border-amber/30 bg-amber/10 text-amber"
    return "border-red/30 bg-red/10 text-red"
  }
  return statusStyles[status] || ""
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <Badge variant="outline" className={cn("font-mono text-[11px]", getStatusStyle(status))}>
      {getStatusLabel(status)}
    </Badge>
  )
}
