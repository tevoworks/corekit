"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function LoginPage() {
  const router = useRouter()
  const [form, setForm] = useState({ email: "", password: "" })
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")
    try {
      const res = await fetch(`${API_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
        credentials: "include",
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.error?.message || "Login failed")
      }
      router.push("/portal")
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Login failed")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-md px-4 py-24">
      <h1 className="text-2xl font-bold text-center mb-8">Log in</h1>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium text-zinc-700">Email</label>
          <input type="email" required value={form.email} onChange={(e) => setForm({...form, email: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        <div>
          <label className="block text-sm font-medium text-zinc-700">Password</label>
          <input type="password" required value={form.password} onChange={(e) => setForm({...form, password: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <button type="submit" disabled={loading}
          className="w-full rounded-lg bg-zinc-900 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 transition-colors">
          {loading ? "Logging in..." : "Log in"}
        </button>
        <p className="text-center text-sm text-zinc-500">
          Don&apos;t have an account? <a href="/register" className="text-zinc-900 underline">Create one</a>
        </p>
      </form>
    </div>
  )
}
