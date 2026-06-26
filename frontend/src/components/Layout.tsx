import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth'
import { useQuery } from '@tanstack/react-query'
import api from '../lib/api'

interface NavItem { to: string; label: string; icon: string }

const navGroups: { label: string; items: NavItem[] }[] = [
  {
    label: 'Main',
    items: [
      { to: '/', label: 'Dashboard', icon: 'dashboard' },
      { to: '/users', label: 'Users', icon: 'people' },
      { to: '/roles', label: 'Roles', icon: 'manage_accounts' },
      { to: '/permissions', label: 'Permissions', icon: 'lock' },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/settings', label: 'Settings', icon: 'settings' },
      { to: '/feature-flags', label: 'Feature Flags', icon: 'flag' },
      { to: '/audit', label: 'Audit Log', icon: 'history' },
      { to: '/apikeys', label: 'API Keys', icon: 'key' },
      { to: '/webhooks', label: 'Webhooks', icon: 'webhook' },
    ],
  },
  {
    label: 'Services',
    items: [
      { to: '/storage', label: 'Storage', icon: 'folder' },
      { to: '/sessions', label: 'Sessions', icon: 'devices' },
      { to: '/jobs', label: 'Jobs', icon: 'assignment' },
      { to: '/notifications', label: 'Notifications', icon: 'notifications' },
    ],
  },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const { data: notifData } = useQuery({
    queryKey: ['unread-count'],
    queryFn: () => api.get('/api/notifications/unread-count').then(r => r.data.data.unread_count),
    refetchInterval: 30000,
  })

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="sidebar-logo">
          <div className="sidebar-logo-icon">C</div>
          <div>
            <div className="sidebar-logo-text">CoreKit</div>
            <div className="sidebar-logo-sub">Admin Console</div>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto -mx-2 px-2">
          {navGroups.map((group) => (
            <div key={group.label}>
              <div className="nav-section-label">{group.label}</div>
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === '/'}
                  className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
                  data-testid={`nav-${item.label.toLowerCase().replace(/\s+/g, '-')}-link`}
                >
                  <span className="material-symbols-outlined nav-item-icon">{item.icon}</span>
                  <span className="nav-item-label">{item.label}</span>
                  {item.to === '/notifications' && notifData !== undefined && notifData > 0 && (
                    <span className="nav-item-badge">{notifData > 99 ? '99+' : notifData}</span>
                  )}
                </NavLink>
              ))}
            </div>
          ))}
        </div>

        <div className="sidebar-footer">
          {user && (
            <div className="px-2 pb-2">
              <NavLink to="/profile" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} data-testid="nav-profile-link">
                <span className="material-symbols-outlined nav-item-icon">person</span>
                <div className="flex-1 min-w-0">
                  <div className="nav-item-label truncate">{user.full_name || 'Profile'}</div>
                  <div className="text-[10px] text-[var(--on-surface-variant)] truncate opacity-60">{user.email}</div>
                </div>
              </NavLink>
            </div>
          )}
          <div className="nav-item text-[var(--on-surface-variant)]" onClick={handleLogout} role="button" tabIndex={0} data-testid="nav-logout-button">
            <span className="material-symbols-outlined nav-item-icon">logout</span>
            <span className="nav-item-label">Logout</span>
          </div>
        </div>
      </aside>

      <div className="main-area">
        <Outlet />
      </div>
    </div>
  )
}
