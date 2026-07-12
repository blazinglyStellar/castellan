"use client"

import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

interface UsageBarChartProps {
  data: { date: string; amount: string }[]
  isProvider?: boolean
}

export function UsageBarChart({ data, isProvider }: UsageBarChartProps) {
  if (data.length === 0) {
    return isProvider ? (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No earnings data for the selected period
      </div>
    ) : (
      <div className="flex h-48 flex-col items-center justify-center gap-3 text-sm text-muted-foreground">
        <p>Earnings tracking is available for registered providers.</p>
      </div>
    )
  }

  const chartData = data
    .map((d) => ({ date: d.date, amount: parseFloat(d.amount) }))
    .sort((a, b) => a.date.localeCompare(b.date))

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
          <XAxis
            dataKey="date"
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(d: string) => {
              const dt = new Date(d + "T00:00:00Z")
              return dt.toLocaleDateString("en-US", {
                month: "short",
                day: "numeric",
              })
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
            }}
            formatter={(value) => [Number(value).toFixed(4), "Earnings"]}
          />
          <Bar
            dataKey="amount"
            fill="var(--color-primary)"
            radius={[4, 4, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
