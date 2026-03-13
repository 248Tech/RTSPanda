import { useMemo, useState } from 'react'
import type { Camera, CreateCameraInput } from '../api/cameras'

export interface CameraFormValues {
  name: string
  rtsp_url: string
  enabled: boolean
  record_enabled: boolean
  tracking_enabled: boolean
  detection_sample_seconds: string
  tracking_min_confidence: string
  tracking_labels: string
  discord_alerts_enabled: boolean
  discord_webhook_url: string
  discord_mention: string
  discord_cooldown_seconds: string
}

export interface CameraFormProps {
  camera?: Camera | null
  onSubmit: (values: CreateCameraInput) => void
  onCancel: () => void
  submitError?: string | null
  isSubmitting?: boolean
}

type FieldName = keyof CameraFormValues
type FieldErrors = Partial<Record<FieldName, string>>

const defaultValues: CameraFormValues = {
  name: '',
  rtsp_url: '',
  enabled: true,
  record_enabled: false,
  tracking_enabled: false,
  detection_sample_seconds: '30',
  tracking_min_confidence: '0.25',
  tracking_labels: '',
  discord_alerts_enabled: false,
  discord_webhook_url: '',
  discord_mention: '',
  discord_cooldown_seconds: '60',
}

function toValues(camera?: Camera | null): CameraFormValues {
  if (!camera) return defaultValues
  return {
    name: camera.name,
    rtsp_url: camera.rtsp_url,
    enabled: camera.enabled,
    record_enabled: camera.record_enabled ?? false,
    tracking_enabled: camera.tracking_enabled ?? false,
    detection_sample_seconds: String(camera.detection_sample_seconds ?? 30),
    tracking_min_confidence: String(camera.tracking_min_confidence ?? 0.25),
    tracking_labels: (camera.tracking_labels ?? []).join(', '),
    discord_alerts_enabled: camera.discord_alerts_enabled ?? false,
    discord_webhook_url: camera.discord_webhook_url ?? '',
    discord_mention: camera.discord_mention ?? '',
    discord_cooldown_seconds: String(camera.discord_cooldown_seconds ?? 60),
  }
}

function validate(values: CameraFormValues): FieldErrors {
  const errors: FieldErrors = {}
  const name = values.name.trim()
  const rtspURL = values.rtsp_url.trim()

  if (!name) errors.name = 'Name is required'
  if (!rtspURL) errors.rtsp_url = 'RTSP URL is required'
  else if (!rtspURL.startsWith('rtsp://')) errors.rtsp_url = 'URL must start with rtsp://'

  if (values.tracking_enabled) {
    const sampleSeconds = Number(values.detection_sample_seconds)
    if (!Number.isInteger(sampleSeconds) || sampleSeconds <= 0) {
      errors.detection_sample_seconds = 'Sample interval must be a whole number greater than 0'
    }

    const confidence = Number(values.tracking_min_confidence)
    if (!Number.isFinite(confidence) || confidence < 0 || confidence > 1) {
      errors.tracking_min_confidence = 'Confidence must be between 0 and 1'
    }
  }

  if (values.discord_alerts_enabled) {
    const webhook = values.discord_webhook_url.trim()
    if (!webhook) {
      errors.discord_webhook_url = 'Discord webhook URL is required when alerts are enabled'
    } else if (!webhook.startsWith('https://')) {
      errors.discord_webhook_url = 'Discord webhook URL must start with https://'
    }

    const cooldown = Number(values.discord_cooldown_seconds)
    if (!Number.isInteger(cooldown) || cooldown < 0) {
      errors.discord_cooldown_seconds = 'Cooldown must be a whole number greater than or equal to 0'
    }
  }

  return errors
}

function hasErrors(errors: FieldErrors): boolean {
  return Object.values(errors).some(Boolean)
}

function parseTrackingLabels(raw: string): string[] {
  return raw
    .split(',')
    .map((label) => label.trim().toLowerCase())
    .filter((label, index, list) => label !== '' && list.indexOf(label) === index)
}

