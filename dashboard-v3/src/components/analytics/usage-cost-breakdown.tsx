"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface CostRow {
  route: string;
  cost: number;
  calls: number;
}

interface UsageCostBreakdownProps {
  events: { request_cost: string; route: string }[];
}

export function UsageCostBreakdown({ events }: UsageCostBreakdownProps) {
  if (events.length === 0) {
    return (
      <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
        No cost data for the selected period
      </div>
    );
  }

  const byRoute = new Map<string, { cost: number; calls: number }>();

  for (const ev of events) {
    const existing = byRoute.get(ev.route) || { cost: 0, calls: 0 };
    existing.cost += parseFloat(ev.request_cost);
    existing.calls += 1;
    byRoute.set(ev.route, existing);
  }

  const rows: CostRow[] = Array.from(byRoute.entries())
    .map(([route, v]) => ({ route, cost: v.cost, calls: v.calls }))
    .sort((a, b) => b.cost - a.cost);

  const totalCost = rows.reduce((s, r) => s + r.cost, 0);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Endpoint</TableHead>
          <TableHead className="text-right">Total Cost</TableHead>
          <TableHead className="text-right">Calls</TableHead>
          <TableHead className="text-right">Share</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => {
          const share = totalCost > 0 ? ((row.cost / totalCost) * 100).toFixed(1) : "0";
          return (
            <TableRow key={row.route}>
              <TableCell className="font-mono text-xs">{row.route}</TableCell>
              <TableCell className="text-right font-medium">
                {row.cost.toFixed(4)}
              </TableCell>
              <TableCell className="text-right text-xs text-muted-foreground">
                {row.calls}
              </TableCell>
              <TableCell className="text-right text-xs text-muted-foreground">
                {share}%
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
