import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Contact } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, Modal, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

const STATUS_OPTIONS = ['', 'new', 'read', 'replied', 'archived']

export default function MessagesPage() {
  const qc = useQueryClient()
  const [statusFilter, setStatusFilter] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allMessages, setAllMessages] = useState<Contact[]>([])
  const [viewing, setViewing] = useState<Contact | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'messages', statusFilter, cursor],
    queryFn: () => api.get(`/api/contact/messages?status=${statusFilter}&limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllMessages(items)
      } else {
        setAllMessages(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/contact/messages/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'messages'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete message'),
  })

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => api.patch(`/api/contact/messages/${id}/status`, { status }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'messages'] }) },
  })

  const assignMutation = useMutation({
    mutationFn: ({ id }: { id: number }) => api.post(`/api/contact/messages/${id}/assign`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'messages'] }) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to assign message'),
  })

  const handleFilterChange = (val: string) => {
    setStatusFilter(val)
    setCursor(0)
  }

  if (isLoading && allMessages.length === 0) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Contact Messages" description="View and manage contact form submissions" />
      <Card testId="messages-loading-card"><LoadingSkeleton rows={5} testId="messages-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Contact Messages" description="View and manage contact form submissions" />

      <div className="mb-4 flex items-center gap-3">
        <select
          value={statusFilter}
          onChange={e => handleFilterChange(e.target.value)}
          className="rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
          data-testid="messages-status-filter"
        >
          <option value="">All Statuses</option>
          <option value="new">New</option>
          <option value="read">Read</option>
          <option value="replied">Replied</option>
          <option value="archived">Archived</option>
        </select>
      </div>

      {viewing && (
        <Modal title={`Message from ${viewing.name}`} onClose={() => setViewing(null)} size="lg" testId="messages-view-modal">
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div><span className="text-[var(--on-surface-variant)]">Name:</span> {viewing.name}</div>
              <div><span className="text-[var(--on-surface-variant)]">Email:</span> {viewing.email}</div>
              <div><span className="text-[var(--on-surface-variant)]">Phone:</span> {viewing.phone || '—'}</div>
              <div><span className="text-[var(--on-surface-variant)]">Subject:</span> {viewing.subject}</div>
              <div><span className="text-[var(--on-surface-variant)]">Status:</span> <StatusBadge status={viewing.status} /></div>
              <div><span className="text-[var(--on-surface-variant)]">Date:</span> {new Date(viewing.created_at).toLocaleString()}</div>
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Message</label>
              <div className="p-3 rounded-lg bg-[var(--surface-container)] text-sm whitespace-pre-wrap">{viewing.message}</div>
            </div>
            <div className="flex gap-2 pt-4 border-t border-[var(--outline-variant)]">
              {viewing.status !== 'read' && (
                <Button variant="secondary" size="sm" onClick={() => { statusMutation.mutate({ id: viewing.id, status: 'read' }); setViewing(prev => prev ? { ...prev, status: 'read' } : null) }} data-testid="messages-mark-read">Mark as Read</Button>
              )}
              {viewing.status !== 'replied' && (
                <Button variant="secondary" size="sm" onClick={() => { statusMutation.mutate({ id: viewing.id, status: 'replied' }); setViewing(prev => prev ? { ...prev, status: 'replied' } : null) }} data-testid="messages-mark-replied">Mark as Replied</Button>
              )}
              {viewing.status !== 'archived' && (
                <Button variant="secondary" size="sm" onClick={() => { statusMutation.mutate({ id: viewing.id, status: 'archived' }); setViewing(prev => prev ? { ...prev, status: 'archived' } : null) }} data-testid="messages-archive">Archive</Button>
              )}
              <Button variant="secondary" size="sm" onClick={() => assignMutation.mutate({ id: viewing.id })} data-testid="messages-assign">Assign to Me</Button>
            </div>
          </div>
        </Modal>
      )}

      {allMessages.length === 0 ? (
        <Card testId="messages-empty-card"><EmptyState icon="inbox" title="No messages yet" description="Contact form submissions will appear here." testId="messages-empty-state" /></Card>
      ) : (
        <>
          <Table headers={['Name', 'Email', 'Subject', 'Status', 'Date', 'Actions']} testId="messages-table">
            {allMessages.map((m: Contact) => (
              <tr key={m.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`messages-row-${m.id}`}>
                <td className="p-3 font-medium">{m.name}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">{m.email}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">{m.subject}</td>
                <td className="p-3"><StatusBadge status={m.status} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{new Date(m.created_at).toLocaleDateString()}</td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="visibility" onClick={() => setViewing(m)} data-testid={`messages-view-${m.id}`} className="mr-1">View</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(m.id)} data-testid={`messages-delete-${m.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allMessages.length >= 50 && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="messages-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Message"
          message="This will permanently remove this contact message."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="messages-delete-confirm-dialog"
        />
      )}
    </main>
  )
}
