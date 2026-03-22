export type StreamStatus = 'online' | 'offline' | 'connecting' | 'initializing'

export interface StatusBadgeProps {
  status: StreamStatus
}

const labels: Record<StreamStatus, string> = {
  online: 'Live',
  offline: 'Offline',
  connecting: 'Connecting',
  initializing: 'Initializing',
}

const styles: Record<StreamStatus, string> = {
  online: 'bg-status-online/10 text-status-online ring-1 ring-status-online/20',
  offline: 'bg-status-offline/10 text-status-offline ring-1 ring-status-offline/20',
  connecting: 'bg-status-connecting/10 text-status-connecting ring-1 ring-status-connecting/20',
  initializing: 'bg-status-connecting/10 text-status-connecting ring-1 ring-status-connecting/20',
}

const dotStyles: Record<StreamStatus, string> = {
  online: 'bg-status-online animate-pulse',
  offline: 'bg-status-offline',
  connecting: 'bg-status-connecting animate-pulse',
  initializing: 'bg-status-connecting animate-pulse',
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${styles[status]}`}
      role="status"
      aria-label={`Stream status: ${labels[status]}`}
    >
      <span
        className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${dotStyles[status]}`}
        aria-hidden
      />
      {labels[status]}
    </span>
  )
}
