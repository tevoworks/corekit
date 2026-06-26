import type { ReactNode, ButtonHTMLAttributes } from 'react'

export function Card({ children, className = '', padding = true, testId }: { children: ReactNode; className?: string; padding?: boolean; testId?: string }) {
  return (
    <div data-testid={testId} className={`bg-white rounded-lg border border-[var(--outline-variant)] ${padding ? 'p-6' : ''} ${className}`}>
      {children}
    </div>
  )
}

export function PageHeader({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="flex items-start justify-between mb-6">
      <div>
        <h1 className="text-3xl font-bold text-[var(--on-surface)]">{title}</h1>
        {description && <p className="text-sm text-[var(--on-surface-variant)] mt-1">{description}</p>}
      </div>
      {action && <div className="flex items-center gap-2 shrink-0 ml-4">{action}</div>}
    </div>
  )
}

type BadgeVariant = 'success' | 'warning' | 'danger' | 'info' | 'neutral' | 'primary'
const badgeVariants: Record<BadgeVariant, string> = {
  success: 'bg-[var(--success-bg)] text-[var(--success-text)]',
  warning: 'bg-[var(--warning-bg)] text-[var(--warning-text)]',
  danger: 'bg-[var(--danger-bg)] text-[var(--danger-text)]',
  info: 'bg-[var(--info-bg)] text-[var(--info-text)]',
  neutral: 'bg-[var(--surface-container)] text-[var(--on-surface-variant)]',
  primary: 'bg-blue-100 text-blue-700',
}

export function Badge({ children, variant = 'neutral', className = '' }: { children: ReactNode; variant?: BadgeVariant; className?: string }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${badgeVariants[variant]} ${className}`}>
      {children}
    </span>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const variant: BadgeVariant =
    status === 'ACTIVE' || status === 'Active' || status === 'done' || status === 'delivered' ? 'success'
    : status === 'failed' || status === 'SUSPENDED' || status === 'Revoked' ? 'danger'
    : status === 'HALTED' || status === 'BANNED' ? 'danger'
    : status === 'processing' || status === 'pending' ? 'warning'
    : 'info'
  return <Badge variant={variant}>{status}</Badge>
}

export function StatCard({ label, value, icon, color = 'bg-blue-50 text-blue-600', testId }: { label: string; value: ReactNode; icon: string; color?: string; testId?: string }) {
  return (
    <Card testId={testId}>
      <div className="flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${color}`}>
          <span className="material-symbols-outlined text-xl">{icon}</span>
        </div>
        <div className="min-w-0">
          <div className="text-xs text-[var(--on-surface-variant)] truncate">{label}</div>
          <div className="text-lg font-bold text-[var(--on-surface)] truncate">{value}</div>
        </div>
      </div>
    </Card>
  )
}

export function EmptyState({ icon = 'inbox', title, description, action, testId }: { icon?: string; title: string; description?: string; action?: ReactNode; testId?: string }) {
  return (
    <div data-testid={testId} className="flex flex-col items-center justify-center py-16 text-center">
      <span className="material-symbols-outlined text-5xl text-[var(--on-surface-variant)] mb-4 opacity-40">{icon}</span>
      <h3 className="text-sm font-semibold text-[var(--on-surface)] mb-1">{title}</h3>
      {description && <p className="text-xs text-[var(--on-surface-variant)] mb-4 max-w-xs">{description}</p>}
      {action}
    </div>
  )
}

export function LoadingSkeleton({ rows = 3, testId }: { rows?: number; testId?: string }) {
  return (
    <div data-testid={testId} className="space-y-3 p-4">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="h-4 bg-[var(--surface-container)] rounded animate-pulse" style={{ width: `${60 + Math.random() * 40}%` }} />
      ))}
    </div>
  )
}

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost'
const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-[var(--primary)] text-white hover:opacity-90',
  secondary: 'bg-white border border-[var(--outline-variant)] text-[var(--on-surface)] hover:bg-[var(--surface-container)]',
  danger: 'bg-[var(--danger)] text-white hover:opacity-90',
  ghost: 'text-[var(--on-surface-variant)] hover:text-[var(--on-surface)] hover:bg-[var(--surface-container)]',
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: 'sm' | 'md'
  loading?: boolean
  icon?: string
}

