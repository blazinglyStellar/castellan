"use client";

import { useTheme } from "next-themes";
import { useAccount } from "@/lib/auth/account-context";
import { Sun, Moon, LogOut, PanelLeft } from "lucide-react";
import { cn } from "@/lib/utils";
import { SidebarTrigger } from "@/components/ui/sidebar";

interface TopBarProps {
  title?: string;
}

export function TopBar({ title }: TopBarProps) {
  const { theme, setTheme } = useTheme();
  const { user, logout } = useAccount();

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-background px-6">
      <div className="flex items-center gap-3">
        <SidebarTrigger className="h-7 w-7 text-muted-foreground hover:text-foreground">
          <PanelLeft size={16} />
        </SidebarTrigger>
        {title && (
          <h1 className="text-sm font-medium text-foreground">{title}</h1>
        )}
      </div>
      <div className="flex items-center gap-3">
        {user && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{user.email}</span>
            <span
              className={cn(
                "rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider",
                user.role === "provider"
                  ? "bg-primary/10 text-primary"
                  : "bg-accent text-accent-foreground"
              )}
            >
              {user.role === "provider" ? "Producer" : "Consumer"}
            </span>
          </div>
        )}
        <button
          type="button"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Toggle theme"
        >
          {theme === "dark" ? <Sun size="16" /> : <Moon size="16" />}
        </button>
        <button
          type="button"
          onClick={logout}
          className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Sign out"
        >
          <LogOut size="16" />
        </button>
      </div>
    </header>
  );
}
