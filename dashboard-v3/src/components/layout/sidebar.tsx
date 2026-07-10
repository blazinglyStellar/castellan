"use client"

import Image from "next/image"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  BarChart3,
  Activity,
  BookOpen,
  Building2,
  Key,
  Wallet,
  Banknote,
  Compass,
} from "lucide-react"
import { useAccount } from "@/lib/auth/account-context"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

interface NavItem {
  label: string
  href: string
  icon: React.ElementType
}

interface NavGroup {
  label?: string
  roles: ("provider" | "consumer")[]
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    roles: ["provider", "consumer"],
    items: [
      { label: "Overview", href: "/overview", icon: LayoutDashboard },
    ],
  },
  {
    label: "Monitoring",
    roles: ["provider", "consumer"],
    items: [
      { label: "Analytics", href: "/analytics", icon: BarChart3 },
      { label: "Usage", href: "/usage", icon: Activity },
      { label: "Ledger", href: "/account/entries", icon: BookOpen },
    ],
  },
  {
    label: "Payments",
    roles: ["provider", "consumer"],
    items: [
      { label: "Deposit", href: "/deposit", icon: Banknote },
      { label: "API Keys", href: "/api-keys", icon: Key },
    ],
  },
  {
    label: "Management",
    roles: ["provider", "consumer"],
    items: [
      { label: "Providers", href: "/provider/providers", icon: Building2 },
      { label: "Settlements", href: "/provider/settlements", icon: Wallet },
    ],
  },
  {
    label: "Explore",
    roles: ["provider", "consumer"],
    items: [
      { label: "Discover", href: "/discover", icon: Compass },
    ],
  },
]

function isActive(pathname: string, href: string): boolean {
  if (href === "") return false
  if (pathname === href) return true
  if (href !== "/" && pathname.startsWith(href + "/")) return true
  if (href !== "/" && pathname.startsWith(href)) return true
  return false
}

export function AppSidebar() {
  const pathname = usePathname()
  const { user } = useAccount()

  if (!user) return null

  const role = user.role

  return (
    <Sidebar collapsible="icon" className="sticky top-0 h-svh self-start">
      <SidebarHeader className="flex h-14 flex-row items-center gap-2 px-4">
        <Link href="/" className="flex items-center gap-2">
          <Image
            src="/logo.svg"
            alt="Castellan"
            width={28}
            height={49}
            className="shrink-0"
          />
          <span className="group-data-[collapsible=icon]:hidden text-sm font-semibold tracking-tight text-sidebar-foreground">
            Castellan
          </span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        {navGroups
          .filter((group) => group.roles.includes(role))
          .map((group) => (
            <SidebarGroup key={group.label ?? "overview"}>
              {group.label && <SidebarGroupLabel>{group.label}</SidebarGroupLabel>}
              <SidebarGroupContent>
                <SidebarMenu>
                  {group.items.map((item) => {
                    const active = isActive(pathname, item.href)
                    const Icon = item.icon

                    return (
                      <SidebarMenuItem key={item.href}>
                        <SidebarMenuButton asChild isActive={active}>
                          <Link href={item.href}>
                            <Icon />
                            <span>{item.label}</span>
                          </Link>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    )
                  })}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          ))}
      </SidebarContent>
    </Sidebar>
  )
}
