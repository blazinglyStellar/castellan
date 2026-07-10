"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DateRangePicker } from "@/components/usage/date-range-picker";

interface FilterOption {
  value: string;
  label: string;
}

interface FilterBarProps {
  providers: FilterOption[];
  endpoints: FilterOption[];
  selectedProvider: string;
  selectedEndpoint: string;
  onProviderChange: (v: string) => void;
  onEndpointChange: (v: string) => void;
  startDate: string;
  endDate: string;
  onStartDateChange: (d: string) => void;
  onEndDateChange: (d: string) => void;
  statusCode: string;
  onStatusCodeChange: (code: string) => void;
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
  return (
    <div className="flex flex-wrap items-end gap-4">
      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">Provider</span>
        <Select value={selectedProvider} onValueChange={onProviderChange}>
          <SelectTrigger className="h-8 w-44">
            <SelectValue placeholder="All providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value=" ">All providers</SelectItem>
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
          onValueChange={onEndpointChange}
          disabled={endpoints.length === 0}
        >
          <SelectTrigger className="h-8 w-48">
            <SelectValue placeholder="All endpoints" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value=" ">All endpoints</SelectItem>
            {endpoints.map((e) => (
              <SelectItem key={e.value} value={e.value}>
                {e.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <DateRangePicker
        startDate={startDate}
        endDate={endDate}
        onStartDateChange={onStartDateChange}
        onEndDateChange={onEndDateChange}
      />

      <div className="space-y-1">
        <span className="text-xs text-muted-foreground">Status</span>
        <Select value={statusCode} onValueChange={onStatusCodeChange}>
          <SelectTrigger className="h-8 w-28">
            <SelectValue placeholder="All" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value=" ">All</SelectItem>
            <SelectItem value="200">2xx</SelectItem>
            <SelectItem value="400">4xx</SelectItem>
            <SelectItem value="500">5xx</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
