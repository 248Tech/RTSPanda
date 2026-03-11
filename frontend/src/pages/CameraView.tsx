import { useCallback, useEffect, useState } from 'react'
import { getCamera, getStreamInfo } from '../api/cameras'
import type { Camera } from '../api/cameras'
import { StatusBadge } from '../components/StatusBadge'
import type { StreamStatus } from '../components/StatusBadge'
import { VideoPlayer } from '../components/VideoPlayer'
import { RecordingsList } from '../components/RecordingsList'

export interface CameraViewProps {
  cameraId: string
  onBack: () => void
  onNavigateSettings: () => void
}

export default function CameraView({ cameraId, onBack, onNavigateSettings }: CameraViewProps) {
  const [camera, setCamera] = useState<Camera | null>(null)
  const [hlsUrl, setHlsUrl] = useState<string | null>(null)
  const [streamStatus, setStreamStatus] = useState<StreamStatus>('connecting')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchCameraAndStream = useCallback(async () => {
    setError(null)
    try {
      const [cameraRes, streamRes] = await Promise.all([
        getCamera(cameraId),
        getStreamInfo(cameraId),
      ])
      setCamera(cameraRes)
      setHlsUrl(streamRes.hls_url)
      setStreamStatus(streamRes.status)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load camera')
      setCamera(null)
      setHlsUrl(null)
      setStreamStatus('offline')
    } finally {
      setLoading(false)
    }
  }, [cameraId])

  useEffect(() => {
    fetchCameraAndStream()
  }, [fetchCameraAndStream])

  const handleRetry = useCallback(() => {
    fetchCameraAndStream()
  }, [fetchCameraAndStream])

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <span
          className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent"
          aria-hidden
        />
        <span className="sr-only">Loading camera…</span>
      </div>
    )
  }

  if (error || !camera) {
    return (
      <div className="space-y-4">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Dashboard
        </button>
        <div className="rounded-lg border border-status-offline/50 bg-status-offline/10 px-4 py-6 text-text-primary">
          <p className="font-medium">Could not load camera</p>
          <p className="mt-1 text-sm text-text-muted">{error ?? 'Not found'}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Dashboard
        </button>
        <button
          type="button"
          onClick={onNavigateSettings}
          className="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          Settings
        </button>
      </div>

      <div className="group">
        <VideoPlayer hlsUrl={hlsUrl} screenshotLabel={camera.name} onRetry={handleRetry} />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-text-primary">{camera.name}</h2>
          <p className="mt-0.5 truncate text-sm text-text-muted" title={camera.rtsp_url}>
            {camera.rtsp_url}
          </p>
        </div>
        <StatusBadge status={streamStatus} />
      </div>

      {/* Recordings */}
      <div className="border-t border-border pt-4">
        <RecordingsList cameraId={camera.id} recordEnabled={camera.record_enabled} />
      </div>
    </div>
  )
}
