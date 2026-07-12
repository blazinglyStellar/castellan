"use client"

import { usePathname } from "next/navigation"
import { LogOut, User } from "lucide-react"
import { useAuth } from "@/lib/auth/auth-context"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

const routeTitles: Record<string, string> = {
  "/overview": "Overview",
  "/analytics": "Analytics",
  "/usage": "Usage",
  "/ledger": "Ledger",
  "/deposits": "Deposit",
  "/api-keys": "API Keys",
  "/settings": "Settings",
  "/discover": "Discover",
  "/providers": "Providers",
  "/providers/settlements": "Settlements",
}

function getPageTitle(pathname: string): string {
  if (pathname.startsWith("/ledger/")) return "Ledger Entry"
  if (pathname.startsWith("/providers/") && pathname.endsWith("/endpoints")) return "Endpoints"
  if (pathname.startsWith("/providers/")) return "Providers"
  return routeTitles[pathname] ?? "Castellan"
}

export function TopBar({ className }: { className?: string }) {
  const { user, isLoading, logout } = useAuth()
  const pathname = usePathname()
  const title = getPageTitle(pathname)

  return (
    <header
      className={cn(
        "sticky top-0 z-40 flex h-16 items-center justify-between border-b border-border bg-background/80 px-8 backdrop-blur-md",
        className,
      )}
    >
      <div className="flex items-center">
        <h2 className="text-sm font-semibold text-muted-foreground">
          {title}
        </h2>
      </div>

      <div className="flex items-center gap-4">
        <DropdownMenu>
          <DropdownMenuTrigger className="flex cursor-pointer items-center gap-3 rounded-full py-1 pl-3 pr-1 transition-colors hover:bg-muted">
              {isLoading ? (
                <div className="flex items-center gap-3">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="size-9 rounded-full" />
                </div>
              ) : (
                <>
                  <div className="hidden text-right sm:block">
                    <p className="text-xs font-semibold text-foreground">
                      {user?.email ?? "User"}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      {user?.role === "provider" ? "Producer" : "Consumer"}
                    </p>
                  </div>
                  <div className="flex size-9 items-center justify-center rounded-full border border-border bg-muted">
                    <User className="size-5 text-muted-foreground" />
                  </div>
                </>
              )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium text-foreground">
                  {user?.email}
                </p>
                <p className="text-xs text-muted-foreground">
                  {user?.role === "provider" ? "Producer" : "Consumer"}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={logout} className="cursor-pointer text-destructive focus:text-destructive">
              <LogOut className="mr-2 size-4" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
