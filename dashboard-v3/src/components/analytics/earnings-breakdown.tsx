"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

interface EarningsBreakdownProps {
  data: { name: string; total: string }[];
}

function opacity(index: number, count: number): number {
  if (count <= 1) return 1;
  return 1 - (index / (count - 1)) * 0.7;
}

export function EarningsBreakdown({ data }: EarningsBreakdownProps) {
  if (data.length === 0) {
    return (
      <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
        No provider earnings data
      </div>
    );
  }

  const sorted = [...data]
    .map((d) => ({ name: d.name, total: parseFloat(d.total) }))
    .sort((a, b) => b.total - a.total);

  const color = "hsl(var(--primary))";

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={sorted} layout="vertical" margin={{ left: 0, right: 8, top: 4, bottom: 4 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-border" horizontal={false} />
          <XAxis
            type="number"
            className="text-xs text-muted-foreground"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(v: number) => v.toFixed(2)}
          />
          <YAxis
            type="category"
            dataKey="name"
            className="text-xs"
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11 }}
            width={140}
          />
          <Tooltip
            cursor={{ fill: "hsl(var(--muted))" }}
            contentStyle={{
              background: "hsl(var(--popover))",
              border: "1px solid hsl(var(--border))",
              borderRadius: "var(--radius)",
              fontSize: 13,
            }}
            formatter={(value: number) => [value.toFixed(4), "Earnings"]}
          />
          <Bar dataKey="total" radius={[0, 4, 4, 0]}>
            {sorted.map((entry, index) => (
              <Cell key={entry.name} fill={color} fillOpacity={opacity(index, sorted.length)} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
