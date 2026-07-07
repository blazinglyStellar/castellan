"use client";

import { Sidebar } from "@/components/layout/sidebar";
import { TopBar } from "@/components/layout/top-bar";
import { usePathname } from "next/navigation";

const routeTitles: Record<string, string> = {
  "/provider/overview": "Overview",
  "/analytics": "Analytics",
  "/usage": "Usage",
  "/provider/settlements": "Settlements",
  "/provider/apis": "My APIs",
  "/provider/api-keys": "API Keys",
  "/consumer/overview": "Overview",
  "/consumer/deposit": "Deposit",
  "/consumer/api-keys": "API Keys",
};

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const title = routeTitles[pathname] || "Dashboard";

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <div className="flex flex-1 flex-col">
        <TopBar title={title} />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
