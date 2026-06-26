"use client"

import { useEffect, useState } from "react"
import { useRouter, usePathname } from "next/navigation"
import Link from "next/link"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const [authed, setAuthed] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch(`${API_URL}/api/me`, { credentials: "include" })
      .then((res) => {
        if (!res.ok) throw new Error("Not authenticated")
        setAuthed(true)
      })
      .catch(() => {
        router.push("/login")
      })
      .finally(() => setLoading(false))
  }, [router])

  if (loading) return <div className="flex items-center justify-center py-24"><p>Loading...</p></div>
  if (!authed) return null

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <nav className="mb-8 flex items-center gap-6 border-b border-zinc-200 pb-4 text-sm font-medium">
        <Link href="/portal" className={pathname === "/portal" ? "text-zinc-900" : "text-zinc-500 hover:text-zinc-900"}>
          Dashboard
        </Link>
        <Link href="/portal/profile" className={pathname === "/portal/profile" ? "text-zinc-900" : "text-zinc-500 hover:text-zinc-900"}>
          Profile
        </Link>
        <Link href="/portal/notifications" className={pathname === "/portal/notifications" ? "text-zinc-900" : "text-zinc-500 hover:text-zinc-900"}>
          Notifications
        </Link>
        <Link href="/" className="ml-auto text-zinc-500 hover:text-zinc-900">← Site</Link>
      </nav>
      {children}
    </div>
  )
}
