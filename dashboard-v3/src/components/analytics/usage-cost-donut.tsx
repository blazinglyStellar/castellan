"use client";

import { useMemo } from "react";
import Image from "next/image";
import { Label, Pie, PieChart } from "recharts";

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

const LOGO = (
  <Image
    src="/stellar-xlm-logo.svg"
    alt="XLM"
    width={14}
    height={12}
    className="inline-block align-middle"
  />
);

const COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
  "hsl(var(--muted-foreground))",
];

const MIN_SHARE_PCT = 5;

interface UsageCostDonutProps {
  events: { request_cost: string; route: string }[];
}

export function UsageCostDonut({ events }: UsageCostDonutProps) {
  const byRoute = useMemo(() => {
    const map = new Map<string, number>();
    for (const ev of events) {
      map.set(ev.route, (map.get(ev.route) || 0) + parseFloat(ev.request_cost));
    }
    return map;
  }, [events]);

  const { chartData, totalCost, chartConfig } = useMemo(() => {
    const raw = [...byRoute.entries()]
      .map(([route, cost]) => ({ route, cost }))
      .sort((a, b) => b.cost - a.cost);

    const grandTotal = raw.reduce((s, r) => s + r.cost, 0);
    if (grandTotal === 0) {
      return { chartData: [], totalCost: 0, chartConfig: {} as ChartConfig };
    }

    const keep: typeof raw = [];
    let othersCost = 0;
    for (const r of raw) {
      if ((r.cost / grandTotal) * 100 < MIN_SHARE_PCT) {
        othersCost += r.cost;
      } else {
        keep.push(r);
      }
    }

    const grouped = othersCost > 0 ? [...keep, { route: "Others", cost: othersCost }] : keep;

    const data = grouped.map((r, i) => ({
      route: r.route,
      cost: r.cost,
      fill: COLORS[i % COLORS.length],
    }));

    const config: ChartConfig = {};
    for (const r of grouped) {
      const i = grouped.indexOf(r);
      config[r.route] = {
        label: r.route,
        color: COLORS[i % COLORS.length],
      };
    }

    return { chartData: data, totalCost: grandTotal, chartConfig: config };
  }, [byRoute]);

  if (events.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
        No cost data for the selected period
      </div>
    );
  }

  return (
    <div className="flex items-center gap-6">
      <ChartContainer config={chartConfig} className="mx-auto aspect-square max-h-[220px] min-w-0 shrink-0 basis-[220px]">
        <PieChart>
          <ChartTooltip
            cursor={false}
            content={
              <ChartTooltipContent
                hideLabel
                formatter={(value) => {
                  if (typeof value === "number") return `${value.toFixed(4)} ${LOGO}`;
                  return String(value);
                }}
              />
            }
          />
          <Pie
            data={chartData}
            dataKey="cost"
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
                        {totalCost.toFixed(2)}
                      </text>
                      <image
                        href="/stellar-xlm-logo.svg"
                        x={-14}
                        y={16}
                        width={28}
                        height={14}
                      />
                    </g>
                  );
                }
              }}
            />
          </Pie>
        </PieChart>
      </ChartContainer>

      <div className="flex min-w-0 flex-col gap-2 text-sm">
        {chartData.map((d) => {
          const share = totalCost > 0 ? ((d.cost / totalCost) * 100).toFixed(1) : "0";
          return (
            <div key={d.route} className="flex items-center gap-2 whitespace-nowrap">
              <span
                className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: d.fill }}
              />
              <span className="truncate text-muted-foreground max-w-40">{d.route}</span>
              <span className="font-medium tabular-nums">
                {d.cost.toFixed(2)} {LOGO}
              </span>
              <span className="text-xs text-muted-foreground">({share}%)</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
