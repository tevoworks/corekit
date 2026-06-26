import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { PermissionRegistry, GlobalTemplate } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, Badge, EmptyState, LoadingSkeleton, StatusBadge, ConfirmDialog } from '../../components/ui'

export default function PermissionsPage() {
  const qc = useQueryClient()
  const [tab, setTab] = useState<'registry' | 'templates'>('registry')
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'registry' | 'template'; id: number } | null>(null)
  const [error, setError] = useState<string | null>(null)

  const { data: registry = [], isLoading: loadReg } = useQuery({ queryKey: ['perm-registry'], queryFn: () => api.get('/api/permissions/registry').then(r => Array.isArray(r.data.data) ? r.data.data : []) })
  const { data: templates = [], isLoading: loadTmpl } = useQuery({ queryKey: ['templates'], queryFn: () => api.get('/api/templates').then(r => Array.isArray(r.data.data) ? r.data.data : []) })
  const { data: byDomain = {} } = useQuery({ queryKey: ['perm-by-domain'], queryFn: () => api.get('/api/permissions/by-feature').then(r => {
    const arr = r.data.data
    if (!Array.isArray(arr)) return {}
    const obj: Record<string, any> = {}
    arr.forEach((item: any) => { obj[item.domain] = item.permissions })
    return obj
  }) })

  const delReg = useMutation({
    mutationFn: (id: number) => api.delete(`/api/permissions/registry/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['perm-registry'] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete'),
  })
  const delTpl = useMutation({
    mutationFn: (id: number) => api.delete(`/api/templates/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['templates'] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete'),
  })

  const handleSync = () => api.post('/api/permissions/sync').then(() => qc.invalidateQueries({ queryKey: ['perm-registry'] }))

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Permission Registry" description="Manage registered permissions and templates" action={<Button icon="sync" variant="secondary" onClick={handleSync} data-testid="permissions-sync-button">Sync from YAML</Button>} />

      <div className="flex gap-1 mb-6 bg-[var(--surface-container)] rounded-lg p-1 w-fit">
        {(['registry', 'templates'] as const).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-all ${tab === t ? 'bg-white text-[var(--primary)] shadow-sm' : 'text-[var(--on-surface-variant)] hover:text-[var(--on-surface)]'}`}
            data-testid={`permissions-tab-${t}`}>
            {t === 'registry' ? 'Registry' : 'Templates'}
          </button>
        ))}
      </div>

      {tab === 'registry' && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            {Object.entries(byDomain).map(([domain, perms]: [string, any]) => (
              <Card key={domain}>
                <h3 className="text-sm font-semibold text-[var(--on-surface)] capitalize mb-3">{domain}</h3>
                {Array.isArray(perms) && perms.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5">
                    {perms.map((p: any) => <Badge key={p.name} variant="primary">{p.name}</Badge>)}
                  </div>
                ) : (
                  <p className="text-xs text-[var(--on-surface-variant)]" data-testid="permissions-domain-empty">No permissions</p>
                )}
              </Card>
            ))}
          </div>

          {loadReg ? <Card testId="permissions-registry-loading-card"><LoadingSkeleton rows={5} testId="permissions-registry-loading" /></Card> : registry.length === 0 ? (
            <Card testId="permissions-registry-empty-card"><EmptyState icon="lock" title="No permissions registered" description="Permissions appear here after they are registered via seed data or API." action={<Button variant="secondary" size="sm" icon="sync" onClick={handleSync} data-testid="permissions-empty-sync-button">Sync from YAML</Button>} testId="permissions-registry-empty" /></Card>
          ) : (
            <Table headers={['Domain', 'Name', 'Description', 'Actions']} testId="permissions-registry-table">
              {registry.map((p: PermissionRegistry) => (
                <tr key={p.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
                  <td className="p-3"><Badge variant="neutral">{p.domain}</Badge></td>
                  <td className="p-3 font-mono text-xs">{p.name}</td>
                  <td className="p-3 text-[var(--on-surface-variant)]">{p.description}</td>
                  <td className="p-3 text-right"><Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete({ type: 'registry', id: p.id })} data-testid={`permissions-delete-registry-${p.id}`}>Delete</Button></td>
                </tr>
              ))}
            </Table>
          )}
        </>
      )}

      {tab === 'templates' && (
        loadTmpl ? <Card testId="permissions-templates-loading-card"><LoadingSkeleton rows={3} testId="permissions-templates-loading" /></Card> : templates.length === 0 ? (
          <Card testId="permissions-templates-empty-card"><EmptyState icon="article" title="No templates yet" description="Templates define reusable permission sets." testId="permissions-templates-empty" /></Card>
        ) : (
          <Table headers={['Name', 'Category', 'Permissions', 'Status', 'Actions']} testId="permissions-templates-table">
            {templates.map((t: GlobalTemplate) => (
              <tr key={t.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
                <td className="p-3 font-medium">{t.name}</td>
                <td className="p-3"><Badge variant="neutral">{t.category || '—'}</Badge></td>
                <td className="p-3"><Badge variant="info">{Array.isArray(t.permissions) ? t.permissions.length : 0}</Badge></td>
                <td className="p-3"><StatusBadge status={t.is_active ? 'ACTIVE' : 'HALTED'} /></td>
                <td className="p-3 text-right"><Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete({ type: 'template', id: t.id })} data-testid={`permissions-delete-template-${t.id}`}>Delete</Button></td>
              </tr>
            ))}
          </Table>
        )
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => {
          if (confirmDelete!.type === 'registry') delReg.mutate(confirmDelete!.id)
          else delTpl.mutate(confirmDelete!.id)
        }}
        title="Delete Entry"
        message="Are you sure you want to delete this entry?"
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={delReg.isPending || delTpl.isPending}
        error={error}
        testId="permissions-delete-modal"
      />
    </main>
  )
}
