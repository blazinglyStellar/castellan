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
        <a
          href={process.env.NEXT_PUBLIC_GITHUB_URL}
          target="_blank"
          rel="noopener noreferrer"
          title="GitHub"
          className="text-muted-foreground transition-colors hover:text-foreground"
        >
          <svg className="size-6 fill-current" viewBox="0 0 24 24">
            <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
          </svg>
        </a>
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
        <button
          onClick={logout}
          className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-destructive"
          title="Log out"
        >
          <LogOut className="size-4" />
          <span className="hidden sm:inline">Log out</span>
        </button>
      </div>
    </header>
  )
}
