import { useCallback, useEffect, useRef, useState } from 'react'
import {
  getCameras,
  addCamera,
  updateCamera,
  deleteCamera,
  type Camera,
  type CreateCameraInput,
} from '../api/cameras'
import {
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  type AlertRule,
  type AlertType,
} from '../api/alerts'
import { getLogs } from '../api/logs'
import {
  getAppSettings,
  updateAppSettings,
  type AppSettings,
  type VideoStorageProvider,
} from '../api/settings'
import { CameraForm } from '../components/CameraForm'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Modal } from '../components/Modal'

type Tab = 'cameras' | 'alerts' | 'integrations' | 'logs'
type CameraModalMode = 'add' | 'edit' | 'delete' | null
type AlertModalMode = 'add' | 'edit' | 'delete' | null

const ALERT_TYPE_LABELS: Record<AlertType, string> = {
  motion: 'Motion Detection',
  connectivity: 'Connectivity',
  object_detection: 'Object Detection (YOLO)',
}

const ALERT_TYPE_DESCRIPTIONS: Record<AlertType, string> = {
  motion: 'Fires when movement is detected in the frame',
  connectivity: 'Fires when the camera goes offline or reconnects',
  object_detection: 'Fires when a specific YOLO object is detected',
}

// ─── Alert Rule Form ───────────────────────────────────────────────────────────

interface AlertFormValues {
  name: string
  type: AlertType
  enabled: boolean
}

interface AlertRuleFormProps {
  rule?: AlertRule | null
  onSubmit: (values: AlertFormValues) => void
  onCancel: () => void
  submitError?: string | null
  isSubmitting?: boolean
}

