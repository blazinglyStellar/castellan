const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

interface User {
  id: string
  email: string
  created_at: string
}

class ApiClientError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    ...options,
  })

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      window.location.href = "/login"
    }
    throw new ApiClientError("Unauthorized", 401)
  }

  if (!res.ok) {
    throw new ApiClientError(`Request failed: ${res.statusText}`, res.status)
  }

  return res.json()
}

export async function getMe(): Promise<User> {
  return request<User>("/api/v1/auth/me")
}

export async function logout(): Promise<void> {
  await request<{ message: string }>("/api/v1/auth/logout", {
    method: "POST",
  })
}

export function getOAuthLoginURL(provider: string): string {
  return `${API_URL}/auth/${provider}/login`
}
