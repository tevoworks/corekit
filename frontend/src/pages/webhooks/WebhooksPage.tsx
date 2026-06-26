import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Webhook } from '../../lib/types'
import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, PageHeader, Table, Button, Input, Badge, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

type SortField = 'name' | 'url' | 'active'
type SortDir = 'asc' | 'desc'

export default function WebhooksPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [editing, setEditing] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState<SortField>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [cursor, setCursor] = useState(0)
  const [allWebhooks, setAllWebhooks] = useState<any[]>([])

  const { isLoading } = useQuery({
    queryKey: ['webhooks', cursor],
    queryFn: () => api.get(`/api/webhooks?limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllWebhooks(items)
      } else {
        setAllWebhooks(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim()
    let list = allWebhooks
    if (q) {
      list = list.filter((w: any) =>
        (w.name || '').toLowerCase().includes(q) ||
        (w.url || '').toLowerCase().includes(q)
      )
    }
    list = [...list].sort((a: any, b: any) => {
      const av = (a[sortField] === undefined ? '' : String(a[sortField])).toLowerCase()
      const bv = (b[sortField] === undefined ? '' : String(b[sortField])).toLowerCase()
      return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
    })
    return list
  }, [allWebhooks, search, sortField, sortDir])

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

  const sortIcon = (field: SortField) => {
    if (sortField !== field) return ' \u2195'
    return sortDir === 'asc' ? ' \u2191' : ' \u2193'
  }

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/webhooks', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['webhooks'] }); setCursor(0); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create webhook'),
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/webhooks/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['webhooks'] }); setCursor(0); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update webhook'),
  })
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/webhooks/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['webhooks'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete webhook'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Webhooks"
        description="Manage outgoing webhooks"
        action={<Button icon="add" onClick={() => setEditing({})} data-testid="webhooks-add-button">Add Webhook</Button>}
      />

      <div className="mb-4 flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[var(--on-surface-variant)]">search</span>
          <input type="text" placeholder="Search by name or URL..." value={search} onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent" data-testid="webhooks-search-input" />
        </div>
      </div>

      {editing && (
        <Card className="mb-4" testId="webhooks-form-card">
          <h2 className="text-lg font-semibold mb-3">{editing?.id ? 'Edit Webhook' : 'New Webhook'}</h2>
          <WebhookFormContent
            webhook={editing?.id ? editing : undefined}
            onSave={(body) => {
              if (editing?.id) updateMutation.mutate({ id: editing.id, ...body })
              else createMutation.mutate(body)
            }}
            onCancel={() => setEditing(null)}
            saving={createMutation.isPending || updateMutation.isPending}
          />
        </Card>
      )}

      {isLoading && allWebhooks.length === 0 ? (
        <Card testId="webhooks-loading-card"><LoadingSkeleton rows={5} testId="webhooks-loading" /></Card>
      ) : allWebhooks.length === 0 ? (
        <Card testId="webhooks-empty-card"><EmptyState icon="webhook" title="No webhooks yet" description="Create your first webhook to receive events." testId="webhooks-empty-state" /></Card>
      ) : filtered.length === 0 && search ? (
        <Card testId="webhooks-no-results-card"><EmptyState icon="search" title="No results" description={`No webhooks matching "${search}".`} testId="webhooks-no-results" /></Card>
      ) : (
        <>
          <Table headers={[
            <button key="name" className="hover:opacity-70" onClick={() => toggleSort('name')} data-testid="webhooks-sort-name">Name{sortIcon('name')}</button>,
            'URL',
            'Events',
            <button key="status" className="hover:opacity-70" onClick={() => toggleSort('active')} data-testid="webhooks-sort-status">Status{sortIcon('active')}</button>,
            'Actions',
          ]} testId="webhooks-table">
            {filtered.map((w: any) => (
              <tr key={w.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`webhooks-row-${w.id}`}>
                <td className="p-3 font-medium">{w.name}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)] max-w-[200px] truncate font-mono">{w.url}</td>
                <td className="p-3">
                  <div className="flex gap-1 flex-wrap">
                    {(w.events || []).map((e: string) => <Badge key={e} variant="neutral">{e}</Badge>)}
                  </div>
                </td>
                <td className="p-3"><StatusBadge status={w.active ? 'Active' : 'Inactive'} /></td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="list_alt" onClick={() => navigate(`/webhooks/${w.id}/deliveries`)} data-testid={`webhooks-deliveries-${w.id}`} className="mr-1">Deliveries</Button>
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(w)} data-testid={`webhooks-edit-${w.id}`} className="mr-1">Edit</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(w.id)} data-testid={`webhooks-delete-${w.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allWebhooks.length >= 50 && filtered.length === allWebhooks.length && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="webhooks-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => deleteMutation.mutate(confirmDelete!)}
        title="Delete Webhook"
        message="Are you sure you want to delete this webhook?"
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={deleteMutation.isPending}
        error={error}
        testId="webhooks-delete-modal"
      />
    </main>
  )
}

function WebhookFormContent({ webhook, onSave, onCancel, saving }: { webhook?: any; onSave: (body: any) => void; onCancel: () => void; saving?: boolean }) {
  const [name, setName] = useState(webhook?.name || '')
  const [url, setUrl] = useState(webhook?.url || '')
  const [events, setEvents] = useState<string[]>(webhook?.events || [])
  const [customEvent, setCustomEvent] = useState('')
  const [active, setActive] = useState(webhook?.active ?? true)

  const toggleEvent = (e: string) => {
    setEvents(prev => prev.includes(e) ? prev.filter(x => x !== e) : [...prev, e])
  }

  const addCustomEvent = () => {
    const trimmed = customEvent.trim()
    if (trimmed && !events.includes(trimmed)) {
      setEvents(prev => [...prev, trimmed])
    }
    setCustomEvent('')
  }

  const presets = [
    'user.created', 'user.updated', 'user.deleted',
    'session.created', 'session.revoked',
    'role.created', 'role.updated', 'role.deleted',
    'settings.updated', 'feature_flag.updated',
    'api_key.created', 'api_key.revoked',
    'file.uploaded', 'file.deleted',
  ]

  return (
    <div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-[480px]">
        <Input label="Name *" value={name} onChange={e => setName(e.target.value)} required data-testid="webhooks-form-name-input" />
        <Input label="URL *" value={url} onChange={e => setUrl(e.target.value.trim())} placeholder="https://..." required data-testid="webhooks-form-url-input" />
      </div>
      <div className="mt-4 max-w-[480px]">
        <label className="block text-sm font-medium text-[var(--on-surface)] mb-2">Events</label>
        <div className="flex flex-wrap gap-1.5 mb-2">
          {presets.map(e => (
            <button key={e} type="button" onClick={() => toggleEvent(e)}
              className={`px-2.5 py-1 rounded-full text-xs font-medium border transition-all ${events.includes(e) ? 'bg-[var(--primary)] text-white border-[var(--primary)]' : 'bg-white text-[var(--on-surface-variant)] border-[var(--outline-variant)] hover:border-[var(--primary)]'}`}
              data-testid={`webhooks-event-preset-${e}`}>
              {e}
            </button>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" value={customEvent} onChange={e => setCustomEvent(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addCustomEvent() } }}
            placeholder="Type custom event and press Enter"
            className="flex-1 px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent" data-testid="webhooks-custom-event-input" />
          <Button variant="secondary" size="sm" onClick={addCustomEvent} disabled={!customEvent.trim()} data-testid="webhooks-custom-event-add-button">Add</Button>
        </div>
        {events.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mt-2">
            {events.map(e => (
              <span key={e} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-200">
                {e}
                <button type="button" onClick={() => toggleEvent(e)} className="hover:text-blue-900" data-testid={`webhooks-event-remove-${e}`}>&times;</button>
              </span>
            ))}
          </div>
        )}
      </div>
      <div className="mt-4">
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input type="checkbox" checked={active} onChange={e => setActive(e.target.checked)} className="accent-[var(--primary)]" data-testid="webhooks-form-active-checkbox" />
          Active
        </label>
      </div>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="webhooks-form-cancel-button">Cancel</Button>
        <Button onClick={() => onSave({ name, url, events, active })} loading={saving} data-testid="webhooks-form-submit-button">
          {webhook ? 'Update' : 'Create'}
        </Button>
      </div>
    </div>
  )
}