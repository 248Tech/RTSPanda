import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  getCamera,
  getStreamInfo,
  updateCamera,
  type Camera,
} from '../api/cameras'
import {
  detectionSnapshotUrl,
  listDetectionEvents,
  sendDiscordRecording,
  sendDiscordScreenshot,
  triggerTestDetection,
  type DetectionEvent,
} from '../api/detections'
import { StatusBadge } from '../components/StatusBadge'
import type { StreamStatus } from '../components/StatusBadge'
import { VideoPlayer } from '../components/VideoPlayer'
import { RecordingsList } from '../components/RecordingsList'

export interface CameraViewProps {
  cameraId: string
  onBack: () => void
  onNavigateSettings: () => void
}

interface DetectionEventGroup {
  key: string
  createdAt: string
  snapshotEventId: string
  events: DetectionEvent[]
}

const DETECTION_POLL_INTERVAL_MS = 3000
const OVERLAY_STALE_AFTER_MS = 12000

function formatDateTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}

function groupDetectionEvents(events: DetectionEvent[]): DetectionEventGroup[] {
  const groups: DetectionEventGroup[] = []
  const indexByKey = new Map<string, number>()

  for (const event of events) {
    const key = `${event.created_at}|${event.snapshot_path}`
    const existing = indexByKey.get(key)
    if (existing === undefined) {
      indexByKey.set(key, groups.length)
      groups.push({
        key,
        createdAt: event.created_at,
        snapshotEventId: event.id,
        events: [event],
      })
    } else {
      groups[existing].events.push(event)
    }
  }

  return groups
}

