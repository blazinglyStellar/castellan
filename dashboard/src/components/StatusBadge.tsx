import { cn } from "@/lib/utils"

interface StatusBadgeProps {
  status: "active" | "inactive" | "completed" | "pending" | "failed" | "revoked"
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const variants: Record<string, string> = {
    active: "bg-success/15 text-success border-success/30",
    inactive: "bg-muted text-muted-foreground border-border",
    completed: "bg-success/15 text-success border-success/30",
    pending: "bg-warning/15 text-warning border-warning/30",
    failed: "bg-destructive/15 text-destructive border-destructive/30",
    revoked: "bg-destructive/15 text-destructive border-destructive/30",
  }
  return (
    <span className={cn("inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium", variants[status], className)}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}
