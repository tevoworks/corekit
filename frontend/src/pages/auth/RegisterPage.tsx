import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../../lib/auth'
import { Card, Button, Input } from '../../components/ui'

export default function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setSubmitting(true)
    setError('')
    try {
      const msg = await register(email, password, fullName)
      navigate(msg ? '/login?registered=1' : '/')
    } catch {
      setError('Registration failed. Please try again.')
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
          <h1 className="text-3xl font-bold text-[var(--on-surface)]">Create Account</h1>
          <p className="text-sm text-[var(--on-surface-variant)] mt-1">Register for CoreKit</p>
        </div>

        <Card className="shadow-sm" padding={false} testId="register-card">
          <form onSubmit={handleSubmit} className="p-6 space-y-4" data-testid="register-form">
            {error && (
              <div className="p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="register-error-banner">
                <span className="material-symbols-outlined text-base">error</span>
                {error}
              </div>
            )}

            <Input label="Full Name" value={fullName} onChange={e => setFullName(e.target.value)} required data-testid="register-name-input" />
            <Input label="Email" type="email" value={email} onChange={e => setEmail(e.target.value.trim())} required data-testid="register-email-input" />
            <Input label="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} required data-testid="register-password-input" />

            <Button type="submit" loading={submitting} className="w-full justify-center" data-testid="register-submit-button">
              {submitting ? 'Registering...' : 'Register'}
            </Button>
          </form>
        </Card>

        <p className="text-center text-sm text-[var(--on-surface-variant)] mt-6">
          Already have an account?{' '}
           <Link to="/login" className="text-[var(--primary)] font-medium hover:underline" data-testid="register-sign-in-link">Sign in</Link>
        </p>
      </div>
    </div>
  )
}
