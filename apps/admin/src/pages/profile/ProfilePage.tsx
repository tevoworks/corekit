import { useAuth } from '../../lib/auth'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import { useState, useEffect, useRef } from 'react'
import type { Session, UserPreference } from '../../lib/types'
import { Card, PageHeader, Button, Input, StatusBadge, Modal, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

function useToast() {
  const [toast, setToast] = useState<{ type: 'success' | 'error' | 'info'; message: string } | null>(null)
  const toastRef = useRef(toast)
  toastRef.current = toast

  useEffect(() => {
    if (!toast) return
    if (toast.type === 'success' || toast.type === 'info') {
      const t = setTimeout(() => setToast(null), 3000)
      return () => clearTimeout(t)
    }
  }, [toast])

  return { toast, setToast }
}

export default function ProfilePage() {
  const { user, setUser: setAuthUser, logout } = useAuth()
  const qc = useQueryClient()
  const { toast, setToast } = useToast()

  const [fullName, setFullName] = useState(user?.full_name || '')
  const [email, setEmail] = useState(user?.email || '')
  const [confirmLogoutAll, setConfirmLogoutAll] = useState(false)
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null)
  const [dialogError, setDialogError] = useState<string | null>(null)

  const isDirty = fullName !== (user?.full_name || '') || email !== (user?.email || '')

  const { data: sessions = [], isLoading: loadSessions } = useQuery({
    queryKey: ['my-sessions'],
    queryFn: () => api.get('/api/sessions').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const { data: prefs = [], isLoading: loadPrefs } = useQuery({
    queryKey: ['my-preferences'],
    queryFn: () => api.get('/api/preferences').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const updateProfile = useMutation({
    mutationFn: (body: any) => api.patch('/api/profile', body),
    onSuccess: (res) => {
      const u = res.data.data
      setAuthUser(u)
      setFullName(u.full_name || '')
      setEmail(u.email || '')
      setToast({ type: 'success', message: 'Profile updated' })
    },
    onError: (err: any) => setToast({ type: 'error', message: err.response?.data?.error?.message || 'Failed to update profile' }),
  })

  const revokeSession = useMutation({
    mutationFn: (tokenId: string) => api.delete(`/api/sessions/${tokenId}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-sessions'] }); setToast({ type: 'success', message: 'Session revoked' }); setConfirmRevoke(null) },
    onError: (err: any) => setDialogError(err.response?.data?.error?.message || 'Failed to revoke session'),
  })

  const logoutAll = useMutation({
    mutationFn: () => api.post('/api/logout-all'),
    onSuccess: () => { qc.clear(); setAuthUser(null); logout() },
    onError: (err: any) => setDialogError(err.response?.data?.error?.message || 'Failed to logout all sessions'),
  })

  return (
    <main className="content-canvas animate-fade-in">
      {toast && (
        <div role="alert" className={`mb-4 p-3 rounded-lg text-sm flex items-center gap-2 ${
          toast.type === 'success' ? 'bg-[var(--success-bg)] text-[var(--success-text)]' 
          : toast.type === 'error' ? 'bg-[var(--danger-bg)] text-[var(--danger-text)]'
          : 'bg-[var(--info-bg)] text-[var(--info-text)]'
        }`} data-testid="profile-toast">
          <span className="material-symbols-outlined text-base">{toast.type === 'success' ? 'check_circle' : toast.type === 'error' ? 'error' : 'info'}</span>
          <span className="flex-1">{toast.message}</span>
          {toast.type === 'error' && (
            <button onClick={() => setToast(null)} className="opacity-60 hover:opacity-100" aria-label="Dismiss" data-testid="profile-toast-dismiss-button"><span className="material-symbols-outlined text-base">close</span></button>
          )}
        </div>
      )}

      <PageHeader title="Profile" description="Manage your account" />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <section aria-label="Account details form">
            <Card padding={false} testId="profile-account-details-card">
              <div className="p-6 pb-0">
                <h2 className="text-lg font-semibold text-[var(--on-surface)]">Account Details</h2>
                <p className="text-xs text-[var(--on-surface-variant)] mt-1 mb-4">Used in audit logs and reports.</p>
              </div>
              <div className="px-6 pb-24">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-[480px]">
                  <Input label="Full Name *" value={fullName} onChange={e => setFullName(e.target.value)} data-testid="profile-name-input" />
                  <Input label="Email *" type="email" value={email} onChange={e => setEmail(e.target.value)} data-testid="profile-email-input" />
                </div>
              </div>
              <div className="sticky bottom-0 bg-white border-t border-[var(--outline-variant)] rounded-b-lg px-6 py-4 flex items-center justify-end gap-2">
                <Button variant="ghost" onClick={() => { setFullName(user?.full_name || ''); setEmail(user?.email || '') }} data-testid="profile-cancel-button">Cancel</Button>
                <Button onClick={() => updateProfile.mutate({ full_name: fullName, email })} loading={updateProfile.isPending} disabled={!isDirty} data-testid="profile-save-button">
                  Save Changes
                </Button>
              </div>
            </Card>
          </section>

          <section aria-label="Active sessions">
            <Card testId="profile-sessions-card">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold">Active Sessions</h2>
                {sessions.length > 1 && (
                  <Button variant="danger" size="sm" icon="devices" onClick={() => setConfirmLogoutAll(true)} data-testid="profile-logout-all-button">
                    Logout All
                  </Button>
                )}
              </div>
              {loadSessions ? (
                <LoadingSkeleton rows={3} testId="profile-sessions-loading" />
              ) : sessions.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 text-center" data-testid="profile-sessions-empty">
                  <span className="material-symbols-outlined text-3xl text-[var(--on-surface-variant)] mb-2 opacity-40">devices</span>
                  <p className="text-sm text-[var(--on-surface-variant)]">No active sessions.</p>
                  <p className="text-xs text-[var(--on-surface-variant)] mt-1 opacity-60">Sessions appear when you log in from a new device.</p>
                </div>
              ) : (
                <div className="divide-y divide-[var(--outline-variant)]">
                  {sessions.map((s: Session) => (
                    <div key={s.id} className="flex items-center justify-between py-3 gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-[var(--on-surface)] truncate">{s.user_agent || 'Unknown device'}</div>
                        <div className="text-xs text-[var(--on-surface-variant)]">{s.ip_address} &middot; <time>{new Date(s.created_at).toLocaleString()}</time></div>
                      </div>
                      {!s.revoked_at && (
                        <Button variant="ghost" size="sm" icon="logout" className="text-[var(--danger)] shrink-0" onClick={() => setConfirmRevoke(s.token_id)} data-testid={`profile-revoke-session-${s.id}`}>
                          Revoke
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </section>
        </div>

        <div className="space-y-6">
          <Card>
            <h2 className="text-lg font-semibold mb-3">Account Info</h2>
            <div className="space-y-3">
              <div>
                <div className="text-xs text-[var(--on-surface-variant)] mb-0.5">Status</div>
                <div><StatusBadge status={user?.status || '—'} /></div>
              </div>
              <div>
                <div className="text-xs text-[var(--on-surface-variant)] mb-0.5">Role</div>
                <div className="text-sm font-medium">{user?.role_name || '—'}</div>
              </div>
              <div>
                <div className="text-xs text-[var(--on-surface-variant)] mb-0.5">Super Admin</div>
                <div className="text-sm font-medium">{user?.is_super_admin ? 'Yes' : 'No'}</div>
              </div>
              <div>
                <div className="text-xs text-[var(--on-surface-variant)] mb-0.5">Joined</div>
                <div className="text-sm font-medium"><time>{user?.created_at ? new Date(user.created_at).toLocaleDateString() : '—'}</time></div>
              </div>
            </div>
          </Card>

          <Card>
            <h2 className="text-lg font-semibold mb-3">Preferences</h2>
            {loadPrefs ? (
              <LoadingSkeleton rows={2} testId="profile-preferences-loading" />
            ) : prefs.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-6 text-center" data-testid="profile-preferences-empty">
                <span className="material-symbols-outlined text-3xl text-[var(--on-surface-variant)] mb-2 opacity-40">tune</span>
                <p className="text-sm text-[var(--on-surface-variant)]">No preferences set.</p>
                <p className="text-xs text-[var(--on-surface-variant)] mt-1 opacity-60">Preferences help personalize your experience.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {prefs.map((p: UserPreference) => (
                  <div key={p.key}>
                    <div className="text-xs text-[var(--on-surface-variant)] mb-0.5">{p.key}</div>
                    <div className="text-sm font-medium">{p.value}</div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={confirmLogoutAll}
        onClose={() => { setConfirmLogoutAll(false); setDialogError(null) }}
        onConfirm={() => logoutAll.mutate()}
        title="Logout All Devices"
        message="This will revoke all your active sessions except the current one."
        confirmLabel="Yes, Logout All"
        loading={logoutAll.isPending}
        error={dialogError}
        testId="profile-logout-all-modal"
      />

      <ConfirmDialog
        open={!!confirmRevoke}
        onClose={() => { setConfirmRevoke(null); setDialogError(null) }}
        onConfirm={() => revokeSession.mutate(confirmRevoke!)}
        title="Revoke Session"
        message="This will log the device out immediately. You cannot undo this action."
        confirmLabel="Yes, Revoke"
        loading={revokeSession.isPending}
        error={dialogError}
        testId="profile-revoke-modal"
      />
    </main>
  )
}
