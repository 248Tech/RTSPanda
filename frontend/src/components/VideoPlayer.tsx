import Hls from 'hls.js'
import { useCallback, useEffect, useRef, useState } from 'react'

/** Captures the current video frame and triggers a PNG download. */
function captureScreenshot(video: HTMLVideoElement, label: string) {
  const canvas = document.createElement('canvas')
  canvas.width = video.videoWidth || 1280
  canvas.height = video.videoHeight || 720
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
  canvas.toBlob((blob) => {
    if (!blob) return
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    a.download = `${label}_${ts}.png`
    a.click()
    URL.revokeObjectURL(url)
  }, 'image/png')
}

export type PlayerState = 'idle' | 'loading' | 'playing' | 'error'

export interface OverlayDetection {
  id: string
  label: string
  confidence: number
  bbox: {
    x: number
    y: number
    width: number
    height: number
  }
  frameWidth?: number
  frameHeight?: number
}

export interface VideoPlayerProps {
  /** HLS manifest URL (e.g. /hls/camera-{id}/index.m3u8). When null, nothing is loaded. */
  hlsUrl: string | null
  /** Label used for screenshot filenames, e.g. camera name. */
  screenshotLabel?: string
  /** Optional live detections to render on top of video. */
  overlayDetections?: OverlayDetection[]
  /** Controls whether live overlay boxes are visible. */
  showOverlay?: boolean
  /** Called when user requests retry after an error. */
  onRetry?: () => void
  className?: string
}

function canPlayNativeHLS(video: HTMLVideoElement): boolean {
  return video.canPlayType('application/vnd.apple.mpegurl') !== ''
}

