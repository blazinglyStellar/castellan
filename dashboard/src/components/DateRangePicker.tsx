"use client"

import { useState } from "react"
import { Calendar } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"

interface DateRangePickerProps {
  className?: string
}

export function DateRangePicker({ className }: DateRangePickerProps) {
  const [date, setDate] = useState("Last 7 days")
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className={cn("gap-2", className)}>
          <Calendar className="h-4 w-4" />
          {date}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-48 p-2" align="end">
        <div className="flex flex-col gap-1">
          {["Last 7 days", "Last 30 days", "Last 90 days", "This year"].map((d) => (
            <Button key={d} variant={date === d ? "secondary" : "ghost"} size="sm" className="justify-start" onClick={() => setDate(d)}>
              {d}
            </Button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
