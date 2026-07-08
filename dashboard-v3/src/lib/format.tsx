export function formatAmount(amount: string): string {
  const num = parseFloat(amount);
  if (isNaN(num)) return "0.0000";
  return num.toFixed(4);
}

export function timeAgo(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  if (isNaN(then)) return "\u2014";

  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}

export function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    active: "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950",
    completed: "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950",
    confirmed: "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950",
    pending: "text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-950",
    inactive: "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950",
    suspended: "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950",
    failed: "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950",
    revoked: "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950",
  };

  return (
    <span
      className={`inline-block rounded px-1.5 py-0.5 font-mono text-[11px] capitalize ${
        colors[status] || colors.inactive
      }`}
    >
      {status}
    </span>
  );
}

export function StatusCodeBadge({ code }: { code?: number | null }) {
  if (code == null) {
    return <span className="text-xs text-muted-foreground">—</span>;
  }

  let color: string;
  if (code >= 200 && code < 300) {
    color = "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950";
  } else if (code >= 400 && code < 500) {
    color = "text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-950";
  } else {
    color = "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-950";
  }

  return (
    <span
      className={`inline-block rounded px-1.5 py-0.5 font-mono text-[11px] ${color}`}
    >
      {code}
    </span>
  );
}
