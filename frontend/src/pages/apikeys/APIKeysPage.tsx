import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { APIKey } from '../../lib/types'
import { useState, useEffect, useRef } from 'react'
import { Card, PageHeader, Table, Button, EmptyState, StatusBadge, LoadingSkeleton, Modal, ConfirmDialog } from '../../components/ui'

export default function APIKeysPage() {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [newKey, setNewKey] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [confirmRevoke, setConfirmRevoke] = useState<number | null>(null)
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const clearNewKey = () => {
    if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current)
    setNewKey('')
  }

  useEffect(() => {
    if (newKey) {
      dismissTimerRef.current = setTimeout(clearNewKey, 15000)
    }
    return () => {
      if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current)
    }
  }, [newKey])

  const { data: keys = [], isLoading } = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => api.get('/api/api-keys').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/api-keys', body),
  })

  const revokeMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/api-keys/${id}`),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="API Keys" description="Manage programmatic access" />

      <Card className="mb-6" testId="api-keys-create-card">
        <h2 className="text-base font-semibold mb-4">Create New Key</h2>
        <div className="flex gap-2 items-end max-w-md">
          <div className="flex-1">
            <label className="block text-xs font-medium text-[var(--on-surface)] mb-1">Key Name</label>
            <input
              type="text" placeholder="e.g. Production API Key" value={name}
              onChange={e => { setName(e.target.value); setNewKey('') }}
              className="w-full px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
              data-testid="api-keys-name-input"
            />
          </div>
          <Button icon="vpn_key" onClick={() => createMutation.mutate({ name }, {
              onSuccess: (res) => {
                qc.invalidateQueries({ queryKey: ['api-keys'] })
                setNewKey(res.data.data.raw_key || 'Key created (copy now)')
                setName('')
              },
              onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create API key'),
            })} disabled={!name.trim()} data-testid="api-keys-generate-button">Generate</Button>
        </div>
        {newKey && (
          <div className="mt-4 p-4 rounded-lg bg-amber-50 border border-amber-200" data-testid="api-keys-new-key-banner">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-amber-800 flex items-center gap-1">
                  <span className="material-symbols-outlined text-base">warning</span>
                  Copy this key now. It will not be shown again.
                </p>
                <div className="mt-2 p-2.5 bg-white rounded border border-amber-200 font-mono text-xs break-all select-all text-[var(--on-surface)]">{newKey}</div>
              </div>
              <button onClick={clearNewKey} className="shrink-0 w-6 h-6 flex items-center justify-center rounded hover:bg-amber-100 text-amber-600" data-testid="api-keys-new-key-dismiss-button">
                <span className="material-symbols-outlined text-sm">close</span>
              </button>
            </div>
          </div>
        )}
      </Card>

      {isLoading ? (
        <Card testId="api-keys-loading-card"><LoadingSkeleton rows={5} testId="api-keys-loading" /></Card>
      ) : keys.length === 0 ? (
        <Card testId="api-keys-empty-card"><EmptyState icon="key" title="No API keys yet" description="Generate your first API key above." testId="api-keys-empty-state" /></Card>
      ) : (
        <Table headers={['Name', 'Prefix', 'Status', 'Last Used', 'Created', 'Actions']} testId="api-keys-table">
          {keys.map((k: APIKey) => (
            <tr key={k.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
              <td className="p-3 font-medium">{k.name}</td>
              <td className="p-3 font-mono text-xs text-[var(--on-surface-variant)]">{k.key_prefix}...</td>
              <td className="p-3"><StatusBadge status={k.revoked_at ? 'Revoked' : 'Active'} /></td>
              <td className="p-3 text-xs text-[var(--on-surface-variant)]">{k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'Never'}</td>
              <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(k.created_at).toLocaleDateString()}</td>
              <td className="p-3 text-right">
                {!k.revoked_at && (
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmRevoke(k.id)} data-testid={`api-keys-revoke-${k.id}`}>Revoke</Button>
                )}
              </td>
            </tr>
          ))}
        </Table>
      )}

      {confirmRevoke && (
        <ConfirmDialog
          open={!!confirmRevoke}
          onClose={() => { setConfirmRevoke(null); setError(null) }}
          onConfirm={() => revokeMutation.mutate(confirmRevoke, {
              onSuccess: () => { qc.invalidateQueries({ queryKey: ['api-keys'] }); setConfirmRevoke(null); setError(null) },
              onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to revoke API key'),
            })}
          title="Revoke API Key"
          message="This will immediately revoke the API key. All applications and services using this key will lose access."
          confirmLabel="Yes, Revoke Key"
          loading={revokeMutation.isPending}
          error={error}
          testId="api-keys-revoke-confirm-dialog"
        />
      )}
    </main>
  )
}
