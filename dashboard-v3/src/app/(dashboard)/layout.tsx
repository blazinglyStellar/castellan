"use client"

import { SidebarProvider, SidebarTrigger, SidebarInset } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/layout/sidebar"
import { TopBar } from "@/components/layout/top-bar"
import { usePathname } from "next/navigation"

const routeTitles: Record<string, string> = {
  "/overview": "Overview",
  "/discover": "Discover",
  "/account/entries": "Ledger",
  "/analytics": "Analytics",
  "/usage": "Usage",
  "/provider/settlements": "Settlements",
  "/provider/providers": "Providers",
  "/deposit": "Deposit",
  "/api-keys": "API Keys",
  "/settings": "Settings",
}

function findTitle(pathname: string): string {
  if (routeTitles[pathname]) return routeTitles[pathname]
  for (const [prefix, title] of Object.entries(routeTitles)) {
    if (pathname.startsWith(prefix + "/") || pathname.startsWith(prefix)) {
      return title
    }
  }
  return "Dashboard"
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  const title = findTitle(pathname)

  return (
    <SidebarProvider defaultOpen={true}>
      <AppSidebar />
      <SidebarInset className="overflow-hidden">
        <TopBar title={title} />
        <div className="flex-1 overflow-y-auto p-6">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  )
}