export default function CameraView({ cameraId, onBack, onNavigateSettings }: CameraViewProps) {
  const [camera, setCamera] = useState<Camera | null>(null)
  const [hlsUrl, setHlsUrl] = useState<string | null>(null)
  const [streamStatus, setStreamStatus] = useState<StreamStatus>('connecting')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [overlayEnabled, setOverlayEnabled] = useState(true)
  const [trackingBusy, setTrackingBusy] = useState(false)
  const [testingDetection, setTestingDetection] = useState(false)
  const [sendingDiscordScreenshot, setSendingDiscordScreenshot] = useState(false)
  const [sendingDiscordRecording, setSendingDiscordRecording] = useState(false)
  const [testMessage, setTestMessage] = useState<string | null>(null)
  const [detectionEvents, setDetectionEvents] = useState<DetectionEvent[]>([])
  const [detectionError, setDetectionError] = useState<string | null>(null)
  const [isLoadingEvents, setIsLoadingEvents] = useState(false)
  const [nowMs, setNowMs] = useState(() => Date.now())

  const fetchCameraAndStream = useCallback(async () => {
    setError(null)
    try {
      const [cameraRes, streamRes] = await Promise.all([
        getCamera(cameraId),
        getStreamInfo(cameraId),
      ])
      setCamera(cameraRes)
      setHlsUrl(streamRes.hls_url)
      setStreamStatus(streamRes.status)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load camera')
      setCamera(null)
      setHlsUrl(null)
      setStreamStatus('offline')
    } finally {
      setLoading(false)
    }
  }, [cameraId])

  const fetchDetectionEvents = useCallback(async (showSpinner = false) => {
    if (showSpinner) {
      setIsLoadingEvents(true)
    }
    try {
      const events = await listDetectionEvents(cameraId, 120)
      setDetectionEvents(events)
      setDetectionError(null)
    } catch (e) {
      setDetectionError(e instanceof Error ? e.message : 'Failed to load detection events')
    } finally {
      if (showSpinner) {
        setIsLoadingEvents(false)
      }
    }
  }, [cameraId])

  useEffect(() => {
    fetchCameraAndStream()
  }, [fetchCameraAndStream])

  useEffect(() => {
    fetchDetectionEvents(true)
  }, [fetchDetectionEvents])

  useEffect(() => {
    if (!camera?.tracking_enabled) return
    const id = setInterval(() => {
      void fetchDetectionEvents()
    }, DETECTION_POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [camera?.tracking_enabled, fetchDetectionEvents])

  useEffect(() => {
    const id = setInterval(() => setNowMs(Date.now()), 1000)
    return () => clearInterval(id)
  }, [])

  const overlayDetections = useMemo(() => {
    if (!overlayEnabled) return []
    if (detectionEvents.length === 0) return []

    const latestGroup = groupDetectionEvents(detectionEvents)[0]
    if (!latestGroup) return []

    const latestTs = new Date(latestGroup.createdAt).getTime()
    if (Number.isNaN(latestTs)) return []
    if (nowMs - latestTs > OVERLAY_STALE_AFTER_MS) return []

    return latestGroup.events.map((event) => ({
      id: event.id,
      label: event.object_label,
      confidence: event.confidence,
      bbox: event.bbox,
      frameWidth: event.frame_width,
      frameHeight: event.frame_height,
    }))
  }, [detectionEvents, nowMs, overlayEnabled])

  const groupedEvents = useMemo(() => {
    return groupDetectionEvents(detectionEvents).slice(0, 30)
  }, [detectionEvents])

  const handleRetry = useCallback(() => {
    void fetchCameraAndStream()
  }, [fetchCameraAndStream])

  const handleToggleTracking = useCallback(async () => {
    if (!camera || trackingBusy) return
    setTrackingBusy(true)
    setTestMessage(null)
    try {
      const updated = await updateCamera(camera.id, {
        tracking_enabled: !camera.tracking_enabled,
      })
      setCamera(updated)
      if (updated.tracking_enabled) {
        await fetchDetectionEvents()
      }
    } catch (e) {
      setTestMessage(e instanceof Error ? e.message : 'Failed to update tracking')
    } finally {
      setTrackingBusy(false)
    }
  }, [camera, fetchDetectionEvents, trackingBusy])

  const handleTestDetection = useCallback(async () => {
    if (!camera || testingDetection) return
    setTestingDetection(true)
    setTestMessage(null)
    try {
      const response = await triggerTestDetection(camera.id)
      const count = response.detections.length
      setTestMessage(count > 0 ? `Test detection found ${count} object(s).` : 'Test completed with no matching detections.')
      await fetchDetectionEvents()
    } catch (e) {
      setTestMessage(e instanceof Error ? e.message : 'Test detection failed')
    } finally {
      setTestingDetection(false)
    }
  }, [camera, fetchDetectionEvents, testingDetection])

  const handleSendDiscordScreenshot = useCallback(async () => {
    if (!camera || sendingDiscordScreenshot) return
    setSendingDiscordScreenshot(true)
    setTestMessage(null)
    try {
      await sendDiscordScreenshot(camera.id, false)
      setTestMessage('Screenshot sent to Discord webhook.')
    } catch (e) {
      setTestMessage(e instanceof Error ? e.message : 'Failed to send screenshot to Discord')
    } finally {
      setSendingDiscordScreenshot(false)
    }
  }, [camera, sendingDiscordScreenshot])

  const handleSendDiscordRecording = useCallback(async () => {
    if (!camera || sendingDiscordRecording) return
    setSendingDiscordRecording(true)
    setTestMessage(null)
    try {
      const durationSeconds = camera.discord_record_duration_seconds ?? 60
      const format = camera.discord_record_format ?? 'webp'
      await sendDiscordRecording(camera.id, durationSeconds, format)
      setTestMessage(`Recording sent to Discord (${durationSeconds}s ${format}).`)
    } catch (e) {
      setTestMessage(e instanceof Error ? e.message : 'Failed to send recording to Discord')
    } finally {
      setSendingDiscordRecording(false)
    }
  }, [camera, sendingDiscordRecording])

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <span
          className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent"
          aria-hidden
        />
        <span className="sr-only">Loading camera…</span>
      </div>
    )
  }

  if (error || !camera) {
    return (
      <div className="space-y-4">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Dashboard
        </button>
        <div className="rounded-lg border border-status-offline/50 bg-status-offline/10 px-4 py-6 text-text-primary">
          <p className="font-medium">Could not load camera</p>
          <p className="mt-1 text-sm text-text-muted">{error ?? 'Not found'}</p>
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
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Dashboard
        </button>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setOverlayEnabled((value) => !value)}
            className={`rounded-lg px-3 py-1.5 text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-accent ${
              overlayEnabled ? 'bg-accent/15 text-accent' : 'bg-card text-text-muted hover:bg-card-hover hover:text-text-primary'
            }`}
          >
            {overlayEnabled ? 'Overlay On' : 'Overlay Off'}
          </button>
          <button
            type="button"
            onClick={handleToggleTracking}
            disabled={trackingBusy}
            className={`rounded-lg px-3 py-1.5 text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60 ${
              camera.tracking_enabled
                ? 'bg-status-online/15 text-status-online hover:bg-status-online/25'
                : 'bg-card text-text-muted hover:bg-card-hover hover:text-text-primary'
            }`}
          >
            {trackingBusy
              ? 'Updating...'
              : camera.tracking_enabled
              ? 'Tracking Enabled'
              : 'Tracking Disabled'}
          </button>
          <button
            type="button"
            onClick={handleTestDetection}
            disabled={testingDetection}
            className="rounded-lg bg-accent px-3 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
          >
            {testingDetection ? 'Testing...' : 'Run Test Detection'}
          </button>
          <button
            type="button"
            onClick={handleSendDiscordScreenshot}
            disabled={sendingDiscordScreenshot || camera.discord_webhook_url.trim() === ''}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-60"
            title={
              camera.discord_webhook_url.trim() === ''
                ? 'Configure camera Discord webhook first'
                : 'Capture and send a screenshot to Discord'
            }
          >
            {sendingDiscordScreenshot ? 'Sending Screenshot...' : 'Screenshot to Discord'}
          </button>
          <button
            type="button"
            onClick={handleSendDiscordRecording}
            disabled={sendingDiscordRecording || camera.discord_webhook_url.trim() === ''}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-60"
            title={
              camera.discord_webhook_url.trim() === ''
                ? 'Configure camera Discord webhook first'
                : 'Record and send a clip to Discord'
            }
          >
            {sendingDiscordRecording ? 'Recording...' : 'Record to Discord'}
          </button>
          <button
            type="button"
            onClick={onNavigateSettings}
            className="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
          >
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Settings
          </button>
        </div>
      </div>

      {testMessage && (
        <div className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-text-primary">
          {testMessage}
        </div>
      )}

      <div className="group">
        <VideoPlayer
          hlsUrl={hlsUrl}
          screenshotLabel={camera.name}
          overlayDetections={overlayDetections}
          showOverlay={overlayEnabled}
          onRetry={handleRetry}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-text-primary">{camera.name}</h2>
          <p className="mt-0.5 truncate text-sm text-text-muted" title={camera.rtsp_url}>
            {camera.rtsp_url}
          </p>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs">
            <span className={`rounded-full px-2 py-0.5 ${camera.tracking_enabled ? 'bg-accent/15 text-accent' : 'bg-card text-text-muted'}`}>
              {camera.tracking_enabled ? 'YOLOv8 tracking enabled' : 'YOLOv8 tracking disabled'}
            </span>
            <span className="rounded-full bg-card px-2 py-0.5 text-text-muted">
              Min confidence: {(camera.tracking_min_confidence ?? 0.25).toFixed(2)}
            </span>
            <span className="rounded-full bg-card px-2 py-0.5 text-text-muted">
              Interval: {camera.detection_sample_seconds ?? 30}s
            </span>
            {camera.discord_alerts_enabled && (
              <span className="rounded-full bg-status-online/15 px-2 py-0.5 text-status-online">
                Discord alerts enabled
              </span>
            )}
          </div>
        </div>
        <StatusBadge status={streamStatus} />
      </div>

      <div className="border-t border-border pt-4">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-base font-semibold text-text-primary">Detection Event History</h3>
          <button
            type="button"
            onClick={() => void fetchDetectionEvents(true)}
            disabled={isLoadingEvents}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent disabled:opacity-60"
          >
            {isLoadingEvents ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>

        {detectionError && (
          <p className="mb-3 text-sm text-status-offline" role="alert">
            {detectionError}
          </p>
        )}

        {!camera.tracking_enabled && (
          <p className="mb-3 rounded-lg border border-border bg-card px-3 py-2 text-sm text-text-muted">
            Tracking is disabled for this camera. Enable tracking to generate YOLOv8 events and overlays.
          </p>
        )}

        {groupedEvents.length === 0 ? (
          <p className="rounded-lg border border-border bg-card px-4 py-6 text-sm text-text-muted">
            No detection events yet.
          </p>
        ) : (
          <ul className="space-y-3">
            {groupedEvents.map((group) => (
              <li key={group.key} className="overflow-hidden rounded-lg border border-border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-2">
                  <p className="text-sm font-medium text-text-primary">{formatDateTime(group.createdAt)}</p>
                  <span className="text-xs text-text-muted">{group.events.length} detection(s)</span>
                </div>
                <div className="grid grid-cols-1 gap-3 p-3 md:grid-cols-[220px_1fr]">
                  <img
                    src={detectionSnapshotUrl(group.snapshotEventId)}
                    alt={`Detection snapshot from ${formatDateTime(group.createdAt)}`}
                    className="aspect-video w-full rounded-md border border-border object-cover"
                    loading="lazy"
                  />
                  <div className="flex flex-wrap content-start gap-2">
                    {group.events.map((event) => (
                      <span
                        key={event.id}
                        className="inline-flex items-center gap-1 rounded-full bg-accent/15 px-2.5 py-1 text-xs text-accent"
                      >
                        <span className="font-medium">{event.object_label}</span>
                        <span>{(event.confidence * 100).toFixed(0)}%</span>
                      </span>
                    ))}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="border-t border-border pt-4">
        <RecordingsList cameraId={camera.id} recordEnabled={camera.record_enabled} />
      </div>
    </div>
  )
}
