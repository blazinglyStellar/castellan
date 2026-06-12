"use client"

import { useState } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import {
  LayoutDashboard,
  Cable,
  BarChart3,
  Banknote,
  Settings,
  Landmark,
  Activity,
  Key,
  ChevronLeft,
  ChevronRight,
  LogOut,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Separator } from "@/components/ui/separator"

const navItems = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard },
  { href: "/dashboard/providers", label: "My APIs", icon: Cable },
  { href: "/dashboard/analytics", label: "Analytics", icon: BarChart3 },
  { href: "/dashboard/settlements", label: "Settlements", icon: Banknote },
  { href: "/dashboard/deposit", label: "Deposit", icon: Landmark },
  { href: "/dashboard/usage", label: "Usage", icon: Activity },
  { href: "/dashboard/api-keys", label: "API Keys", icon: Key },
  { href: "/dashboard/settings", label: "Settings", icon: Settings },
]

interface DashboardShellProps {
  children: React.ReactNode
  role?: "provider" | "consumer" | "both"
}

export function DashboardShell({ children, role = "both" }: DashboardShellProps) {
  const pathname = usePathname()
  const [collapsed, setCollapsed] = useState(false)
  const isProvider = role === "provider" || role === "both"
  const isConsumer = role === "consumer" || role === "both"
  const filteredNav = navItems.filter((item) => {
    if (item.href === "/dashboard/providers" || item.href === "/dashboard/analytics" || item.href === "/dashboard/settlements") return isProvider
    if (item.href === "/dashboard/usage" || item.href === "/dashboard/deposit") return isConsumer || isProvider
    return true
  })

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <aside
        className={cn(
          "flex flex-col border-r bg-card transition-all duration-200",
          collapsed ? "w-16" : "w-60"
        )}
      >
        <div className="flex h-14 items-center gap-2 border-b px-4">
          {!collapsed && (
            <Link href="/dashboard" className="text-lg font-bold tracking-tight">
              <span className="text-primary">Flow</span>Gate
            </Link>
          )}
          <Button
            variant="ghost"
            size="icon"
            className={cn("ml-auto h-7 w-7", collapsed && "mx-auto")}
            onClick={() => setCollapsed(!collapsed)}
          >
            {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
          </Button>
        </div>
        <nav className="flex-1 space-y-1 p-2">
          {filteredNav.map((item) => {
            const Icon = item.icon
            const active = pathname === item.href || (item.href !== "/dashboard" && pathname.startsWith(item.href))
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  collapsed && "justify-center px-2"
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </Link>
            )
          })}
        </nav>
        <Separator />
        <div className="p-3">
          <div className={cn("flex items-center gap-3", collapsed && "justify-center")}>
            <Avatar className="h-8 w-8">
              <AvatarFallback>CA</AvatarFallback>
            </Avatar>
            {!collapsed && (
              <div className="flex-1 truncate">
                <p className="text-sm font-medium truncate">user@castellan.io</p>
                <p className="text-xs text-muted-foreground capitalize">{role}</p>
              </div>
            )}
          </div>
          <Button variant="ghost" size="sm" className={cn("mt-2 w-full justify-start gap-2 text-muted-foreground", collapsed && "justify-center px-0")}>
            <LogOut className="h-4 w-4" />
            {!collapsed && "Sign Out"}
          </Button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-7xl p-6">{children}</div>
      </main>
    </div>
  )
}
