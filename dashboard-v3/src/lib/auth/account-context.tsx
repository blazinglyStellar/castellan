"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { useRouter, usePathname } from "next/navigation";
import { api, ApiError } from "@/lib/api/client";
import type { DashboardMeResponse } from "@/lib/api/types";

interface AccountUser {
  id: string;
  email: string;
  role: "provider" | "consumer";
  accountId: string;
}

interface AccountContextValue {
  user: AccountUser | null;
  isLoading: boolean;
  error: Error | null;
  logout: () => Promise<void>;
}

const AccountContext = createContext<AccountContextValue | null>(null);

export function AccountProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<AccountUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchUser = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const profile = await api.get<DashboardMeResponse>("/api/v1/me");
      if (!profile.id) {
        throw new Error("Invalid user profile");
      }
      setUser({
        id: profile.id,
        email: profile.email,
        role: (profile.role === "provider" ? "provider" : "consumer"),
        accountId: profile.id,
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setUser(null);
        if (pathname !== "/login") {
          router.push("/login");
        }
        return;
      }
      setError(err instanceof Error ? err : new Error("Failed to load user"));
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, [router, pathname]);

  useEffect(() => {
    fetchUser();
  }, [fetchUser]);

  useEffect(() => {
    const handler = () => {
      setUser(null);
      if (pathname !== "/login") {
        router.push("/login");
      }
    };
    window.addEventListener("auth:unauthorized", handler);
    return () => window.removeEventListener("auth:unauthorized", handler);
  }, [router, pathname]);

  const logout = useCallback(async () => {
    try {
      await api.post("/api/v1/auth/logout");
    } catch {
      /* proceed even if server call fails */
    }
    setUser(null);
    router.push("/login");
  }, [router]);

  return (
    <AccountContext.Provider value={{ user, isLoading, error, logout }}>
      {children}
    </AccountContext.Provider>
  );
}

export function useAccount(): AccountContextValue {
  const ctx = useContext(AccountContext);
  if (!ctx) {
    throw new Error("useAccount must be used within an AccountProvider");
  }
  return ctx;
}
