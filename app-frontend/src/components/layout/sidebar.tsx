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
  ChevronLeft,
  Menu,
  LogOut,
} from "lucide-react"
import { useAuth } from "@/lib/auth/auth-context"
import { useSidebarStore } from "@/stores/sidebar"
import { cn } from "@/lib/utils"
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
  const { logout } = useAuth()
  const pathname = usePathname()
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
          <div className="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary">
            <img src="/logo3.png" alt="Castellan" className="size-[2.6rem] object-contain" />
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
        <NavItemLink
          item={{ label: "Settings", href: "/settings", icon: Settings }}
          collapsed={collapsed}
          isActive={pathname === "/settings"}
        />
        <button
          onClick={logout}
          className={cn(
            "group mt-0.5 flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200",
            collapsed ? "justify-center px-2" : "",
            "text-sidebar-muted-foreground hover:bg-muted hover:text-destructive",
          )}
          title={collapsed ? "Log out" : undefined}
        >
          <LogOut className="size-[20px] shrink-0" />
          {!collapsed && <span>Log out</span>}
        </button>

        <div className={cn("mt-2 flex", collapsed ? "flex-col items-center gap-1" : "gap-1")}>
          <button
            onClick={toggle}
            className={cn(
              "flex items-center justify-center rounded-lg text-sidebar-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
              collapsed ? "size-10" : "flex-1 px-3 py-2",
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
  const { logout } = useAuth()
  const pathname = usePathname()
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
        <div className="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary">
          <img src="/logo3.png" alt="Castellan" className="size-[2.6rem] object-contain" />
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
        <button
          onClick={logout}
          className="mt-0.5 flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-sidebar-muted-foreground transition-all duration-200 hover:bg-muted hover:text-destructive"
        >
          <LogOut className="size-[20px] shrink-0" />
          <span>Log out</span>
        </button>
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
