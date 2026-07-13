"use client"

import { useEffect } from "react"
import { usePathname, useRouter } from "next/navigation"
import { Sidebar } from "@/components/layout/sidebar"
import { TopBar } from "@/components/layout/top-bar"
import { useAuth } from "@/lib/auth/auth-context"
import { useSidebarStore } from "@/stores/sidebar"
import { cn } from "@/lib/utils"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { collapsed } = useSidebarStore()
  const { user, isLoading } = useAuth()
  const pathname = usePathname()
  const router = useRouter()

  useEffect(() => {
    if (isLoading) return
    if (!user) return
    if (user.onboarding_completed) return
    if (pathname === "/onboarding") return

    router.replace("/onboarding")
  }, [user, isLoading, pathname, router])

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
