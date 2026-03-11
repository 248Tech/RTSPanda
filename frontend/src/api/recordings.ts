const BASE = '/api/v1'

export interface Recording {
  filename: string
  camera_id: string
  size_bytes: number
  created_at: string
}

export async function listRecordings(cameraId: string): Promise<Recording[]> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/recordings`)
  if (!res.ok) throw new Error(`listRecordings: ${res.status}`)
  return res.json()
}

export function downloadRecordingUrl(cameraId: string, filename: string): string {
  return `${BASE}/cameras/${cameraId}/recordings/${encodeURIComponent(filename)}`
}

export async function deleteRecording(cameraId: string, filename: string): Promise<void> {
  const res = await fetch(`${BASE}/cameras/${cameraId}/recordings/${encodeURIComponent(filename)}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error(`deleteRecording: ${res.status}`)
}

/** Format bytes as human-readable size string */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
