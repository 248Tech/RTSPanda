const BASE = '/api/v1'

export interface DetectionBBox {
  x: number
  y: number
  width: number
  height: number
}

export interface DetectionEvent {
  id: string
  camera_id: string
  object_label: string
  confidence: number
  bbox: DetectionBBox
  snapshot_path: string
  frame_width?: number
  frame_height?: number
  created_at: string
}

export interface DetectionHealth {
  status: string
  detector_url: string
  detector_healthy: boolean
  queue_depth: number
  queue_capacity: number
  sampler_enabled: boolean
  worker_concurrency: number
}

export interface TriggerDetectionResponse {
  camera_id?: string
  timestamp: string
  image_width?: number
  image_height?: number
  detections: Array<{
    label: string
    confidence: number
    bbox: DetectionBBox
  }>
  snapshot_path?: string
}

export async function getDetectionHealth(): Promise<DetectionHealth> {
  const res = await fetch(`${BASE}/detections/health`)
  if (!res.ok) throw new Error(`getDetectionHealth: ${res.status}`)
  return res.json()
}

export async function listDetectionEvents(
  cameraId?: string,
  limit = 100
): Promise<DetectionEvent[]> {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  if (cameraId) {
    params.set('camera_id', cameraId)
  }

  const res = await fetch(`${BASE}/detection-events?${params.toString()}`)
  if (!res.ok) throw new Error(`listDetectionEvents: ${res.status}`)
  return res.json()
}

export async function triggerTestDetection(
  cameraId: string
): Promise<TriggerDetectionResponse> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/detections/test`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(`triggerTestDetection: ${res.status}`)
  return res.json()
}

export function detectionSnapshotUrl(eventId: string): string {
  return `${BASE}/detection-events/${eventId}/snapshot`
}
