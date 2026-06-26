"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function RegisterPage() {
  const router = useRouter()
  const [form, setForm] = useState({ email: "", password: "", full_name: "" })
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")
    try {
      const res = await fetch(`${API_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
        credentials: "include",
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.error?.message || "Registration failed")
      }
      router.push("/portal")
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Registration failed")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-md px-4 py-24">
      <h1 className="text-2xl font-bold text-center mb-8">Create an account</h1>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium text-zinc-700">Full name</label>
          <input type="text" required value={form.full_name} onChange={(e) => setForm({...form, full_name: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        <div>
          <label className="block text-sm font-medium text-zinc-700">Email</label>
          <input type="email" required value={form.email} onChange={(e) => setForm({...form, email: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        <div>
          <label className="block text-sm font-medium text-zinc-700">Password</label>
          <input type="password" required minLength={8} value={form.password} onChange={(e) => setForm({...form, password: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <button type="submit" disabled={loading}
          className="w-full rounded-lg bg-zinc-900 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 transition-colors">
          {loading ? "Creating account..." : "Create account"}
        </button>
        <p className="text-center text-sm text-zinc-500">
          Already have an account? <a href="/login" className="text-zinc-900 underline">Log in</a>
        </p>
      </form>
    </div>
  )
}
