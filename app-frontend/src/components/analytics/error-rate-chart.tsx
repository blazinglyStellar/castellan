"use client"

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import type { UsageEvent } from "@/lib/api/types"

interface ErrorRateChartProps {
  events: UsageEvent[]
}

interface RouteErrorRate {
  route: string
  rate: number
  errors: number
  total: number
}

const BAR_COLOR = "var(--color-destructive)"

export function ErrorRateChart({ events }: ErrorRateChartProps) {
  const withStatus = events.filter((e) => e.status_code != null)
  if (withStatus.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No error data for the selected period
      </div>
    )
  }

  const routeStats = new Map<string, { errors: number; total: number }>()

  for (const ev of withStatus) {
    const s = routeStats.get(ev.route) || { errors: 0, total: 0 }
    s.total += 1
    if (ev.status_code! >= 400) s.errors += 1
    routeStats.set(ev.route, s)
  }

  const totalErrors = [...routeStats.values()].reduce((s, r) => s + r.errors, 0)
  if (totalErrors === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        <span className="flex items-center gap-2">
          <span className="size-2 rounded-full bg-green-500" />
          No errors recorded in this period
        </span>
      </div>
    )
  }

  const sorted = [...routeStats.entries()]
    .map(([route, s]) => ({
      route,
      errors: s.errors,
      total: s.total,
      rate: s.total > 0 ? (s.errors / s.total) * 100 : 0,
    }))
    .sort((a, b) => b.rate - a.rate)

  const top5 = sorted.slice(0, 5)
  const others = sorted.slice(5)

  const chartData: RouteErrorRate[] =
    others.length > 0
      ? [
          ...top5,
          {
            route: "Others",
            rate:
              (others.reduce((s, r) => s + r.errors, 0) /
                others.reduce((s, r) => s + r.total, 0)) *
              100,
            errors: others.reduce((s, r) => s + r.errors, 0),
            total: others.reduce((s, r) => s + r.total, 0),
          },
        ]
      : top5

  const maxRate = Math.max(...chartData.map((d) => d.rate), 1)

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={chartData}
          layout="vertical"
          margin={{ left: 100, right: 16 }}
          barCategoryGap={0}
        >
          <CartesianGrid
            strokeDasharray="3 3"
            className="stroke-border"
            horizontal={false}
          />
          <XAxis
            type="number"
            domain={[0, 100]}
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(v: number) => `${v}%`}
          />
          <YAxis
            type="category"
            dataKey="route"
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            width={90}
          />
          <Tooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius)",
              fontSize: 13,
              color: "var(--color-popover-foreground)",
            }}
            formatter={(value) => {
              if (typeof value === "number")
                return [`${value.toFixed(1)}%`, "Error Rate"]
              return [String(value)]
            }}
            labelFormatter={() => ""}
          />
          <Bar dataKey="rate" radius={[0, 4, 4, 0]}>
            {chartData.map((entry) => (
              <Cell
                key={entry.route}
                fill={BAR_COLOR}
                fillOpacity={0.25 + 0.75 * (entry.rate / maxRate)}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
