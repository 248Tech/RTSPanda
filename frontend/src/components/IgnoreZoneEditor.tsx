import { useEffect, useMemo, useRef, useState } from 'react'
import type { IgnorePolygon } from '../api/cameras'

export interface IgnoreZoneEditorProps {
  snapshotDataUrl: string
  initialPolygons: IgnorePolygon[]
  onCancel: () => void
  onSave: (polygons: IgnorePolygon[]) => void | Promise<void>
  isSaving?: boolean
  error?: string | null
}

function clamp01(value: number): number {
  if (value < 0) return 0
  if (value > 1) return 1
  return value
}

function pointsToSvg(points: IgnorePolygon): string {
  return points.map((pt) => `${pt.x * 100} ${pt.y * 100}`).join(' ')
}

export function IgnoreZoneEditor({
  snapshotDataUrl,
  initialPolygons,
  onCancel,
  onSave,
  isSaving = false,
  error,
}: IgnoreZoneEditorProps) {
  const [polygons, setPolygons] = useState<IgnorePolygon[]>(() => initialPolygons ?? [])
  const [draftPoints, setDraftPoints] = useState<IgnorePolygon>([])
  const [localError, setLocalError] = useState<string | null>(null)
  const overlayRef = useRef<SVGSVGElement | null>(null)

  useEffect(() => {
    setPolygons(initialPolygons ?? [])
    setDraftPoints([])
    setLocalError(null)
  }, [initialPolygons, snapshotDataUrl])

  const totalZones = polygons.length + (draftPoints.length >= 3 ? 1 : 0)

  const draftPolyline = useMemo(() => {
    if (draftPoints.length === 0) return ''
    return pointsToSvg(draftPoints)
  }, [draftPoints])

  const addPoint = (event: React.MouseEvent<SVGSVGElement>) => {
    const svg = overlayRef.current
    if (!svg) return
    const rect = svg.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return

    const x = clamp01((event.clientX - rect.left) / rect.width)
    const y = clamp01((event.clientY - rect.top) / rect.height)
    setLocalError(null)
    setDraftPoints((prev) => [...prev, { x, y }])
  }

  const handleClosePolygon = () => {
    if (draftPoints.length < 3) {
      setLocalError('Add at least 3 points before closing a polygon.')
      return
    }
    setPolygons((prev) => [...prev, draftPoints])
    setDraftPoints([])
    setLocalError(null)
  }

  const handleUndoPoint = () => {
    setDraftPoints((prev) => prev.slice(0, -1))
    setLocalError(null)
  }

  const handleDeleteLastZone = () => {
    setPolygons((prev) => prev.slice(0, -1))
    setLocalError(null)
  }

  const handleClearAll = () => {
    setPolygons([])
    setDraftPoints([])
    setLocalError(null)
  }

  const handleSave = async () => {
    if (draftPoints.length > 0 && draftPoints.length < 3) {
      setLocalError('Complete or clear the current polygon before saving.')
      return
    }

    const next = draftPoints.length >= 3 ? [...polygons, draftPoints] : polygons
    setLocalError(null)
    await onSave(next)
  }

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-sm text-text-primary">
          Click the frame to place polygon points around noisy areas you want ignored (flags, tree branches, car cover).
        </p>
        <p className="mt-1 text-xs text-text-muted">
          Detections whose center point lands inside any ignore zone will be dropped.
        </p>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-black">
        <div className="relative">
          <img src={snapshotDataUrl} alt="Ignore zone editor frame" className="block w-full select-none" draggable={false} />
          <svg
            ref={overlayRef}
            viewBox="0 0 100 100"
            preserveAspectRatio="none"
            className="absolute inset-0 h-full w-full cursor-crosshair"
            onClick={addPoint}
          >
            {polygons.map((polygon, idx) => (
              <polygon
                key={`zone-${idx}`}
                points={pointsToSvg(polygon)}
                fill="rgba(239, 68, 68, 0.18)"
                stroke="rgba(239, 68, 68, 0.95)"
                strokeWidth={0.5}
                strokeDasharray="1.4 1"
              />
            ))}

            {draftPoints.length >= 2 && (
              <polyline
                points={draftPolyline}
                fill="none"
                stroke="rgba(34, 211, 238, 0.95)"
                strokeWidth={0.5}
                strokeDasharray="1.2 1"
              />
            )}

            {draftPoints.map((pt, idx) => (
              <circle
                key={`draft-point-${idx}`}
                cx={pt.x * 100}
                cy={pt.y * 100}
                r={0.85}
                fill="rgba(34, 211, 238, 1)"
                stroke="rgba(8, 145, 178, 1)"
                strokeWidth={0.25}
              />
            ))}
          </svg>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={handleClosePolygon}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent"
        >
          Close Polygon
        </button>
        <button
          type="button"
          onClick={handleUndoPoint}
          disabled={draftPoints.length === 0}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          Undo Point
        </button>
        <button
          type="button"
          onClick={() => setDraftPoints([])}
          disabled={draftPoints.length === 0}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          Clear Draft
        </button>
        <button
          type="button"
          onClick={handleDeleteLastZone}
          disabled={polygons.length === 0}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          Delete Last Zone
        </button>
        <button
          type="button"
          onClick={handleClearAll}
          disabled={polygons.length === 0 && draftPoints.length === 0}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          Clear All
        </button>
        <span className="ml-auto text-xs text-text-muted">
          Saved zones: {polygons.length} | Draft points: {draftPoints.length} | Total after save: {totalZones}
        </span>
      </div>

      {(localError || error) && (
        <p className="text-sm text-status-offline" role="alert">
          {localError ?? error}
        </p>
      )}

      <div className="flex justify-end gap-2 border-t border-border pt-3">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSaving}
          className="rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={isSaving}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
        >
          {isSaving ? 'Saving…' : 'Save Ignore Zones'}
        </button>
      </div>
    </div>
  )
}
