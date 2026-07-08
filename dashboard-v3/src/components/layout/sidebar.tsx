"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAccount } from "@/lib/auth/account-context";
import { cn } from "@/lib/utils";

interface NavItem {
  label: string;
  href: string;
  roles: ("provider" | "consumer")[];
}

const navItems: NavItem[] = [
  { label: "Overview", href: "/provider/overview", roles: ["provider"] },
  { label: "Discover", href: "/discover", roles: ["provider", "consumer"] },
  { label: "Ledger", href: "/account/entries", roles: ["provider", "consumer"] },
  { label: "Analytics", href: "/analytics", roles: ["provider"] },
  { label: "Usage", href: "/usage", roles: ["provider"] },
  { label: "Settlements", href: "/provider/settlements", roles: ["provider"] },
  { label: "Providers", href: "/provider/providers", roles: ["provider"] },
  { label: "My APIs", href: "/provider/apis", roles: ["provider"] },
  { label: "API Keys", href: "/provider/api-keys", roles: ["provider"] },
  { label: "Overview", href: "/consumer/overview", roles: ["consumer"] },
  { label: "Analytics", href: "/analytics", roles: ["consumer"] },
  { label: "Deposit", href: "/consumer/deposit", roles: ["consumer"] },
  { label: "Usage", href: "/usage", roles: ["consumer"] },
  { label: "API Keys", href: "/consumer/api-keys", roles: ["consumer"] },
  { label: "Settings", href: "/settings", roles: ["provider", "consumer"] },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user } = useAccount();

  const visible = navItems.filter(
    (item) => user && item.roles.includes(user.role)
  );

  return (
    <aside className="flex h-full w-56 flex-col border-r border-sidebar-border bg-sidebar">
      <div className="flex h-14 items-center border-b border-sidebar-border px-5">
        <Link href="/" className="text-sm font-semibold tracking-tight text-sidebar-foreground">
          Castellan
        </Link>
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {visible.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              "block rounded-md px-3 py-2 text-sm font-medium transition-colors",
              pathname === item.href
                ? "bg-sidebar-accent text-sidebar-foreground"
                : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-foreground"
            )}
          >
            {item.label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}
