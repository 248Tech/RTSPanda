import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { updateAppSettings, type AppSettings } from './settings'

describe('updateAppSettings', () => {
  const fetchMock = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('sends PUT request with JSON body and returns settings payload', async () => {
    const response: AppSettings = {
      openai_enabled: true,
      openai_model: 'gpt-4o-mini',
      openai_api_key_set: true,
      video_storage_enabled: false,
      video_storage_provider: 'local_server',
      video_storage_local_path: 'C:/videos',
      video_storage_remote_name: '',
      video_storage_remote_path: 'RTSPanda',
      video_storage_sync_interval_seconds: 300,
      video_storage_min_file_age_seconds: 120,
    }

    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => response,
    })

    const input = { openai_enabled: true, openai_model: 'gpt-4o-mini' }
    const result = await updateAppSettings(input)

    expect(result).toEqual(response)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  })

  it('uses API error payload when present', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: 'video_storage_local_path is required' }),
    })

    await expect(updateAppSettings({ video_storage_enabled: true })).rejects.toThrow(
      'video_storage_local_path is required'
    )
  })

  it('falls back to status code message when error payload cannot be parsed', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 503,
      json: async () => {
        throw new Error('not-json')
      },
    })

    await expect(updateAppSettings({})).rejects.toThrow('updateAppSettings: 503')
  })
})
