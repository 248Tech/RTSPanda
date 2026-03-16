import { useCallback, useEffect, useState } from 'react'
import CameraView from './pages/CameraView'
import Dashboard from './pages/Dashboard'
import MultiCameraView from './pages/MultiCameraView'
import Settings from './pages/Settings'

function usePath() {
  const [path, setPath] = useState(() => window.location.pathname)

  useEffect(() => {
    const handlePopState = () => setPath(window.location.pathname)
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const navigate = useCallback((to: string) => {
    window.history.pushState({}, '', to)
    setPath(to)
  }, [])

  return [path, navigate] as const
}

function NavItem({
  label,
  active,
  onClick,
  children,
}: {
  label: string
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      aria-label={label}
      className={`group relative flex h-10 w-10 items-center justify-center rounded-lg transition-all focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base ${
        active
          ? 'bg-accent/15 text-accent'
          : 'text-text-muted hover:bg-card-hover hover:text-text-primary'
      }`}
    >
      {active && (
        <span className="absolute left-0 h-5 w-0.5 rounded-r bg-accent" aria-hidden />
      )}
      {children}
    </button>
  )
}

function Sidebar({
  path,
  onNavigateHome,
  onNavigateMulti,
  onNavigateSettings,
}: {
  path: string
  onNavigateHome: () => void
  onNavigateMulti: () => void
  onNavigateSettings: () => void
}) {
  const isHome = path === '/' || path.startsWith('/cameras/')
  const isMulti = path === '/multi'
  const isSettings = path === '/settings'

  return (
    <aside className="fixed inset-y-0 left-0 z-20 flex w-14 flex-col border-r border-border-muted bg-surface">
      {/* Logo */}
      <button
        type="button"
        onClick={onNavigateHome}
        title="RTSPanda"
        aria-label="RTSPanda home"
        className="flex h-14 items-center justify-center text-xl focus:outline-none focus:ring-2 focus:ring-accent focus:ring-inset"
      >
        <span role="img" aria-hidden>🐼</span>
      </button>

      {/* Nav items */}
      <nav className="flex flex-1 flex-col items-center gap-1 px-2 py-2" aria-label="Main navigation">
        {/* Dashboard */}
        <NavItem label="Dashboard" active={isHome} onClick={onNavigateHome}>
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <rect x="3" y="3" width="7" height="7" rx="1" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
            <rect x="14" y="3" width="7" height="7" rx="1" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
            <rect x="3" y="14" width="7" height="7" rx="1" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
            <rect x="14" y="14" width="7" height="7" rx="1" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </NavItem>

        {/* Multi-view */}
        <NavItem label="Multi-Camera View" active={isMulti} onClick={onNavigateMulti}>
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z" />
          </svg>
        </NavItem>
      </nav>

      {/* Bottom: Chrome Extension + Settings */}
      <div className="flex flex-col items-center gap-1 px-2 py-2">
        <a
          href="/downloads/rtspanda-chrome-pip-extension.zip"
          download
          title="Download Chrome PiP Extension"
          aria-label="Download Chrome PiP Extension"
          className="flex h-10 w-10 items-center justify-center rounded-lg text-text-muted transition-all hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
        >
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 2C6.477 2 2 6.477 2 12s4.477 10 10 10 10-4.477 10-10S17.523 2 12 2z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h8M12 8v8" />
          </svg>
        </a>

        <NavItem label="Settings" active={isSettings} onClick={onNavigateSettings}>
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </NavItem>
      </div>
    </aside>
  )
}

export default function App() {
  const [path, navigate] = usePath()

  const onNavigateHome = useCallback(() => navigate('/'), [navigate])
  const onNavigateSettings = useCallback(() => navigate('/settings'), [navigate])
  const onNavigateMulti = useCallback(() => navigate('/multi'), [navigate])

  const onSelectCamera = useCallback(
    (cameraId: string) => navigate(`/cameras/${cameraId}`),
    [navigate]
  )

  const cameraIdMatch = path.startsWith('/cameras/') && path.length > 9
  const cameraId = cameraIdMatch ? path.slice(9).split('/')[0] || null : null

  return (
    <div className="min-h-screen bg-base font-sans text-text-primary">
      <Sidebar
        path={path}
        onNavigateHome={onNavigateHome}
        onNavigateMulti={onNavigateMulti}
        onNavigateSettings={onNavigateSettings}
      />
      <main className="pl-14">
        <div className="mx-auto max-w-7xl px-5 py-6">
          {path === '/' && (
            <Dashboard
              onSelectCamera={onSelectCamera}
              onNavigateSettings={onNavigateSettings}
              onNavigateMulti={onNavigateMulti}
            />
          )}
          {path === '/multi' && (
            <MultiCameraView
              onBack={() => navigate('/')}
              onSelectCamera={onSelectCamera}
            />
          )}
          {path === '/settings' && <Settings />}
          {cameraId && (
            <CameraView
              cameraId={cameraId}
              onBack={() => navigate('/')}
              onNavigateSettings={onNavigateSettings}
            />
          )}
          {path !== '/' && path !== '/settings' && path !== '/multi' && !cameraIdMatch && (
            <p className="text-text-muted">Not found.</p>
          )}
        </div>
      </main>
    </div>
  )
}
