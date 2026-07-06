"use client"

import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { TrendingUp, TrendingDown } from "lucide-react"
import type { LucideIcon } from "lucide-react"

interface StatCardProps {
  title: string
  value: string | React.ReactNode
  subtitle?: string
  icon?: LucideIcon
  trend?: "up" | "down"
  loading?: boolean
}

export function StatCard({ title, value, subtitle, icon: Icon, trend, loading }: StatCardProps) {
  if (loading) {
    return (
      <Card>
        <CardContent className="flex flex-col gap-2 p-4">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-7 w-32" />
          <Skeleton className="h-3 w-20" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-1.5 p-4">
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {Icon && <Icon className="size-3.5" />}
          <span>{title}</span>
        </div>
        <div className="flex items-baseline gap-2">
          {typeof value === "string" ? (
            <span className="text-2xl font-semibold tracking-tight">{value}</span>
          ) : (
            value
          )}
          {trend && (
            <span className={cn("flex items-center text-xs", trend === "up" ? "text-green" : "text-red")}>
              {trend === "up" ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
            </span>
          )}
        </div>
        {subtitle && <span className="text-xs text-muted-foreground">{subtitle}</span>}
      </CardContent>
    </Card>
  )
}
