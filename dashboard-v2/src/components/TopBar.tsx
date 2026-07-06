"use client"

import { SearchCommand } from "@/components/SearchCommand"
import { NotificationBell } from "@/components/NotificationBell"
import { BalanceBadge } from "@/components/BalanceBadge"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Settings, LogOut, Moon, Sun } from "lucide-react"
import { useState, useEffect } from "react"
import type { Theme } from "@/lib/types"

interface TopBarProps {
  balance: number
}

export function TopBar({ balance }: TopBarProps) {
  const [theme, setTheme] = useState<Theme>("dark")

  useEffect(() => {
    document.documentElement.classList.remove("dark", "light")
    document.documentElement.classList.add(theme)
  }, [theme])

  const toggleTheme = () => {
    setTheme((prev) => (prev === "dark" ? "light" : "dark"))
  }

  return (
    <header className="flex h-12 items-center gap-3 border-b px-4">
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold tracking-tight md:hidden">Castellan</span>
      </div>

      <div className="hidden sm:block">
        <SearchCommand />
      </div>

      <div className="ml-auto flex items-center gap-1.5">
        <BalanceBadge balance={balance} />

        <Button variant="ghost" size="icon-sm" onClick={toggleTheme} className="text-muted-foreground">
          {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </Button>

        <NotificationBell />

        <DropdownMenu>
          <DropdownMenuTrigger
            render={<Button variant="ghost" size="icon-sm" className="rounded-full" />}
          >
            <Avatar size="sm">
              <AvatarFallback>U</AvatarFallback>
            </Avatar>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel>user@castellan.io</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <Settings className="size-4" />
              Settings
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive">
              <LogOut className="size-4" />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
