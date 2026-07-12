"use client"

import { useMemo } from "react"
import { Label, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts"

import type { EndpointEarning } from "@/lib/api/types"
import { formatAmount } from "@/lib/format"

const COLORS = [
  "var(--color-chart-1)",
  "var(--color-chart-2)",
  "var(--color-chart-3)",
  "var(--color-chart-4)",
  "var(--color-chart-5)",
  "var(--color-muted-foreground)",
]

const MIN_SHARE_PCT = 5

export function RevenueDistributionChart({ data }: { data: EndpointEarning[] }) {
  const { chartData, totalRevenue } = useMemo(() => {
    const raw = [...data]
      .map((d) => ({ route: d.route, total: parseFloat(d.total) }))
      .sort((a, b) => b.total - a.total)

    const grandTotal = raw.reduce((s, r) => s + r.total, 0)
    if (grandTotal === 0) {
      return { chartData: [], totalRevenue: 0 }
    }

    const keep: typeof raw = []
    let othersTotal = 0
    for (const r of raw) {
      if ((r.total / grandTotal) * 100 < MIN_SHARE_PCT) {
        othersTotal += r.total
      } else {
        keep.push(r)
      }
    }

    const grouped = othersTotal > 0 ? [...keep, { route: "Others", total: othersTotal }] : keep

    const typed = grouped.map((r, i) => ({
      route: r.route,
      total: r.total,
      fill: COLORS[i % COLORS.length],
    }))

    return { chartData: typed, totalRevenue: grandTotal }
  }, [data])

  if (data.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No revenue data for the selected period
      </div>
    )
  }

  return (
    <div className="flex items-center gap-6">
      <div className="mx-auto aspect-square max-h-[220px] min-w-0 shrink-0 basis-[220px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Tooltip
              contentStyle={{
                background: "var(--color-popover)",
                border: "1px solid var(--color-border)",
                borderRadius: "var(--radius)",
                fontSize: 13,
              }}
              formatter={(value) => {
                if (typeof value === "number") return [`${formatAmount(value.toFixed(7))} XLM`]
                return [String(value)]
              }}
            />
            <Pie
              data={chartData}
              dataKey="total"
              nameKey="route"
              innerRadius={55}
              strokeWidth={4}
            >
              <Label
                content={({ viewBox }) => {
                  if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                    return (
                      <g transform={`translate(${viewBox.cx}, ${viewBox.cy})`}>
                        <text
                          textAnchor="middle"
                          dominantBaseline="middle"
                          className="fill-foreground text-2xl font-bold"
                        >
                          {totalRevenue.toFixed(2)}
                        </text>
                      </g>
                    )
                  }
                }}
              />
            </Pie>
          </PieChart>
        </ResponsiveContainer>
      </div>

      <div className="flex min-w-0 flex-col gap-2 text-sm">
        {chartData.map((d) => {
          const share = totalRevenue > 0 ? ((d.total / totalRevenue) * 100).toFixed(1) : "0"
          return (
            <div key={d.route} className="flex items-center gap-2 whitespace-nowrap">
              <span
                className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: d.fill }}
              />
              <span className="truncate text-muted-foreground max-w-40">{d.route}</span>
              <span className="font-medium tabular-nums">
                {formatAmount(d.total.toFixed(7))} XLM
              </span>
              <span className="text-xs text-muted-foreground">({share}%)</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
