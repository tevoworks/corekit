import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { NewsletterSubscriber } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function SubscribersPage() {
  const qc = useQueryClient()
  const [cursor, setCursor] = useState(0)
  const [allSubs, setAllSubs] = useState<NewsletterSubscriber[]>([])
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'subscribers', cursor],
    queryFn: () => api.get(`/api/contact/subscribers?limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllSubs(items)
      } else {
        setAllSubs(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/contact/subscribers/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'subscribers'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete subscriber'),
  })

  if (isLoading && allSubs.length === 0) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Newsletter Subscribers" description="Manage email subscribers" />
      <Card testId="subscribers-loading-card"><LoadingSkeleton rows={5} testId="subscribers-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Newsletter Subscribers" description="Manage email subscribers" />

      {allSubs.length === 0 ? (
        <Card testId="subscribers-empty-card"><EmptyState icon="mail" title="No subscribers yet" description="Newsletter subscribers will appear here." testId="subscribers-empty-state" /></Card>
      ) : (
        <>
          <Table headers={['Email', 'Name', 'Source', 'Status', 'Subscribed At', 'Actions']} testId="subscribers-table">
            {allSubs.map((s: NewsletterSubscriber) => (
              <tr key={s.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`subscribers-row-${s.id}`}>
                <td className="p-3 font-medium">{s.email}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">{s.name || '—'}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">{s.source || '—'}</td>
                <td className="p-3"><StatusBadge status={s.unsubscribed_at ? 'unsubscribed' : 'Active'} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{new Date(s.subscribed_at).toLocaleDateString()}</td>
                <td className="p-3 text-right">
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(s.id)} data-testid={`subscribers-delete-${s.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allSubs.length >= 50 && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="subscribers-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Subscriber"
          message="This will permanently remove this subscriber from the list."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="subscribers-delete-confirm-dialog"
        />
      )}
    </main>
  )
}
