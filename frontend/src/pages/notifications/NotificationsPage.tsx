import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Notification } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Button, Badge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function NotificationsPage() {
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { data: notifications = [], isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.get('/api/notifications').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const readMutation = useMutation({
    mutationFn: (id: number) => api.patch(`/api/notifications/${id}/read`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to mark as read'),
  })

  const readAllMutation = useMutation({
    mutationFn: () => api.post('/api/notifications/read-all'),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to mark all as read'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/notifications/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['notifications'] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete notification'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Notifications"
        description="Your notifications"
        action={notifications.length > 0 ? (
          <Button icon="done_all" variant="secondary" onClick={() => readAllMutation.mutate()} data-testid="notifications-mark-all-read-button">Mark all as read</Button>
        ) : undefined}
      />

      {isLoading ? (
        <Card testId="notifications-loading-card"><LoadingSkeleton rows={4} testId="notifications-loading" /></Card>
      ) : notifications.length === 0 ? (
        <Card testId="notifications-empty-card"><EmptyState icon="notifications" title="No notifications" description="You're all caught up!" testId="notifications-empty-state" /></Card>
      ) : (
        <div className="space-y-2">
          {notifications.map((n: Notification) => (
            <Card key={n.id} className={`transition-all ${!n.is_read ? 'border-[var(--primary)] ring-1 ring-[var(--primary)]/20' : ''}`} testId={`notifications-card-${n.id}`}>
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="font-medium text-sm text-[var(--on-surface)]">{n.title}</h3>
                    {!n.is_read && <span className="w-2 h-2 rounded-full bg-[var(--primary)] shrink-0" />}
                    <Badge variant="neutral">{n.type}</Badge>
                  </div>
                  <p className="text-sm text-[var(--on-surface-variant)] mt-1">{n.body}</p>
                  <span className="text-xs text-[var(--on-surface-variant)] mt-2 block">{new Date(n.created_at).toLocaleString()}</span>
                </div>
                <div className="flex gap-2 shrink-0">
                  {!n.is_read && (
                    <Button variant="ghost" size="sm" icon="mark_email_read" onClick={() => readMutation.mutate(n.id)} data-testid={`notifications-read-${n.id}`}>Read</Button>
                  )}
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(n.id)} data-testid={`notifications-delete-${n.id}`}>Delete</Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => deleteMutation.mutate(confirmDelete!)}
        title="Delete Notification"
        message="Are you sure you want to delete this notification?"
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={deleteMutation.isPending}
        error={error}
        testId="notifications-delete-modal"
      />
    </main>
  )
}
