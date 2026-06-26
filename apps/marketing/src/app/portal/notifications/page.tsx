"use client"

import { useEffect, useState } from "react"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

interface Notification {
  id: number
  title: string
  message: string
  is_read: boolean
  created_at: string
}

export default function NotificationsPage() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)

  const fetchNotifications = () => {
    fetch(`${API_URL}/api/notifications`, { credentials: "include" })
      .then((r) => r.json())
      .then((d) => setNotifications(d.data || []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchNotifications()
  }, [])

  const markAsRead = async (id: number) => {
    await fetch(`${API_URL}/api/notifications/${id}/read`, {
      method: "PATCH",
      credentials: "include",
    })
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, is_read: true } : n))
    )
  }

  if (loading) return <p>Loading...</p>

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Notifications</h1>
      {notifications.length === 0 ? (
        <p className="text-zinc-500">No notifications.</p>
      ) : (
        <div className="space-y-3">
          {notifications.map((n) => (
            <div
              key={n.id}
              className={`rounded-xl border p-4 ${n.is_read ? "border-zinc-200 bg-white" : "border-zinc-300 bg-zinc-50"}`}
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="font-medium text-zinc-900">{n.title}</h3>
                  <p className="mt-1 text-sm text-zinc-600">{n.message}</p>
                  <p className="mt-1 text-xs text-zinc-400">{new Date(n.created_at).toLocaleString()}</p>
                </div>
                {!n.is_read && (
                  <button
                    onClick={() => markAsRead(n.id)}
                    className="shrink-0 text-xs font-medium text-zinc-500 hover:text-zinc-900 transition-colors"
                  >
                    Mark read
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
