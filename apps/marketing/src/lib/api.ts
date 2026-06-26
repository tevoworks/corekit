const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  })
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  const json = await res.json()
  if (json.error) {
    throw new Error(json.error.message || "API error")
  }
  return json.data as T
}

export async function apiFetchEnvelope<T>(path: string, options?: RequestInit): Promise<{ data: T; meta?: Record<string, unknown> }> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  })
  return res.json()
}
