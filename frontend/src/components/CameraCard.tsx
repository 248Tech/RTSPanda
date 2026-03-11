import { useEffect, useState } from 'react'
import type { Camera } from '../api/cameras'
import { getStreamInfo } from '../api/cameras'
import type { StreamStatus } from './StatusBadge'
import { StatusBadge } from './StatusBadge'

export interface CameraCardProps {
  camera: Camera
  onSelect: (cameraId: string) => void
}

export function CameraCard({ camera, onSelect }: CameraCardProps) {
  const [status, setStatus] = useState<StreamStatus>('connecting')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    getStreamInfo(camera.id)
      .then((info) => {
        if (!cancelled) {
          setStatus(info.status)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setStatus('offline')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [camera.id])

  const displayStatus = loading ? 'connecting' : status

  return (
    <button
      type="button"
      onClick={() => onSelect(camera.id)}
      className="group w-full rounded-lg border border-border bg-card text-left transition-all hover:border-accent/50 hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
      aria-label={`View ${camera.name} camera`}
    >
      {/* 16:9 thumbnail area — placeholder only (no video on grid) */}
      <div
        className={`relative aspect-video w-full overflow-hidden rounded-t-lg bg-card-hover transition-opacity ${
          displayStatus === 'offline' ? 'opacity-40 grayscale' : ''
        }`}
      >
        <div className="absolute inset-0 flex items-center justify-center">
          {displayStatus === 'connecting' ? (
            <span
              className="h-8 w-8 animate-spin rounded-full border-2 border-status-connecting border-t-transparent"
              aria-hidden
            />
          ) : displayStatus === 'offline' ? (
            <svg
              className="h-12 w-12 text-text-muted/50"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
              />
            </svg>
          ) : (
            <svg
              className="h-12 w-12 text-text-muted/60 transition-colors group-hover:text-text-muted"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
              />
            </svg>
          )}
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 p-3">
        <span className="min-w-0 truncate font-medium text-text-primary">
          {camera.name}
        </span>
        <StatusBadge status={displayStatus} />
      </div>
    </button>
  )
}
