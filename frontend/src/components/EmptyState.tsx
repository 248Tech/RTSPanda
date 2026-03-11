export interface EmptyStateProps {
  onAddCamera: () => void
}

export function EmptyState({ onAddCamera }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-card px-6 py-16 text-center">
      <span className="mb-4 text-5xl" aria-hidden>
        📷
      </span>
      <h2 className="text-xl font-semibold text-text-primary">
        No cameras configured
      </h2>
      <p className="mt-2 max-w-sm text-text-muted">
        Add your first camera to get started.
      </p>
      <button
        type="button"
        onClick={onAddCamera}
        className="mt-6 rounded-lg bg-accent px-4 py-2.5 font-medium text-white transition-colors hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-base"
      >
        + Add Camera
      </button>
    </div>
  )
}
