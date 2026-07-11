export function formatAmount(amount: string): string {
  const num = parseFloat(amount)
  if (isNaN(num)) return "0.0000"
  return num.toFixed(4)
}

export function formatShortDateTime(timestamp: string): string {
  const d = new Date(timestamp)
  if (isNaN(d.getTime())) return "\u2014"
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function timeAgo(timestamp: string): string {
  const now = Date.now()
  const then = new Date(timestamp).getTime()
  if (isNaN(then)) return "\u2014"

  const diffSec = Math.floor((now - then) / 1000)

  if (diffSec < 60) return "just now"
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`
  return `${Math.floor(diffSec / 86400)}d ago`
}

const statusDotColors: Record<string, string> = {
  active: "bg-green-500",
  completed: "bg-green-500",
  confirmed: "bg-green-500",
  pending: "bg-yellow-500",
  inactive: "bg-gray-400",
  suspended: "bg-red-500",
  failed: "bg-red-500",
  revoked: "bg-red-500",
  reserved: "bg-blue-500",
}

const statusBgColors: Record<string, string> = {
  active: "text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-950",
  completed: "text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-950",
  confirmed: "text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-950",
  pending: "text-yellow-700 bg-yellow-50 dark:text-yellow-300 dark:bg-yellow-950",
  inactive: "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950",
  suspended: "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-950",
  failed: "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-950",
  revoked: "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-950",
  reserved: "text-blue-700 bg-blue-50 dark:text-blue-300 dark:bg-blue-950",
}

export function StatusBadge({ status }: { status: string }) {
  const s = status.toLowerCase()
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold capitalize ${
        statusBgColors[s] || statusBgColors.inactive
      }`}
    >
      <span className={`size-1.5 rounded-full ${statusDotColors[s] || statusDotColors.inactive}`} />
      {status}
    </span>
  )
}

export function StatusDot({ status }: { status: string }) {
  const s = status.toLowerCase()
  return <span className={`inline-block size-2 rounded-full ${statusDotColors[s] || statusDotColors.inactive}`} />
}

export function UsageStatusBadge({ status }: { status: string }) {
  const s = status.toLowerCase()
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold capitalize ${
        statusBgColors[s] ||
        "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950"
      }`}
    >
      <span className={`size-1.5 rounded-full ${statusDotColors[s] || statusDotColors.inactive}`} />
      {status}
    </span>
  )
}

export function formatBytes(bytes?: number | null): string {
  if (bytes == null) return "\u2014"
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

const statusCodeColorMap: Record<string, string> = {
  "2": "text-green-700 bg-green-50 dark:text-green-300 dark:bg-green-950",
  "4": "text-yellow-700 bg-yellow-50 dark:text-yellow-300 dark:bg-yellow-950",
  "5": "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-950",
}

const statusCodeDotMap: Record<string, string> = {
  "2": "bg-green-500",
  "4": "bg-yellow-500",
  "5": "bg-red-500",
}

export function StatusCodeBadge({ code }: { code?: number | null }) {
  if (code == null) {
    return <span className="text-xs text-muted-foreground">\u2014</span>
  }

  const prefix = String(code)[0]
  const color = statusCodeColorMap[prefix] || statusCodeColorMap["5"]
  const dot = statusCodeDotMap[prefix] || statusCodeDotMap["5"]

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 font-mono text-[11px] font-semibold ${color}`}
    >
      <span className={`size-1.5 rounded-full ${dot}`} />
      {code}
    </span>
  )
}
