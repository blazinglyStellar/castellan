const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
  ) {
    super(message)
    this.name = "ApiError"
  }
}

export class UnauthorizedError extends ApiError {
  constructor() {
    super("Unauthorized", 401)
    this.name = "UnauthorizedError"
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_BASE}${path}`

  const response = await fetch(url, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    ...options,
  })

  if (!response.ok) {
    if (response.status === 401) {
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("auth:unauthorized"))
      }
      throw new UnauthorizedError()
    }

    const body = await response.json().catch(() => ({ error: response.statusText }))
    throw new ApiError(body.error ?? "Unknown error", response.status)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json()
}

export function get<T>(path: string, signal?: AbortSignal) {
  return request<T>(path, { method: "GET", signal })
}

export function post<T>(path: string, body?: unknown) {
  return request<T>(path, {
    method: "POST",
    body: body ? JSON.stringify(body) : undefined,
  })
}

export function patch<T>(path: string, body: unknown) {
  return request<T>(path, {
    method: "PATCH",
    body: JSON.stringify(body),
  })
}

export function del<T>(path: string) {
  return request<T>(path, { method: "DELETE" })
}
