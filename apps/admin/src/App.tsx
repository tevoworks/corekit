import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './lib/auth'
import { setNavigate } from './lib/api'
import Layout from './components/Layout'
import ErrorBoundary from './components/ErrorBoundary'
import LoginPage from './pages/auth/LoginPage'
import RegisterPage from './pages/auth/RegisterPage'
import DashboardPage from './pages/dashboard/DashboardPage'
import UsersPage from './pages/users/UsersPage'
import RolesPage from './pages/roles/RolesPage'
import PermissionsPage from './pages/permissions/PermissionsPage'
import SettingsPage from './pages/settings/SettingsPage'
import FeatureFlagsPage from './pages/settings/FeatureFlagsPage'
import AuditPage from './pages/audit/AuditPage'
import APIKeysPage from './pages/apikeys/APIKeysPage'
import WebhooksPage from './pages/webhooks/WebhooksPage'
import WebhookDeliveriesPage from './pages/webhooks/WebhookDeliveriesPage'
import StoragePage from './pages/storage/StoragePage'
import SessionsPage from './pages/sessions/SessionsPage'
import JobsPage from './pages/jobs/JobsPage'
import NotificationsPage from './pages/notifications/NotificationsPage'
import ProfilePage from './pages/profile/ProfilePage'
import PagesPage from './pages/cms/PagesPage'
import PageFormPage from './pages/cms/PageFormPage'
import SectionsPage from './pages/cms/SectionsPage'
import BlogPage from './pages/cms/BlogPage'
import PostFormPage from './pages/cms/PostFormPage'
import MessagesPage from './pages/contact/MessagesPage'
import SubscribersPage from './pages/contact/SubscribersPage'
import type { ReactNode } from 'react'
import { useEffect } from 'react'

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="h-screen flex items-center justify-center text-sm text-[var(--on-surface-variant)]">Loading...</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

function PublicRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="h-screen flex items-center justify-center text-sm text-[var(--on-surface-variant)]">Loading...</div>
  if (user) return <Navigate to="/" replace />
  return <>{children}</>
}

function NavigateProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate()
  useEffect(() => { setNavigate(navigate) }, [navigate])
  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <NavigateProvider>
          <Routes>
            <Route path="/login" element={<PublicRoute><ErrorBoundary><LoginPage /></ErrorBoundary></PublicRoute>} />
            <Route path="/register" element={<PublicRoute><ErrorBoundary><RegisterPage /></ErrorBoundary></PublicRoute>} />
            <Route element={<ProtectedRoute><Layout /></ProtectedRoute>}>
              <Route path="/" element={<ErrorBoundary><DashboardPage /></ErrorBoundary>} />
              <Route path="/users" element={<ErrorBoundary><UsersPage /></ErrorBoundary>} />
              <Route path="/roles" element={<ErrorBoundary><RolesPage /></ErrorBoundary>} />
              <Route path="/permissions" element={<ErrorBoundary><PermissionsPage /></ErrorBoundary>} />
              <Route path="/settings" element={<ErrorBoundary><SettingsPage /></ErrorBoundary>} />
              <Route path="/feature-flags" element={<ErrorBoundary><FeatureFlagsPage /></ErrorBoundary>} />
              <Route path="/audit" element={<ErrorBoundary><AuditPage /></ErrorBoundary>} />
              <Route path="/apikeys" element={<ErrorBoundary><APIKeysPage /></ErrorBoundary>} />
              <Route path="/webhooks" element={<ErrorBoundary><WebhooksPage /></ErrorBoundary>} />
              <Route path="/webhooks/:id/deliveries" element={<ErrorBoundary><WebhookDeliveriesPage /></ErrorBoundary>} />
              <Route path="/storage" element={<ErrorBoundary><StoragePage /></ErrorBoundary>} />
              <Route path="/sessions" element={<ErrorBoundary><SessionsPage /></ErrorBoundary>} />
              <Route path="/jobs" element={<ErrorBoundary><JobsPage /></ErrorBoundary>} />
              <Route path="/notifications" element={<ErrorBoundary><NotificationsPage /></ErrorBoundary>} />
              <Route path="/cms/pages" element={<ErrorBoundary><PagesPage /></ErrorBoundary>} />
              <Route path="/cms/pages/new" element={<ErrorBoundary><PageFormPage /></ErrorBoundary>} />
              <Route path="/cms/pages/:id/edit" element={<ErrorBoundary><PageFormPage /></ErrorBoundary>} />
              <Route path="/cms/pages/:pageId/sections" element={<ErrorBoundary><SectionsPage /></ErrorBoundary>} />
              <Route path="/cms/blog" element={<ErrorBoundary><BlogPage /></ErrorBoundary>} />
              <Route path="/cms/blog/new" element={<ErrorBoundary><PostFormPage /></ErrorBoundary>} />
              <Route path="/cms/blog/:id/edit" element={<ErrorBoundary><PostFormPage /></ErrorBoundary>} />
              <Route path="/contact/messages" element={<ErrorBoundary><MessagesPage /></ErrorBoundary>} />
              <Route path="/contact/subscribers" element={<ErrorBoundary><SubscribersPage /></ErrorBoundary>} />
              <Route path="/profile" element={<ErrorBoundary><ProfilePage /></ErrorBoundary>} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </NavigateProvider>
      </AuthProvider>
    </BrowserRouter>
  )
}
