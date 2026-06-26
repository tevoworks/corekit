import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import type { User } from './types'
import api from './api'

interface AuthContextType {
  user: User | null
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, fullName: string) => Promise<string | null>
  logout: () => void
  loading: boolean
  setUser: (user: User | null) => void
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get('/api/me')
      .then((res) => {
        setUser(res.data.data)
      })
      .catch(() => {
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [])

  const login = async (email: string, password: string) => {
    const res = await api.post('/api/auth/login', { email, password })
    const u = res.data.data.user
    setUser(u)
  }

  const register = async (email: string, password: string, fullName: string) => {
    const res = await api.post('/api/auth/register', { email, password, full_name: fullName })
    const d = res.data.data
    if (d.user) {
      setUser(d.user)
    }
    return d.token ? null : (d.message || 'Registration successful')
  }

  const logout = async () => {
    try {
      await api.post('/api/logout')
    } catch (e) {
      console.error('Logout request failed', e)
    }
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, login, register, logout, loading, setUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
