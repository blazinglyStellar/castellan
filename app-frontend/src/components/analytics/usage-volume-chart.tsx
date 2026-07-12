"use client"

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

interface DailyVolume {
  date: string
  cost: number
  calls: number
}

interface UsageVolumeChartProps {
  events: { request_cost: string; timestamp: string }[]
}

export function UsageVolumeChart({ events }: UsageVolumeChartProps) {
  if (events.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No usage data for the selected period
      </div>
    )
  }

  const byDate = new Map<string, { cost: number; calls: number }>()

  for (const ev of events) {
    const date = ev.timestamp.slice(0, 10)
    const existing = byDate.get(date) || { cost: 0, calls: 0 }
    existing.cost += parseFloat(ev.request_cost)
    existing.calls += 1
    byDate.set(date, existing)
  }

  const chartData: DailyVolume[] = Array.from(byDate.entries())
    .map(([date, v]) => ({ date, cost: v.cost, calls: v.calls }))
    .sort((a, b) => a.date.localeCompare(b.date))

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData}>
          <defs>
            <linearGradient id="usage" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.3} />
              <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
          <XAxis
            dataKey="date"
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(d: string) => {
              const dt = new Date(d + "T00:00:00Z")
              return dt.toLocaleDateString("en-US", { month: "short", day: "numeric" })
            }}
          />
          <YAxis
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(v: number) => v.toFixed(2)}
          />
          <Tooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius)",
              fontSize: 13,
              color: "var(--color-popover-foreground)",
            }}
            formatter={(value) => [Number(value).toFixed(4), "Cost"]}
          />
          <Area
            type="monotone"
            dataKey="cost"
            stroke="var(--color-primary)"
            fill="url(#usage)"
            strokeWidth={2}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
