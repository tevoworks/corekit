import { useQuery } from '@tanstack/react-query'
import api from '../../lib/api'
import type { AuditLog } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, Badge, EmptyState, LoadingSkeleton } from '../../components/ui'

export default function AuditPage() {
  const [actorFilter, setActorFilter] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const buildUrl = () => {
    const params = new URLSearchParams()
    if (actorFilter) params.set('actor_id', actorFilter)
    if (actionFilter) params.set('action', actionFilter)
    if (dateFrom) params.set('date_from', dateFrom + 'T00:00:00Z')
    if (dateTo) params.set('date_to', dateTo + 'T23:59:59Z')
    params.set('limit', '100')
    return `/api/audit-logs?${params.toString()}`
  }

  const { data: logs = [], isLoading, refetch } = useQuery({
    queryKey: ['audit-logs', actorFilter, actionFilter, dateFrom, dateTo],
    queryFn: () => api.get(buildUrl()).then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Audit Log"
        description="Track all system mutations"
        action={<Button icon="refresh" variant="secondary" onClick={() => refetch()} data-testid="audit-refresh-button">Refresh</Button>}
      />

      <Card className="mb-6" padding={false}>
        <div className="p-4 flex flex-wrap items-end gap-3">
          <div className="space-y-1 min-w-[120px]">
            <label className="block text-xs font-medium text-[var(--on-surface)]">Actor ID</label>
            <input
              type="text" value={actorFilter} onChange={e => setActorFilter(e.target.value)}
              placeholder="e.g. 1"
              className="w-full px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
              data-testid="audit-actor-input"
            />
          </div>
          <div className="space-y-1 min-w-[140px]">
            <label className="block text-xs font-medium text-[var(--on-surface)]">Action</label>
            <input
              type="text" value={actionFilter} onChange={e => setActionFilter(e.target.value)}
              placeholder="e.g. LOGIN"
              className="w-full px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
              data-testid="audit-action-input"
            />
          </div>
          <div className="space-y-1">
            <label className="block text-xs font-medium text-[var(--on-surface)]">From</label>
            <input type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)}
              className="px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent" data-testid="audit-date-from-input" />
          </div>
          <div className="space-y-1">
            <label className="block text-xs font-medium text-[var(--on-surface)]">To</label>
            <input type="date" value={dateTo} onChange={e => setDateTo(e.target.value)}
              className="px-3 py-2 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent" data-testid="audit-date-to-input" />
          </div>
          {(actorFilter || actionFilter || dateFrom || dateTo) && (
            <Button variant="ghost" size="sm" onClick={() => { setActorFilter(''); setActionFilter(''); setDateFrom(''); setDateTo('') }} data-testid="audit-clear-button">Clear</Button>
          )}
        </div>
      </Card>

      {isLoading ? (
        <Card testId="audit-loading-card"><LoadingSkeleton rows={8} testId="audit-loading" /></Card>
      ) : logs.length === 0 ? (
        <Card testId="audit-empty-card"><EmptyState icon="history" title="No audit logs found" description="Try adjusting your filters." testId="audit-empty-state" /></Card>
      ) : (
        <Table headers={['Time', 'Actor', 'Action', 'Target']} testId="audit-table">
          {logs.map((log: AuditLog) => (
            <tr key={log.id} className="hover:bg-[var(--surface-container-low)] transition-colors">
              <td className="p-3 text-xs text-[var(--on-surface-variant)] whitespace-nowrap">{new Date(log.created_at).toLocaleString()}</td>
              <td className="p-3">{log.actor_name || log.actor_email || <span className="text-[var(--on-surface-variant)]">#{log.actor_id}</span>}</td>
              <td className="p-3"><Badge variant="neutral">{log.action}</Badge></td>
              <td className="p-3 text-[var(--on-surface-variant)]">{log.target_entity}</td>
            </tr>
          ))}
        </Table>
      )}
    </main>
  )
}
