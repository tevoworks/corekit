"use client"

import { useState } from "react"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function ContactPage() {
  const [form, setForm] = useState({ name: "", email: "", subject: "", message: "" })
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle")

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setStatus("loading")
    try {
      const res = await fetch(`${API_URL}/api/public/contact`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      })
      if (!res.ok) throw new Error("Failed")
      setStatus("success")
      setForm({ name: "", email: "", subject: "", message: "" })
    } catch {
      setStatus("error")
    }
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl mb-8">Contact us</h1>
      {status === "success" ? (
        <div className="rounded-lg bg-green-50 p-6 text-green-800">
          <p className="font-semibold">Message sent!</p>
          <p className="mt-1 text-sm">We&apos;ll get back to you as soon as possible.</p>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-zinc-700">Name</label>
            <input type="text" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-700">Email</label>
            <input type="email" required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-700">Subject</label>
            <input type="text" required value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })}
              className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-700">Message</label>
            <textarea rows={5} required value={form.message} onChange={(e) => setForm({ ...form, message: e.target.value })}
              className="mt-1 block w-full rounded-lg border border-zinc-300 px-4 py-2 text-sm focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 outline-none" />
          </div>
          {status === "error" && <p className="text-sm text-red-600">Something went wrong. Please try again.</p>}
          <button type="submit" disabled={status === "loading"}
            className="rounded-lg bg-zinc-900 px-6 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 transition-colors">
            {status === "loading" ? "Sending..." : "Send message"}
          </button>
        </form>
      )}
    </div>
  )
}
