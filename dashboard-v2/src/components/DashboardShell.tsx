"use client"

import { useState } from "react"
import { Sidebar } from "@/components/Sidebar"
import { TopBar } from "@/components/TopBar"
import { LowBalanceBanner } from "@/components/LowBalanceBanner"
import { Toaster } from "@/components/ui/sonner"
import type { Role } from "@/lib/types"
import { MOCK_CONSUMER_OVERVIEW } from "@/lib/mock-data"

interface DashboardShellProps {
  children: React.ReactNode
  role?: Role
}

export function DashboardShell({ children, role = "both" }: DashboardShellProps) {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div className="flex h-screen">
      <Sidebar role={role} collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar balance={MOCK_CONSUMER_OVERVIEW.balance} />
        <div className="flex-1 overflow-y-auto">
          <LowBalanceBanner
            balance={MOCK_CONSUMER_OVERVIEW.balance}
            visible={MOCK_CONSUMER_OVERVIEW.isLowBalance}
          />
          <main className="p-6">{children}</main>
        </div>
      </div>
      <Toaster />
    </div>
  )
}
