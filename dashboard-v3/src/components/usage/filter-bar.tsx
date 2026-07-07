"use client";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DateRangePicker } from "@/components/usage/date-range-picker";

interface FilterBarProps {
  role: "provider" | "consumer";
  onRoleChange: (role: "provider" | "consumer") => void;
  startDate: string;
  endDate: string;
  onStartDateChange: (d: string) => void;
  onEndDateChange: (d: string) => void;
  statusCode: string;
  onStatusCodeChange: (code: string) => void;
}

export function FilterBar({
  role,
  onRoleChange,
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
        <span className="text-xs text-muted-foreground">View as</span>
        <div className="flex gap-0">
          <Button
            variant={role === "consumer" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleChange("consumer")}
            className="rounded-r-none"
          >
            Consumer
          </Button>
          <Button
            variant={role === "provider" ? "default" : "outline"}
            size="sm"
            onClick={() => onRoleChange("provider")}
            className="rounded-l-none"
          >
            Provider
          </Button>
        </div>
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