export function CameraForm({
  camera,
  onSubmit,
  onCancel,
  submitError,
  isSubmitting = false,
}: CameraFormProps) {
  const [values, setValues] = useState<CameraFormValues>(() => toValues(camera))
  const [touched, setTouched] = useState<Partial<Record<FieldName, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)

  const errors = useMemo(() => validate(values), [values])

  const setField = <K extends FieldName>(key: K, value: CameraFormValues[K]) => {
    setValues((prev) => ({ ...prev, [key]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitted(true)
    if (hasErrors(errors)) return

    const sampleSeconds = Number(values.detection_sample_seconds)
    const trackingConfidence = Number(values.tracking_min_confidence)
    const discordCooldown = Number(values.discord_cooldown_seconds)

    onSubmit({
      name: values.name.trim(),
      rtsp_url: values.rtsp_url.trim(),
      enabled: values.enabled,
      record_enabled: values.record_enabled,
      tracking_enabled: values.tracking_enabled,
      detection_sample_seconds:
        Number.isInteger(sampleSeconds) && sampleSeconds > 0 ? sampleSeconds : undefined,
      tracking_min_confidence:
        Number.isFinite(trackingConfidence) ? trackingConfidence : undefined,
      tracking_labels: parseTrackingLabels(values.tracking_labels),
      discord_alerts_enabled: values.discord_alerts_enabled,
      discord_webhook_url: values.discord_webhook_url.trim(),
      discord_mention: values.discord_mention.trim(),
      discord_cooldown_seconds:
        Number.isInteger(discordCooldown) && discordCooldown >= 0 ? discordCooldown : undefined,
    })
  }

  const showError = (field: FieldName): boolean => Boolean((submitted || touched[field]) && errors[field])
  const isEdit = Boolean(camera)

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div>
        <label htmlFor="camera-name" className="mb-1 block text-sm font-medium text-text-primary">
          Name
        </label>
        <input
          id="camera-name"
          type="text"
          value={values.name}
          onChange={(e) => setField('name', e.target.value)}
          onBlur={() => setTouched((prev) => ({ ...prev, name: true }))}
          placeholder="e.g. Front Door"
          className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          autoFocus
          aria-invalid={showError('name')}
        />
        {showError('name') && (
          <p className="mt-1 text-sm text-status-offline" role="alert">
            {errors.name}
          </p>
        )}
      </div>

      <div>
        <label htmlFor="camera-rtsp-url" className="mb-1 block text-sm font-medium text-text-primary">
          RTSP URL
        </label>
        <input
          id="camera-rtsp-url"
          type="url"
          value={values.rtsp_url}
          onChange={(e) => setField('rtsp_url', e.target.value)}
          onBlur={() => setTouched((prev) => ({ ...prev, rtsp_url: true }))}
          placeholder="rtsp://admin:password@host:554/stream"
          className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          aria-invalid={showError('rtsp_url')}
        />
        {showError('rtsp_url') && (
          <p className="mt-1 text-sm text-status-offline" role="alert">
            {errors.rtsp_url}
          </p>
        )}
      </div>

      <div className="space-y-2 rounded-lg border border-border bg-card p-3">
        <label className="flex items-center gap-2 text-sm text-text-primary">
          <input
            id="camera-enabled"
            type="checkbox"
            checked={values.enabled}
            onChange={(e) => setField('enabled', e.target.checked)}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          Enabled
        </label>
        <label className="flex items-center gap-2 text-sm text-text-primary">
          <input
            id="camera-record"
            type="checkbox"
            checked={values.record_enabled}
            onChange={(e) => setField('record_enabled', e.target.checked)}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          Record to disk
          <span className="text-xs text-text-muted">(saves 1-hour MP4 segments)</span>
        </label>
      </div>

      <section className="space-y-3 rounded-lg border border-border bg-card p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold text-text-primary">YOLOv8 Tracking</h3>
            <p className="text-xs text-text-muted">Per-camera sampling, filtering, and overlay/history events.</p>
          </div>
          <label className="flex items-center gap-2 text-sm text-text-primary">
            <input
              type="checkbox"
              checked={values.tracking_enabled}
              onChange={(e) => setField('tracking_enabled', e.target.checked)}
              className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
            />
            Enabled
          </label>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label htmlFor="tracking-sample-seconds" className="mb-1 block text-sm text-text-primary">
              Sample interval (seconds)
            </label>
            <input
              id="tracking-sample-seconds"
              type="number"
              min={1}
              step={1}
              value={values.detection_sample_seconds}
              onChange={(e) => setField('detection_sample_seconds', e.target.value)}
              onBlur={() => setTouched((prev) => ({ ...prev, detection_sample_seconds: true }))}
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              aria-invalid={showError('detection_sample_seconds')}
            />
            {showError('detection_sample_seconds') && (
              <p className="mt-1 text-sm text-status-offline" role="alert">
                {errors.detection_sample_seconds}
              </p>
            )}
          </div>
          <div>
            <label htmlFor="tracking-confidence" className="mb-1 block text-sm text-text-primary">
              Minimum confidence (0-1)
            </label>
            <input
              id="tracking-confidence"
              type="number"
              min={0}
              max={1}
              step={0.01}
              value={values.tracking_min_confidence}
              onChange={(e) => setField('tracking_min_confidence', e.target.value)}
              onBlur={() => setTouched((prev) => ({ ...prev, tracking_min_confidence: true }))}
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              aria-invalid={showError('tracking_min_confidence')}
            />
            {showError('tracking_min_confidence') && (
              <p className="mt-1 text-sm text-status-offline" role="alert">
                {errors.tracking_min_confidence}
              </p>
            )}
          </div>
        </div>

        <div>
          <label htmlFor="tracking-labels" className="mb-1 block text-sm text-text-primary">
            Track labels (comma-separated, optional)
          </label>
          <input
            id="tracking-labels"
            type="text"
            value={values.tracking_labels}
            onChange={(e) => setField('tracking_labels', e.target.value)}
            placeholder="person, car, dog"
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
          <p className="mt-1 text-xs text-text-muted">Leave empty to keep all labels from the model.</p>
        </div>
      </section>

      <section className="space-y-3 rounded-lg border border-border bg-card p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold text-text-primary">Discord Rich Alerts</h3>
            <p className="text-xs text-text-muted">Attach snapshot media and detection details to webhook events.</p>
          </div>
          <label className="flex items-center gap-2 text-sm text-text-primary">
            <input
              type="checkbox"
              checked={values.discord_alerts_enabled}
              onChange={(e) => setField('discord_alerts_enabled', e.target.checked)}
              className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
            />
            Enabled
          </label>
        </div>

        <div>
          <label htmlFor="discord-webhook-url" className="mb-1 block text-sm text-text-primary">
            Webhook URL
          </label>
          <input
            id="discord-webhook-url"
            type="url"
            value={values.discord_webhook_url}
            onChange={(e) => setField('discord_webhook_url', e.target.value)}
            onBlur={() => setTouched((prev) => ({ ...prev, discord_webhook_url: true }))}
            placeholder="https://discord.com/api/webhooks/..."
            className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            aria-invalid={showError('discord_webhook_url')}
          />
          {showError('discord_webhook_url') && (
            <p className="mt-1 text-sm text-status-offline" role="alert">
              {errors.discord_webhook_url}
            </p>
          )}
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label htmlFor="discord-mention" className="mb-1 block text-sm text-text-primary">
              Mention (optional)
            </label>
            <input
              id="discord-mention"
              type="text"
              value={values.discord_mention}
              onChange={(e) => setField('discord_mention', e.target.value)}
              placeholder="@here or <@123456789>"
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label htmlFor="discord-cooldown-seconds" className="mb-1 block text-sm text-text-primary">
              Cooldown (seconds)
            </label>
            <input
              id="discord-cooldown-seconds"
              type="number"
              min={0}
              step={1}
              value={values.discord_cooldown_seconds}
              onChange={(e) => setField('discord_cooldown_seconds', e.target.value)}
              onBlur={() => setTouched((prev) => ({ ...prev, discord_cooldown_seconds: true }))}
              className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              aria-invalid={showError('discord_cooldown_seconds')}
            />
            {showError('discord_cooldown_seconds') && (
              <p className="mt-1 text-sm text-status-offline" role="alert">
                {errors.discord_cooldown_seconds}
              </p>
            )}
          </div>
        </div>
      </section>

      {submitError && (
        <p className="text-sm text-status-offline" role="alert">
          {submitError}
        </p>
      )}

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
          {isSubmitting ? 'Saving…' : isEdit ? 'Save' : 'Add Camera'}
        </button>
      </div>
    </form>
  )
}
