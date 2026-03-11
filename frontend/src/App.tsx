import { useCallback, useEffect, useState } from 'react'
import CameraView from './pages/CameraView'
import Dashboard from './pages/Dashboard'
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

function Navbar({
  onNavigateHome,
  onNavigateSettings,
}: {
  onNavigateHome: () => void
  onNavigateSettings: () => void
}) {
  return (
    <header className="fixed left-0 right-0 top-0 z-10 flex h-14 items-center justify-between border-b border-border bg-card px-4">
      <button
        type="button"
        onClick={onNavigateHome}
        className="flex items-center gap-2 font-semibold text-text-primary focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
      >
        <span aria-hidden role="img" aria-label="panda">🐼</span>
        <span>RTSPanda</span>
      </button>
      <button
        type="button"
        onClick={onNavigateSettings}
        className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
      >
        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
        Settings
      </button>
    </header>
  )
}

export default function App() {
  const [path, navigate] = usePath()

  const onNavigateHome = useCallback(() => navigate('/'), [navigate])
  const onNavigateSettings = useCallback(() => {
    navigate('/settings')
  }, [navigate])

  const onSelectCamera = useCallback(
    (cameraId: string) => {
      navigate(`/cameras/${cameraId}`)
    },
    [navigate]
  )

  const cameraIdMatch = path.startsWith('/cameras/') && path.length > 9
  const cameraId = cameraIdMatch ? path.slice(9).split('/')[0] || null : null

  return (
    <div className="min-h-screen bg-base text-text-primary">
      <Navbar onNavigateHome={onNavigateHome} onNavigateSettings={onNavigateSettings} />
      <main className="pt-14">
        <div className="mx-auto max-w-7xl px-4 py-6">
          {path === '/' && (
            <Dashboard
              onSelectCamera={onSelectCamera}
              onNavigateSettings={onNavigateSettings}
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
          {path !== '/' && path !== '/settings' && !cameraIdMatch && (
            <p className="text-text-muted">Not found.</p>
          )}
        </div>
      </main>
    </div>
  )
}
