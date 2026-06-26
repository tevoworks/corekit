import { useQuery } from '@tanstack/react-query'
import api from '../../lib/api'
import { useAuth } from '../../lib/auth'
import { Card, StatCard, PageHeader, LoadingSkeleton, EmptyState } from '../../components/ui'

export default function DashboardPage() {
  const { user } = useAuth()
  const { data: users, isLoading: loadUsers } = useQuery({ queryKey: ['users'], queryFn: () => api.get('/api/users?limit=1').then(r => ({ total: r.data.meta?.total ?? r.data.data?.length ?? 0 })) })
  const { data: roles, isLoading: loadRoles } = useQuery({ queryKey: ['roles'], queryFn: () => api.get('/api/roles').then(r => r.data.data || []) })
  const { data: audit, isLoading: loadAudit } = useQuery({ queryKey: ['audit-logs'], queryFn: () => api.get('/api/audit-logs?limit=5').then(r => r.data.data || []) })

  const stats = [
    { label: 'Total Users', value: users?.total ?? 0, icon: 'people', color: 'bg-blue-50 text-blue-600' },
    { label: 'Roles', value: roles?.length || 0, icon: 'manage_accounts', color: 'bg-purple-50 text-purple-600' },
    { label: 'Your Role', value: user?.role_name || '—', icon: 'badge', color: 'bg-emerald-50 text-emerald-600' },
    { label: 'Status', value: user?.status || '—', icon: 'circle', color: 'bg-amber-50 text-amber-600' },
  ]

  const isLoading = loadUsers || loadRoles

  const actionColors: Record<string, string> = {
    CREATE: 'text-green-600', DELETE: 'text-red-600', UPDATE: 'text-blue-600',
    LOGIN: 'text-purple-600', SESSION: 'text-orange-600',
  }

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Dashboard"
        description={`Welcome back, ${user?.full_name || 'User'}`}
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => (
              <Card key={i} testId={`dashboard-stat-loading-${i}`}>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-[var(--surface-container)] animate-pulse" />
                  <div className="space-y-2 flex-1">
                    <div className="h-3 bg-[var(--surface-container)] rounded animate-pulse w-16" />
                    <div className="h-5 bg-[var(--surface-container)] rounded animate-pulse w-12" />
                  </div>
                </div>
              </Card>
            ))
          : stats.map((s) => (
              <StatCard key={s.label} {...s} testId={`dashboard-stat-${s.label.toLowerCase().replace(/\s+/g, '-')}`} />
            ))
        }
      </div>

      <Card testId="dashboard-activity-card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-[var(--on-surface)]" data-testid="dashboard-recent-activity-heading">Recent Activity</h2>
          <span className="text-xs text-[var(--on-surface-variant)]">Last 5 entries</span>
        </div>
        {loadAudit ? (
          <LoadingSkeleton rows={5} testId="dashboard-activity-loading" />
        ) : !audit || audit.length === 0 ? (
          <EmptyState icon="history" title="No recent activity" description="Actions performed in the system will appear here." testId="dashboard-empty-activity" />
        ) : (
          <div className="space-y-1">
            {audit.map((log: any, idx: number) => {
              const action = (log.action || '').split('_')[0]
              return (
                <div key={log.id} className="flex items-center gap-3 py-2.5 px-2 rounded-lg hover:bg-[var(--surface-container-low)] transition-colors -mx-2" data-testid={`dashboard-activity-row-${idx}`}>
                  <div className={`w-8 h-8 rounded-lg flex items-center justify-center bg-[var(--surface-container)] ${actionColors[action] || 'text-[var(--on-surface-variant)]'}`}>
                    <span className="material-symbols-outlined text-lg">history</span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <span className="text-sm font-medium text-[var(--on-surface)]">{log.action}</span>
                    <span className="text-sm text-[var(--on-surface-variant)] ml-1">on {log.target_entity}</span>
                  </div>
                  <span className="text-xs text-[var(--on-surface-variant)] whitespace-nowrap">{new Date(log.created_at).toLocaleString()}</span>
                </div>
              )
            })}
          </div>
        )}
      </Card>
    </main>
  )
}
