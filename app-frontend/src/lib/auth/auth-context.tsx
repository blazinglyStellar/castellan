"use client"

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react"
import { get, post, setAuthToken, SESSION_TOKEN_KEY, UnauthorizedError } from "@/lib/api/client"

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"

export type UserRole = "consumer" | "provider"

export interface User {
  id: string
  email: string
  role: UserRole
  deposit_memo: string
  payout_stellar_address?: string
  onboarding_completed: boolean
}

interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  refreshUser: () => Promise<void>
  updateUser: (partial: Partial<User>) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isReady, setIsReady] = useState(false)

  const fetchUser = useCallback(async () => {
    try {
      const data = await get<User>("/api/v1/me")
      setUser(data)
    } catch {
      setUser(null)
    } finally {
      setIsLoading(false)
    }
  }, [])

  const PUBLIC_PATHS = ["/login"]

  useEffect(() => {
    const storedToken = sessionStorage.getItem(SESSION_TOKEN_KEY)
    if (storedToken) {
      setAuthToken(storedToken)
    }

    const params = new URLSearchParams(window.location.search)
    const urlToken = params.get("token")
    if (urlToken) {
      setAuthToken(urlToken)
      window.history.replaceState(null, "", window.location.pathname)
    }

    setIsReady(true)
    fetchUser()

    const handleUnauthorized = () => {
      setAuthToken(null)
      if (PUBLIC_PATHS.includes(window.location.pathname)) {
        setUser(null)
        return
      }

      const redirect = encodeURIComponent(window.location.origin + "/login")
      window.location.href = API_BASE + "/api/v1/auth/logout?redirect=" + redirect
    }

    window.addEventListener("auth:unauthorized", handleUnauthorized)
    return () => window.removeEventListener("auth:unauthorized", handleUnauthorized)
  }, [fetchUser])

  const updateUser = useCallback((partial: Partial<User>) => {
    setUser((prev) => (prev ? { ...prev, ...partial } : prev))
  }, [])

  const logout = useCallback(async () => {
    try {
      await post("/api/v1/auth/logout")
    } catch {
      // proceed with client-side logout regardless
    }
    setUser(null)
    setAuthToken(null)
    document.cookie = "session_token=; path=/; max-age=0"
    window.location.href = "/login"
  }, [])

  if (!isReady) {
    return null
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        refreshUser: fetchUser,
        updateUser,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return ctx
}
