import { useCallback, useEffect, useState } from 'react'
import { getCameras } from '../api/cameras'
import type { Camera } from '../api/cameras'
import { CameraGrid } from '../components/CameraGrid'
import { EmptyState } from '../components/EmptyState'

const POLL_INTERVAL_MS = 30_000

export interface DashboardProps {
  onSelectCamera: (cameraId: string) => void
  onNavigateSettings: () => void
}

export default function Dashboard({
  onSelectCamera,
  onNavigateSettings,
}: DashboardProps) {
  const [cameras, setCameras] = useState<Camera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchCameras = useCallback(async () => {
    try {
      const list = await getCameras()
      setCameras(list)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load cameras')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchCameras()
  }, [fetchCameras])

  useEffect(() => {
    const id = setInterval(fetchCameras, POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [fetchCameras])

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <span
          className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent"
          aria-hidden
        />
        <span className="sr-only">Loading cameras…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-lg border border-status-offline/50 bg-status-offline/10 px-4 py-6 text-text-primary">
        <p className="font-medium">Could not load cameras</p>
        <p className="mt-1 text-sm text-text-muted">{error}</p>
      </div>
    )
  }

  if (cameras.length === 0) {
    return (
      <EmptyState
        onAddCamera={onNavigateSettings}
      />
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-text-primary">Cameras</h1>
        <span className="text-sm text-text-muted">{cameras.length} configured</span>
      </div>
      <CameraGrid
        cameras={cameras}
        onSelectCamera={onSelectCamera}
      />
    </div>
  )
}
