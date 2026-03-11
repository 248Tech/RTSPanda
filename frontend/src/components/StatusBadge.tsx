export type StreamStatus = 'online' | 'offline' | 'connecting'

export interface StatusBadgeProps {
  status: StreamStatus
}

const labels: Record<StreamStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  connecting: 'Connecting',
}

const dotColors: Record<StreamStatus, string> = {
  online: 'bg-status-online',
  offline: 'bg-status-offline',
  connecting: 'bg-status-connecting animate-pulse',
}

const labelColors: Record<StreamStatus, string> = {
  online: 'text-status-online',
  offline: 'text-status-offline',
  connecting: 'text-status-connecting',
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 text-xs font-medium ${labelColors[status]}`}
      role="status"
      aria-label={`Stream status: ${labels[status]}`}
    >
      <span
        className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${dotColors[status]}`}
        aria-hidden
      />
      {labels[status]}
    </span>
  )
}
