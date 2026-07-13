"use client"

import type { UsageEvent } from "@/lib/api/types"

interface RouteStatus {
  route: string
  total: number
  s2xx: number
  s4xx: number
  s5xx: number
}

export function StatusDistribution({ events }: { events: UsageEvent[] }) {
  const withStatus = events.filter((e) => e.status_code != null)
  if (withStatus.length === 0) {
    return (
      <div className="flex h-24 items-center justify-center text-sm text-muted-foreground">
        No status data for the selected period
      </div>
    )
  }

  const byRoute = new Map<string, RouteStatus>()

  for (const ev of withStatus) {
    const existing = byRoute.get(ev.route) || {
      route: ev.route,
      total: 0,
      s2xx: 0,
      s4xx: 0,
      s5xx: 0,
    }
    existing.total += 1
    const code = ev.status_code!
    if (code < 300) existing.s2xx += 1
    else if (code < 500) existing.s4xx += 1
    else existing.s5xx += 1
    byRoute.set(ev.route, existing)
  }

  const rows = [...byRoute.values()].sort((a, b) => b.total - a.total)

  return (
    <div className="overflow-x-auto">
      <table className="w-full border-separate border-spacing-y-1 px-6 pb-2 text-left">
        <thead>
          <tr className="text-[11px] font-bold uppercase tracking-widest text-muted-foreground">
            <th className="px-4 py-2">Endpoint</th>
            <th className="px-4 py-2 text-right">2xx / 4xx / 5xx</th>
          </tr>
        </thead>
        <tbody className="text-sm">
          {rows.map((row) => (
            <tr
              key={row.route}
              className="rounded-lg bg-muted/30 transition-colors hover:bg-muted/60"
            >
              <td className="px-4 py-3 font-mono text-xs">
                {row.route}
              </td>
              <td className="px-4 py-3 text-right text-xs tabular-nums">
                <span className="text-green-600 dark:text-green-400">
                  {row.s2xx}
                </span>
                {" / "}
                <span className="text-yellow-600 dark:text-yellow-400">
                  {row.s4xx}
                </span>
                {" / "}
                <span className="text-red-600 dark:text-red-400">
                  {row.s5xx}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
