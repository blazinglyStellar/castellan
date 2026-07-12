"use client"

import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  BarChart3,
  Activity,
  Wallet,
  PlusCircle,
  Key,
  Store,
  HandshakeIcon,
  Compass,
  Settings,
  Shield,
  ChevronLeft,
  Moon,
  Sun,
  Menu,
} from "lucide-react"
import { useTheme } from "next-themes"
import { useSidebarStore } from "@/stores/sidebar"
import { cn } from "@/lib/utils"
import { useEffect, useState } from "react"
import {
  Sheet,
  SheetContent,
  SheetTrigger,
} from "@/components/ui/sheet"

interface NavItem {
  label: string
  href: string
  icon: React.ElementType
}

interface NavGroup {
  label?: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    items: [{ label: "Overview", href: "/overview", icon: LayoutDashboard }],
  },
  {
    label: "Monitoring",
    items: [
      { label: "Analytics", href: "/analytics", icon: BarChart3 },
      { label: "Usage", href: "/usage", icon: Activity },
      { label: "Ledger", href: "/ledger", icon: Wallet },
    ],
  },
  {
    label: "Payments",
    items: [
      { label: "Deposit", href: "/deposits", icon: PlusCircle },
      { label: "API Keys", href: "/api-keys", icon: Key },
    ],
  },
  {
    label: "Management",
    items: [
      { label: "Providers", href: "/providers", icon: Store },
      { label: "Settlements", href: "/providers/settlements", icon: HandshakeIcon },
    ],
  },
  {
    label: "Explore",
    items: [{ label: "Discover", href: "/discover", icon: Compass }],
  },
]

function NavItemLink({
  item,
  collapsed,
  isActive,
}: {
  item: NavItem
  collapsed: boolean
  isActive: boolean
}) {
  const Icon = item.icon
  return (
    <a
      href={item.href}
      className={cn(
        "group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200",
        isActive
          ? "bg-sidebar-accent text-sidebar-accent-foreground"
          : "text-sidebar-muted-foreground hover:bg-muted hover:text-foreground",
        collapsed && "justify-center px-2",
      )}
      title={collapsed ? item.label : undefined}
    >
      <Icon className="size-[20px] shrink-0" />
      {!collapsed && <span>{item.label}</span>}
    </a>
  )
}

