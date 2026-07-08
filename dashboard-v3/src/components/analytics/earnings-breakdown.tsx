"use client";

import { formatAmount } from "@/lib/format";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface EarningsBreakdownProps {
  data: { route: string; total: string }[];
}

export function EarningsBreakdown({ data }: EarningsBreakdownProps) {
  if (data.length === 0) {
    return (
      <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
        No breakdown data available
      </div>
    );
  }

  const total = data.reduce((sum, e) => sum + parseFloat(e.total), 0);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Endpoint</TableHead>
          <TableHead className="text-right">Earnings</TableHead>
          <TableHead className="text-right">Share</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.map((entry) => {
          const share = total > 0 ? ((parseFloat(entry.total) / total) * 100).toFixed(1) : "0";
          return (
            <TableRow key={entry.route}>
              <TableCell className="font-mono text-xs">{entry.route}</TableCell>
              <TableCell className="text-right font-medium">
                {formatAmount(entry.total)}
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


