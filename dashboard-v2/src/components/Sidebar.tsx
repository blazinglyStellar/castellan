"use client"

import { usePathname } from "next/navigation"
import Link from "next/link"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  LayoutDashboard,
  Globe,
  BarChart3,
  Wallet,
  Compass,
  Banknote,
  Key,
  Settings,
  BookOpen,
  ChevronLeft,
  ChevronRight,
} from "lucide-react"
import type { Role } from "@/lib/types"

interface SidebarProps {
  role: Role
  collapsed: boolean
  onToggle: () => void
}

const providerLinks = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard },
  { href: "/dashboard/providers", label: "My APIs", icon: Globe },
  { href: "/dashboard/analytics", label: "Analytics", icon: BarChart3 },
  { href: "/dashboard/settlements", label: "Settlements", icon: Wallet },
]

const consumerLinks = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard },
  { href: "/dashboard/discover", label: "Discover", icon: Compass },
  { href: "/dashboard/deposit", label: "Deposit", icon: Banknote },
  { href: "/dashboard/usage", label: "Usage", icon: BarChart3 },
  { href: "/dashboard/api-keys", label: "API Keys", icon: Key },
]

const sharedLinks = [
  { href: "/dashboard/settings", label: "Settings", icon: Settings },
  { href: "/docs", label: "API Docs", icon: BookOpen },
]

export function Sidebar({ role, collapsed, onToggle }: SidebarProps) {
  const pathname = usePathname()

  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard"
    return pathname.startsWith(href)
  }

  const renderLink = (link: { href: string; label: string; icon: React.ComponentType<{ className?: string }> }) => (
    <Link
      key={link.href}
      href={link.href}
      className={cn(
        "flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-sm font-medium transition-colors",
        isActive(link.href)
          ? "bg-accent text-accent-foreground"
          : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
        collapsed && "justify-center px-1.5",
      )}
    >
      <link.icon className="size-4 shrink-0" />
      {!collapsed && <span>{link.label}</span>}
    </Link>
  )

  return (
    <aside
      className={cn(
        "flex flex-col border-r bg-sidebar transition-all duration-200",
        collapsed ? "w-14" : "w-56",
      )}
    >
      <div className={cn("flex items-center border-b px-3 py-3", collapsed && "justify-center px-0")}>
        {!collapsed && (
          <span className="text-sm font-semibold tracking-tight">
            Castellan
          </span>
        )}
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={onToggle}
          className={cn("ml-auto text-muted-foreground", collapsed && "ml-0")}
        >
          {collapsed ? <ChevronRight className="size-3.5" /> : <ChevronLeft className="size-3.5" />}
        </Button>
      </div>

      <nav className="flex-1 space-y-4 overflow-y-auto p-2">
        {(role === "provider" || role === "both") && (
          <div className="space-y-0.5">
            {!collapsed && (
              <p className="px-2.5 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                Provider
              </p>
            )}
            {providerLinks.map(renderLink)}
          </div>
        )}

        {(role === "consumer" || role === "both") && (
          <div className="space-y-0.5">
            {!collapsed && (
              <p className="px-2.5 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                Consumer
              </p>
            )}
            {consumerLinks.map(renderLink)}
          </div>
        )}

        <Separator />

        <div className="space-y-0.5">
          {!collapsed && (
            <p className="px-2.5 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
              General
            </p>
          )}
          {sharedLinks.map(renderLink)}
        </div>
      </nav>
    </aside>
  )
}