export function VideoPlayer({
  hlsUrl,
  screenshotLabel = 'screenshot',
  overlayDetections = [],
  showOverlay = true,
  onRetry,
  className = '',
}: VideoPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const hlsUrlRef = useRef<string | null>(null)
  const [state, setState] = useState<PlayerState>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [videoSize, setVideoSize] = useState<{ width: number; height: number }>({ width: 0, height: 0 })
  const [containerSize, setContainerSize] = useState<{ width: number; height: number }>({ width: 0, height: 0 })

  useEffect(() => {
    hlsUrlRef.current = hlsUrl
  }, [hlsUrl])

  const loadSource = useCallback((url: string) => {
    const video = videoRef.current
    if (!video) return

    setState('loading')
    setErrorMessage(null)

    if (Hls.isSupported()) {
      const hls = new Hls({
        enableWorker: true,
        lowLatencyMode: true,
      })
      hlsRef.current = hls

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        setState((s) => (s === 'error' ? s : 'loading'))
      })
      hls.on(Hls.Events.FRAG_BUFFERED, () => {
        setState((s) => (s === 'error' ? s : 'playing'))
      })
      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal) {
          setState('error')
          setErrorMessage(
            data.type === Hls.ErrorTypes.NETWORK_ERROR ? 'Network error' : 'Playback error'
          )
        }
      })

      hls.loadSource(url)
      hls.attachMedia(video)
      return
    }

    if (canPlayNativeHLS(video)) {
      video.src = url
      video.addEventListener('loadeddata', () => setState('playing'), { once: true })
      video.addEventListener('canplay', () => setState('playing'), { once: true })
      video.addEventListener('error', () => {
        setState('error')
        setErrorMessage('Failed to load stream')
      }, { once: true })
      return
    }

    setState('error')
    setErrorMessage('HLS not supported in this browser')
  }, [])

  useEffect(() => {
    if (!hlsUrl) {
      return
    }
    const video = videoRef.current
    if (!video) return

    const loadTimer = window.setTimeout(() => {
      loadSource(hlsUrl)
    }, 0)

    return () => {
      window.clearTimeout(loadTimer)
      const hls = hlsRef.current
      if (hls) {
        hls.destroy()
        hlsRef.current = null
      }
      video.removeAttribute('src')
      video.load()
      setState('idle')
      setErrorMessage(null)
    }
  }, [hlsUrl, loadSource])

  useEffect(() => {
    const video = videoRef.current
    if (!video || !hlsUrl) return
    const onWaiting = () => setState((s) => (s === 'playing' ? 'loading' : s))
    const onPlaying = () => setState((s) => (s === 'error' ? s : 'playing'))
    const onMetadata = () => {
      setVideoSize({
        width: video.videoWidth || 0,
        height: video.videoHeight || 0,
      })
    }
    video.addEventListener('waiting', onWaiting)
    video.addEventListener('playing', onPlaying)
    video.addEventListener('loadedmetadata', onMetadata)
    onMetadata()
    return () => {
      video.removeEventListener('waiting', onWaiting)
      video.removeEventListener('playing', onPlaying)
      video.removeEventListener('loadedmetadata', onMetadata)
    }
  }, [hlsUrl])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const updateSize = () => {
      const rect = container.getBoundingClientRect()
      setContainerSize({ width: rect.width, height: rect.height })
    }

    updateSize()
    const observer = new ResizeObserver(updateSize)
    observer.observe(container)
    return () => observer.disconnect()
  }, [])

  const handleRetry = useCallback(() => {
    const url = hlsUrlRef.current
    if (url) {
      const hls = hlsRef.current
      if (hls) {
        setState('loading')
        setErrorMessage(null)
        hls.loadSource(url)
      } else {
        const video = videoRef.current
        if (video && canPlayNativeHLS(video)) {
          setState('loading')
          setErrorMessage(null)
          video.src = url
        }
      }
    }
    onRetry?.()
  }, [onRetry])

  const handleScreenshot = useCallback(() => {
    const video = videoRef.current
    if (!video) return
    captureScreenshot(video, screenshotLabel)
  }, [screenshotLabel])

  const getOverlayBox = useCallback((detection: OverlayDetection) => {
    const cw = containerSize.width
    const ch = containerSize.height
    if (cw <= 0 || ch <= 0) {
      return null
    }

    const sourceWidth = detection.frameWidth && detection.frameWidth > 0
      ? detection.frameWidth
      : videoSize.width
    const sourceHeight = detection.frameHeight && detection.frameHeight > 0
      ? detection.frameHeight
      : videoSize.height
    if (sourceWidth <= 0 || sourceHeight <= 0) {
      return null
    }

    const videoAspectWidth = videoSize.width > 0 ? videoSize.width : sourceWidth
    const videoAspectHeight = videoSize.height > 0 ? videoSize.height : sourceHeight
    const scale = Math.min(cw / videoAspectWidth, ch / videoAspectHeight)
    const renderedWidth = videoAspectWidth * scale
    const renderedHeight = videoAspectHeight * scale
    const offsetX = (cw - renderedWidth) / 2
    const offsetY = (ch - renderedHeight) / 2

    const xNorm = detection.bbox.x / sourceWidth
    const yNorm = detection.bbox.y / sourceHeight
    const wNorm = detection.bbox.width / sourceWidth
    const hNorm = detection.bbox.height / sourceHeight

    return {
      left: offsetX + xNorm * renderedWidth,
      top: offsetY + yNorm * renderedHeight,
      width: wNorm * renderedWidth,
      height: hNorm * renderedHeight,
    }
  }, [containerSize.height, containerSize.width, videoSize.height, videoSize.width])

  const colorForLabel = useCallback((label: string) => {
    const palette = ['#f97316', '#22c55e', '#06b6d4', '#eab308', '#a855f7', '#ef4444']
    const seed = label
      .split('')
      .reduce((acc, char) => acc + char.charCodeAt(0), 0)
    return palette[seed % palette.length]
  }, [])

  return (
    <div ref={containerRef} className={`relative aspect-video w-full overflow-hidden rounded-lg bg-black ${className}`}>
      <video
        ref={videoRef}
        className="h-full w-full object-contain"
        playsInline
        muted
        autoPlay
        controls
        aria-label="Camera stream"
      />
      {showOverlay && state === 'playing' && overlayDetections.length > 0 && (
        <div className="pointer-events-none absolute inset-0">
          {overlayDetections.map((detection) => {
            const box = getOverlayBox(detection)
            if (!box) return null
            const color = colorForLabel(detection.label)
            return (
              <div
                key={detection.id}
                className="absolute rounded-md border-2 shadow-[0_0_0_1px_rgba(0,0,0,0.4)]"
                style={{
                  left: `${box.left}px`,
                  top: `${box.top}px`,
                  width: `${box.width}px`,
                  height: `${box.height}px`,
                  borderColor: color,
                }}
              >
                <div
                  className="absolute -top-6 left-0 rounded px-1.5 py-0.5 text-[10px] font-semibold text-white"
                  style={{ backgroundColor: color }}
                >
                  {`${detection.label} ${(detection.confidence * 100).toFixed(0)}%`}
                </div>
              </div>
            )
          })}
        </div>
      )}
      {/* Screenshot button — only shown when playing */}
      {state === 'playing' && (
        <div className="absolute right-2 top-2 opacity-0 transition-opacity hover:opacity-100 group-hover:opacity-100 [.group:hover_&]:opacity-100">
          <button
            type="button"
            onClick={handleScreenshot}
            title="Save screenshot"
            className="flex items-center gap-1.5 rounded-lg bg-black/60 px-2.5 py-1.5 text-xs font-medium text-white backdrop-blur-sm transition-colors hover:bg-black/80 focus:outline-none focus:ring-2 focus:ring-accent"
          >
            <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Screenshot
          </button>
        </div>
      )}
      {state === 'loading' && (
        <div
          className="absolute inset-0 flex items-center justify-center bg-base/80"
          aria-hidden
        >
          <span className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent" />
          <span className="sr-only">Loading stream…</span>
        </div>
      )}
      {state === 'error' && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-base/95 p-4">
          <p className="text-center text-sm text-text-primary">{errorMessage ?? 'Playback failed'}</p>
          {onRetry && (
            <button
              type="button"
              onClick={handleRetry}
              className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent"
            >
              Retry
            </button>
          )}
        </div>
      )}
    </div>
  )
}
