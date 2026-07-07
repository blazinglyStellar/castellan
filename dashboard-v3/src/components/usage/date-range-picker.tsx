"use client";

import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";

interface DateRangePickerProps {
  startDate: string;
  endDate: string;
  onStartDateChange: (date: string) => void;
  onEndDateChange: (date: string) => void;
}

export function DateRangePicker({
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
}: DateRangePickerProps) {
  return (
    <div className="flex items-end gap-2">
      <div className="space-y-1">
        <Label htmlFor="start-date" className="text-xs text-muted-foreground">
          From
        </Label>
        <Input
          id="start-date"
          type="date"
          value={startDate}
          onChange={(e) => onStartDateChange(e.target.value)}
          className="h-8 w-40"
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="end-date" className="text-xs text-muted-foreground">
          To
        </Label>
        <Input
          id="end-date"
          type="date"
          value={endDate}
          onChange={(e) => onEndDateChange(e.target.value)}
          className="h-8 w-40"
        />
      </div>
    </div>
  );
}
