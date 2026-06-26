import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Role, Permission } from '../../lib/types'
import { useState, useMemo, useCallback } from 'react'
import { Card, PageHeader, Table, Button, Modal, Input, Badge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

type SortField = 'name' | 'description' | 'permission_count'
type SortDir = 'asc' | 'desc'

export default function RolesPage() {
  const qc = useQueryClient()
  const [editing, setEditing] = useState<Role | null>(null)
  const [permEditor, setPermEditor] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState<SortField>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [cursor, setCursor] = useState(0)
  const [allRoles, setAllRoles] = useState<any[]>([])

  const { isLoading: loadRoles } = useQuery({
    queryKey: ['roles', cursor],
    queryFn: () => api.get(`/api/roles?limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllRoles(items)
      } else {
        setAllRoles(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const { data: allPermissions = [] } = useQuery({ queryKey: ['permissions'], queryFn: () => api.get('/api/permissions').then(r => Array.isArray(r.data.data) ? r.data.data : []) })
  const { data: byFeature } = useQuery({ queryKey: ['perm-by-domain'], queryFn: () => api.get('/api/permissions/by-feature').then(r => {
    const arr = r.data.data
    if (!Array.isArray(arr)) return {}
    const obj: Record<string, any> = {}
    arr.forEach((item: any) => { obj[item.domain] = item.permissions })
    return obj
  }) })

  const permNameToId = new Map<string, number>(allPermissions.map((p: Permission) => [p.name, p.id]))

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim()
    let list = allRoles
    if (q) {
      list = list.filter((r: any) => (r.name || '').toLowerCase().includes(q))
    }
    list = [...list].sort((a: any, b: any) => {
      let av = a[sortField], bv = b[sortField]
      if (sortField === 'description') { av = av || ''; bv = bv || '' }
      av = av?.toString().toLowerCase() || ''
      bv = bv?.toString().toLowerCase() || ''
      return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
    })
    return list
  }, [allRoles, search, sortField, sortDir])

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
    mutationFn: (body: any) => api.post('/api/roles', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['roles'] }); setCursor(0); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create role'),
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/roles/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['roles'] }); setCursor(0); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update role'),
  })
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/roles/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['roles'] }); setCursor(0); setConfirmDelete(null); setError(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete role'),
  })

  const groups = byFeature || {}

  if (loadRoles && allRoles.length === 0) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Roles & Permissions" description="Manage roles and assign permissions" />
      <Card testId="roles-loading-card"><LoadingSkeleton rows={5} testId="roles-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Roles & Permissions"
        description="Manage roles and assign permissions"
        action={
          <Button icon="add" onClick={() => setEditing({ id: 0, name: '', description: '' } as Role)} data-testid="roles-add-button">Add Role</Button>
        }
      />

      <div className="mb-4 flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[var(--on-surface-variant)]">search</span>
          <input
            type="text" placeholder="Search roles..."
            value={search} onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
            data-testid="roles-search-input"
          />
        </div>
      </div>

      {editing && (
        <Modal title={editing.id > 0 ? 'Edit Role' : 'New Role'} onClose={() => { setError(null); setEditing(null) }} testId="roles-form-modal">
          <RoleFormContent role={editing} error={error} onClearError={() => setError(null)} onSave={(body) => {
            setError(null)
            if (editing.id > 0) updateMutation.mutate({ id: editing.id, ...body })
            else createMutation.mutate(body)
          }} onCancel={() => { setError(null); setEditing(null) }} saving={createMutation.isPending || updateMutation.isPending} />
        </Modal>
      )}

      {allRoles.length === 0 ? (
        <Card testId="roles-empty-card"><EmptyState icon="manage_accounts" title="No roles yet" description="Create your first role to get started." action={<Button icon="add" variant="secondary" size="sm" onClick={() => setEditing({ id: 0, name: '', description: '' } as Role)} data-testid="roles-empty-add-button">Add Role</Button>} testId="roles-empty-state" /></Card>
      ) : filtered.length === 0 && search ? (
        <Card testId="roles-no-results-card"><EmptyState icon="search" title="No results" description={`No roles matching "${search}".`} testId="roles-no-results" /></Card>
      ) : (
        <>
          <Table headers={[
            <button key="name" className="hover:opacity-70" onClick={() => toggleSort('name')} data-testid="roles-sort-name">Name{sortIcon('name')}</button>,
            <button key="desc" className="hover:opacity-70" onClick={() => toggleSort('description')} data-testid="roles-sort-description">Description{sortIcon('description')}</button>,
            <button key="perms" className="hover:opacity-70" onClick={() => toggleSort('permission_count')} data-testid="roles-sort-permissions">Permissions{sortIcon('permission_count')}</button>,
            'Actions',
          ]} testId="roles-table">
            {filtered.map((r: any) => (
              <tr key={r.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`roles-row-${r.id}`}>
                <td className="p-3 font-medium">{r.name}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">{r.description || '—'}</td>
                <td className="p-3"><Badge variant="info">{r.permission_count ?? 0} assigned</Badge></td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="lock" onClick={() => setPermEditor({ role: r, assignedPerms: (r.permissions || []).map((p: any) => p.name || p) })} data-testid={`roles-permissions-${r.id}`} className="mr-1">Permissions</Button>
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(r)} data-testid={`roles-edit-${r.id}`} className="mr-1">Edit</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(r.id)} data-testid={`roles-delete-${r.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allRoles.length >= 50 && filtered.length === allRoles.length && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="roles-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Role"
          message="This will permanently remove this role. Users currently assigned this role will lose all associated permissions."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="roles-delete-confirm-dialog"
        />
      )}

      {permEditor && (
        <Modal title={`Permissions: ${permEditor.role.name}`} onClose={() => setPermEditor(null)} size="lg" testId="roles-permissions-modal">
          <PermissionEditorContent
            role={permEditor.role}
            assignedPerms={permEditor.assignedPerms}
            allPermissions={allPermissions}
            groups={groups}
            permNameToId={permNameToId}
            qc={qc}
            onClose={() => { setPermEditor(null); setCursor(0) }}
          />
        </Modal>
      )}
    </main>
  )
}

function RoleFormContent({ role, onSave, onCancel, saving, error, onClearError }: {
  role: any; onSave: (body: any) => void; onCancel: () => void; saving?: boolean; error?: string | null; onClearError?: () => void
}) {
  const [name, setName] = useState(role.name || '')
  const [desc, setDesc] = useState(role.description || '')
  return (
    <div>
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="roles-form-error-banner">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70" data-testid="roles-form-error-dismiss-button">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="space-y-4">
        <Input label="Role Name *" value={name} onChange={e => setName(e.target.value)} required data-testid="roles-form-name-input" />
        <Input label="Description" value={desc} onChange={e => setDesc(e.target.value)} data-testid="roles-form-description-input" />
      </div>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="roles-form-cancel-button">Cancel</Button>
        <Button onClick={() => onSave({ name, description: desc })} loading={saving} data-testid="roles-form-submit-button">{role.id > 0 ? 'Update' : 'Create'}</Button>
      </div>
    </div>
  )
}

function PermissionEditorContent({ role, assignedPerms, allPermissions, groups, permNameToId, qc, onClose }: {
  role: any; assignedPerms: string[]; allPermissions: Permission[]; groups: Record<string, any>; permNameToId: Map<string, number>; qc: any; onClose: () => void
}) {
  const [selected, setSelected] = useState<Set<string>>(new Set(assignedPerms))
  const [saving, setSaving] = useState(false)
  const [permError, setPermError] = useState<string | null>(null)

  const toggle = (name: string) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  const handleSave = useCallback(async () => {
    setSaving(true)
    const assigned = new Set(assignedPerms)
    const toAdd = [...selected].filter(n => !assigned.has(n))
    const toRemove = [...assigned].filter(n => !selected.has(n))
    const promises = [
      ...toAdd.map(name => api.post(`/api/roles/${role.id}/permissions`, { permission_id: permNameToId.get(name) }).catch(() => {})),
      ...toRemove.map(name => api.delete(`/api/roles/${role.id}/permissions/${permNameToId.get(name)}`).catch(() => {})),
    ]
    await Promise.allSettled(promises)
    qc.invalidateQueries({ queryKey: ['roles'] })
    setSaving(false)
    onClose()
  }, [selected, assignedPerms, role.id, qc, onClose])

  const permsList = Object.keys(groups).length > 0 ? groups : { all: allPermissions }

  return (
    <div className="space-y-4">
      {permError && <div className="p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-xs" data-testid="roles-perm-error-banner">{permError}</div>}
      <div className="space-y-4 max-h-[420px] overflow-y-auto pr-1">
        {Object.entries(permsList).map(([domain, perms]: [string, any]) => (
          <div key={domain}>
            <h3 className="text-sm font-semibold text-[var(--on-surface)] capitalize mb-2">{domain}</h3>
            <div className="flex flex-wrap gap-2">
              {Array.isArray(perms) && perms.map((p: any) => (
                <label key={p.name} className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--outline-variant)] text-xs cursor-pointer hover:bg-[var(--surface-container)] has-[:checked]:border-[var(--primary)] has-[:checked]:bg-blue-50 has-[:checked]:text-[var(--primary)] transition-all">
                  <input type="checkbox" checked={selected.has(p.name)} onChange={() => toggle(p.name)} className="accent-[var(--primary)]" data-testid={`roles-perm-checkbox-${p.name}`} />
                  {p.name}
                </label>
              ))}
            </div>
          </div>
        ))}
      </div>
      <div className="flex gap-2 justify-end pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onClose} data-testid="roles-permissions-cancel-button">Cancel</Button>
        <Button onClick={handleSave} loading={saving} data-testid="roles-permissions-save-button">Save Permissions</Button>
      </div>
    </div>
  )
}