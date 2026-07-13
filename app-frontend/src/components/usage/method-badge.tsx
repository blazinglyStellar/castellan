"use client"

import { cn } from "@/lib/utils"

const colorMap: Record<string, string> = {
  GET: "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950",
  POST: "text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-950",
  PUT: "text-orange-600 bg-orange-100 dark:text-orange-400 dark:bg-orange-950",
  PATCH: "text-purple-600 bg-purple-100 dark:text-purple-400 dark:bg-purple-950",
  DELETE: "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950",
}

export function MethodBadge({ method }: { method: string }) {
  return (
    <span
      className={cn(
        "inline-block rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase",
        colorMap[method.toUpperCase()] ||
          "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-800",
      )}
    >
      {method}
    </span>
  )
}
