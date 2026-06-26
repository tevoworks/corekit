import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../lib/auth'
import { Card, Button, Input } from '../../components/ui'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const registered = searchParams.get('registered')
  const expired = searchParams.get('expired')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setSubmitting(true)
    setError('')
    try {
      await login(email, password)
      navigate('/')
    } catch (err: any) {
      if (err.response?.status === 429) {
        setError('Too many attempts. Please try again later.')
      } else if (err.response?.status === 401) {
        setError('Invalid email or password')
      } else if (err.code === 'ERR_NETWORK') {
        setError('Unable to connect to server. Please check your connection.')
      } else {
        setError('An unexpected error occurred. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-[#f0f4ff] to-[#faf8ff] p-4">
      <div className="w-full max-w-sm animate-fade-in">
        <div className="text-center mb-8">
          <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-[var(--primary)] to-[var(--primary-container)] text-white flex items-center justify-center text-xl font-bold mx-auto mb-3 shadow-lg shadow-blue-200">
            C
          </div>
          <h1 className="text-3xl font-bold text-[var(--on-surface)]">CoreKit</h1>
          <p className="text-sm text-[var(--on-surface-variant)] mt-1">Sign in to your admin console</p>
        </div>

        <Card className="shadow-sm" padding={false} testId="login-card">
          <form onSubmit={handleSubmit} className="p-6 space-y-4" data-testid="login-form">
            {registered && (
              <div className="p-3 rounded-lg bg-[var(--success-bg)] text-[var(--success-text)] text-sm flex items-center gap-2" data-testid="login-success-banner">
                <span className="material-symbols-outlined text-base">check_circle</span>
                Registration successful! You can now sign in.
              </div>
            )}
            {expired && (
              <div className="p-3 rounded-lg bg-[var(--warning-bg)] text-[var(--warning-text)] text-sm flex items-center gap-2" data-testid="login-expired-banner">
                <span className="material-symbols-outlined text-base">timer_off</span>
                Your session has expired. Please sign in again.
              </div>
            )}
            {error && (
              <div className="p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="login-error-banner">
                <span className="material-symbols-outlined text-base">error</span>
                {error}
              </div>
            )}

            <Input label="Email *" type="email" value={email} onChange={e => setEmail(e.target.value.trim())} placeholder="admin@example.com" autoComplete="email" required data-testid="login-email-input" />
            <Input label="Password *" type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Enter your password" autoComplete="current-password" required data-testid="login-password-input" />

            <Button type="submit" loading={submitting} className="w-full justify-center py-3" data-testid="login-sign-in-button">
              {submitting ? 'Signing in...' : 'Sign in'}
            </Button>
          </form>
        </Card>

        <p className="text-center text-xs text-[var(--on-surface-variant)] mt-6 opacity-60">
          CoreKit Admin Console
        </p>
      </div>
    </div>
  )
}