function DesktopSidebar({ collapsed, toggle }: { collapsed: boolean; toggle: () => void }) {
  const pathname = usePathname()
  const { theme, setTheme, resolvedTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const isActive = (href: string) => {
    if (pathname === href) return true
    if (!pathname.startsWith(href)) return false
    const next = pathname[href.length]
    if (next !== "/") return false
    if (href === "/providers" && pathname.startsWith("/providers/settlements")) return false
    return true
  }

  return (
    <aside
      className={cn(
        "hidden lg:flex lg:flex-col fixed inset-y-0 left-0 z-50 border-r border-sidebar-border bg-sidebar transition-all duration-300",
        collapsed ? "w-16" : "w-60",
      )}
    >
      <div className="flex flex-col px-3 pt-6 pb-4">
        <div className={cn("flex items-center", collapsed ? "justify-center" : "mb-8 gap-3 px-2")}>
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary">
            <Shield className="size-4 text-primary-foreground" />
          </div>
          {!collapsed && (
            <div>
              <h1 className="text-lg font-bold tracking-tight text-foreground">
                Castellan
              </h1>
            </div>
          )}
        </div>

        <nav className="flex-1 space-y-6">
          {navGroups.map((group, i) => (
            <div key={i}>
              {group.label && !collapsed && (
                <p className="mb-1 px-3 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                  {group.label}
                </p>
              )}
              <div className="space-y-0.5">
                {group.items.map((item) => (
                  <NavItemLink
                    key={item.href}
                    item={item}
                    collapsed={collapsed}
                    isActive={isActive(item.href)}
                  />
                ))}
              </div>
            </div>
          ))}
        </nav>
      </div>

      <div className="mt-auto border-t border-sidebar-border p-3">
        {!collapsed && (
          <NavItemLink
            item={{ label: "Settings", href: "/settings", icon: Settings }}
            collapsed={false}
            isActive={pathname === "/settings"}
          />
        )}

        <div className={cn("mt-2 flex", collapsed ? "flex-col items-center gap-2" : "gap-1")}>
          {mounted && (
            <button
              onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
              className={cn(
                "flex items-center justify-center rounded-lg text-sidebar-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                collapsed ? "size-10" : "flex-1 px-3 py-2 text-sm font-medium",
              )}
              title="Toggle theme"
            >
              {resolvedTheme === "dark" ? (
                <Sun className="size-[18px]" />
              ) : (
                <Moon className="size-[18px]" />
              )}
              {!collapsed && <span className="ml-2">Appearance</span>}
            </button>
          )}
          <button
            onClick={toggle}
            className={cn(
              "flex items-center justify-center rounded-lg text-sidebar-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
              collapsed ? "size-10" : "px-3 py-2",
            )}
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            <ChevronLeft
              className={cn(
                "size-[18px] transition-transform duration-200",
                collapsed && "rotate-180",
              )}
            />
          </button>
        </div>
      </div>
    </aside>
  )
}

export function MobileSidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname()
  const { theme, setTheme, resolvedTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const isActive = (href: string) => {
    if (pathname === href) return true
    if (!pathname.startsWith(href)) return false
    const next = pathname[href.length]
    if (next !== "/") return false
    if (href === "/providers" && pathname.startsWith("/providers/settlements")) return false
    return true
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 px-4 py-5">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary">
          <Shield className="size-4 text-primary-foreground" />
        </div>
        <h1 className="text-lg font-bold tracking-tight text-foreground">Castellan</h1>
      </div>

      <nav className="flex-1 space-y-6 px-3">
        {navGroups.map((group, i) => (
          <div key={i}>
            {group.label && (
              <p className="mb-1 px-3 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                {group.label}
              </p>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const Icon = item.icon
                return (
                  <a
                    key={item.href}
                    href={item.href}
                    onClick={onNavigate}
                    className={cn(
                      "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200",
                      isActive(item.href)
                        ? "bg-sidebar-accent text-sidebar-accent-foreground"
                        : "text-sidebar-muted-foreground hover:bg-muted hover:text-foreground",
                    )}
                  >
                    <Icon className="size-[20px] shrink-0" />
                    <span>{item.label}</span>
                  </a>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-sidebar-border p-3">
        <a
          href="/settings"
          onClick={onNavigate}
          className={cn(
            "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200",
            pathname === "/settings"
              ? "bg-sidebar-accent text-sidebar-accent-foreground"
              : "text-sidebar-muted-foreground hover:bg-muted hover:text-foreground",
          )}
        >
          <Settings className="size-[20px] shrink-0" />
          <span>Settings</span>
        </a>
        <div className="mt-2 flex gap-1">
          {mounted && (
            <button
              onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
              className="flex flex-1 items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-sidebar-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {resolvedTheme === "dark" ? <Sun className="size-[18px]" /> : <Moon className="size-[18px]" />}
              <span>Appearance</span>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export function Sidebar() {
  const { collapsed, toggle } = useSidebarStore()

  return (
    <>
      <DesktopSidebar collapsed={collapsed} toggle={toggle} />
      <Sheet>
        <div className="fixed top-4 left-4 z-50 lg:hidden">
          <SheetTrigger className="flex size-9 items-center justify-center rounded-lg border border-border bg-background text-muted-foreground shadow-sm transition-colors hover:text-foreground">
            <Menu className="size-5" />
          </SheetTrigger>
        </div>
        <SheetContent side="left" className="w-72 p-0">
          <MobileSidebarNav />
        </SheetContent>
      </Sheet>
    </>
  )
}
