"use client"

import { Sidebar } from "@/components/layout/sidebar"
import { TopBar } from "@/components/layout/top-bar"
import { useSidebarStore } from "@/stores/sidebar"
import { cn } from "@/lib/utils"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { collapsed } = useSidebarStore()

  return (
    <div className="relative flex min-h-screen">
      <Sidebar />
      <div
        className={cn(
          "flex flex-1 flex-col transition-all duration-300",
          collapsed ? "lg:ml-16" : "lg:ml-60",
        )}
      >
        <TopBar />
        <main className="flex-1 p-8">
          {children}
        </main>
      </div>
    </div>
  )
}
