import type { Camera } from '../api/cameras'
import { CameraCard } from './CameraCard'

export interface CameraGridProps {
  cameras: Camera[]
  onSelectCamera: (cameraId: string) => void
}

export function CameraGrid({ cameras, onSelectCamera }: CameraGridProps) {
  if (cameras.length === 0) {
    return null
  }
  return (
    <div
      className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      role="list"
    >
      {cameras.map((camera) => (
        <div key={camera.id} role="listitem">
          <CameraCard camera={camera} onSelect={onSelectCamera} />
        </div>
      ))}
    </div>
  )
}
