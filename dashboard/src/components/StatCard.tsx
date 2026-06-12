import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

interface StatCardProps {
  title: string
  value: string
  subtitle?: string
  icon?: React.ReactNode
  trend?: "up" | "down"
  loading?: boolean
  className?: string
}

export function StatCard({ title, value, subtitle, icon, trend, loading, className }: StatCardProps) {
  if (loading) {
    return (
      <Card className={className}>
        <CardContent className="p-6">
          <Skeleton className="mb-2 h-4 w-24" />
          <Skeleton className="mb-1 h-7 w-32" />
          <Skeleton className="h-3 w-20" />
        </CardContent>
      </Card>
    )
  }
  return (
    <Card className={cn("", className)}>
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          {icon && <div className="text-muted-foreground">{icon}</div>}
        </div>
        <p className="mt-2 text-2xl font-bold tracking-tight">{value}</p>
        {subtitle && (
          <p className={cn("mt-1 text-xs", trend === "up" ? "text-success" : trend === "down" ? "text-destructive" : "text-muted-foreground")}>
            {subtitle}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
