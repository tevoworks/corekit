"use client"

import { useEffect, useState } from "react"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

interface User {
  id: number; email: string; full_name: string; role_name: string | null; status: string
}

export default function PortalDashboard() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch(`${API_URL}/api/me`, { credentials: "include" })
      .then((r) => r.json())
      .then((d) => setUser(d.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <p>Loading...</p>
  if (!user) return <p>Could not load profile.</p>

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Welcome, {user.full_name}</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div className="rounded-xl border border-zinc-200 p-6">
          <p className="text-sm text-zinc-500">Email</p>
          <p className="mt-1 font-medium">{user.email}</p>
        </div>
        <div className="rounded-xl border border-zinc-200 p-6">
          <p className="text-sm text-zinc-500">Role</p>
          <p className="mt-1 font-medium">{user.role_name || "—"}</p>
        </div>
        <div className="rounded-xl border border-zinc-200 p-6">
          <p className="text-sm text-zinc-500">Status</p>
          <p className="mt-1 font-medium">{user.status}</p>
        </div>
      </div>
    </div>
  )
}
