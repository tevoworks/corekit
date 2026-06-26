import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Job } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function JobsPage() {
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  const [confirmCancel, setConfirmCancel] = useState<number | null>(null)

  const { data: jobs = [], isLoading } = useQuery({
    queryKey: ['jobs'],
    queryFn: () => api.get('/api/jobs').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const retryMutation = useMutation({
    mutationFn: (id: number) => api.post(`/api/jobs/${id}/retry`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['jobs'] }),
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to retry job'),
  })

  const cancelMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/jobs/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['jobs'] }); setConfirmCancel(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to cancel job'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Background Jobs" description="Monitor and manage queue jobs" />

      {isLoading ? (
        <Card testId="jobs-loading-card"><LoadingSkeleton rows={5} testId="jobs-loading" /></Card>
      ) : jobs.length === 0 ? (
        <Card testId="jobs-empty-card"><EmptyState icon="assignment" title="No jobs found" description="Background jobs will appear here when they are queued." testId="jobs-empty-state" /></Card>
      ) : (
        <Table headers={['Type', 'Status', 'Retries', 'Error', 'Created', 'Actions']} testId="jobs-table">
          {jobs.map((j: Job) => (
            <tr key={j.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
              <td className="p-3 font-mono text-xs">{j.type}</td>
              <td className="p-3"><StatusBadge status={j.status} /></td>
              <td className="p-3 text-xs text-[var(--on-surface-variant)]">{j.retry_count}/{j.max_retries}</td>
              <td className="p-3 text-xs text-[var(--danger)] max-w-[200px] truncate">{j.error_message || <span className="text-[var(--on-surface-variant)]">—</span>}</td>
              <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(j.created_at).toLocaleString()}</td>
              <td className="p-3 text-right">
                {j.status === 'failed' && (
                  <Button variant="ghost" size="sm" icon="refresh" onClick={() => retryMutation.mutate(j.id)} data-testid={`jobs-retry-${j.id}`} className="mr-1">Retry</Button>
                )}
                {(j.status === 'pending' || j.status === 'processing') && (
                  <Button variant="danger" size="sm" icon="cancel" onClick={() => setConfirmCancel(j.id)} data-testid={`jobs-cancel-${j.id}`}>Cancel</Button>
                )}
              </td>
            </tr>
          ))}
        </Table>
      )}

      <ConfirmDialog
        open={confirmCancel !== null}
        onClose={() => setConfirmCancel(null)}
        onConfirm={() => cancelMutation.mutate(confirmCancel!)}
        title="Cancel Job"
        message="Are you sure you want to cancel this job? It will be removed from the queue."
        confirmLabel="Delete Job"
        confirmVariant="danger"
        loading={cancelMutation.isPending}
        error={error}
        testId="jobs-cancel-modal"
      />
    </main>
  )
}
