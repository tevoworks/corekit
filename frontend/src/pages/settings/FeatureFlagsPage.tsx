import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import { useState, useMemo } from 'react'
import { Card, PageHeader, Table, Button, Modal, Input, Badge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function FeatureFlagsPage() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<any>(null)
  const [confirmDelete, setConfirmDelete] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allFlags, setAllFlags] = useState<any[]>([])

  const { data, isLoading } = useQuery({
    queryKey: ['feature-flags', cursor],
    queryFn: () => api.get(`/api/feature-flags?limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllFlags(items)
      } else {
        setAllFlags(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim()
    if (!q) return allFlags
    return allFlags.filter((f: any) =>
      (f.name || '').toLowerCase().includes(q) ||
      (f.key || '').toLowerCase().includes(q)
    )
  }, [allFlags, search])

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/feature-flags', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['feature-flags'] }); setShowForm(false); setCursor(0) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create flag'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/feature-flags/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['feature-flags'] }); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update flag'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/feature-flags/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['feature-flags'] }); setConfirmDelete(null); setCursor(0) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete flag'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Feature Flags"
        description="Toggle application features on and off"
        action={<Button icon="add" onClick={() => setShowForm(true)} data-testid="feature-flags-add-button">Add Flag</Button>}
      />

      <div className="mb-4 flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[var(--on-surface-variant)]">search</span>
          <input
            type="text"
            placeholder="Search by name or key..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
            data-testid="feature-flags-search-input"
          />
        </div>
      </div>

      {isLoading && allFlags.length === 0 ? (
        <Card testId="feature-flags-loading-card"><LoadingSkeleton rows={4} testId="feature-flags-loading" /></Card>
      ) : filtered.length === 0 ? (
        <Card testId="feature-flags-empty-card"><EmptyState icon="flag" title="No feature flags" description={search ? 'Try adjusting your search.' : 'Create feature flags to control feature availability.'} action={!search ? <Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="feature-flags-empty-add-button">Add Flag</Button> : undefined} testId="feature-flags-empty-state" /></Card>
      ) : (
        <>
          <Table headers={['Name', 'Key', 'Status', 'Actions']} testId="feature-flags-table">
            {filtered.map((f: any) => (
              <tr key={f.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
                <td className="p-3 font-medium">{f.name}</td>
                <td className="p-3 font-mono text-xs">{f.key}</td>
                <td className="p-3">
                  <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${f.enabled ? 'bg-[var(--success-bg)] text-[var(--success-text)]' : 'bg-[var(--surface-container)] text-[var(--on-surface-variant)]'}`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${f.enabled ? 'bg-green-500' : 'bg-gray-400'}`} />
                    {f.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </td>
                <td className="p-3 text-right">
                  <Button variant="ghost" size="sm" icon={f.enabled ? 'toggle_off' : 'toggle_on'} onClick={() => updateMutation.mutate({ id: f.id, name: f.name, key: f.key, description: f.description || '', enabled: !f.enabled })} data-testid={`feature-flags-toggle-${f.id}`} className="mr-1">
                    {f.enabled ? 'Disable' : 'Enable'}
                  </Button>
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(f)} data-testid={`feature-flags-edit-${f.id}`} className="mr-1">Edit</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(f)} data-testid={`feature-flags-delete-${f.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {data && data.next_cursor > 0 && filtered.length === allFlags.length && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(data.next_cursor)} data-testid="feature-flags-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {showForm && (
        <Modal title="Add Feature Flag" onClose={() => setShowForm(false)} testId="feature-flags-add-modal">
          {error && <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm" role="alert">{error}</div>}
          <FlagForm onSave={(body) => createMutation.mutate(body)} onCancel={() => setShowForm(false)} saving={createMutation.isPending} />
        </Modal>
      )}

      {editing && (
        <Modal title="Edit Feature Flag" onClose={() => setEditing(null)} testId="feature-flags-edit-modal">
          {error && <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm" role="alert">{error}</div>}
          <FlagForm flag={editing} onSave={(body) => updateMutation.mutate({ id: editing.id, ...body })} onCancel={() => setEditing(null)} saving={updateMutation.isPending} />
        </Modal>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => deleteMutation.mutate(confirmDelete.id)}
        title="Delete Feature Flag"
        message={`Are you sure you want to delete ${confirmDelete?.name}? This action cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={deleteMutation.isPending}
        error={error}
        testId="feature-flags-delete-modal"
      />
    </main>
  )
}

function FlagForm({ flag, onSave, onCancel, saving }: {
  flag?: any
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
}) {
  const [name, setName] = useState(flag?.name || '')
  const [key, setKey] = useState(flag?.key || '')
  const [description, setDescription] = useState(flag?.description || '')
  const [enabled, setEnabled] = useState(flag?.enabled ?? true)
  return (
    <div className="space-y-4">
      <Input label="Name" value={name} onChange={e => setName(e.target.value)} placeholder="Dark Mode" required data-testid="feature-flags-form-name-input" />
      <Input label="Key" value={key} onChange={e => setKey(e.target.value)} placeholder="dark_mode" required data-testid="feature-flags-form-key-input" />
      <Input label="Description" value={description} onChange={e => setDescription(e.target.value)} placeholder="Enable dark mode theme" data-testid="feature-flags-form-description-input" />
      <label className="flex items-center gap-2 text-sm cursor-pointer">
        <input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="accent-[var(--primary)]" data-testid="feature-flags-form-enabled-checkbox" />
        Enabled on creation
      </label>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="feature-flags-form-cancel-button">Cancel</Button>
        <Button onClick={() => onSave({ name, key, description, enabled })} loading={saving} data-testid="feature-flags-form-submit-button">
          {flag ? 'Update' : 'Create'}
        </Button>
      </div>
    </div>
  )
}