"use client"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface FilterOption {
  value: string
  label: string
}

interface FilterBarProps {
  providers: FilterOption[]
  endpoints: FilterOption[]
  selectedProvider: string
  selectedEndpoint: string
  onProviderChange: (v: string) => void
  onEndpointChange: (v: string) => void
  startDate: string
  endDate: string
  onStartDateChange: (d: string) => void
  onEndDateChange: (d: string) => void
  statusCode: string
  onStatusCodeChange: (code: string) => void
}

export function FilterBar({
  providers,
  endpoints,
  selectedProvider,
  selectedEndpoint,
  onProviderChange,
  onEndpointChange,
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  statusCode,
  onStatusCodeChange,
}: FilterBarProps) {
  const handleProviderChange = (v: string | null) => onProviderChange(v ?? "")
  const handleEndpointChange = (v: string | null) => onEndpointChange(v ?? "")
  const handleStatusCodeChange = (v: string | null) =>
    onStatusCodeChange(v ?? "")

  return (
    <div className="flex flex-wrap items-end gap-4">
      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">Provider</span>
          <Select
            value={selectedProvider}
            onValueChange={handleProviderChange}
          >
            <SelectTrigger className="h-8 w-44 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All providers</SelectItem>
            {providers.map((p) => (
              <SelectItem key={p.value} value={p.value}>
                {p.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">Endpoint</span>
          <Select
            value={selectedEndpoint}
            onValueChange={handleEndpointChange}
          >
            <SelectTrigger className="h-8 w-48 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All endpoints" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All endpoints</SelectItem>
            {endpoints.map((e) => (
              <SelectItem key={e.value} value={e.value}>
                {e.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="space-y-1">
          <span className="text-xs text-muted-foreground">From</span>
          <input
            type="date"
            value={startDate}
            onChange={(e) => onStartDateChange(e.target.value)}
            className="h-8 rounded-lg border border-input bg-background px-2.5 text-sm text-foreground transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
          />
        </div>
        <div className="space-y-1">
          <span className="text-xs text-muted-foreground">To</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => onEndDateChange(e.target.value)}
            className="h-8 rounded-lg border border-input bg-background px-2.5 text-sm text-foreground transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
          />
        </div>
      </div>

      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">Status</span>
        <Select value={statusCode} onValueChange={handleStatusCodeChange}>
          <SelectTrigger className="h-8 w-28 bg-background data-placeholder:text-foreground">
            <SelectValue placeholder="All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All</SelectItem>
            <SelectItem value="200">2xx</SelectItem>
            <SelectItem value="400">4xx</SelectItem>
            <SelectItem value="500">5xx</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
