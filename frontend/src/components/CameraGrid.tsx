import type { Camera, StreamStatusMap } from '../api/cameras'
import type { StreamStatus } from './StatusBadge'
import { CameraCard } from './CameraCard'

export interface CameraGridProps {
  cameras: Camera[]
  onSelectCamera: (cameraId: string) => void
  /** Pre-fetched batch stream status — eliminates per-card API calls on load. */
  statusMap?: StreamStatusMap
}

export function CameraGrid({ cameras, onSelectCamera, statusMap }: CameraGridProps) {
  if (cameras.length === 0) {
    return null
  }
  return (
    <div
      className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      role="list"
    >
      {cameras.map((camera) => {
        const initialStatus = statusMap?.[camera.id]?.status as StreamStatus | undefined
        return (
          <div key={camera.id} role="listitem">
            <CameraCard camera={camera} onSelect={onSelectCamera} initialStatus={initialStatus} />
          </div>
        )
      })}
    </div>
  )
}
