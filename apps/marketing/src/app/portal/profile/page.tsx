"use client"

import { useEffect, useState } from "react"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

interface Profile {
  email: string
  full_name: string
  bio?: string
}

export default function ProfilePage() {
  const [, setProfile] = useState<Profile | null>(null)
  const [form, setForm] = useState({ email: "", full_name: "" })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null)

  useEffect(() => {
    fetch(`${API_URL}/api/me`, { credentials: "include" })
      .then((r) => r.json())
      .then((d) => {
        setProfile(d.data)
        setForm({ email: d.data.email, full_name: d.data.full_name })
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setMessage(null)
    try {
      const res = await fetch(`${API_URL}/api/profile`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
        credentials: "include",
      })
      if (!res.ok) throw new Error("Failed to update profile")
      setMessage({ type: "success", text: "Profile updated successfully." })
    } catch {
      setMessage({ type: "error", text: "Something went wrong." })
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <p>Loading...</p>

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Edit profile</h1>
      <form onSubmit={handleSubmit} className="max-w-md space-y-5">
        <div>
          <label className="block text-sm font-medium text-zinc-700">Email</label>
          <input type="email" required value={form.email} onChange={(e) => setForm({...form, email: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        <div>
          <label className="block text-sm font-medium text-zinc-700">Full name</label>
          <input type="text" required value={form.full_name} onChange={(e) => setForm({...form, full_name: e.target.value})}
            className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
        </div>
        {message && (
          <p className={`text-sm ${message.type === "success" ? "text-green-600" : "text-red-600"}`}>{message.text}</p>
        )}
        <button type="submit" disabled={saving}
          className="rounded-lg bg-zinc-900 px-6 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 transition-colors">
          {saving ? "Saving..." : "Save changes"}
        </button>
      </form>
    </div>
  )
}
