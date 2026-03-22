import { useCallback, useEffect, useRef, useState } from 'react'
import type { Camera } from '../api/cameras'
import { getStreamInfo } from '../api/cameras'
import type { StreamStatus } from './StatusBadge'
import { StatusBadge } from './StatusBadge'

const STATUS_POLL_MS = 10_000

export interface CameraCardProps {
  camera: Camera
  onSelect: (cameraId: string) => void
  /** When provided by a parent batch fetch, the card skips its own API call. */
  initialStatus?: StreamStatus
}

export function CameraCard({ camera, onSelect, initialStatus }: CameraCardProps) {
  const [status, setStatus] = useState<StreamStatus>(initialStatus ?? 'connecting')
  const [loading, setLoading] = useState(!initialStatus)
  const cancelledRef = useRef(false)

  const fetchStatus = useCallback(() => {
    getStreamInfo(camera.id)
      .then((info) => { if (!cancelledRef.current) setStatus(info.status) })
      .catch(() => { if (!cancelledRef.current) setStatus('offline') })
      .finally(() => { if (!cancelledRef.current) setLoading(false) })
  }, [camera.id])

  useEffect(() => {
    // If the parent supplied a status, don't fetch on mount — just poll for updates.
    cancelledRef.current = false
    if (!initialStatus) {
      fetchStatus()
    }
    const id = setInterval(fetchStatus, STATUS_POLL_MS)
    return () => {
      cancelledRef.current = true
      clearInterval(id)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchStatus])

  const displayStatus = loading ? 'connecting' : status
  const isOffline = displayStatus === 'offline'

  return (
    <button
      type="button"
      onClick={() => onSelect(camera.id)}
      className="group relative w-full overflow-hidden rounded-xl border border-border-muted bg-card text-left transition-all duration-200 hover:border-border hover:shadow-lg focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
      aria-label={`View ${camera.name} camera`}
    >
      {/* Thumbnail */}
      <div
        className={`relative aspect-video w-full bg-surface transition-all duration-300 ${
          isOffline ? 'opacity-40 grayscale' : ''
        }`}
      >
        {/* Placeholder graphic */}
        <div className="absolute inset-0 flex items-center justify-center">
          {displayStatus === 'connecting' || displayStatus === 'initializing' ? (
            <span
              className="h-7 w-7 animate-spin rounded-full border-2 border-text-subtle border-t-transparent"
              aria-hidden
            />
          ) : isOffline ? (
            <svg className="h-10 w-10 text-text-subtle" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
          ) : (
            <svg className="h-10 w-10 text-text-subtle transition-colors group-hover:text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          )}
        </div>

        {/* Subtle grid overlay for depth */}
        <div
          className="absolute inset-0 opacity-[0.03]"
          style={{ backgroundImage: 'linear-gradient(rgba(255,255,255,.15) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.15) 1px, transparent 1px)', backgroundSize: '24px 24px' }}
          aria-hidden
        />

        {/* Status badge — top right */}
        <div className="absolute right-2 top-2">
          <StatusBadge status={displayStatus} />
        </div>
      </div>

      {/* Info bar */}
      <div className="flex items-center justify-between gap-2 px-3 py-2.5">
        <span className="min-w-0 truncate text-sm font-medium text-text-primary">
          {camera.name}
        </span>

        {/* Feature indicator dots */}
        <div className="flex shrink-0 items-center gap-1.5" aria-label="Camera features">
          {camera.record_enabled && (
            <span title="Recording enabled" aria-label="Recording enabled" className="flex h-5 w-5 items-center justify-center rounded bg-red-500/10 text-red-400">
              <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 24 24" aria-hidden>
                <circle cx="12" cy="12" r="6" />
              </svg>
            </span>
          )}
          {camera.tracking_enabled && (
            <span title="AI detection enabled" aria-label="AI detection enabled" className="flex h-5 w-5 items-center justify-center rounded bg-violet-500/10 text-violet-400">
              <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18" />
              </svg>
            </span>
          )}
          {camera.discord_alerts_enabled && (
            <span title="Discord alerts enabled" aria-label="Discord alerts enabled" className="flex h-5 w-5 items-center justify-center rounded bg-indigo-500/10 text-indigo-400">
              <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 24 24" aria-hidden>
                <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03z" />
              </svg>
            </span>
          )}
        </div>
      </div>
    </button>
  )
}
