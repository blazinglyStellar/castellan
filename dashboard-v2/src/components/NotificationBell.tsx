"use client"

import { Bell } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

export function NotificationBell() {
  return (
    <Popover>
      <PopoverTrigger
        render={<Button variant="ghost" size="icon-sm" className="text-muted-foreground" />}
      >
        <Bell className="size-4" />
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-0">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-xs font-medium">Notifications</span>
        </div>
        <div className="flex flex-col items-center justify-center gap-1 py-8 text-center">
          <Bell className="size-8 text-muted-foreground/40" />
          <p className="text-xs text-muted-foreground">No notifications yet</p>
        </div>
      </PopoverContent>
    </Popover>
  )
}
