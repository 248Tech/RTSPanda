const BASE = '/api/v1'

export type VideoStorageProvider =
  | 'local_server'
  | 'dropbox'
  | 'google_drive'
  | 'onedrive'
  | 'proton_drive'

export interface AppSettings {
  openai_enabled: boolean
  openai_model: string
  openai_api_key_set: boolean
  video_storage_enabled: boolean
  video_storage_provider: VideoStorageProvider
  video_storage_local_path: string
  video_storage_remote_name: string
  video_storage_remote_path: string
  video_storage_sync_interval_seconds: number
  video_storage_min_file_age_seconds: number
}

export interface UpdateAppSettingsInput {
  openai_enabled?: boolean
  openai_model?: string
  openai_api_key?: string
  clear_openai_api_key?: boolean
  video_storage_enabled?: boolean
  video_storage_provider?: VideoStorageProvider
  video_storage_local_path?: string
  video_storage_remote_name?: string
  video_storage_remote_path?: string
  video_storage_sync_interval_seconds?: number
  video_storage_min_file_age_seconds?: number
}

export async function getAppSettings(): Promise<AppSettings> {
  const res = await fetch(`${BASE}/settings`)
  if (!res.ok) throw new Error(`getAppSettings: ${res.status}`)
  return res.json()
}

export async function updateAppSettings(data: UpdateAppSettingsInput): Promise<AppSettings> {
  const res = await fetch(`${BASE}/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    let message = `updateAppSettings: ${res.status}`
    try {
      const payload = await res.json() as { error?: string }
      if (payload.error) {
        message = payload.error
      }
    } catch {
      // ignore parse failure and keep status fallback
    }
    throw new Error(message)
  }
  return res.json()
}