export function Button({ children, variant = 'primary', size = 'md', loading, icon, className = '', ...props }: ButtonProps) {
  const sizeClass = size === 'sm' ? 'px-3 py-1.5 text-xs' : 'px-4 py-2 text-sm'
  return (
    <button
      className={`inline-flex items-center gap-1.5 rounded-lg font-medium transition-all duration-150 disabled:opacity-50 ${buttonVariants[variant]} ${sizeClass} ${className}`}
      disabled={loading || props.disabled}
      {...props}
    >
      {loading ? (
        <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
      ) : icon ? (
        <span className="material-symbols-outlined text-lg">{icon}</span>
      ) : null}
      {children}
    </button>
  )
}

export function Table({ headers, children, className = '', testId }: { headers: ReactNode[]; children: ReactNode; className?: string; testId?: string }) {
  return (
    <div data-testid={testId} className={`bg-white rounded-lg border border-[var(--outline-variant)] overflow-hidden ${className}`}>
      <table className="w-full text-sm">
        <thead className="sticky top-0 z-10">
          <tr className="border-b bg-[var(--surface-container)]">
            {headers.map((h, i) => (
              <th key={i} className={`p-3 font-medium text-[var(--on-surface-variant)] text-xs uppercase tracking-wider ${i === headers.length - 1 ? 'text-right' : 'text-left'}`}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--outline-variant)]">
          {children}
        </tbody>
      </table>
    </div>
  )
}

export function Modal({ title, children, onClose, size = 'md', testId }: { title: string; children: ReactNode; onClose: () => void; size?: 'sm' | 'md' | 'lg'; testId?: string }) {
  const widthClass = size === 'sm' ? 'max-w-sm' : size === 'lg' ? 'max-w-2xl' : 'max-w-md'
  return (
    <div data-testid={testId} className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm" role="dialog" aria-modal="true" onClick={onClose}>
      <div className={`bg-white shadow-2xl w-full ${widthClass} mx-4 max-h-[90vh] overflow-y-auto animate-in zoom-in-95 duration-200`} onClick={e => e.stopPropagation()} style={{ borderRadius: '20px' }}>
        <div className="flex items-center justify-between px-6 pt-6 pb-2">
          <h2 className="text-lg font-semibold text-[var(--on-surface)] tracking-tight">{title}</h2>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-xl hover:bg-[var(--surface-container)] text-[var(--on-surface-variant)] transition-colors" data-testid="modal-close-button">
            <span className="material-symbols-outlined text-xl">close</span>
          </button>
        </div>
        <div className="px-6 pb-6">
          {children}
        </div>
      </div>
    </div>
  )
}

export function ConfirmDialog({ open, onClose, onConfirm, title, message, confirmLabel = 'Confirm', confirmVariant = 'danger', loading, error, testId }: {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title?: string
  message: string
  confirmLabel?: string
  confirmVariant?: ButtonVariant
  loading?: boolean
  error?: string | null
  testId?: string
}) {
  if (!open) return null
  return (
    <Modal title={title || 'Confirm Action'} onClose={onClose} size="sm" testId={testId}>
      <div className="flex flex-col items-center text-center pt-2 pb-4">
        <div className="w-10 h-10 rounded-xl bg-[var(--danger-bg)] flex items-center justify-center mb-4">
          <span className="material-symbols-outlined text-2xl text-[var(--danger-text)]">gpp_maybe</span>
        </div>
        <p className="text-sm leading-relaxed text-[var(--on-surface-variant)] max-w-xs">{message}</p>
      </div>
      {error && (
        <div className="mb-2 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm text-center" data-testid={`${testId}-error`}>
          {error}
        </div>
      )}
      <div className="flex gap-3 justify-end pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onClose} className="px-5">Cancel</Button>
        <Button variant={confirmVariant} onClick={onConfirm} loading={loading} className="px-5">{confirmLabel}</Button>
      </div>
    </Modal>
  )
}

let inputIdCounter = 0

export function Input({ label, error, id, ...props }: { label: string; error?: string; id?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  const inputId = id || `input-${++inputIdCounter}`
  return (
    <div className="space-y-1">
      <label htmlFor={inputId} className="block text-sm font-medium text-[var(--on-surface)]">{label}</label>
      <input id={inputId} className={`w-full px-3 py-3 rounded-lg border text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all ${error ? 'border-red-400' : 'border-[var(--outline-variant)]'}`} {...props} />
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  )
}
