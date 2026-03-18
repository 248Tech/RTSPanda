import { useCallback, useEffect, useState } from 'react'
import { getCameras, getStreamStatusMap, resetAllStreams } from '../api/cameras'
import type { Camera, StreamStatusMap } from '../api/cameras'
import { CameraGrid } from '../components/CameraGrid'
import { EmptyState } from '../components/EmptyState'

const POLL_INTERVAL_MS = 30_000

export interface DashboardProps {
  onSelectCamera: (cameraId: string) => void
  onNavigateSettings: () => void
  onNavigateMulti: () => void
}

export default function Dashboard({
  onSelectCamera,
  onNavigateSettings,
  onNavigateMulti,
}: DashboardProps) {
  const [cameras, setCameras] = useState<Camera[]>([])
  const [statusMap, setStatusMap] = useState<StreamStatusMap>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [resetting, setResetting] = useState(false)
  const [resetMsg, setResetMsg] = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    try {
      // Parallel fetch: camera list + all stream statuses in one mediamtx call
      const [list, map] = await Promise.all([getCameras(), getStreamStatusMap()])
      setCameras(list)
      setStatusMap(map)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load cameras')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])
  useEffect(() => {
    const id = setInterval(fetchAll, POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [fetchAll])

  const handleResetNetwork = useCallback(async () => {
    if (resetting) return
    setResetting(true)
    setResetMsg(null)
    try {
      await resetAllStreams()
      setResetMsg('Network reset — streams reconnecting')
    } catch {
      setResetMsg('Reset failed — check logs')
    } finally {
      setResetting(false)
      setTimeout(() => setResetMsg(null), 4000)
    }
  }, [resetting])

  if (loading) {
    return (
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <div className="h-6 w-28 animate-pulse rounded-md bg-card" />
          <div className="h-8 w-36 animate-pulse rounded-lg bg-card" />
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="overflow-hidden rounded-xl border border-border-muted bg-card">
              <div className="aspect-video animate-pulse bg-surface" />
              <div className="flex items-center gap-2 px-3 py-2.5">
                <div className="h-4 w-2/3 animate-pulse rounded bg-surface" />
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl border border-status-offline/20 bg-status-offline/5 px-5 py-5">
        <p className="text-sm font-medium text-status-offline">Failed to load cameras</p>
        <p className="mt-1 text-xs text-text-muted">{error}</p>
        <button
          type="button"
          onClick={fetchAll}
          className="mt-3 text-xs text-accent hover:underline focus:outline-none"
        >
          Try again
        </button>
      </div>
    )
  }

  if (cameras.length === 0) {
    return <EmptyState onAddCamera={onNavigateSettings} />
  }

  const enabled = cameras.filter((c) => c.enabled).length

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <h1 className="text-base font-semibold text-text-primary">Cameras</h1>
          <span className="rounded-full border border-border-muted px-2 py-0.5 text-xs text-text-muted">
            {enabled}/{cameras.length} active
          </span>
        </div>
        <div className="flex items-center gap-2">
          {resetMsg && (
            <span className="text-xs text-text-muted">{resetMsg}</span>
          )}
          <button
            type="button"
            onClick={handleResetNetwork}
            disabled={resetting}
            title="Force all camera streams to reconnect"
            className="inline-flex items-center gap-2 rounded-lg border border-border-muted bg-card px-3 py-1.5 text-xs font-medium text-text-muted transition-colors hover:border-border hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
          >
            {resetting ? (
              <span className="h-3.5 w-3.5 animate-spin rounded-full border border-text-subtle border-t-transparent" aria-hidden />
            ) : (
              <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            )}
            Reset Network
          </button>
          <button
            type="button"
            onClick={onNavigateMulti}
            className="inline-flex items-center gap-2 rounded-lg border border-border-muted bg-card px-3 py-1.5 text-xs font-medium text-text-muted transition-colors hover:border-border hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
          >
            <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
            Multi-view
          </button>
        </div>
      </div>

      <CameraGrid cameras={cameras} onSelectCamera={onSelectCamera} statusMap={statusMap} />
    </div>
  )
}
