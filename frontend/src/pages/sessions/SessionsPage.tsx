import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import { useAuth } from '../../lib/auth'
import type { Session } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function SessionsPage() {
  const { user } = useAuth()
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null)

  if (!user?.is_super_admin) {
    return (
      <main className="content-canvas animate-fade-in">
        <div className="flex flex-col items-center justify-center py-20 text-center" data-testid="sessions-access-denied">
          <span className="material-symbols-outlined text-5xl text-[var(--danger)] mb-4">lock</span>
          <h1 className="text-xl font-bold text-[var(--on-surface)] mb-2">Access Denied</h1>
          <p className="text-sm text-[var(--on-surface-variant)]">You do not have permission to view this page.</p>
        </div>
      </main>
    )
  }

  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['sessions-all'],
    queryFn: () => api.get('/api/sessions/all').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const revokeMutation = useMutation({
    mutationFn: (tokenId: string) => api.delete(`/api/sessions/all/${tokenId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sessions-all'] }),
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to revoke session'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="All Sessions" description="Manage active user sessions" />

      {isLoading ? (
        <Card testId="sessions-loading-card"><LoadingSkeleton rows={6} testId="sessions-loading" /></Card>
      ) : sessions.length === 0 ? (
        <Card testId="sessions-empty-card"><EmptyState icon="devices" title="No sessions found" description="Active sessions will appear here when users log in." testId="sessions-empty-state" /></Card>
      ) : (
        <Table headers={['User', 'IP Address', 'Device', 'Created', 'Expires', 'Status', 'Actions']} testId="sessions-table">
          {sessions.map((s: Session) => {
            const isExpired = new Date(s.expires_at) < new Date()
            const status = s.revoked_at ? 'failed' : isExpired ? 'warning' : 'ACTIVE'
            return (
              <tr key={s.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
                <td className="p-3 font-medium">#{s.user_id}</td>
                <td className="p-3 font-mono text-xs text-[var(--on-surface-variant)]">{s.ip_address}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)] max-w-[200px] truncate">{s.user_agent || '—'}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(s.created_at).toLocaleString()}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(s.expires_at).toLocaleString()}</td>
                <td className="p-3"><StatusBadge status={status} /></td>
                <td className="p-3 text-right">
                  {!s.revoked_at && (
                    <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmRevoke(s.token_id)} data-testid={`sessions-revoke-${s.id}`}>Revoke</Button>
                  )}
                </td>
              </tr>
            )
          })}
        </Table>
      )}

      <ConfirmDialog
        open={confirmRevoke !== null}
        onClose={() => setConfirmRevoke(null)}
        onConfirm={() => revokeMutation.mutate(confirmRevoke!)}
        title="Revoke Session"
        message="Are you sure you want to revoke this session?"
        confirmLabel="Revoke"
        confirmVariant="danger"
        loading={revokeMutation.isPending}
        error={error}
        testId="sessions-revoke-modal"
      />
    </main>
  )
}
