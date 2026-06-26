import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { User, Role } from '../../lib/types'
import { useState, useMemo } from 'react'
import { Card, PageHeader, Table, StatusBadge, Button, Modal, Input, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

type SortField = 'full_name' | 'email' | 'status' | 'role_name'
type SortDir = 'asc' | 'desc'

const STATUS_OPTIONS = ['ACTIVE', 'SUSPENDED', 'HALTED', 'PENDING_VERIFICATION'] as const

export default function UsersPage() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [statusEdit, setStatusEdit] = useState<{ id: number; current: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState<SortField>('full_name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: () => api.get('/api/users').then(r => {
      const d = r.data.data
      return Array.isArray(d) ? d : []
    }),
  })

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim()
    let list = Array.isArray(users) ? users : []
    if (q) {
      list = list.filter((u: User) =>
        (u.full_name || '').toLowerCase().includes(q) ||
        (u.email || '').toLowerCase().includes(q)
      )
    }
    list = [...list].sort((a: any, b: any) => {
      const av = (a[sortField] || '').toString().toLowerCase()
      const bv = (b[sortField] || '').toString().toLowerCase()
      return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
    })
    return list
  }, [users, search, sortField, sortDir])

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
    mutationFn: (body: any) => api.post('/api/users', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); setShowForm(false) },
    onError: (err: any) => setFormError(err.response?.data?.error?.message || 'Failed to create user'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/users/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }) },
    onError: (err: any) => setFormError(err.response?.data?.error?.message || 'Failed to update user'),
  })

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => api.patch(`/api/users/${id}/status`, { status }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); setStatusEdit(null) },
    onError: (err: any) => setStatusError(err.response?.data?.error?.message || 'Failed to change status'),
  })

  const { data: roles = [] } = useQuery({
    queryKey: ['roles'],
    queryFn: () => api.get('/api/roles?limit=100').then(r => {
      const d = r.data.data
      return Array.isArray(d) ? d : []
    }),
  })

  const [roleError, setRoleError] = useState<string | null>(null)
  const roleMutation = useMutation({
    mutationFn: ({ id, role_id }: { id: number; role_id: number | null }) => api.put(`/api/users/${id}/role`, { role_id }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }) },
    onError: (err: any) => setRoleError(err.response?.data?.error?.message || 'Failed to update role'),
  })

  const [pendingEditBody, setPendingEditBody] = useState<any>(null)

  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/users/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete user'),
  })

  if (isLoading) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Users" description="Manage user accounts" />
      <Card testId="users-loading-card"><LoadingSkeleton rows={6} testId="users-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
        <PageHeader
          title="Users"
          description="Manage user accounts"
          action={
            <Button icon="add" onClick={() => setShowForm(true)} data-testid="users-add-button">Add User</Button>
          }
        />

      <div className="mb-4 flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[var(--on-surface-variant)]">search</span>
          <input
            type="text"
            placeholder="Search by name or email..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
            data-testid="users-search-input"
          />
        </div>
        {filtered.length !== users.length && (
          <span className="text-xs text-[var(--on-surface-variant)]">{filtered.length} of {users.length}</span>
        )}
      </div>

      {!Array.isArray(users) || users.length === 0 ? (
        <Card testId="users-empty-card">
          <EmptyState icon="people" title="No users found" description="Create your first user to get started." action={<Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="users-empty-add-button">Add User</Button>} testId="users-empty-state" />
        </Card>
      ) : filtered.length === 0 && search ? (
        <Card testId="users-no-results-card">
          <EmptyState icon="search" title="No results" description={`No users matching "${search}".`} testId="users-no-results" />
        </Card>
      ) : (
        <Table headers={[
          <button key="name" className="hover:opacity-70" onClick={() => toggleSort('full_name')} data-testid="users-sort-name">Name{sortIcon('full_name')}</button>,
          <button key="email" className="hover:opacity-70" onClick={() => toggleSort('email')} data-testid="users-sort-email">Email{sortIcon('email')}</button>,
          <button key="role" className="hover:opacity-70" onClick={() => toggleSort('role_name')} data-testid="users-sort-role">Role{sortIcon('role_name')}</button>,
          <button key="status" className="hover:opacity-70" onClick={() => toggleSort('status')} data-testid="users-sort-status">Status{sortIcon('status')}</button>,
          'Actions',
        ]} testId="users-table">
          {filtered.map((u: User) => (
            <tr key={u.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`users-row-${u.id}`}>
              <td className="p-3 font-medium">{u.full_name}</td>
              <td className="p-3 text-[var(--on-surface-variant)]">{u.email}</td>
              <td className="p-3">{u.role_name || <span className="text-[var(--on-surface-variant)]">&mdash;</span>}</td>
              <td className="p-3"><StatusBadge status={u.status} /></td>
              <td className="p-3 text-right">
                <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(u)} data-testid={`users-edit-${u.id}`} className="mr-1">Edit</Button>
                <Button variant="ghost" size="sm" icon="flag" onClick={() => setStatusEdit({ id: u.id, current: u.status })} data-testid={`users-status-${u.id}`} className="mr-1">Status</Button>
                <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(u.id)} data-testid={`users-delete-${u.id}`}>Delete</Button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {showForm && (
        <Modal title="Create User" onClose={() => { setShowForm(false); setFormError(null) }} testId="users-create-modal">
          <UserFormContent
            onSave={(body) => createMutation.mutate(body)}
            onCancel={() => { setShowForm(false); setFormError(null) }}
            saving={createMutation.isPending}
            error={formError}
            onClearError={() => setFormError(null)}
          />
        </Modal>
      )}
      {editing && (
        <Modal title="Edit User" onClose={() => { setEditing(null); setFormError(null); setRoleError(null) }} testId="users-edit-modal">
          <UserFormContent
            user={editing}
            roles={roles}
            onSave={(body) => setPendingEditBody(body)}
            onCancel={() => { setEditing(null); setFormError(null); setRoleError(null) }}
            saving={updateMutation.isPending || roleMutation.isPending}
            error={formError}
            roleError={roleError}
            onClearError={() => { setFormError(null); setRoleError(null) }}
          />
        </Modal>
      )}

      {pendingEditBody && editing && (
        <ConfirmDialog
          open={!!pendingEditBody}
          onClose={() => { setPendingEditBody(null); setRoleError(null) }}
          onConfirm={async () => {
            const body = pendingEditBody
            setPendingEditBody(null)
            setFormError(null)
            setRoleError(null)
            try {
              await updateMutation.mutateAsync({ id: editing.id, email: body.email, full_name: body.full_name })
              if (body.role_id !== undefined && body.role_id !== editing.role_id) {
                await roleMutation.mutateAsync({ id: editing.id, role_id: body.role_id })
              }
              setEditing(null)
            } catch {
              // errors are handled by each mutation's onError
            }
          }}
          title="Update User"
          message="Are you sure you want to update this user's information? This change will take effect immediately."
          confirmLabel="Yes, Update"
          testId="users-update-confirm-dialog"
        />
      )}

      {statusEdit && (
        <Modal title="Change User Status" onClose={() => { setStatusEdit(null); setStatusError(null) }} testId="users-status-modal">
          <ChangeUserStatusContent
            userId={statusEdit.id}
            currentStatus={statusEdit.current}
            onConfirm={(status) => statusMutation.mutate({ id: statusEdit.id, status })}
            onCancel={() => { setStatusEdit(null); setStatusError(null) }}
            saving={statusMutation.isPending}
            error={statusError}
            onClearError={() => setStatusError(null)}
          />
        </Modal>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete User"
          message="This action is irreversible. The user and all associated data will be permanently removed."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="users-delete-confirm-dialog"
        />
      )}
    </main>
  )
}

function ChangeUserStatusContent({ userId, currentStatus, onConfirm, onCancel, saving, error, onClearError }: {
  userId: number
  currentStatus: string
  onConfirm: (status: string) => void
  onCancel: () => void
  saving?: boolean
  error?: string | null
  onClearError?: () => void
}) {
  const [selected, setSelected] = useState(currentStatus)
  const [showConfirm, setShowConfirm] = useState(false)

  const handleConfirm = () => {
    if (selected !== currentStatus) {
      setShowConfirm(true)
    }
  }

  return (
    <div>
      <p className="text-xs text-[var(--on-surface-variant)] mb-4 tracking-wide uppercase">Select new status</p>
      <div className="space-y-1.5">
        {STATUS_OPTIONS.map(s => (
          <label key={s} className={`flex items-center gap-3.5 p-3.5 rounded-xl border-2 cursor-pointer transition-all ${
            selected === s
              ? 'border-[var(--primary)] bg-[var(--primary)]/5'
              : 'border-transparent bg-[var(--surface-container)] hover:bg-[var(--surface-container-high)]'
          }`}>
            <input type="radio" name="status" value={s} checked={selected === s} onChange={() => setSelected(s)} className="sr-only" data-testid={`users-status-radio-${s}`} />
            <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0 ${selected === s ? 'border-[var(--primary)]' : 'border-[var(--outline-variant)]'}`}>
              {selected === s && <div className="w-2 h-2 rounded-full bg-[var(--primary)]" />}
            </div>
            <div>
              <div className="text-sm font-medium text-[var(--on-surface)]">{s}</div>
              {s === currentStatus && <div className="text-xs text-[var(--on-surface-variant)] mt-0.5">Current status</div>}
            </div>
          </label>
        ))}
      </div>

      {error && (
        <div className="mt-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="users-status-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="flex gap-3 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="users-status-cancel-button" className="px-5">Cancel</Button>
        <Button onClick={handleConfirm} loading={saving} disabled={selected === currentStatus} data-testid="users-status-confirm-button" className="px-5">
          Confirm
        </Button>
      </div>

      <ConfirmDialog
        open={showConfirm}
        onClose={() => setShowConfirm(false)}
        onConfirm={() => { setShowConfirm(false); onConfirm(selected) }}
        title="Change User Status"
        message={`Are you sure you want to change status from ${currentStatus} to ${selected}? This action affects user access.`}
        confirmLabel="Yes, Change Status"
        testId="users-status-confirm-dialog"
      />
    </div>
  )
}

function UserFormContent({ user, roles, onSave, onCancel, saving, error, roleError, onClearError }: {
  user?: User | null
  roles?: Role[]
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
  error?: string | null
  roleError?: string | null
  onClearError?: () => void
}) {
  const [email, setEmail] = useState(user?.email || '')
  const [fullName, setFullName] = useState(user?.full_name || '')
  const [roleId, setRoleId] = useState<number | null>(user?.role_id ?? null)
  const hasChanges = email !== (user?.email || '') || fullName !== (user?.full_name || '') || roleId !== (user?.role_id ?? null)
  return (
    <div>
      <div className="space-y-4">
        <Input label="Email *" type="email" value={email} onChange={e => setEmail(e.target.value)} required data-testid="users-form-email-input" />
        <Input label="Full Name *" value={fullName} onChange={e => setFullName(e.target.value)} required data-testid="users-form-name-input" />
        {user && (
          <div>
            <label className="block text-xs font-medium text-[var(--on-surface-variant)] mb-1.5">Role</label>
            <select
              value={roleId ?? ''}
              onChange={e => setRoleId(e.target.value ? Number(e.target.value) : null)}
              className="w-full rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
              data-testid="users-form-role-select"
            >
              <option value="">&mdash; No role &mdash;</option>
              {(roles || []).map((r: Role) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>
        )}
      </div>
      {error && (
        <div className="mt-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="users-form-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      {roleError && (
        <div className="mt-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="users-role-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{roleError}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="users-form-cancel-button">Cancel</Button>
        {user ? (
          <Button onClick={() => onSave({ email, full_name: fullName, role_id: roleId })} disabled={!hasChanges} loading={saving} data-testid="users-form-submit-button">Update</Button>
        ) : (
          <Button onClick={() => onSave({ email, full_name: fullName })} loading={saving} data-testid="users-form-submit-button">Create</Button>
        )}
      </div>
    </div>
  )
}