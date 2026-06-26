import axios from 'axios'

function getCSRFToken(): string | null {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/)
  return match ? decodeURIComponent(match[1]) : null
}

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
  withCredentials: true,
})

// Attach CSRF token to state-changing requests using Double Submit Cookie pattern.
// If the csrf_token cookie is missing (CSRF disabled on backend), no header is sent.
api.interceptors.request.use((config) => {
  if (config.method && !['get', 'head', 'options'].includes(config.method)) {
    const token = getCSRFToken()
    if (token) {
      config.headers['X-CSRF-Token'] = token
    }
  }
  return config
})

let isRefreshing = false
let pendingRequests: Array<() => void> = []
let navigate: ((path: string) => void) | null = null

export function setNavigate(nav: (path: string) => void) {
  navigate = nav
}

// Extract error message from backend response envelope { error: { code, message } }
export function getApiError(err: any): string {
  if (err.response?.data?.error?.message) {
    return err.response.data.error.message
  }
  if (err.response?.data?.message) {
    return err.response.data.message
  }
  if (err.message === 'Network Error') {
    return 'Unable to connect to server. Please check your connection.'
  }
  return err.message || 'An unexpected error occurred. Please try again.'
}

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    const originalRequest = err.config

    if (err.response?.status === 401 && !originalRequest._retry) {
      if (window.location.pathname === '/login') {
        return Promise.reject(err)
      }

      // Never retry the refresh endpoint itself — would cause deadlock
      if (originalRequest.url?.includes('/auth/refresh')) {
        return Promise.reject(err)
      }

      if (isRefreshing) {
        return new Promise((resolve) => {
          pendingRequests.push(() => resolve(api(originalRequest)))
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        await api.post('/api/auth/refresh')
        pendingRequests.forEach((cb) => cb())
        pendingRequests = []
        return api(originalRequest)
      } catch {
        pendingRequests = []
        if (navigate) {
          navigate('/login?expired=1')
        } else {
          window.location.href = '/login?expired=1'
        }
        return Promise.reject(err)
      } finally {
        isRefreshing = false
      }
    }

    return Promise.reject(err)
  },
)

export default api
