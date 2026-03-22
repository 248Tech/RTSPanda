import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { getCameras, getStreamInfo, type Camera, type StreamInfo } from '../api/cameras'
import { sendDiscordRecording, sendDiscordScreenshot } from '../api/detections'
import { StatusBadge } from '../components/StatusBadge'
import { VideoPlayer, captureVideoScreenshot } from '../components/VideoPlayer'

const MAX_MULTI_CAMERAS = 4
const CAMERA_POLL_INTERVAL_MS = 30_000
const STREAM_POLL_INTERVAL_MS = 5_000

interface MultiCameraViewProps {
  onBack: () => void
  onSelectCamera: (cameraId: string) => void
}

export default function MultiCameraView({ onBack, onSelectCamera }: MultiCameraViewProps) {
  const [cameras, setCameras] = useState<Camera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [streamInfoByCameraId, setStreamInfoByCameraId] = useState<Record<string, StreamInfo>>({})

  const [sendingDiscordScreenshots, setSendingDiscordScreenshots] = useState(false)
  const [sendingDiscordRecordings, setSendingDiscordRecordings] = useState(false)
  const [showPicker, setShowPicker] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)

  const videoRefs = useRef<Record<string, HTMLVideoElement | null>>({})

  const selectedCameras = useMemo(() => {
    const byId = new Map(cameras.map((camera) => [camera.id, camera]))
    return selectedIds
      .map((id) => byId.get(id))
      .filter((camera): camera is Camera => Boolean(camera))
  }, [cameras, selectedIds])

  const fetchCameras = useCallback(async () => {
    try {
      const list = await getCameras()
      setCameras(list)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load cameras')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchCameras()
  }, [fetchCameras])

  useEffect(() => {
    const id = setInterval(fetchCameras, CAMERA_POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [fetchCameras])

  useEffect(() => {
    setSelectedIds((current) => {
      const validCurrent = current.filter((id) => cameras.some((camera) => camera.id === id)).slice(0, MAX_MULTI_CAMERAS)
      if (validCurrent.length > 0) return validCurrent

      const enabledFirst = cameras.filter((camera) => camera.enabled).slice(0, MAX_MULTI_CAMERAS)
      if (enabledFirst.length > 0) return enabledFirst.map((camera) => camera.id)
      return cameras.slice(0, MAX_MULTI_CAMERAS).map((camera) => camera.id)
    })
  }, [cameras])

  const fetchStreamInfo = useCallback(async () => {
    if (selectedIds.length === 0) {
      setStreamInfoByCameraId({})
      return
    }
    const pairs = await Promise.all(
      selectedIds.map(async (cameraID): Promise<[string, StreamInfo | null]> => {
        try {
          const info = await getStreamInfo(cameraID)
          return [cameraID, info]
        } catch {
          return [cameraID, null]
        }
      })
    )

    const next: Record<string, StreamInfo> = {}
    for (const [cameraID, info] of pairs) {
      if (info) {
        next[cameraID] = info
      } else {
        next[cameraID] = {
          hls_url: `/hls/camera-${cameraID}/index.m3u8`,
          status: 'offline',
        }
      }
    }
    setStreamInfoByCameraId(next)
  }, [selectedIds])

  useEffect(() => {
    fetchStreamInfo()
  }, [fetchStreamInfo])

  useEffect(() => {
    const id = setInterval(fetchStreamInfo, STREAM_POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [fetchStreamInfo])

  useEffect(() => {
    if (!showPicker) return
    const handler = (e: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setShowPicker(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showPicker])

  const toggleCamera = useCallback((cameraId: string) => {
    setMessage(null)
    setSelectedIds((current) => {
      if (current.includes(cameraId)) {
        if (expandedId === cameraId) {
          setExpandedId(null)
        }
        return current.filter((id) => id !== cameraId)
      }
      if (current.length >= MAX_MULTI_CAMERAS) {
        setMessage(`You can select up to ${MAX_MULTI_CAMERAS} cameras at once.`)
        return current
      }
      return [...current, cameraId]
    })
  }, [expandedId])

  const addCamera = useCallback((cameraId: string) => {
    setShowPicker(false)
    setMessage(null)
    setSelectedIds((current) => {
      if (current.includes(cameraId) || current.length >= MAX_MULTI_CAMERAS) return current
      return [...current, cameraId]
    })
  }, [])

  const handleScreenshotAll = useCallback(() => {
    let captured = 0
    let skipped = 0
    for (const camera of selectedCameras) {
      const video = videoRefs.current[camera.id]
      if (!video || video.readyState < 2 || video.videoWidth === 0 || video.videoHeight === 0) {
        skipped++
        continue
      }
      captureVideoScreenshot(video, camera.name)
      captured++
    }
    setMessage(`Screenshots captured: ${captured}. Skipped: ${skipped}.`)
  }, [selectedCameras])

  const handleScreenshotAllDiscord = useCallback(async () => {
    if (selectedCameras.length === 0 || sendingDiscordScreenshots) return
    setSendingDiscordScreenshots(true)
    setMessage(null)
    const results = await Promise.allSettled(
      selectedCameras.map((camera) => sendDiscordScreenshot(camera.id, false))
    )
    const success = results.filter((result) => result.status === 'fulfilled').length
    const failed = results.length - success
    setMessage(`Discord screenshots started for ${success} camera(s). Failed: ${failed}.`)
    setSendingDiscordScreenshots(false)
  }, [selectedCameras, sendingDiscordScreenshots])

  const handleRecordAllDiscord = useCallback(async () => {
    if (selectedCameras.length === 0 || sendingDiscordRecordings) return
    setSendingDiscordRecordings(true)
    setMessage(null)
    const results = await Promise.allSettled(
      selectedCameras.map((camera) =>
        sendDiscordRecording(
          camera.id,
          camera.discord_record_duration_seconds ?? 60,
          camera.discord_record_format ?? 'webp'
        )
      )
    )
    const success = results.filter((result) => result.status === 'fulfilled').length
    const failed = results.length - success
    setMessage(`Discord recordings started for ${success} camera(s). Failed: ${failed}.`)
    setSendingDiscordRecordings(false)
  }, [selectedCameras, sendingDiscordRecordings])

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <span className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent" aria-hidden />
        <span className="sr-only">Loading cameras…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
        >
          Dashboard
        </button>
        <div className="rounded-lg border border-status-offline/50 bg-status-offline/10 px-4 py-6 text-text-primary">
          <p className="font-medium">Could not load cameras</p>
          <p className="mt-1 text-sm text-text-muted">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
        >
          Dashboard
        </button>
        <div className="text-sm text-text-muted">
          Multi-camera mode ({selectedIds.length}/{MAX_MULTI_CAMERAS})
        </div>
      </div>

      <div className="rounded-lg border border-border bg-card p-4">
        <h2 className="text-sm font-semibold text-text-primary">Select cameras (up to 4)</h2>
        <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {cameras.map((camera) => {
            const checked = selectedIds.includes(camera.id)
            return (
              <label key={camera.id} className="flex cursor-pointer items-center gap-2 rounded border border-border px-3 py-2 text-sm text-text-primary">
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => toggleCamera(camera.id)}
                  className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
                />
                <span className="truncate" title={camera.name}>{camera.name}</span>
              </label>
            )
          })}
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handleScreenshotAll}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent"
          >
            Screenshot All
          </button>
          <button
            type="button"
            onClick={handleScreenshotAllDiscord}
            disabled={sendingDiscordScreenshots}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
          >
            {sendingDiscordScreenshots ? 'Sending Screenshots…' : 'Screenshot All to Discord'}
          </button>
          <button
            type="button"
            onClick={handleRecordAllDiscord}
            disabled={sendingDiscordRecordings}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
          >
            {sendingDiscordRecordings ? 'Starting Recordings…' : 'Record All to Discord'}
          </button>
        </div>

        {message && <p className="mt-3 text-sm text-text-muted">{message}</p>}
      </div>

      {selectedCameras.length === 0 ? (
        <p className="rounded-lg border border-border bg-card px-4 py-6 text-sm text-text-muted">
          Select one or more cameras to start multi-camera mode.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {selectedCameras.map((camera) => {
            const stream = streamInfoByCameraId[camera.id] ?? {
              hls_url: '',
              status: 'initializing' as const,
            }
            const expanded = expandedId === camera.id
            return (
              <section
                key={camera.id}
                className={`rounded-lg border border-border bg-card p-3 transition-all ${expanded ? 'lg:col-span-2' : ''}`}
              >
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <div className="flex min-w-0 items-center gap-2">
                    <h3 className="truncate text-sm font-semibold text-text-primary" title={camera.name}>{camera.name}</h3>
                    <StatusBadge status={stream.status} />
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => setExpandedId((current) => current === camera.id ? null : camera.id)}
                      className="rounded-lg border border-border bg-card px-2.5 py-1 text-xs text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent"
                    >
                      {expanded ? 'Shrink' : 'Enlarge'}
                    </button>
                    <button
                      type="button"
                      onClick={() => onSelectCamera(camera.id)}
                      className="rounded-lg border border-border bg-card px-2.5 py-1 text-xs text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent"
                    >
                      Open
                    </button>
                    <button
                      type="button"
                      onClick={() => toggleCamera(camera.id)}
                      title="Remove from view"
                      className="rounded-lg border border-border bg-card px-2.5 py-1 text-xs text-text-muted transition-colors hover:border-status-offline/40 hover:bg-status-offline/10 hover:text-status-offline focus:outline-none focus:ring-2 focus:ring-accent"
                    >
                      ✕
                    </button>
                  </div>
                </div>
                <VideoPlayer
                  hlsUrl={stream.hls_url || null}
                  screenshotLabel={camera.name}
                  showOverlay={false}
                  onVideoElement={(video) => { videoRefs.current[camera.id] = video }}
                />
              </section>
            )
          })}

          {/* Add camera card — shown when slots remain and unselected cameras exist */}
          {selectedIds.length < MAX_MULTI_CAMERAS && cameras.some((c) => !selectedIds.includes(c.id)) && (
            <div ref={pickerRef} className="relative">
              <button
                type="button"
                onClick={() => setShowPicker((v) => !v)}
                className="flex aspect-video w-full flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-border bg-card text-text-muted transition-colors hover:border-accent/50 hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
              >
                <svg className="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4v16m8-8H4" />
                </svg>
                <span className="text-sm font-medium">Add Camera</span>
              </button>

              {showPicker && (
                <div className="absolute left-0 right-0 top-full z-10 mt-1 overflow-hidden rounded-lg border border-border bg-card shadow-lg">
                  {cameras
                    .filter((c) => !selectedIds.includes(c.id))
                    .map((c) => (
                      <button
                        key={c.id}
                        type="button"
                        onClick={() => addCamera(c.id)}
                        className="w-full px-4 py-2.5 text-left text-sm text-text-primary transition-colors hover:bg-card-hover focus:bg-card-hover focus:outline-none"
                      >
                        {c.name}
                      </button>
                    ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
