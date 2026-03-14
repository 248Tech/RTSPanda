export interface Camera {
  id: string
  name: string
  rtsp_url: string
  enabled: boolean
  record_enabled: boolean
  detection_sample_seconds?: number
  tracking_enabled: boolean
  tracking_min_confidence: number
  tracking_labels: string[]
  discord_alerts_enabled: boolean
  discord_webhook_url: string
  discord_mention: string
  discord_cooldown_seconds: number
  discord_trigger_on_detection: boolean
  discord_trigger_on_interval: boolean
  discord_screenshot_interval_seconds: number
  discord_include_motion_clip: boolean
  discord_motion_clip_seconds: number
  discord_record_format: 'webp' | 'webm' | 'gif'
  discord_record_duration_seconds: number
  position: number
  created_at: string
  updated_at: string
}

export interface CreateCameraInput {
  name: string
  rtsp_url: string
  enabled?: boolean
  record_enabled?: boolean
  detection_sample_seconds?: number
  tracking_enabled?: boolean
  tracking_min_confidence?: number
  tracking_labels?: string[]
  discord_alerts_enabled?: boolean
  discord_webhook_url?: string
  discord_mention?: string
  discord_cooldown_seconds?: number
  discord_trigger_on_detection?: boolean
  discord_trigger_on_interval?: boolean
  discord_screenshot_interval_seconds?: number
  discord_include_motion_clip?: boolean
  discord_motion_clip_seconds?: number
  discord_record_format?: 'webp' | 'webm' | 'gif'
  discord_record_duration_seconds?: number
}

export interface UpdateCameraInput {
  name?: string
  rtsp_url?: string
  enabled?: boolean
  record_enabled?: boolean
  detection_sample_seconds?: number
  tracking_enabled?: boolean
  tracking_min_confidence?: number
  tracking_labels?: string[]
  discord_alerts_enabled?: boolean
  discord_webhook_url?: string
  discord_mention?: string
  discord_cooldown_seconds?: number
  discord_trigger_on_detection?: boolean
  discord_trigger_on_interval?: boolean
  discord_screenshot_interval_seconds?: number
  discord_include_motion_clip?: boolean
  discord_motion_clip_seconds?: number
  discord_record_format?: 'webp' | 'webm' | 'gif'
  discord_record_duration_seconds?: number
  position?: number
}

export interface StreamInfo {
  hls_url: string
  status: 'online' | 'offline' | 'connecting'
}

const BASE = '/api/v1'

export async function getCameras(): Promise<Camera[]> {
  const res = await fetch(`${BASE}/cameras`)
  if (!res.ok) throw new Error(`getCameras: ${res.status}`)
  return res.json()
}

export async function getCamera(id: string): Promise<Camera> {
  const res = await fetch(`${BASE}/cameras/${id}`)
  if (!res.ok) throw new Error(`getCamera: ${res.status}`)
  return res.json()
}

export async function addCamera(data: CreateCameraInput): Promise<Camera> {
  const res = await fetch(`${BASE}/cameras`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`addCamera: ${res.status}`)
  return res.json()
}

export async function updateCamera(id: string, data: UpdateCameraInput): Promise<Camera> {
  const res = await fetch(`${BASE}/cameras/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`updateCamera: ${res.status}`)
  return res.json()
}

export async function deleteCamera(id: string): Promise<void> {
  const res = await fetch(`${BASE}/cameras/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteCamera: ${res.status}`)
}

export async function getStreamInfo(id: string): Promise<StreamInfo> {
  const res = await fetch(`${BASE}/cameras/${id}/stream`)
  if (!res.ok) throw new Error(`getStreamInfo: ${res.status}`)
  return res.json()
}
