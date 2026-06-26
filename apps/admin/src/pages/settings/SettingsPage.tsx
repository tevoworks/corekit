import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import { useState, useMemo } from 'react'
import { Card, PageHeader, Table, Button, Modal, Input, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function SettingsPage() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<any>(null)
  const [confirmDelete, setConfirmDelete] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  const { data: settings = [], isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: () => api.get('/api/settings').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim()
    if (!q) return settings
    return settings.filter((s: any) =>
      (s.key || '').toLowerCase().includes(q) ||
      (s.value || '').toLowerCase().includes(q)
    )
  }, [settings, search])

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/settings', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); setShowForm(false) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to save setting'),
  })

  const updateMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/settings', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update setting'),
  })

  const deleteMutation = useMutation({
    mutationFn: (key: string) => api.delete(`/api/settings/${encodeURIComponent(key)}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete setting'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Settings"
        description="Manage system settings"
        action={<Button icon="add" onClick={() => setShowForm(true)} data-testid="settings-add-button">Add Setting</Button>}
      />

      <div className="mb-4 flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[var(--on-surface-variant)]">search</span>
          <input
            type="text"
            placeholder="Search by key or value..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
            data-testid="settings-search-input"
          />
        </div>
      </div>

      {isLoading ? (
        <Card testId="settings-loading-card"><LoadingSkeleton rows={4} testId="settings-loading" /></Card>
      ) : filtered.length === 0 ? (
        <Card testId="settings-empty-card"><EmptyState icon="settings" title="No settings" description={search ? 'Try adjusting your search.' : 'Add system settings to configure your application.'} action={!search ? <Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="settings-empty-add-button">Add Setting</Button> : undefined} testId="settings-empty-state" /></Card>
      ) : (
        <Table headers={['Key', 'Value', 'Updated', 'Actions']} testId="settings-table">
          {filtered.map((s: any) => (
            <tr key={s.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
              <td className="p-3 font-mono text-xs font-medium">{s.key}</td>
              <td className="p-3 text-[var(--on-surface-variant)] max-w-md truncate">{s.value}</td>
              <td className="p-3 text-[var(--on-surface-variant)] text-xs">{s.updated_at ? new Date(s.updated_at).toLocaleString() : '—'}</td>
              <td className="p-3 text-right">
                <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(s)} data-testid={`settings-edit-${s.id}`} className="mr-1">Edit</Button>
                <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(s)} data-testid={`settings-delete-${s.id}`}>Delete</Button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {showForm && (
        <Modal title="Add Setting" onClose={() => setShowForm(false)} testId="settings-add-modal">
          {error && <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm" role="alert">{error}</div>}
          <SettingForm onSave={(body) => createMutation.mutate(body)} onCancel={() => setShowForm(false)} saving={createMutation.isPending} />
        </Modal>
      )}

      {editing && (
        <Modal title="Edit Setting" onClose={() => setEditing(null)} testId="settings-edit-modal">
          {error && <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm" role="alert">{error}</div>}
          <SettingForm setting={editing} onSave={(body) => updateMutation.mutate(body)} onCancel={() => setEditing(null)} saving={updateMutation.isPending} />
        </Modal>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => deleteMutation.mutate(confirmDelete.key)}
        title="Delete Setting"
        message={`Are you sure you want to delete ${confirmDelete?.key}?`}
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={deleteMutation.isPending}
        error={error}
        testId="settings-delete-modal"
      />
    </main>
  )
}

function SettingForm({ setting, onSave, onCancel, saving }: {
  setting?: any
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
}) {
  const [key, setKey] = useState(setting?.key || '')
  const [value, setValue] = useState(setting?.value || '')
  return (
    <div className="space-y-4">
      <Input label="Key" value={key} onChange={e => setKey(e.target.value)} placeholder="site_name" required disabled={!!setting} data-testid="settings-form-key-input" />
      <Input label="Value" value={value} onChange={e => setValue(e.target.value)} placeholder="My App" required data-testid="settings-form-value-input" />
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="settings-form-cancel-button">Cancel</Button>
        <Button onClick={() => onSave({ key, value })} loading={saving} data-testid="settings-form-submit-button">{setting ? 'Update' : 'Save'}</Button>
      </div>
    </div>
  )
}