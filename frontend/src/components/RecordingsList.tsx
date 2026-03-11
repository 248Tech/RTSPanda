import { useCallback, useEffect, useState } from 'react'
import { listRecordings, deleteRecording, downloadRecordingUrl, formatBytes } from '../api/recordings'
import type { Recording } from '../api/recordings'

export interface RecordingsListProps {
  cameraId: string
  recordEnabled: boolean
}

export function RecordingsList({ cameraId, recordEnabled }: RecordingsListProps) {
  const [recordings, setRecordings] = useState<Recording[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [deletingFile, setDeletingFile] = useState<string | null>(null)

  const fetchRecordings = useCallback(async () => {
    try {
      const list = await listRecordings(cameraId)
      setRecordings(list)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load recordings')
    } finally {
      setLoading(false)
    }
  }, [cameraId])

  useEffect(() => {
    fetchRecordings()
  }, [fetchRecordings])

  const handleDelete = useCallback(async (filename: string) => {
    setDeletingFile(filename)
    try {
      await deleteRecording(cameraId, filename)
      setRecordings((prev) => prev.filter((r) => r.filename !== filename))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed')
    } finally {
      setDeletingFile(null)
    }
  }, [cameraId])

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-text-primary">Recordings</h3>
        <div className="flex items-center gap-2">
          {recordEnabled ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-status-online/10 px-2 py-0.5 text-xs font-medium text-status-online">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-status-online" aria-hidden />
              Recording
            </span>
          ) : (
            <span className="text-xs text-text-muted">Recording off</span>
          )}
          <button
            type="button"
            onClick={fetchRecordings}
            className="rounded p-1 text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
            title="Refresh"
          >
            <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {error && (
        <p className="text-xs text-status-offline" role="alert">{error}</p>
      )}

      {loading ? (
        <div className="flex items-center gap-2 py-3 text-xs text-text-muted">
          <span className="h-4 w-4 animate-spin rounded-full border-2 border-accent border-t-transparent" aria-hidden />
          Loading…
        </div>
      ) : recordings.length === 0 ? (
        <p className="py-4 text-center text-sm text-text-muted">
          {recordEnabled ? 'No recordings yet — segments appear after the first hour.' : 'Enable recording in camera settings to save footage.'}
        </p>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border bg-card">
          {recordings.map((rec) => (
            <li key={rec.filename} className="flex items-center justify-between gap-3 px-3 py-2.5">
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium text-text-primary" title={rec.filename}>
                  {rec.filename}
                </p>
                <p className="mt-0.5 text-xs text-text-muted">
                  {formatBytes(rec.size_bytes)} &middot; {new Date(rec.created_at).toLocaleString()}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-1.5">
                <a
                  href={downloadRecordingUrl(cameraId, rec.filename)}
                  download={rec.filename}
                  className="rounded p-1.5 text-text-muted transition-colors hover:bg-card-hover hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
                  title="Download"
                >
                  <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                </a>
                <button
                  type="button"
                  onClick={() => handleDelete(rec.filename)}
                  disabled={deletingFile === rec.filename}
                  className="rounded p-1.5 text-text-muted transition-colors hover:bg-status-offline/10 hover:text-status-offline focus:outline-none focus:ring-2 focus:ring-status-offline disabled:opacity-50"
                  title="Delete"
                >
                  {deletingFile === rec.filename ? (
                    <span className="block h-3.5 w-3.5 animate-spin rounded-full border-2 border-status-offline border-t-transparent" aria-hidden />
                  ) : (
                    <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  )}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
