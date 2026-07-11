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

interface LatencyChartProps {
  events: UsageEvent[]
}

interface RouteLatency {
  route: string
  avg: number
  calls: number
}

const BAR_COLOR = "hsl(var(--chart-1))"

export function LatencyChart({ events }: LatencyChartProps) {
  const withLatency = events.filter((e) => e.latency_ms != null)
  if (withLatency.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No latency data for the selected period
      </div>
    )
  }

  const routeStats = new Map<string, { sum: number; calls: number }>()

  for (const ev of withLatency) {
    const s = routeStats.get(ev.route) || { sum: 0, calls: 0 }
    s.sum += ev.latency_ms!
    s.calls += 1
    routeStats.set(ev.route, s)
  }

  const sorted = [...routeStats.entries()]
    .map(([route, s]) => ({
      route,
      avg: s.sum / s.calls,
      calls: s.calls,
    }))
    .sort((a, b) => b.avg - a.avg)

  const top5 = sorted.slice(0, 5)
  const others = sorted.slice(5)

  const chartData: RouteLatency[] =
    others.length > 0
      ? [
          ...top5,
          {
            route: "Others",
            avg:
              others.reduce((s, r) => s + r.avg * r.calls, 0) /
              others.reduce((s, r) => s + r.calls, 0),
            calls: others.reduce((s, r) => s + r.calls, 0),
          },
        ]
      : top5

  const maxAvg = Math.max(...chartData.map((d) => d.avg), 1)

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
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(v: number) => `${Math.round(v)}ms`}
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
              background: "hsl(var(--popover))",
              border: "1px solid hsl(var(--border))",
              borderRadius: "var(--radius)",
              fontSize: 13,
            }}
            formatter={(value) => {
              if (typeof value === "number")
                return [`${Math.round(value)}ms`, "Avg Latency"]
              return [String(value)]
            }}
            labelFormatter={() => ""}
          />
          <Bar dataKey="avg" radius={[0, 4, 4, 0]}>
            {chartData.map((entry) => (
              <Cell
                key={entry.route}
                fill={BAR_COLOR}
                fillOpacity={0.25 + 0.75 * (entry.avg / maxAvg)}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