function AlertRuleForm({ rule, onSubmit, onCancel, submitError, isSubmitting = false }: AlertRuleFormProps) {
  const [values, setValues] = useState<AlertFormValues>(() => ({
    name: rule?.name ?? '',
    type: rule?.type ?? 'connectivity',
    enabled: rule?.enabled ?? true,
  }))
  const [nameError, setNameError] = useState<string | null>(null)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!values.name.trim()) {
      setNameError('Name is required')
      return
    }
    setNameError(null)
    onSubmit(values)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="alert-name" className="mb-1 block text-sm font-medium text-text-primary">
          Rule name
        </label>
        <input
          id="alert-name"
          type="text"
          value={values.name}
          onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
          placeholder="e.g. Front door offline"
          autoFocus
          className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
        />
        {nameError && <p className="mt-1 text-sm text-status-offline" role="alert">{nameError}</p>}
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-text-primary">Alert type</label>
        <div className="space-y-2">
          {(Object.keys(ALERT_TYPE_LABELS) as AlertType[]).map((type) => (
            <label
              key={type}
              className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                values.type === type ? 'border-accent bg-accent/5' : 'border-border hover:bg-card-hover'
              }`}
            >
              <input
                type="radio"
                name="alert-type"
                value={type}
                checked={values.type === type}
                onChange={() => setValues((v) => ({ ...v, type }))}
                className="mt-0.5 text-accent focus:ring-accent"
              />
              <div>
                <p className="text-sm font-medium text-text-primary">{ALERT_TYPE_LABELS[type]}</p>
                <p className="text-xs text-text-muted">{ALERT_TYPE_DESCRIPTIONS[type]}</p>
              </div>
            </label>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <input
          id="alert-enabled"
          type="checkbox"
          checked={values.enabled}
          onChange={(e) => setValues((v) => ({ ...v, enabled: e.target.checked }))}
          className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
        />
        <label htmlFor="alert-enabled" className="text-sm text-text-primary">Enabled</label>
      </div>

      {submitError && <p className="text-sm text-status-offline" role="alert">{submitError}</p>}

      <div className="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base disabled:opacity-50"
        >
          {isSubmitting ? 'Saving…' : rule ? 'Save' : 'Add Rule'}
        </button>
      </div>
    </form>
  )
}

// ─── Alerts panel ─────────────────────────────────────────────────────────────

function AlertsPanel({ cameras }: { cameras: Camera[] }) {
  const [selectedCameraId, setSelectedCameraId] = useState<string>(() => cameras[0]?.id ?? '')
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [modalMode, setModalMode] = useState<AlertModalMode>(null)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [deletingRule, setDeletingRule] = useState<AlertRule | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const fetchRules = useCallback(async (cameraId: string) => {
    if (!cameraId) return
    setLoading(true)
    setError(null)
    try {
      const list = await listAlertRules(cameraId)
      setRules(list)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load alert rules')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (selectedCameraId) fetchRules(selectedCameraId)
  }, [selectedCameraId, fetchRules])

  const openAdd = () => { setFormError(null); setModalMode('add') }
  const openEdit = (rule: AlertRule) => { setEditingRule(rule); setFormError(null); setModalMode('edit') }
  const openDelete = (rule: AlertRule) => { setDeletingRule(rule); setModalMode('delete') }
  const closeModal = () => {
    setModalMode(null); setEditingRule(null); setDeletingRule(null)
    setFormError(null)
  }

  const handleToggleEnabled = useCallback(async (rule: AlertRule) => {
    try {
      const updated = await updateAlertRule(rule.id, { enabled: !rule.enabled })
      setRules((prev) => prev.map((r) => (r.id === updated.id ? updated : r)))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Update failed')
    }
  }, [])

  const handleFormSubmit = useCallback(async (values: AlertFormValues) => {
    setFormError(null)
    setIsSubmitting(true)
    try {
      if (editingRule) {
        const updated = await updateAlertRule(editingRule.id, { name: values.name, type: values.type, enabled: values.enabled })
        setRules((prev) => prev.map((r) => (r.id === updated.id ? updated : r)))
      } else {
        const created = await createAlertRule(selectedCameraId, { name: values.name, type: values.type, enabled: values.enabled })
        setRules((prev) => [...prev, created])
      }
      closeModal()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : 'Request failed')
    } finally {
      setIsSubmitting(false)
    }
  }, [editingRule, selectedCameraId])

  const handleDeleteConfirm = useCallback(async () => {
    if (!deletingRule) return
    try {
      await deleteAlertRule(deletingRule.id)
      setRules((prev) => prev.filter((r) => r.id !== deletingRule.id))
      closeModal()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed')
    }
  }, [deletingRule])

  if (cameras.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-text-muted">
        Add a camera first, then configure alert rules for it.
      </p>
    )
  }

  return (
    <div className="space-y-5">
      {/* Camera selector */}
      <div className="flex flex-wrap items-center gap-3">
        <label htmlFor="alert-camera-select" className="text-sm font-medium text-text-primary shrink-0">
          Camera
        </label>
        <select
          id="alert-camera-select"
          value={selectedCameraId}
          onChange={(e) => setSelectedCameraId(e.target.value)}
          className="rounded-lg border border-border bg-base px-3 py-1.5 text-sm text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
        >
          {cameras.map((c) => (
            <option key={c.id} value={c.id}>{c.name}</option>
          ))}
        </select>
        <div className="ml-auto">
          <button
            type="button"
            onClick={openAdd}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
          >
            + Add Rule
          </button>
        </div>
      </div>

      {/* Callout: legacy compatibility note */}
      <div className="flex gap-3 rounded-lg border border-accent/20 bg-accent/5 px-4 py-3 text-sm">
        <svg className="mt-0.5 h-4 w-4 shrink-0 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div className="text-text-muted">
          <span className="font-medium text-text-primary">Legacy alert rules.</span>{' '}
          Primary alerting is now provider-based Discord alerts (YOLO or Frigate) in camera settings. This endpoint remains for compatibility:{' '}
          <code className="rounded bg-card px-1 font-mono text-xs text-accent">
            POST /api/v1/alerts/&#123;id&#125;/events
          </code>
        </div>
      </div>

      {error && <p className="text-sm text-status-offline" role="alert">{error}</p>}

      {loading ? (
        <div className="flex items-center gap-2 py-4 text-sm text-text-muted">
          <span className="h-4 w-4 animate-spin rounded-full border-2 border-accent border-t-transparent" aria-hidden />
          Loading rules…
        </div>
      ) : rules.length === 0 ? (
        <div className="rounded-lg border border-border bg-card px-6 py-10 text-center">
          <p className="text-text-muted">No alert rules yet.</p>
          <button type="button" onClick={openAdd} className="mt-4 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent">
            + Add Rule
          </button>
        </div>
      ) : (
        <ul className="space-y-2">
          {rules.map((rule) => (
            <li key={rule.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-card px-4 py-3">
              <div className="min-w-0 flex-1">
                <p className="font-medium text-text-primary">{rule.name}</p>
                <p className="text-xs text-text-muted">{ALERT_TYPE_LABELS[rule.type]}</p>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  onClick={() => handleToggleEnabled(rule)}
                  className={`rounded-full px-2.5 py-1 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-accent ${
                    rule.enabled
                      ? 'bg-status-online/10 text-status-online hover:bg-status-online/20'
                      : 'bg-card-hover text-text-muted hover:bg-card-hover'
                  }`}
                >
                  {rule.enabled ? 'Enabled' : 'Disabled'}
                </button>
                <button
                  type="button"
                  onClick={() => openEdit(rule)}
                  className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => openDelete(rule)}
                  className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-status-offline transition-colors hover:bg-status-offline/10 focus:outline-none focus:ring-2 focus:ring-status-offline focus:ring-offset-2 focus:ring-offset-base"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {modalMode === 'add' && (
        <Modal title="Add Alert Rule" onClose={closeModal}>
          <AlertRuleForm
            onSubmit={handleFormSubmit}
            onCancel={closeModal}
            submitError={formError}
            isSubmitting={isSubmitting}
          />
        </Modal>
      )}
      {modalMode === 'edit' && editingRule && (
        <Modal title="Edit Alert Rule" onClose={closeModal}>
          <AlertRuleForm
            rule={editingRule}
            onSubmit={handleFormSubmit}
            onCancel={closeModal}
            submitError={formError}
            isSubmitting={isSubmitting}
          />
        </Modal>
      )}
      {modalMode === 'delete' && deletingRule && (
        <Modal onClose={closeModal}>
          <ConfirmDialog
            title="Delete alert rule?"
            message={`"${deletingRule.name}" will be removed and all its event history will be lost.`}
            confirmLabel="Delete"
            cancelLabel="Cancel"
            variant="danger"
            onConfirm={handleDeleteConfirm}
            onCancel={closeModal}
          />
        </Modal>
      )}
    </div>
  )
}

// ─── Logs panel ──────────────────────────────────────────────────────────────

function LogsPanel() {
  const [lines, setLines] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const fetchLogs = useCallback(async () => {
    setError(null)
    try {
      const { lines: next } = await getLogs()
      setLines(next)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load logs')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchLogs()
  }, [fetchLogs])

  useEffect(() => {
    if (!autoRefresh) return
    const id = setInterval(fetchLogs, 3000)
    return () => clearInterval(id)
  }, [autoRefresh, fetchLogs])

  useEffect(() => {
    if (containerRef.current && lines.length > 0) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [lines])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-text-muted">
          Recent server log output (streams, cameras, mediamtx). Last 1000 lines.
        </p>
        <div className="flex items-center gap-3">
          <label className="flex cursor-pointer items-center gap-2 text-sm text-text-primary">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
            />
            Auto-refresh
          </label>
          <button
            type="button"
            onClick={() => { setLoading(true); fetchLogs() }}
            disabled={loading}
            className="rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base disabled:opacity-50"
          >
            {loading ? 'Loading…' : 'Refresh'}
          </button>
        </div>
      </div>

      {error && (
        <p className="text-sm text-status-offline" role="alert">{error}</p>
      )}

      <div
        ref={containerRef}
        className="max-h-[60vh] min-h-[200px] overflow-auto rounded-lg border border-border bg-card p-4"
      >
        {lines.length === 0 && !loading && !error ? (
          <p className="text-sm text-text-muted">No log lines yet.</p>
        ) : (
          <pre className="font-mono text-xs text-text-primary whitespace-pre-wrap break-all">
            {lines.join('\n') || '\n'}
          </pre>
        )}
      </div>
    </div>
  )
}

// ─── Integrations panel ───────────────────────────────────────────────────────

const VIDEO_STORAGE_PROVIDERS: Array<{ value: VideoStorageProvider; label: string }> = [
  { value: 'local_server', label: 'Local Server (NAS/SMB/NFS)' },
  { value: 'dropbox', label: 'Dropbox (rclone)' },
  { value: 'google_drive', label: 'Google Drive (rclone)' },
  { value: 'onedrive', label: 'OneDrive (rclone)' },
  { value: 'proton_drive', label: 'Proton Drive (rclone)' },
]

function IntegrationsPanel() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const [settings, setSettings] = useState<AppSettings | null>(null)
  const [openAIEnabled, setOpenAIEnabled] = useState(false)
  const [openAIModel, setOpenAIModel] = useState('gpt-4o-mini')
  const [openAIAPIKey, setOpenAIAPIKey] = useState('')
  const [clearAPIKey, setClearAPIKey] = useState(false)
  const [videoStorageEnabled, setVideoStorageEnabled] = useState(false)
  const [videoStorageProvider, setVideoStorageProvider] = useState<VideoStorageProvider>('local_server')
  const [videoStorageLocalPath, setVideoStorageLocalPath] = useState('')
  const [videoStorageRemoteName, setVideoStorageRemoteName] = useState('')
  const [videoStorageRemotePath, setVideoStorageRemotePath] = useState('RTSPanda')
  const [videoStorageSyncIntervalSec, setVideoStorageSyncIntervalSec] = useState('300')
  const [videoStorageMinFileAgeSec, setVideoStorageMinFileAgeSec] = useState('120')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const cfg = await getAppSettings()
      setSettings(cfg)
      setOpenAIEnabled(cfg.openai_enabled)
      setOpenAIModel(cfg.openai_model || 'gpt-4o-mini')
      setOpenAIAPIKey('')
      setClearAPIKey(false)
      setVideoStorageEnabled(cfg.video_storage_enabled)
      setVideoStorageProvider(cfg.video_storage_provider || 'local_server')
      setVideoStorageLocalPath(cfg.video_storage_local_path || '')
      setVideoStorageRemoteName(cfg.video_storage_remote_name || '')
      setVideoStorageRemotePath(cfg.video_storage_remote_path || 'RTSPanda')
      setVideoStorageSyncIntervalSec(String(cfg.video_storage_sync_interval_seconds || 300))
      setVideoStorageMinFileAgeSec(String(cfg.video_storage_min_file_age_seconds || 120))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load app settings')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleSave = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      const syncInterval = Number.parseInt(videoStorageSyncIntervalSec, 10)
      if (!Number.isFinite(syncInterval) || syncInterval < 30 || syncInterval > 86400) {
        throw new Error('Video storage sync interval must be between 30 and 86400 seconds.')
      }
      const minFileAge = Number.parseInt(videoStorageMinFileAgeSec, 10)
      if (!Number.isFinite(minFileAge) || minFileAge < 15 || minFileAge > 3600) {
        throw new Error('Video storage minimum file age must be between 15 and 3600 seconds.')
      }

      const payload: Parameters<typeof updateAppSettings>[0] = {
        openai_enabled: openAIEnabled,
        openai_model: openAIModel.trim() || 'gpt-4o-mini',
        video_storage_enabled: videoStorageEnabled,
        video_storage_provider: videoStorageProvider,
        video_storage_local_path: videoStorageLocalPath.trim(),
        video_storage_remote_name: videoStorageRemoteName.trim(),
        video_storage_remote_path: videoStorageRemotePath.trim() || 'RTSPanda',
        video_storage_sync_interval_seconds: syncInterval,
        video_storage_min_file_age_seconds: minFileAge,
      }
      if (openAIAPIKey.trim() !== '') {
        payload.openai_api_key = openAIAPIKey.trim()
      }
      if (clearAPIKey) {
        payload.clear_openai_api_key = true
      }

      const updated = await updateAppSettings(payload)
      setSettings(updated)
      setOpenAIEnabled(updated.openai_enabled)
      setOpenAIModel(updated.openai_model || 'gpt-4o-mini')
      setOpenAIAPIKey('')
      setClearAPIKey(false)
      setVideoStorageEnabled(updated.video_storage_enabled)
      setVideoStorageProvider(updated.video_storage_provider || 'local_server')
      setVideoStorageLocalPath(updated.video_storage_local_path || '')
      setVideoStorageRemoteName(updated.video_storage_remote_name || '')
      setVideoStorageRemotePath(updated.video_storage_remote_path || 'RTSPanda')
      setVideoStorageSyncIntervalSec(String(updated.video_storage_sync_interval_seconds || 300))
      setVideoStorageMinFileAgeSec(String(updated.video_storage_min_file_age_seconds || 120))
      setMessage('Integration settings saved.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save app settings')
    } finally {
      setSaving(false)
    }
  }, [
    clearAPIKey,
    openAIAPIKey,
    openAIEnabled,
    openAIModel,
    videoStorageEnabled,
    videoStorageProvider,
    videoStorageLocalPath,
    videoStorageRemoteName,
    videoStorageRemotePath,
    videoStorageSyncIntervalSec,
    videoStorageMinFileAgeSec,
  ])

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-4 text-sm text-text-muted">
        <span className="h-4 w-4 animate-spin rounded-full border-2 border-accent border-t-transparent" aria-hidden />
        Loading integrations…
      </div>
    )
  }

  return (
    <form onSubmit={handleSave} className="space-y-5">
      <div className="rounded-lg border border-border bg-card p-4">
        <h3 className="text-base font-semibold text-text-primary">OpenAI Vision Captions</h3>
        <p className="mt-1 text-sm text-text-muted">
          Generate short and verbose scene text for Discord screenshot alerts (example: person at front door, package in driveway).
        </p>
        <p className="mt-2 text-xs text-text-muted">
          This sends snapshot images to OpenAI when enabled.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 rounded-lg border border-border bg-card p-4">
        <label className="flex items-center gap-2 text-sm text-text-primary">
          <input
            type="checkbox"
            checked={openAIEnabled}
            onChange={(e) => setOpenAIEnabled(e.target.checked)}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          Enable OpenAI captions for Discord screenshot alerts
        </label>

        <div>
          <label htmlFor="openai-model" className="mb-1 block text-sm text-text-primary">
            Model
          </label>
          <input
            id="openai-model"
            type="text"
            value={openAIModel}
            onChange={(e) => setOpenAIModel(e.target.value)}
            placeholder="gpt-4o-mini"
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div>
          <label htmlFor="openai-key" className="mb-1 block text-sm text-text-primary">
            OpenAI API key
          </label>
          <input
            id="openai-key"
            type="password"
            value={openAIAPIKey}
            onChange={(e) => setOpenAIAPIKey(e.target.value)}
            placeholder={settings?.openai_api_key_set ? 'Configured (enter new key to rotate)' : 'sk-...'}
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
          <p className="mt-1 text-xs text-text-muted">
            {settings?.openai_api_key_set ? 'A key is currently configured.' : 'No key configured yet.'}
          </p>
        </div>

        <label className="flex items-center gap-2 text-sm text-text-primary">
          <input
            type="checkbox"
            checked={clearAPIKey}
            onChange={(e) => setClearAPIKey(e.target.checked)}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          Clear saved OpenAI API key
        </label>
      </div>

      <div className="rounded-lg border border-border bg-card p-4">
        <h3 className="text-base font-semibold text-text-primary">External Video Storage</h3>
        <p className="mt-1 text-sm text-text-muted">
          Auto-sync completed recording files to a local server path or a cloud provider (Dropbox, Google Drive, OneDrive, Proton Drive).
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 rounded-lg border border-border bg-card p-4">
        <label className="flex items-center gap-2 text-sm text-text-primary">
          <input
            type="checkbox"
            checked={videoStorageEnabled}
            onChange={(e) => setVideoStorageEnabled(e.target.checked)}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          Enable external video storage sync
        </label>

        <div>
          <label htmlFor="video-storage-provider" className="mb-1 block text-sm text-text-primary">
            Provider
          </label>
          <select
            id="video-storage-provider"
            value={videoStorageProvider}
            onChange={(e) => setVideoStorageProvider(e.target.value as VideoStorageProvider)}
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          >
            {VIDEO_STORAGE_PROVIDERS.map((provider) => (
              <option key={provider.value} value={provider.value}>
                {provider.label}
              </option>
            ))}
          </select>
        </div>

        {videoStorageProvider === 'local_server' ? (
          <div>
            <label htmlFor="video-storage-local-path" className="mb-1 block text-sm text-text-primary">
              Local server destination path
            </label>
            <input
              id="video-storage-local-path"
              type="text"
              value={videoStorageLocalPath}
              onChange={(e) => setVideoStorageLocalPath(e.target.value)}
              placeholder="\\\\nas\\camera-archive\\rtspanda or /mnt/nas/rtspanda"
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        ) : (
          <>
            <div>
              <label htmlFor="video-storage-remote-name" className="mb-1 block text-sm text-text-primary">
                rclone remote name
              </label>
              <input
                id="video-storage-remote-name"
                type="text"
                value={videoStorageRemoteName}
                onChange={(e) => setVideoStorageRemoteName(e.target.value)}
                placeholder="dropbox, gdrive, onedrive, protondrive"
                className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
            <div>
              <label htmlFor="video-storage-remote-path" className="mb-1 block text-sm text-text-primary">
                Remote folder path
              </label>
              <input
                id="video-storage-remote-path"
                type="text"
                value={videoStorageRemotePath}
                onChange={(e) => setVideoStorageRemotePath(e.target.value)}
                placeholder="RTSPanda"
                className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
          </>
        )}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label htmlFor="video-storage-sync-interval" className="mb-1 block text-sm text-text-primary">
              Sync interval (seconds)
            </label>
            <input
              id="video-storage-sync-interval"
              type="number"
              min={30}
              max={86400}
              step={1}
              value={videoStorageSyncIntervalSec}
              onChange={(e) => setVideoStorageSyncIntervalSec(e.target.value)}
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label htmlFor="video-storage-min-age" className="mb-1 block text-sm text-text-primary">
              Min file age before upload (seconds)
            </label>
            <input
              id="video-storage-min-age"
              type="number"
              min={15}
              max={3600}
              step={1}
              value={videoStorageMinFileAgeSec}
              onChange={(e) => setVideoStorageMinFileAgeSec(e.target.value)}
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>

        <p className="text-xs text-text-muted">
          Cloud providers use <code className="rounded bg-base px-1">rclone</code>. Set <code className="rounded bg-base px-1">RCLONE_BIN</code> if the binary is not on PATH.
        </p>
      </div>

      {error && <p className="text-sm text-status-offline" role="alert">{error}</p>}
      {message && <p className="text-sm text-status-online">{message}</p>}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={saving}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save Integrations'}
        </button>
      </div>
    </form>
  )
}

// ─── Main Settings Page ────────────────────────────────────────────────────────

export default function Settings() {
  const [tab, setTab] = useState<Tab>('cameras')
  const [cameras, setCameras] = useState<Camera[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalMode, setModalMode] = useState<CameraModalMode>(null)
  const [editingCamera, setEditingCamera] = useState<Camera | null>(null)
  const [deletingCamera, setDeletingCamera] = useState<Camera | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

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

  const openAdd = useCallback(() => { setFormError(null); setModalMode('add') }, [])
  const openEdit = useCallback((camera: Camera) => { setEditingCamera(camera); setFormError(null); setModalMode('edit') }, [])
  const openDelete = useCallback((camera: Camera) => { setDeletingCamera(camera); setModalMode('delete') }, [])
  const closeModal = useCallback(() => {
    setModalMode(null); setEditingCamera(null); setDeletingCamera(null)
    setFormError(null); setDeleteError(null)
  }, [])

  const handleFormSubmit = useCallback(
    async (values: CreateCameraInput) => {
      setFormError(null)
      setIsSubmitting(true)
      try {
        if (editingCamera) {
          const updated = await updateCamera(editingCamera.id, {
            name: values.name,
            rtsp_url: values.rtsp_url,
            enabled: values.enabled,
            record_enabled: values.record_enabled,
            detection_sample_seconds: values.detection_sample_seconds,
            tracking_enabled: values.tracking_enabled,
            tracking_min_confidence: values.tracking_min_confidence,
            tracking_labels: values.tracking_labels,
            discord_alerts_enabled: values.discord_alerts_enabled,
            discord_webhook_url: values.discord_webhook_url,
            discord_mention: values.discord_mention,
            discord_cooldown_seconds: values.discord_cooldown_seconds,
            discord_trigger_on_detection: values.discord_trigger_on_detection,
            discord_trigger_on_interval: values.discord_trigger_on_interval,
            discord_screenshot_interval_seconds: values.discord_screenshot_interval_seconds,
            discord_include_motion_clip: values.discord_include_motion_clip,
            discord_motion_clip_seconds: values.discord_motion_clip_seconds,
            discord_record_format: values.discord_record_format,
            discord_record_duration_seconds: values.discord_record_duration_seconds,
            discord_detection_provider: values.discord_detection_provider,
            frigate_camera_name: values.frigate_camera_name,
          })
          setCameras((prev) => prev.map((c) => (c.id === updated.id ? updated : c)))
        } else {
          const created = await addCamera(values)
          setCameras((prev) => [...prev, created].sort((a, b) => a.position - b.position))
        }
        closeModal()
      } catch (e) {
        setFormError(e instanceof Error ? e.message : 'Request failed')
      } finally {
        setIsSubmitting(false)
      }
    },
    [editingCamera, closeModal]
  )

  const handleDeleteConfirm = useCallback(async () => {
    if (!deletingCamera) return
    setDeleteError(null)
    try {
      await deleteCamera(deletingCamera.id)
      setCameras((prev) => prev.filter((c) => c.id !== deletingCamera.id))
      closeModal()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Delete failed')
    }
  }, [deletingCamera, closeModal])

  if (loading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <span className="h-10 w-10 animate-spin rounded-full border-2 border-accent border-t-transparent" aria-hidden />
        <span className="sr-only">Loading…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-lg border border-status-offline/50 bg-status-offline/10 px-4 py-6 text-text-primary">
        <p className="font-medium">Could not load cameras</p>
        <p className="mt-1 text-sm text-text-muted">{error}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Tab bar */}
      <div className="flex items-center justify-between gap-4 border-b border-border pb-1">
        <nav className="flex gap-1" aria-label="Settings sections">
          {(['cameras', 'alerts', 'integrations', 'logs'] as Tab[]).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={`rounded-lg px-4 py-2 text-sm font-medium capitalize transition-colors focus:outline-none focus:ring-2 focus:ring-accent ${
                tab === t
                  ? 'bg-accent/10 text-accent'
                  : 'text-text-muted hover:bg-card-hover hover:text-text-primary'
              }`}
            >
              {t === 'alerts' ? 'Alert Rules' : t === 'integrations' ? 'Integrations' : t === 'logs' ? 'Logs' : 'Cameras'}
            </button>
          ))}
        </nav>
        {tab === 'cameras' && (
          <button
            type="button"
            onClick={openAdd}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
          >
            + Add Camera
          </button>
        )}
      </div>

      {/* Cameras tab */}
      {tab === 'cameras' && (
        <>
          {cameras.length === 0 ? (
            <div className="rounded-lg border border-border bg-card px-6 py-12 text-center">
              <p className="text-text-muted">No cameras yet. Add one to get started.</p>
              <button type="button" onClick={openAdd} className="mt-4 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent">
                + Add Camera
              </button>
            </div>
          ) : (
            <ul className="space-y-2">
              {cameras.map((camera) => (
                <li key={camera.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-card px-4 py-3">
                  <div className="min-w-0 flex-1">
                    <p className="font-medium text-text-primary">{camera.name}</p>
                    <p className="truncate text-sm text-text-muted" title={camera.rtsp_url}>{camera.rtsp_url}</p>
                    <div className="mt-1 flex flex-wrap gap-2">
                      <span className={`inline-flex items-center gap-1 text-xs ${camera.enabled ? 'text-status-online' : 'text-text-muted'}`}>
                        <span className={`inline-block h-1.5 w-1.5 rounded-full ${camera.enabled ? 'bg-status-online' : 'bg-text-muted'}`} aria-hidden />
                        {camera.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                      {camera.record_enabled && (
                        <span className="inline-flex items-center gap-1 text-xs text-status-connecting">
                          <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                            <circle cx="12" cy="12" r="3" fill="currentColor" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14" />
                          </svg>
                          Recording
                        </span>
                      )}
                      {camera.tracking_enabled && (
                        <span className="inline-flex items-center gap-1 text-xs text-accent">
                          <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12h2m14 0h2M12 3v2m0 14v2M5.636 5.636l1.414 1.414m9.9 9.9l1.414 1.414m0-12.728l-1.414 1.414m-9.9 9.9l-1.414 1.414M12 8a4 4 0 100 8 4 4 0 000-8z" />
                          </svg>
                          YOLOv8
                        </span>
                      )}
                      {camera.discord_alerts_enabled && (
                        <span className="inline-flex items-center gap-1 text-xs text-status-online">
                          <svg className="h-3 w-3" fill="currentColor" viewBox="0 0 24 24" aria-hidden>
                            <path d="M20.317 4.369A19.791 19.791 0 0015.41 3a13.595 13.595 0 00-.63 1.295 18.256 18.256 0 00-5.557 0A13.595 13.595 0 008.593 3a19.736 19.736 0 00-4.908 1.37C.533 9.066-.32 13.64.099 18.146A19.9 19.9 0 006.13 21a14.487 14.487 0 001.292-2.116 12.885 12.885 0 01-2.035-.97c.171-.125.338-.257.5-.396a14.235 14.235 0 0012.23 0c.164.139.33.271.5.396a12.85 12.85 0 01-2.04.972A14.43 14.43 0 0017.87 21a19.886 19.886 0 006.03-2.854c.5-5.228-.837-9.76-3.583-13.777zM8.02 15.332c-1.184 0-2.155-1.085-2.155-2.418 0-1.333.952-2.418 2.155-2.418 1.206 0 2.178 1.105 2.156 2.418 0 1.333-.95 2.418-2.156 2.418zm7.96 0c-1.184 0-2.156-1.085-2.156-2.418 0-1.333.952-2.418 2.156-2.418 1.205 0 2.177 1.105 2.155 2.418 0 1.333-.95 2.418-2.155 2.418z" />
                          </svg>
                          {camera.discord_detection_provider === 'frigate' ? 'Discord + Frigate' : 'Discord + YOLO'}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <button type="button" onClick={() => openEdit(camera)} className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-text-primary transition-colors hover:bg-card-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base">
                      Edit
                    </button>
                    <button type="button" onClick={() => openDelete(camera)} className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-status-offline transition-colors hover:bg-status-offline/10 focus:outline-none focus:ring-2 focus:ring-status-offline focus:ring-offset-2 focus:ring-offset-base">
                      Delete
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      {/* Alert rules tab */}
      {tab === 'alerts' && <AlertsPanel cameras={cameras} />}

      {/* Integrations tab */}
      {tab === 'integrations' && <IntegrationsPanel />}

      {/* Logs tab */}
      {tab === 'logs' && <LogsPanel />}

      {/* Modals */}
      {modalMode === 'add' && (
        <Modal title="Add Camera" onClose={closeModal} size="xl">
          <CameraForm onSubmit={handleFormSubmit} onCancel={closeModal} submitError={formError} isSubmitting={isSubmitting} />
        </Modal>
      )}
      {modalMode === 'edit' && editingCamera && (
        <Modal title="Edit Camera" onClose={closeModal} size="xl">
          <CameraForm camera={editingCamera} onSubmit={handleFormSubmit} onCancel={closeModal} submitError={formError} isSubmitting={isSubmitting} />
        </Modal>
      )}
      {modalMode === 'delete' && deletingCamera && (
        <Modal onClose={closeModal}>
          <div className="space-y-4">
            {deleteError && <p className="text-sm text-status-offline" role="alert">{deleteError}</p>}
            <ConfirmDialog
              title="Delete camera?"
              message={`"${deletingCamera.name}" will be removed. Stream will stop. This cannot be undone.`}
              confirmLabel="Delete"
              cancelLabel="Cancel"
              variant="danger"
              onConfirm={handleDeleteConfirm}
              onCancel={closeModal}
            />
          </div>
        </Modal>
      )}
    </div>
  )
}
