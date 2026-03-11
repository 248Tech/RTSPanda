import { useState } from 'react'
import type { Camera } from '../api/cameras'
import type { CreateCameraInput } from '../api/cameras'

export interface CameraFormValues {
  name: string
  rtsp_url: string
  enabled: boolean
  record_enabled: boolean
}

export interface CameraFormProps {
  /** When provided, form is in edit mode with prefilled values. */
  camera?: Camera | null
  onSubmit: (values: CreateCameraInput) => void
  onCancel: () => void
  /** Optional error message from parent (e.g. API error). */
  submitError?: string | null
  /** When true, disable submit and show saving state. */
  isSubmitting?: boolean
}

const defaultValues: CameraFormValues = {
  name: '',
  rtsp_url: '',
  enabled: true,
  record_enabled: false,
}

function validate(values: CameraFormValues): { name?: string; rtsp_url?: string } {
  const err: { name?: string; rtsp_url?: string } = {}
  const name = values.name.trim()
  const url = values.rtsp_url.trim()
  if (!name) err.name = 'Name is required'
  if (!url) err.rtsp_url = 'RTSP URL is required'
  else if (!url.startsWith('rtsp://')) err.rtsp_url = 'URL must start with rtsp://'
  return err
}

export function CameraForm({
  camera,
  onSubmit,
  onCancel,
  submitError,
  isSubmitting = false,
}: CameraFormProps) {
  const [values, setValues] = useState<CameraFormValues>(() =>
    camera
      ? { name: camera.name, rtsp_url: camera.rtsp_url, enabled: camera.enabled, record_enabled: camera.record_enabled ?? false }
      : defaultValues
  )
  const [touched, setTouched] = useState<Partial<Record<keyof CameraFormValues, boolean>>>({})
  const errors = validate(values)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setTouched({ name: true, rtsp_url: true })
    if (errors.name || errors.rtsp_url) return
    onSubmit({
      name: values.name.trim(),
      rtsp_url: values.rtsp_url.trim(),
      enabled: values.enabled,
      record_enabled: values.record_enabled,
    })
  }

  const isEdit = Boolean(camera)

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="camera-name" className="mb-1 block text-sm font-medium text-text-primary">
          Name
        </label>
        <input
          id="camera-name"
          type="text"
          value={values.name}
          onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
          onBlur={() => setTouched((t) => ({ ...t, name: true }))}
          placeholder="e.g. Front Door"
          className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          autoFocus
          aria-invalid={touched.name && Boolean(errors.name)}
          aria-describedby={touched.name && errors.name ? 'name-error' : undefined}
        />
        {touched.name && errors.name && (
          <p id="name-error" className="mt-1 text-sm text-status-offline" role="alert">
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
          onChange={(e) => setValues((v) => ({ ...v, rtsp_url: e.target.value }))}
          onBlur={() => setTouched((t) => ({ ...t, rtsp_url: true }))}
          placeholder="rtsp://admin:password@host:554/stream (or other path)"
          className="w-full rounded-lg border border-border bg-base px-3 py-2 text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          aria-invalid={touched.rtsp_url && Boolean(errors.rtsp_url)}
          aria-describedby={touched.rtsp_url && errors.rtsp_url ? 'rtsp-url-error' : undefined}
        />
        {touched.rtsp_url && errors.rtsp_url && (
          <p id="rtsp-url-error" className="mt-1 text-sm text-status-offline" role="alert">
            {errors.rtsp_url}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <input
            id="camera-enabled"
            type="checkbox"
            checked={values.enabled}
            onChange={(e) => setValues((v) => ({ ...v, enabled: e.target.checked }))}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          <label htmlFor="camera-enabled" className="text-sm text-text-primary">
            Enabled
          </label>
        </div>
        <div className="flex items-center gap-2">
          <input
            id="camera-record"
            type="checkbox"
            checked={values.record_enabled}
            onChange={(e) => setValues((v) => ({ ...v, record_enabled: e.target.checked }))}
            className="h-4 w-4 rounded border-border bg-base text-accent focus:ring-accent"
          />
          <label htmlFor="camera-record" className="text-sm text-text-primary">
            Record to disk
          </label>
          <span className="text-xs text-text-muted">(saves 1-hour MP4 segments)</span>
        </div>
      </div>
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
